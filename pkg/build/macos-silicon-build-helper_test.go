package build

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestMacOsDownloadArm64Uefi(t *testing.T) {
	t.Parallel()

	scriptPath := "scripts/macos/download-arm64-uefi.sh"
	err := RunMacOsSiliconBuildHelperScript(t, scriptPath)
	if err != nil {
		t.Fatalf("Failed to run %s: %v", scriptPath, err)
	}
}

func RunMacOsSiliconBuildHelperScript(t *testing.T, scriptPath string, args ...string) error {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping MacOS Silicon build helper script test on non-MacOS host")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	cmd := exec.CommandContext(ctx, "bash", append([]string{scriptPath}, args...)...)
	cmd.Dir = "../../"

	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	output, err := cmd.CombinedOutput()
	t.Logf("Output of %s:\n%s", scriptPath, string(output))
	return err
}
