package deploy

// When this tests are executed by VS Code integration it might be that bash scripts are not working as intended.
// Issues with bash expressions have been observed. Running the tests directly with `go test` should work as expected.

import (
	"runtime"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestPrintSystemOsArch(t *testing.T) {
	t.Logf("Running on OS: %s, ARCH: %s", runtime.GOOS, runtime.GOARCH)
}

func TestDeployUtmUbuntuServerArm64OnMacos(t *testing.T) {

	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "server",
	}
	RunUtmDeployOnMacOS(VirtualMachineConfig)
}

func TestDeployUtmUbuntuServerAmd64OnMacos(t *testing.T) {

	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "server",
	}
	RunUtmDeployOnMacOS(VirtualMachineConfig)
}

func TestDeployUtmUbuntuDesktopArm64OnMacos(t *testing.T) {

	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "desktop",
	}
	RunUtmDeployOnMacOS(VirtualMachineConfig)
}

func TestDeployUtmUbuntuDesktopAmd64OnMacos(t *testing.T) {
	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "desktop",
	}
	RunUtmDeployOnMacOS(VirtualMachineConfig)
}

func TestDeployUtmWindows11Arm64OnMacos(t *testing.T) {

	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:   "windows11",
		Arch: "arm64",
	}
	RunUtmDeployOnMacOS(VirtualMachineConfig)
}

func TestDeployUtmWindows11Amd64OnMacos(t *testing.T) {

	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:   "windows11",
		Arch: "amd64",
	}
	RunUtmDeployOnMacOS(VirtualMachineConfig)
}
