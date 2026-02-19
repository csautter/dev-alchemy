package build

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationDependencyReconciliation(t *testing.T) {
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
	_, err := getWindows11Download("amd64", filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/windows11/iso/win11_25h2_english_amd64.iso"), false)
	if err != nil {
		t.Fatalf("Failed to get Windows 11 download: %v", err)
	}
}

func TestGetWindows11DownloadArm64(t *testing.T) {
	_, err := getWindows11Download("arm64", filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/windows11/iso/win11_25h2_english_arm64.iso"), false)
	if err != nil {
		t.Fatalf("Failed to get Windows 11 download: %v", err)
	}
}

func TestResolveDebianPackageURL(t *testing.T) {
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
	if err := downloadWebFileDependency(dep); err != nil {
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
