package build

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestMacOsDownloadArm64Uefi(t *testing.T) {
	t.Parallel()

	// Remove files matching vendor/qemu-efi*
	matches, err := filepath.Glob("../../vendor/qemu-efi*")
	if err != nil {
		t.Fatalf("Failed to glob ../../vendor/qemu-efi*: %v", err)
	}
	for _, file := range matches {
		if err := os.RemoveAll(file); err != nil {
			t.Fatalf("Failed to remove %s: %v", file, err)
		}
	}

	// Remove folders matching vendor/qemu-uefi
	matches, err = filepath.Glob("../../vendor/qemu-uefi")
	if err != nil {
		t.Fatalf("Failed to glob ../../vendor/qemu-uefi: %v", err)
	}
	for _, folder := range matches {
		info, err := os.Stat(folder)
		if err != nil {
			t.Fatalf("Failed to stat %s: %v", folder, err)
		}
		if info.IsDir() {
			if err := os.RemoveAll(folder); err != nil {
				t.Fatalf("Failed to remove folder %s: %v", folder, err)
			}
		}
	}

	scriptPath := "scripts/macos/download-arm64-uefi.sh"
	err = RunMacOsSiliconBuildHelperScript(t, scriptPath)
	if err != nil {
		t.Fatalf("Failed to run %s: %v", scriptPath, err)
	}

	if _, err := os.Stat("../../vendor/qemu-uefi/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("Expected file ../../vendor/qemu-uefi/usr/share/qemu-efi-aarch64/QEMU_EFI.fd to exist, but it does not")
		} else {
			t.Fatalf("Failed to stat ../../vendor/qemu-uefi/usr/share/qemu-efi-aarch64/QEMU_EFI.fd: %v", err)
		}
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
