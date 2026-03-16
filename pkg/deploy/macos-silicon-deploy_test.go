//go:build darwin
// +build darwin

package deploy

// When this tests are executed by VS Code integration it might be that bash scripts are not working as intended.
// Issues with bash expressions have been observed. Running the tests directly with `go test` should work as expected.

import (
	"os"
	"runtime"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestPrintSystemOsArch(t *testing.T) {
	t.Logf("Running on OS: %s, ARCH: %s", runtime.GOOS, runtime.GOARCH)
}

func TestDeployUtmUbuntuServerArm64OnMacos(t *testing.T) {
	requireUtmDeploySmokePrereqs(t, alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "server",
	})

	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "server",
	}
	if err := RunUtmDeployOnMacOS(VirtualMachineConfig); err != nil {
		t.Fatalf("expected UTM deploy to succeed: %v", err)
	}
}

func TestDeployUtmUbuntuServerAmd64OnMacos(t *testing.T) {
	requireUtmDeploySmokePrereqs(t, alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "server",
	})

	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "server",
	}
	if err := RunUtmDeployOnMacOS(VirtualMachineConfig); err != nil {
		t.Fatalf("expected UTM deploy to succeed: %v", err)
	}
}

func TestDeployUtmUbuntuDesktopArm64OnMacos(t *testing.T) {
	requireUtmDeploySmokePrereqs(t, alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "desktop",
	})

	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "arm64",
		UbuntuType: "desktop",
	}
	if err := RunUtmDeployOnMacOS(VirtualMachineConfig); err != nil {
		t.Fatalf("expected UTM deploy to succeed: %v", err)
	}
}

func TestDeployUtmUbuntuDesktopAmd64OnMacos(t *testing.T) {
	requireUtmDeploySmokePrereqs(t, alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "desktop",
	})

	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		Arch:       "amd64",
		UbuntuType: "desktop",
	}
	if err := RunUtmDeployOnMacOS(VirtualMachineConfig); err != nil {
		t.Fatalf("expected UTM deploy to succeed: %v", err)
	}
}

func TestDeployUtmWindows11Arm64OnMacos(t *testing.T) {
	requireUtmDeploySmokePrereqs(t, alchemy_build.VirtualMachineConfig{
		OS:   "windows11",
		Arch: "arm64",
	})

	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:   "windows11",
		Arch: "arm64",
	}
	if err := RunUtmDeployOnMacOS(VirtualMachineConfig); err != nil {
		t.Fatalf("expected UTM deploy to succeed: %v", err)
	}
}

func TestDeployUtmWindows11Amd64OnMacos(t *testing.T) {
	requireUtmDeploySmokePrereqs(t, alchemy_build.VirtualMachineConfig{
		OS:   "windows11",
		Arch: "amd64",
	})

	VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
		OS:   "windows11",
		Arch: "amd64",
	}
	if err := RunUtmDeployOnMacOS(VirtualMachineConfig); err != nil {
		t.Fatalf("expected UTM deploy to succeed: %v", err)
	}
}

func requireUtmDeploySmokePrereqs(t *testing.T, config alchemy_build.VirtualMachineConfig) {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") == "" || os.Getenv("RUN_MACOS_UTM_DEPLOY_SMOKE") == "" {
		t.Skip("skipping UTM deploy smoke test; set RUN_INTEGRATION_TESTS=1 and RUN_MACOS_UTM_DEPLOY_SMOKE=1")
	}

	artifactPath := resolveUtmExpectedArtifact(config)
	if artifactPath == "" {
		t.Skipf("skipping UTM deploy smoke test: no expected artifact path found for %s:%s:%s", config.OS, config.UbuntuType, config.Arch)
	}
	if _, err := os.Stat(artifactPath); err != nil {
		t.Skipf("skipping UTM deploy smoke test: qcow2 artifact not available at %q: %v", artifactPath, err)
	}
}

func resolveUtmExpectedArtifact(config alchemy_build.VirtualMachineConfig) string {
	for _, candidate := range alchemy_build.AvailableVirtualMachineConfigs() {
		if candidate.HostOs != alchemy_build.HostOsDarwin {
			continue
		}
		if candidate.VirtualizationEngine != alchemy_build.VirtualizationEngineUtm {
			continue
		}
		if candidate.OS != config.OS || candidate.Arch != config.Arch || candidate.UbuntuType != config.UbuntuType {
			continue
		}
		if len(candidate.ExpectedBuildArtifacts) == 0 {
			return ""
		}
		return candidate.ExpectedBuildArtifacts[0]
	}
	return ""
}
