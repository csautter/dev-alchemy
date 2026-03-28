package deploy

import (
	"path"
	"path/filepath"
	"strconv"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestResolveHypervVagrantDeploySettings_UbuntuIncludesConfigResources(t *testing.T) {
	projectDir := t.TempDir()
	config := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "server",
		Arch:       "amd64",
		Cpus:       6,
		MemoryMB:   12288,
	}

	settings, err := resolveHypervVagrantDeploySettings(config, projectDir)
	if err != nil {
		t.Fatalf("resolveHypervVagrantDeploySettings returned error: %v", err)
	}

	if settings.BoxName != "linux-ubuntu-server-packer" {
		t.Fatalf("expected ubuntu box name to include type, got %q", settings.BoxName)
	}

	wantVagrantDir := filepath.Join(projectDir, "deployments", "vagrant", "linux-ubuntu-hyperv")
	if settings.VagrantDir != wantVagrantDir {
		t.Fatalf("expected vagrant dir %q, got %q", wantVagrantDir, settings.VagrantDir)
	}

	env := envListToMap(settings.VagrantEnv)
	if env[hypervVagrantBoxNameEnvVar] != "linux-ubuntu-server-packer" {
		t.Fatalf("expected %s env var to match box name, got %q", hypervVagrantBoxNameEnvVar, env[hypervVagrantBoxNameEnvVar])
	}
	if env[hypervVagrantVMNameEnvVar] != "linux-ubuntu-server-packer" {
		t.Fatalf("expected %s env var to match box name, got %q", hypervVagrantVMNameEnvVar, env[hypervVagrantVMNameEnvVar])
	}
	expectedDotfilePath := path.Join(".vagrant", "linux-ubuntu-server-packer")
	if env[hypervVagrantDotfileEnvVar] != expectedDotfilePath {
		t.Fatalf("expected %s env var to match isolated state path, got %q", hypervVagrantDotfileEnvVar, env[hypervVagrantDotfileEnvVar])
	}
	expectedCPU := strconv.Itoa(alchemy_build.GetVmCpuCount(config))
	if env[hypervVagrantCpuEnvVar] != expectedCPU {
		t.Fatalf(
			"expected cpu env var %s to match resolved value, got %q",
			expectedCPU,
			env[hypervVagrantCpuEnvVar],
		)
	}
	if env[hypervVagrantMemoryEnvVar] != "12288" {
		t.Fatalf("expected memory env var to match config value, got %q", env[hypervVagrantMemoryEnvVar])
	}
}

func TestResolveHypervVagrantDeploySettings_WindowsIncludesConfigResources(t *testing.T) {
	projectDir := t.TempDir()
	config := alchemy_build.VirtualMachineConfig{
		OS:       "windows11",
		Arch:     "amd64",
		Cpus:     2,
		MemoryMB: 4096,
	}

	settings, err := resolveHypervVagrantDeploySettings(config, projectDir)
	if err != nil {
		t.Fatalf("resolveHypervVagrantDeploySettings returned error: %v", err)
	}

	if settings.BoxName != windowsHypervVagrantBoxName {
		t.Fatalf("expected windows box name %q, got %q", windowsHypervVagrantBoxName, settings.BoxName)
	}

	wantVagrantDir := filepath.Join(projectDir, "deployments", "vagrant", "ansible-windows")
	if settings.VagrantDir != wantVagrantDir {
		t.Fatalf("expected vagrant dir %q, got %q", wantVagrantDir, settings.VagrantDir)
	}

	env := envListToMap(settings.VagrantEnv)
	if env[hypervVagrantBoxNameEnvVar] != windowsHypervVagrantBoxName {
		t.Fatalf("expected %s env var to match windows box name, got %q", hypervVagrantBoxNameEnvVar, env[hypervVagrantBoxNameEnvVar])
	}
	if env[hypervVagrantVMNameEnvVar] != windowsHypervVagrantBoxName {
		t.Fatalf("expected %s env var to match windows vm name, got %q", hypervVagrantVMNameEnvVar, env[hypervVagrantVMNameEnvVar])
	}
	expectedDotfilePath := path.Join(".vagrant", windowsHypervVagrantBoxName)
	if env[hypervVagrantDotfileEnvVar] != expectedDotfilePath {
		t.Fatalf("expected %s env var to match isolated state path, got %q", hypervVagrantDotfileEnvVar, env[hypervVagrantDotfileEnvVar])
	}
	expectedCPU := strconv.Itoa(alchemy_build.GetVmCpuCount(config))
	if env[hypervVagrantCpuEnvVar] != expectedCPU {
		t.Fatalf(
			"expected cpu env var %s to match resolved value, got %q",
			expectedCPU,
			env[hypervVagrantCpuEnvVar],
		)
	}
	if env[hypervVagrantMemoryEnvVar] != "4096" {
		t.Fatalf("expected memory env var to match config value, got %q", env[hypervVagrantMemoryEnvVar])
	}
}

func TestResolveHypervVagrantDeploySettings_UbuntuVariantsUseDistinctDotfilePaths(t *testing.T) {
	projectDir := t.TempDir()

	serverSettings, err := resolveHypervVagrantDeploySettings(alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "server",
		Arch:       "amd64",
	}, projectDir)
	if err != nil {
		t.Fatalf("resolveHypervVagrantDeploySettings(server) returned error: %v", err)
	}

	desktopSettings, err := resolveHypervVagrantDeploySettings(alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "desktop",
		Arch:       "amd64",
	}, projectDir)
	if err != nil {
		t.Fatalf("resolveHypervVagrantDeploySettings(desktop) returned error: %v", err)
	}

	serverEnv := envListToMap(serverSettings.VagrantEnv)
	desktopEnv := envListToMap(desktopSettings.VagrantEnv)
	if serverEnv[hypervVagrantDotfileEnvVar] == desktopEnv[hypervVagrantDotfileEnvVar] {
		t.Fatalf("expected ubuntu variants to use different %s values, both were %q", hypervVagrantDotfileEnvVar, serverEnv[hypervVagrantDotfileEnvVar])
	}
}

func envListToMap(envList []string) map[string]string {
	env := make(map[string]string, len(envList))
	for _, entry := range envList {
		for index := 0; index < len(entry); index++ {
			if entry[index] == '=' {
				env[entry[:index]] = entry[index+1:]
				break
			}
		}
	}
	return env
}
