package build

import (
	"path/filepath"
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
