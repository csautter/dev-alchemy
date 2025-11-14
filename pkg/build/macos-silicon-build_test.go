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
	RunQemuUbuntuBuildOnMacOS(VirtualMachineConfig)
}

func TestBuildQemuUbuntuServerAmd64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "server",
		VncPort:    5902,
	}
	RunQemuUbuntuBuildOnMacOS(VirtualMachineConfig)
}

func TestBuildQemuUbuntuDesktopArm64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "desktop",
		VncPort:    5903,
	}
	RunQemuUbuntuBuildOnMacOS(VirtualMachineConfig)
}

func TestBuildQemuUbuntuDesktopAmd64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "desktop",
		VncPort:    5904,
	}
	RunQemuUbuntuBuildOnMacOS(VirtualMachineConfig)
}

func TestBuildQemuWindows11Arm64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:      "windows11",
		Arch:    "arm64",
		VncPort: 5911,
	}
	RunQemuWindowsBuildOnMacOS(VirtualMachineConfig)
}

func TestBuildQemuWindows11Amd64OnMacos(t *testing.T) {
	t.Parallel()

	VirtualMachineConfig := VirtualMachineConfig{
		OS:      "windows11",
		Arch:    "amd64",
		VncPort: 5912,
	}
	RunQemuWindowsBuildOnMacOS(VirtualMachineConfig)
}
