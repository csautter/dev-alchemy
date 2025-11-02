package build

import (
	"bufio"
	"os/exec"
	"runtime"
	"testing"
)



func TestPrintSystemOsArch(t *testing.T) {
	t.Logf("Running on OS: %s, ARCH: %s", runtime.GOOS, runtime.GOARCH)
}

func TestRunLinuxUbuntuOnMacOS(t *testing.T) {
	scriptPath := "../../build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh"
	args := []string{"--arch", "arm64", "--ubuntu-type", "server"}
	cmd := exec.Command("bash", append([]string{scriptPath}, args...)...)

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
			t.Logf("stdout: %s", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			t.Logf("stderr: %s", scanner.Text())
		}
	}()

	err = cmd.Wait()
	if err != nil {
		t.Fatalf("Script failed: %v", err)
	}
	t.Logf("Script finished successfully.")
}