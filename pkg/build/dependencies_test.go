package build

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBootstrapPythonEnv_VenvCreationFails verifies that bootstrapPythonEnv returns
// a wrapped error (not nil) when the python executable does not exist and the venv
// directory is absent, so callers can fail fast with an actionable message.
func TestBootstrapPythonEnv_VenvCreationFails(t *testing.T) {
	workdir := t.TempDir()
	// Point at a path that is guaranteed not to exist.
	badPython := filepath.Join(t.TempDir(), "nonexistent-python")

	err := bootstrapPythonEnv(workdir, badPython)

	if err == nil {
		t.Fatal("expected error when Python executable does not exist, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create Python venv") {
		t.Errorf("expected error to mention venv creation failure, got: %v", err)
	}
}

// TestBootstrapPythonEnv_PipInstallFails verifies that bootstrapPythonEnv returns
// a wrapped error when the venv directory already exists but contains no real
// Python/pip binaries, so the playwright pip install step fails immediately.
func TestBootstrapPythonEnv_PipInstallFails(t *testing.T) {
	workdir := t.TempDir()
	// Create the .venv directory so venv creation is skipped, but leave it empty
	// so that venvPython and pipPath do not exist.
	if err := os.MkdirAll(filepath.Join(workdir, ".venv"), 0755); err != nil {
		t.Fatalf("could not create mock venv dir: %v", err)
	}

	// pythonExe is irrelevant here because the venv dir already exists.
	err := bootstrapPythonEnv(workdir, "ignored")

	if err == nil {
		t.Fatal("expected error when venv Python/pip binaries are missing, got nil")
	}
	if !strings.Contains(err.Error(), "failed to install Windows 11 download script requirements") {
		t.Errorf("expected error to mention requirements install failure, got: %v", err)
	}
}

func TestIntegrationDependencyReconciliation(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test; set RUN_INTEGRATION_TESTS=1 to run")
	}
	tests := []VirtualMachineConfig{
		{
			OS:   "windows11",
			Arch: "amd64",
		},
		{
			OS:   "windows11",
			Arch: "arm64",
		},
		{
			OS:         "ubuntu",
			Arch:       "amd64",
			UbuntuType: "desktop",
		},
		{
			OS:         "ubuntu",
			Arch:       "arm64",
			UbuntuType: "desktop",
		},
		{
			OS:         "ubuntu",
			Arch:       "amd64",
			UbuntuType: "server",
		},
		{
			OS:         "ubuntu",
			Arch:       "arm64",
			UbuntuType: "server",
		},
	}

	for _, vmconfig := range tests {
		DependencyReconciliation(vmconfig)
	}
}

func TestGetWindows11DownloadAmd64(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test; set RUN_INTEGRATION_TESTS=1 to run")
	}
	_, err := getWindows11Download("amd64", GetDirectoriesInstance().CachePath("windows11", "iso", "win11_25h2_english_amd64.iso"), false)
	if err != nil {
		t.Fatalf("Failed to get Windows 11 download: %v", err)
	}
}

func TestGetWindows11DownloadArm64(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test; set RUN_INTEGRATION_TESTS=1 to run")
	}
	_, err := getWindows11Download("arm64", GetDirectoriesInstance().CachePath("windows11", "iso", "win11_25h2_english_arm64.iso"), false)
	if err != nil {
		t.Fatalf("Failed to get Windows 11 download: %v", err)
	}
}

func TestResolveDebianPackageURL(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test; set RUN_INTEGRATION_TESTS=1 to run")
	}
	url, err := resolveDebianPackageURL("trixie", "qemu-efi-aarch64")
	if err != nil {
		t.Fatalf("resolveDebianPackageURL returned error: %v", err)
	}
	if !strings.HasPrefix(url, "https://deb.debian.org/debian/pool/") {
		t.Errorf("unexpected URL prefix: %s", url)
	}
	if !strings.Contains(url, "qemu-efi-aarch64") {
		t.Errorf("URL does not contain package name: %s", url)
	}
	t.Logf("Resolved URL: %s", url)

	// Verify the resolved URL is actually reachable
	resp, err := http.Head(url)
	if err != nil {
		t.Fatalf("HTTP HEAD request to resolved URL failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("resolved URL returned HTTP %d: %s", resp.StatusCode, url)
	}
	t.Logf("HTTP HEAD status: %d", resp.StatusCode)
}

func TestResolveAndDownloadQemuEfiAarch64(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test; set RUN_INTEGRATION_TESTS=1 to run")
	}
	url, err := resolveDebianPackageURL("trixie", "qemu-efi-aarch64")
	if err != nil {
		t.Fatalf("resolveDebianPackageURL returned error: %v", err)
	}
	t.Logf("Resolved URL: %s", url)

	destPath := filepath.Join(t.TempDir(), "qemu-efi-aarch64_all.deb")
	dep := WebFileDependency{
		LocalPath: destPath,
		Source:    url,
	}
	if err := downloadWebFileDependency(nil, dep); err != nil {
		t.Fatalf("downloadWebFileDependency failed: %v", err)
	}

	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("downloaded file not found at %s: %v", destPath, err)
	}
	if info.Size() == 0 {
		t.Errorf("downloaded file is empty: %s", destPath)
	}
	t.Logf("Downloaded %s (%d bytes)", destPath, info.Size())
}

func TestDownloadWebFileDependencyWithoutProgressBar(t *testing.T) {
	expected := []byte("test deb payload")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(expected)
	}))
	defer server.Close()

	destPath := filepath.Join(t.TempDir(), "downloaded.deb")
	dep := WebFileDependency{
		LocalPath: destPath,
		Source:    server.URL + "/qemu-efi-aarch64_all.deb",
	}

	if err := downloadWebFileDependency(nil, dep); err != nil {
		t.Fatalf("downloadWebFileDependency failed without progress bar: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if !bytes.Equal(got, expected) {
		t.Fatalf("downloaded file contents mismatch: got %q want %q", string(got), string(expected))
	}
}
