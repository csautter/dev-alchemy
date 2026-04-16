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
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

func RunBuildScript(config VirtualMachineConfig, executable string, args []string) error {
	skipBuild, cleanupArtifacts, err := prepareBuildArtifactsForBuild(config)
	if err != nil {
		return err
	}
	if skipBuild {
		return nil
	}
	buildSucceeded := false
	defer cleanupArtifacts(buildSucceeded)

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

	fmt.Printf("Running Build with executable %s and args %v\n", executable, sanitizeCommandArgs(args))
	// #nosec G204 -- executable and args are constructed by internal build flows; no shell is invoked.
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = GetDirectoriesInstance().GetDirectories().ProjectDir
	cmd.Env = append(os.Environ(), GetDirectoriesInstance().ManagedEnv()...)
	if config.Verbose {
		cmd.Env = append(cmd.Env, "PACKER_LOG=1")
	}

	readAndPrintStdoutStderr(cmd, config)

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start command: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		log.Printf("Waiting for command to finish...")
		err := cmd.Wait()
		log.Printf("Command finished.")
		done <- err
	}()

	vnc_recording_config := VncRecordingConfig{Password: "packer"}
	openVncViewerOnMacosDarwin(ctx, config, vnc_recording_config)

	// Start Screen Capture to record the VM build process
	vnc_snapshot_done := make(chan struct{})
	vnc_interrupt_retry_chan := make(chan bool)

	startVncScreenCaptureOnMacosDarwin(ctx, config, timeout, vnc_interrupt_retry_chan, &vnc_recording_config, vnc_snapshot_done)

	select {
	case err := <-done:
		stopVncScreenCaptureOnMacosDarwin(vnc_interrupt_retry_chan)

		if err != nil {
			runFfmpegOnMacosDarwin(vnc_snapshot_done, config, &vnc_recording_config)
			log.Printf("Script failed: %v", err)
			return err
		}
		buildSucceeded = true
		runFfmpegOnMacosDarwin(vnc_snapshot_done, config, &vnc_recording_config)
		log.Printf("Script finished successfully.")
	case <-ctx.Done():
		// Kill the process if context is done (timeout or cancellation)
		_ = cmd.Process.Kill()

		stopVncScreenCaptureOnMacosDarwin(vnc_interrupt_retry_chan)

		runFfmpegOnMacosDarwin(vnc_snapshot_done, config, &vnc_recording_config)
		log.Printf("Script terminated due to timeout or interruption: %v", ctx.Err())
		return ctx.Err()
	case sig := <-sigs:
		_ = cmd.Process.Kill()

		stopVncScreenCaptureOnMacosDarwin(vnc_interrupt_retry_chan)

		runFfmpegOnMacosDarwin(vnc_snapshot_done, config, &vnc_recording_config)
		log.Printf("Script terminated due to signal: %v", sig)
		return fmt.Errorf("script terminated due to signal: %v", sig)
	}

	return nil
}

func resolveExpectedBuildArtifacts(config VirtualMachineConfig) ([]string, error) {
	if len(config.ExpectedBuildArtifacts) > 0 {
		return config.ExpectedBuildArtifacts, nil
	}

	for _, vm := range AvailableVirtualMachineConfigs() {
		if string(vm.HostOs) == string(config.HostOs) && vm.OS == config.OS && vm.UbuntuType == config.UbuntuType && vm.Arch == config.Arch && string(vm.VirtualizationEngine) == string(config.VirtualizationEngine) {
			return vm.ExpectedBuildArtifacts, nil
		}
	}

	return nil, errors.New("no build artifacts defined for the given configuration")
}

func stopVncScreenCaptureOnMacosDarwin(vnc_interrupt_retry_chan chan bool) {
	if runtime.GOOS != "darwin" {
		return
	}
	log.Printf("stopping VNC snapshot...")
	select {
	case vnc_interrupt_retry_chan <- true:
	default:
		// VNC goroutine already exited (e.g. vncsnapshot finished successfully),
		// so there is no receiver on the channel. Skip the send to avoid deadlock.
		log.Printf("VNC snapshot already stopped, skipping interrupt signal.")
	}
	log.Printf("VNC snapshot stopped.")
}

// FFmpeg integration:
// - FFmpeg is useful for generating a video from the VNC recording, allowing playback and sharing of the build process.
func runFfmpegOnMacosDarwin(vnc_snapshot_done chan struct{}, config VirtualMachineConfig, vnc_recording_config *VncRecordingConfig) {
	if runtime.GOOS != "darwin" {
		return
	}
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
	RunFfmpegVideoGenerationProcess(config, ctx, RunProcessConfig{Timeout: timeout}, vnc_recording_config)
}

func startVncScreenCaptureOnMacosDarwin(ctx context.Context, config VirtualMachineConfig, timeout time.Duration, vnc_interrupt_retry_chan chan bool, vnc_recording_config *VncRecordingConfig, vnc_snapshot_done chan struct{}) {
	if runtime.GOOS != "darwin" {
		return
	}
	go func() {
		vnc_snapshot_ctx := RunVncSnapshotProcess(config, ctx, RunProcessConfig{Timeout: timeout, Retries: 30, InterruptRetryChan: vnc_interrupt_retry_chan, RetryInterval: 10 * time.Second}, vnc_recording_config)
		if vnc_snapshot_ctx != nil {
			<-vnc_snapshot_ctx.Done()
		}
		close(vnc_snapshot_done)
	}()
}

// VNC integration:
// - Opening a VNC viewer (Screen Sharing) is useful for observing the VM build process in real time.
// - VNC recording enables capturing the build process for later review or debugging.
func openVncViewerOnMacosDarwin(ctx context.Context, config VirtualMachineConfig, vnc_recording_config VncRecordingConfig) {
	if runtime.GOOS != "darwin" {
		return
	}
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
}

func readAndPrintStdoutStderr(cmd *exec.Cmd, config VirtualMachineConfig) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to get stderr: %v", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Printf("%s:%s:%s stdout:  %s", config.OS, config.UbuntuType, config.Arch, sanitizeSensitiveText(scanner.Text()))
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("%s:%s:%s stderr:  %s", config.OS, config.UbuntuType, config.Arch, sanitizeSensitiveText(scanner.Text()))
		}
	}()
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
			if closeErr := ln.Close(); closeErr != nil {
				log.Printf("Failed to release test listener on %s: %v", addr, closeErr)
			}
			break
		}
		port++
	}
	config.VncPort = port
	log.Printf("Using VNC port: %d", config.VncPort)
	return port
}

func checkIfBuildArtifactsExist(config VirtualMachineConfig) (bool, error) {
	return BuildArtifactsExist(config)
}

type buildArtifactBackup struct {
	originalPath string
	backupPath   string
}

func prepareBuildArtifactsForBuild(config VirtualMachineConfig) (bool, func(bool), error) {
	if !config.NoCache {
		buildArtifactExists, err := checkIfBuildArtifactsExist(config)
		if err != nil {
			return false, nil, err
		}
		if buildArtifactExists {
			return true, func(bool) {}, nil
		}
		return false, func(bool) {}, nil
	}

	artifacts, err := resolveExpectedBuildArtifacts(config)
	if err != nil {
		return false, nil, err
	}

	backups, err := backupBuildArtifacts(artifacts)
	if err != nil {
		return false, nil, err
	}

	cleanup := func(success bool) {
		if success {
			removeBackedUpArtifacts(backups)
			return
		}
		RemoveBuildArtifacts(artifacts)
		restoreBackedUpArtifacts(backups)
	}

	return false, cleanup, nil
}

func BuildArtifactsExist(config VirtualMachineConfig) (bool, error) {
	return buildArtifactsExist(config, true)
}

func BuildArtifactsExistQuiet(config VirtualMachineConfig) (bool, error) {
	return buildArtifactsExist(config, false)
}

func buildArtifactsExist(config VirtualMachineConfig, verbose bool) (bool, error) {
	artifacts, err := resolveExpectedBuildArtifacts(config)
	if err != nil {
		if verbose {
			log.Printf("No build artifacts defined. Aborting build.")
		}
		return false, err
	}

	artifacts_exist := true
	for _, artifact := range artifacts {
		if verbose {
			log.Printf("Checking artifact: %s", artifact)
		}
		if _, err := os.Stat(artifact); os.IsNotExist(err) {
			artifacts_exist = false
			if verbose {
				log.Printf("Expected build artifact does not exist: %s", artifact)
				log.Printf("Proceeding with build...")
			}
			break
		}
		if err != nil {
			return false, err
		}
	}
	if artifacts_exist {
		if verbose {
			log.Printf("Build artifacts already exist, skipping build: %v", artifacts)
		}
		return true, nil
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
	artifacts, err := resolveExpectedBuildArtifacts(config)
	if err != nil {
		log.Printf("Failed to resolve build artifacts for cleanup: %v", err)
		return
	}
	RemoveBuildArtifacts(artifacts)
}

func backupBuildArtifacts(artifacts []string) ([]buildArtifactBackup, error) {
	backups := make([]buildArtifactBackup, 0, len(artifacts))
	for _, artifact := range artifacts {
		if _, err := os.Stat(artifact); os.IsNotExist(err) {
			continue
		} else if err != nil {
			restoreBackedUpArtifacts(backups)
			return nil, err
		}

		backupPath := fmt.Sprintf("%s.dev-alchemy-backup-%d", artifact, time.Now().UnixNano())
		if err := os.Rename(artifact, backupPath); err != nil {
			restoreBackedUpArtifacts(backups)
			return nil, fmt.Errorf("failed to back up build artifact %s: %w", artifact, err)
		}

		log.Printf("Backed up existing build artifact: %s -> %s", artifact, backupPath)
		backups = append(backups, buildArtifactBackup{
			originalPath: artifact,
			backupPath:   backupPath,
		})
	}

	return backups, nil
}

func restoreBackedUpArtifacts(backups []buildArtifactBackup) {
	for _, backup := range backups {
		if _, err := os.Stat(backup.backupPath); os.IsNotExist(err) {
			continue
		} else if err != nil {
			log.Printf("Failed to stat backup artifact %s: %v", backup.backupPath, err)
			continue
		}

		// #nosec G301 -- restored cache artifact directories must remain traversable for non-root CI steps after sudo-created builds.
		if err := os.MkdirAll(filepath.Dir(backup.originalPath), 0755); err != nil {
			log.Printf("Failed to recreate artifact directory for %s: %v", backup.originalPath, err)
			continue
		}

		if err := os.Rename(backup.backupPath, backup.originalPath); err != nil {
			log.Printf("Failed to restore backup artifact %s: %v", backup.originalPath, err)
			continue
		}
		log.Printf("Restored build artifact backup: %s", backup.originalPath)
	}
}

func removeBackedUpArtifacts(backups []buildArtifactBackup) {
	for _, backup := range backups {
		if _, err := os.Stat(backup.backupPath); os.IsNotExist(err) {
			continue
		} else if err != nil {
			log.Printf("Failed to stat backup artifact %s: %v", backup.backupPath, err)
			continue
		}

		if err := os.RemoveAll(backup.backupPath); err != nil {
			log.Printf("Failed to remove backup artifact %s: %v", backup.backupPath, err)
			continue
		}
		log.Printf("Removed build artifact backup: %s", backup.backupPath)
	}
}
