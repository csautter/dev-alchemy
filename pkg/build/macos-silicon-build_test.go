package build

import (
	"runtime"
	"testing"
)

func TestPrintSystemOsArch(t *testing.T) {
	t.Logf("Running on OS: %s, ARCH: %s", runtime.GOOS, runtime.GOARCH)
}

func TestBuildQemuUbuntuServerArm64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "server",
		VncPort:    5901,
	}
	err := RunQemuUbuntuBuildOnMacOS(VirtualMachineConfig)
	if err != nil {
		t.Fatalf("Failed to build QEMU Ubuntu Server Arm64 on macOS: %v", err)
	}
}

func TestBuildQemuUbuntuServerAmd64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "server",
		VncPort:    5902,
	}
	err := RunQemuUbuntuBuildOnMacOS(VirtualMachineConfig)
	if err != nil {
		t.Fatalf("Failed to build QEMU Ubuntu Server Amd64 on macOS: %v", err)
	}
}

func TestBuildQemuUbuntuDesktopArm64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "desktop",
		VncPort:    5903,
	}
	err := RunQemuUbuntuBuildOnMacOS(VirtualMachineConfig)
	if err != nil {
		t.Fatalf("Failed to build QEMU Ubuntu Desktop Arm64 on macOS: %v", err)
	}
}

func TestBuildQemuUbuntuDesktopAmd64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "desktop",
		VncPort:    5904,
	}
	err := RunQemuUbuntuBuildOnMacOS(VirtualMachineConfig)
	if err != nil {
		t.Fatalf("Failed to build QEMU Ubuntu Desktop Amd64 on macOS: %v", err)
	}
}

func TestBuildQemuWindows11Arm64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:      "windows11",
		Arch:    "arm64",
		VncPort: 5911,
	}
	err := RunQemuWindowsBuildOnMacOS(VirtualMachineConfig)
	if err != nil {
		t.Fatalf("Failed to build QEMU Windows 11 Arm64 on macOS: %v", err)
	}
}

func TestBuildQemuWindows11Amd64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:      "windows11",
		Arch:    "amd64",
		VncPort: 5912,
	}
	err := RunQemuWindowsBuildOnMacOS(VirtualMachineConfig)
	if err != nil {
		t.Fatalf("Failed to build QEMU Windows 11 Amd64 on macOS: %v", err)
	}
}
