//go:build windows
// +build windows

package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestGetHypervWindowsBoxPath_UsesExpectedBuildArtifact(t *testing.T) {
	artifactPath := filepath.Join(t.TempDir(), "custom-hyperv.box")
	config := alchemy_build.VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifactPath},
	}

	if got := getHypervWindowsBoxPath(config); got != artifactPath {
		t.Fatalf("expected box path %q, got %q", artifactPath, got)
	}
}

func TestGetHypervWindowsBoxPath_FallsBackToCachePath(t *testing.T) {
	dirs := alchemy_build.GetDirectoriesInstance()
	originalCacheDir := dirs.CacheDir
	cacheDir := filepath.Join(t.TempDir(), "cache-root")
	dirs.CacheDir = cacheDir
	t.Cleanup(func() {
		dirs.CacheDir = originalCacheDir
	})

	config := alchemy_build.VirtualMachineConfig{}
	want := filepath.Join(cacheDir, "windows11", "hyperv-windows11-amd64.box")
	if got := getHypervWindowsBoxPath(config); got != want {
		t.Fatalf("expected fallback box path %q, got %q", want, got)
	}
}

func TestGetHypervUbuntuBoxPath_UsesExpectedBuildArtifact(t *testing.T) {
	artifactPath := filepath.Join(t.TempDir(), "custom-ubuntu-hyperv.box")
	config := alchemy_build.VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifactPath},
	}

	if got := getHypervUbuntuBoxPath(config); got != artifactPath {
		t.Fatalf("expected box path %q, got %q", artifactPath, got)
	}
}

func TestGetHypervUbuntuBoxPath_FallsBackToCachePath(t *testing.T) {
	dirs := alchemy_build.GetDirectoriesInstance()
	originalCacheDir := dirs.CacheDir
	cacheDir := filepath.Join(t.TempDir(), "cache-root")
	dirs.CacheDir = cacheDir
	t.Cleanup(func() {
		dirs.CacheDir = originalCacheDir
	})

	config := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "desktop",
	}
	want := filepath.Join(cacheDir, "ubuntu", "hyperv-ubuntu-desktop-amd64.box")
	if got := getHypervUbuntuBoxPath(config); got != want {
		t.Fatalf("expected fallback box path %q, got %q", want, got)
	}
}

func TestRunHypervVagrantDeployOnWindows_Smoke(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" || os.Getenv("RUN_WINDOWS_HYPERV_DEPLOY_SMOKE") == "" {
		t.Skip("skipping smoke test; set RUN_INTEGRATION_TESTS=1 and RUN_WINDOWS_HYPERV_DEPLOY_SMOKE=1")
	}

	if _, err := exec.LookPath("vagrant"); err != nil {
		t.Skipf("skipping smoke test: vagrant executable not found: %v", err)
	}

	boxPath := os.Getenv("WINDOWS_HYPERV_BOX_PATH")
	if boxPath == "" {
		boxPath = filepath.Join(
			alchemy_build.GetDirectoriesInstance().CacheDir,
			"windows11",
			"hyperv-windows11-amd64.box",
		)
	}

	if _, err := os.Stat(boxPath); err != nil {
		t.Skipf("skipping smoke test: Hyper-V box not available at %q: %v", boxPath, err)
	}

	config := alchemy_build.VirtualMachineConfig{
		OS:                     "windows11",
		Arch:                   "amd64",
		ExpectedBuildArtifacts: []string{boxPath},
	}
	if err := RunHypervVagrantDeployOnWindows(config); err != nil {
		t.Fatalf("Hyper-V deploy smoke test failed: %v", err)
	}
}
