package build

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestPrintSystemOsArch(t *testing.T) {
	t.Logf("Running on OS: %s, ARCH: %s", runtime.GOOS, runtime.GOARCH)
}

func RunQemuUbuntuBuildOnMacOS(t *testing.T, config VirtualMachineConfig) {
	scriptPath := "./build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh"
	args := []string{"--arch", config.Arch, "--ubuntu-type", config.UbuntuType, "--vnc-port", fmt.Sprintf("%d", config.VncPort), "--headless"}
	RunBashScript(t, config, scriptPath, args)
}

func RunQemuWindowsBuildOnMacOS(t *testing.T, config VirtualMachineConfig) {
	scriptPath := "./build/packer/windows/windows11-on-macos.sh"
	args := []string{"--arch", config.Arch, "--vnc-port", fmt.Sprintf("%d", config.VncPort), "--headless"}
	RunBashScript(t, config, scriptPath, args)
}

func RunBashScript(t *testing.T, config VirtualMachineConfig, scriptPath string, args []string) {
	// Set a timeout for the script execution (adjust as needed)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", append([]string{scriptPath}, args...)...)
	cmd.Dir = "../../"

	go func() {
		time.Sleep(3 * time.Minute) // Wait for 3 minutes before starting Screen Sharing
		// Retry starting Screen Sharing for up to 5 minutes, every 20 seconds
		screenSharingStarted := false
		startTime := time.Now()
		// Retry starting Screen Sharing for up to 5 minutes, every 60 seconds
		// This is to handle cases where the VNC server might take time to become available
		// Unfortunately, there's no direct way to check if the VNC client can connect, so we rely on retries
		for time.Since(startTime) < 5*time.Minute {
			cmdVNC := exec.CommandContext(ctx, "open", "-a", "Screen Sharing", fmt.Sprintf("vnc://localhost:%d", config.VncPort))
			if err := cmdVNC.Start(); err != nil {
				t.Logf("Failed to start Screen Sharing: %v. Retrying in 60s...", err)
				time.Sleep(60 * time.Second)
			} else {
				t.Logf("Started Screen Sharing on port %d", config.VncPort)
				defer cmdVNC.Process.Kill()
				screenSharingStarted = true
			}
		}
		if !screenSharingStarted {
			t.Logf("Could not start Screen Sharing after 5 minutes of retries.")
		}
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

	done := make(chan error, 1)

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

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Script failed: %v", err)
		}
		t.Logf("Script finished successfully.")
	case <-ctx.Done():
		// Kill the process if context is done (timeout or cancellation)
		_ = cmd.Process.Kill()
		t.Fatalf("Script terminated due to timeout or interruption: %v", ctx.Err())
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
