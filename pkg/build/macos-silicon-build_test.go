package build

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestPrintSystemOsArch(t *testing.T) {
	t.Logf("Running on OS: %s, ARCH: %s", runtime.GOOS, runtime.GOARCH)
}

func RunQemuUbuntuBuildOnMacOS(t *testing.T, config VirtualMachineConfig) {
	scriptPath := "./build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh"
	args := []string{"--arch", config.Arch, "--ubuntu-type", config.UbuntuType, "--vnc-port", fmt.Sprintf("%d", config.VncPort), "--headless"}
	RunBuildScript(t, config, scriptPath, args)
}

func RunQemuWindowsBuildOnMacOS(t *testing.T, config VirtualMachineConfig) {
	scriptPath := "./build/packer/windows/windows11-on-macos.sh"
	args := []string{"--arch", config.Arch, "--vnc-port", fmt.Sprintf("%d", config.VncPort), "--headless"}
	RunBuildScript(t, config, scriptPath, args)
}

func RunBuildScript(t *testing.T, config VirtualMachineConfig, scriptPath string, args []string) {
	// Set a timeout for the script execution (adjust as needed)
	timeout := 120 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(sigs)

	cmd := exec.CommandContext(ctx, "bash", append([]string{scriptPath}, args...)...)
	cmd.Dir = "../../"

	// Start Screen Sharing to monitor the VM build process via VNC
	go func() {
		config := RunProcessConfig{
			ExecutablePath: "open",
			Args:           []string{"-a", "Screen Sharing", fmt.Sprintf("vnc://localhost:%d", config.VncPort)},
			Timeout:        5 * time.Minute,
			WorkingDir:     "",
			Context:        ctx,
			FailOnError:    false,
			Retries:        5,
			RetryInterval:  60 * time.Second,
		}
		RunExternalProcessWithRetries(config)
	}()

	// Start Screen Capture to record the VM build process
	vnc_recording_config := VncRecordingConfig{}
	var vnc_snapshot_ctx context.Context
	var ffmpeg_ctx context.Context
	vnc_snapshot_done := make(chan struct{})
	vnc_interrupt_retry_chan := make(chan bool)
	go func() {
		vnc_snapshot_ctx = RunVncSnapshotProcess(config, ctx, RunProcessConfig{Timeout: timeout, InterruptRetryChan: vnc_interrupt_retry_chan}, &vnc_recording_config)
		if vnc_snapshot_ctx != nil {
			<-vnc_snapshot_ctx.Done()
		}
		close(vnc_snapshot_done)
	}()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to get stderr: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			t.Logf("%s:%s stdout:  %s", config.UbuntuType, config.Arch, scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			t.Logf("%s:%s stderr:  %s", config.UbuntuType, config.Arch, scanner.Text())
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

	var ffmpeg_run = func() {
		// Wait for vnc_snapshot to finish
		<-vnc_snapshot_done
		// Always run ffmpeg after vnc_snapshot is done
		ffmpeg_ctx = RunFfmpegVideoGenerationProcess(config, ctx, RunProcessConfig{Timeout: 10 * time.Minute}, &vnc_recording_config)
		if ffmpeg_ctx != nil {
			<-ffmpeg_ctx.Done()
		}
	}

	select {
	case err := <-done:
		vnc_interrupt_retry_chan <- true
		if err != nil {
			ffmpeg_run()
			t.Fatalf("Script failed: %v", err)
		}
		ffmpeg_run()
		t.Logf("Script finished successfully.")
	case <-ctx.Done():
		// Kill the process if context is done (timeout or cancellation)
		_ = cmd.Process.Kill()
		vnc_interrupt_retry_chan <- true
		ffmpeg_run()
		t.Fatalf("Script terminated due to timeout or interruption: %v", ctx.Err())
	case sig := <-sigs:
		_ = cmd.Process.Kill()
		vnc_interrupt_retry_chan <- true
		ffmpeg_run()
		t.Fatalf("Script terminated due to signal: %v", sig)
	}
}

func TestBuildQemuUbuntuServerArm64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "server",
		VncPort:    5901,
	}
	RunQemuUbuntuBuildOnMacOS(t, VirtualMachineConfig)
}

func TestBuildQemuUbuntuServerAmd64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "server",
		VncPort:    5902,
	}
	RunQemuUbuntuBuildOnMacOS(t, VirtualMachineConfig)
}

func TestBuildQemuUbuntuDesktopArm64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "desktop",
		VncPort:    5903,
	}
	RunQemuUbuntuBuildOnMacOS(t, VirtualMachineConfig)
}

func TestBuildQemuUbuntuDesktopAmd64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "desktop",
		VncPort:    5904,
	}
	RunQemuUbuntuBuildOnMacOS(t, VirtualMachineConfig)
}

func TestBuildQemuWindows11Arm64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:      "windows11",
		Arch:    "arm64",
		VncPort: 5911,
	}
	RunQemuWindowsBuildOnMacOS(t, VirtualMachineConfig)
}

func TestBuildQemuWindows11Amd64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:      "windows11",
		Arch:    "amd64",
		VncPort: 5912,
	}
	RunQemuWindowsBuildOnMacOS(t, VirtualMachineConfig)
}
