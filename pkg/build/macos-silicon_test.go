package build

import (
	"bufio"
	"context"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

type VirtualMachineConfig struct {
	OS         string
	Arch       string
	UbuntuType string
}

func TestPrintSystemOsArch(t *testing.T) {
	t.Logf("Running on OS: %s, ARCH: %s", runtime.GOOS, runtime.GOARCH)
}

func RunQemuUbuntuBuildOnMacOS(t *testing.T, config VirtualMachineConfig) {
	scriptPath := "../../build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh"
	args := []string{"--arch", config.Arch, "--ubuntu-type", config.UbuntuType}

	// Set a timeout for the script execution (adjust as needed)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", append([]string{scriptPath}, args...)...)

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
			t.Logf("stdout: %s", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			t.Logf("stderr: %s", scanner.Text())
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
	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "debian",
		Arch:       "arm64",
		UbuntuType: "server",
	}
	RunQemuUbuntuBuildOnMacOS(t, VirtualMachineConfig)
}

func TestBuildQemuUbuntuServerAmd64OnMacos(t *testing.T) {
	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "debian",
		Arch:       "amd64",
		UbuntuType: "server",
	}
	RunQemuUbuntuBuildOnMacOS(t, VirtualMachineConfig)
}

func TestBuildQemuUbuntuDesktopArm64OnMacos(t *testing.T) {
	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "debian",
		Arch:       "arm64",
		UbuntuType: "desktop",
	}
	RunQemuUbuntuBuildOnMacOS(t, VirtualMachineConfig)
}

func TestBuildQemuUbuntuDesktopAmd64OnMacos(t *testing.T) {
	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "debian",
		Arch:       "amd64",
		UbuntuType: "desktop",
	}
	RunQemuUbuntuBuildOnMacOS(t, VirtualMachineConfig)
}
