package build

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func RunBuildScript(config VirtualMachineConfig, executable string, args []string) error {
	// Ensure that the build artifact does not already exist

	// if no ExpectedBuildArtifacts are provided, use the defaults for the given config
	build_artifact_exists, err := checkIfBuildArtifactsExist(config)
	if err != nil {
		return err
	}
	if build_artifact_exists {
		return nil
	}

	// Ensure all required dependencies are present
	DependencyReconciliation(config)

	// Check if VNC port is free, if not, increment until a free port is found
	_ = getFreeVncPort(&config)

	// Set a timeout for the script execution (adjust as needed)
	timeout := 240 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(sigs)

	printCurrentWorkingDirectory()

	fmt.Printf("Running Build with executable %s and args %v\n", executable, args)
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = GetDirectoriesInstance().GetDirectories().ProjectDir

	// VNC integration:
	// - Opening a VNC viewer (Screen Sharing) is useful for observing the VM build process in real time.
	// - VNC recording enables capturing the build process for later review or debugging.
	vnc_recording_config := VncRecordingConfig{Password: "packer"}
	go func() {
		config := RunProcessConfig{
			ExecutablePath:   "open",
			Args:             []string{"-a", "Screen Sharing", fmt.Sprintf("vnc://:%s@localhost:%d", vnc_recording_config.Password, config.VncPort)},
			Timeout:          5 * time.Minute,
			WorkingDir:       "",
			Context:          ctx,
			FailOnError:      false,
			Retries:          5,
			RetryInterval:    time.Minute,
			DelayBeforeStart: time.Minute,
		}
		RunExternalProcessWithRetries(config)
	}()

	// Start Screen Capture to record the VM build process
	var vnc_snapshot_ctx context.Context
	vnc_snapshot_done := make(chan struct{})
	vnc_interrupt_retry_chan := make(chan bool)
	go func() {
		vnc_snapshot_ctx = RunVncSnapshotProcess(config, ctx, RunProcessConfig{Timeout: timeout, Retries: 30, InterruptRetryChan: vnc_interrupt_retry_chan, RetryInterval: 10 * time.Second}, &vnc_recording_config)
		if vnc_snapshot_ctx != nil {
			<-vnc_snapshot_ctx.Done()
		}
		close(vnc_snapshot_done)
	}()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to get stderr: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start command: %v", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Printf("%s:%s stdout:  %s", config.UbuntuType, config.Arch, scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("%s:%s stderr:  %s", config.UbuntuType, config.Arch, scanner.Text())
		}
	}()

	done := make(chan error, 1)

	go func() {
		err := cmd.Wait()
		if vnc_snapshot_ctx != nil {
			vnc_snapshot_ctx.Done()
		}
		done <- err
	}()

	// FFmpeg integration:
	// - FFmpeg is useful for generating a video from the VNC recording, allowing playback and sharing of the build process.
	var ffmpeg_run = func() {
		// Wait for vnc_snapshot to finish
		_, ok := <-vnc_snapshot_done
		if !ok {
			// Channel is closed, proceed
		} else {
			// Channel not closed, wait for it
			<-vnc_snapshot_done
		}
		// Always run ffmpeg after vnc_snapshot is done
		timeout := 10 * time.Minute
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		RunFfmpegVideoGenerationProcess(config, ctx, RunProcessConfig{Timeout: timeout}, &vnc_recording_config)
	}

	select {
	case err := <-done:
		vnc_interrupt_retry_chan <- true
		if err != nil {
			RemoveBuildArtifactsForConfig(config)
			ffmpeg_run()
			log.Printf("Script failed: %v", err)
			return err
		}
		ffmpeg_run()
		log.Printf("Script finished successfully.")
	case <-ctx.Done():
		// Kill the process if context is done (timeout or cancellation)
		_ = cmd.Process.Kill()
		vnc_interrupt_retry_chan <- true
		RemoveBuildArtifactsForConfig(config)
		ffmpeg_run()
		log.Printf("Script terminated due to timeout or interruption: %v", ctx.Err())
		return ctx.Err()
	case sig := <-sigs:
		_ = cmd.Process.Kill()
		vnc_interrupt_retry_chan <- true
		RemoveBuildArtifactsForConfig(config)
		ffmpeg_run()
		log.Printf("Script terminated due to signal: %v", sig)
		return fmt.Errorf("script terminated due to signal: %v", sig)
	}

	return nil

	// TODO: check for vnc recording files and video generation success
}

func printCurrentWorkingDirectory() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	log.Printf("Current working directory: %s", cwd)
}

func getFreeVncPort(config *VirtualMachineConfig) int {
	port := config.VncPort
	for {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			break
		}
		port++
	}
	config.VncPort = port
	log.Printf("Using VNC port: %d", config.VncPort)
	return port
}

func checkIfBuildArtifactsExist(config VirtualMachineConfig) (bool, error) {
	if len(config.ExpectedBuildArtifacts) == 0 {
		for _, vm := range AvailableVirtualMachineConfigs() {
			if string(vm.HostOs) == string(config.HostOs) && vm.OS == config.OS && vm.UbuntuType == config.UbuntuType && vm.Arch == config.Arch {
				config.ExpectedBuildArtifacts = vm.ExpectedBuildArtifacts
				break
			}
		}
	}

	if len(config.ExpectedBuildArtifacts) == 0 {
		log.Printf("No build artifacts defined. Aborting build.")
		return false, errors.New("no build artifacts defined for the given configuration")
	}

	if len(config.ExpectedBuildArtifacts) > 0 {
		artifacts_exist := true
		for _, artifact := range config.ExpectedBuildArtifacts {
			log.Printf("Checking artifact: %s", artifact)
			if _, err := os.Stat(artifact); os.IsNotExist(err) {
				artifacts_exist = false
				log.Printf("Expected build artifact does not exist: %s", artifact)
				log.Printf("Proceeding with build...")
				break
			}
		}
		if artifacts_exist {
			log.Printf("Build artifacts already exist, skipping build: %v", config.ExpectedBuildArtifacts)
			return true, nil
		}
	}
	return false, nil
}

func RemoveBuildArtifacts(artifacts []string) {
	for _, artifact := range artifacts {
		if _, err := os.Stat(artifact); err == nil {
			log.Printf("Removing existing build artifact: %s", artifact)
			if err := os.RemoveAll(artifact); err != nil {
				log.Fatalf("Failed to remove build artifact %s: %v", artifact, err)
			}
		}
	}
}

func RemoveBuildArtifactsForConfig(config VirtualMachineConfig) {
	RemoveBuildArtifacts(config.ExpectedBuildArtifacts)
}
