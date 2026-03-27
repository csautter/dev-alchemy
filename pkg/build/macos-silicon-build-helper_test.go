//go:build darwin
// +build darwin

package build

import (
	"context"
	"errors"
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
	requireIntegrationTests(t)
	t.Parallel()
	appDataDir := t.TempDir()
	cacheDir := filepath.Join(appDataDir, "cache")
	t.Setenv(devAlchemyAppDataEnvVar, appDataDir)
	t.Setenv(devAlchemyCacheEnvVar, cacheDir)
	t.Setenv(devAlchemyPackerCacheEnvVar, filepath.Join(appDataDir, "packer_cache"))

	// Remove files matching cache/qemu-efi*
	matches, err := filepath.Glob(filepath.Join(cacheDir, "qemu-efi*"))
	if err != nil {
		t.Fatalf("Failed to glob ../../cache/qemu-efi*: %v", err)
	}
	for _, file := range matches {
		if err := os.RemoveAll(file); err != nil {
			t.Fatalf("Failed to remove %s: %v", file, err)
		}
	}

	// Remove folders matching cache/qemu-uefi
	matches, err = filepath.Glob(filepath.Join(cacheDir, "qemu-uefi"))
	if err != nil {
		t.Fatalf("Failed to glob ../../cache/qemu-uefi: %v", err)
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

	if _, err := os.Stat(filepath.Join(cacheDir, "qemu-uefi", "usr", "share", "qemu-efi-aarch64", "QEMU_EFI.fd")); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("Expected qemu-uefi firmware file to exist in %s, but it does not", cacheDir)
		} else {
			t.Fatalf("Failed to stat qemu-uefi firmware file in %s: %v", cacheDir, err)
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

func TestRunVncSnapshotProcess(t *testing.T) {
	requireIntegrationTests(t)

	if runtime.GOOS != "darwin" {
		t.Skip("Skipping VNC snapshot process test on non-MacOS host")
	}

	vmConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "server",
		VncPort:    5901,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ctx_sub := RunVncSnapshotProcess(vmConfig, ctx, RunProcessConfig{Timeout: 2 * time.Second, Retries: 20, RetryInterval: time.Second}, &VncRecordingConfig{})
	if ctx_sub == nil {
		t.Fatalf("Expected context deadline to be exceeded, but got no error")
	}
	if !errors.Is(ctx_sub.Err(), context.Canceled) {
		t.Fatalf("Expected context canceled to be exceeded, got: %v, error: %v", ctx_sub.Err(), ctx_sub)
	}
}
