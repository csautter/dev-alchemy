package deploy

import (
	"bufio"
	"context"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestPrintSystemOsArch(t *testing.T) {
	t.Logf("Running on OS: %s, ARCH: %s", runtime.GOOS, runtime.GOARCH)
}

func RunUtmUbuntuDeployOnMacOS(t *testing.T, config VirtualMachineConfig) {
	scriptPath := "./deployments/utm/create-utm-vm.sh"
	args := []string{"--arch", config.Arch, "--os", config.OS + "-" + config.UbuntuType}

	// Set a timeout for the script execution (adjust as needed)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", append([]string{scriptPath}, args...)...)
	cmd.Dir = "../../"

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

func TestDeployUtmUbuntuServerArm64OnMacos(t *testing.T) {

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "server",
	}
	RunUtmUbuntuDeployOnMacOS(t, VirtualMachineConfig)
}

func TestDeployUtmUbuntuServerAmd64OnMacos(t *testing.T) {

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "server",
	}
	RunUtmUbuntuDeployOnMacOS(t, VirtualMachineConfig)
}

func TestDeployUtmUbuntuDesktopArm64OnMacos(t *testing.T) {

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "desktop",
	}
	RunUtmUbuntuDeployOnMacOS(t, VirtualMachineConfig)
}

func TestDeployUtmUbuntuDesktopAmd64OnMacos(t *testing.T) {
	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "desktop",
	}
	RunUtmUbuntuDeployOnMacOS(t, VirtualMachineConfig)
}
