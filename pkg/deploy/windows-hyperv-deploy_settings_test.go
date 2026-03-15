package deploy

import (
	"path/filepath"
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
	if env[hypervVagrantCpuEnvVar] != "6" {
		t.Fatalf("expected cpu env var to match config value, got %q", env[hypervVagrantCpuEnvVar])
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
	if env[hypervVagrantCpuEnvVar] != "2" {
		t.Fatalf("expected cpu env var to match config value, got %q", env[hypervVagrantCpuEnvVar])
	}
	if env[hypervVagrantMemoryEnvVar] != "4096" {
		t.Fatalf("expected memory env var to match config value, got %q", env[hypervVagrantMemoryEnvVar])
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
