//go:build windows
// +build windows

package build

import (
	"os"
	"testing"
)

func TestBuildHypervWindows11Amd64OnWindows(t *testing.T) {
	t.Parallel()

	memoryMB := 4096
	if os.Getenv("GITHUB_ACTIONS") != "" {
		memoryMB = 0
	}

	VirtualMachineConfig := VirtualMachineConfig{
		OS:                   "windows11",
		Arch:                 "amd64",
		VncPort:              5912,
		HostOs:               HostOsWindows,
		VirtualizationEngine: VirtualizationEngineHyperv,
		Cpus:                 4,
		MemoryMB:             memoryMB,
	}
	err := RunHypervWindowsBuildOnWindows(VirtualMachineConfig)
	if err != nil {
		t.Fatalf("Failed to build Hyper-V Windows 11 Amd64 on Windows: %v", err)
	}
}

func TestBuildVirtualBoxWindows11Amd64OnWindows(t *testing.T) {
	t.Parallel()

	memoryMB := 4096
	if os.Getenv("GITHUB_ACTIONS") != "" {
		memoryMB = 0
	}

	VirtualMachineConfig := VirtualMachineConfig{
		OS:                   "windows11",
		Arch:                 "amd64",
		VncPort:              5913,
		HostOs:               HostOsWindows,
		VirtualizationEngine: VirtualizationEngineVirtualBox,
		Cpus:                 4,
		MemoryMB:             memoryMB,
	}
	err := RunVirtualBoxWindowsBuildOnWindows(VirtualMachineConfig)
	if err != nil {
		t.Fatalf("Failed to build VirtualBox Windows 11 Amd64 on Windows: %v", err)
	}
}
