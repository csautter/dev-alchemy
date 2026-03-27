package deploy

import (
	"path"
	"path/filepath"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestResolveHypervVagrantExecutionSettings_UsesUbuntuTypeSpecificMetadata(t *testing.T) {
	dirs := alchemy_build.GetDirectoriesInstance()
	originalProjectDir := dirs.ProjectDir
	dirs.ProjectDir = t.TempDir()
	t.Cleanup(func() {
		dirs.ProjectDir = originalProjectDir
	})

	settings, err := ResolveHypervVagrantExecutionSettings(alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "desktop",
		Arch:       "amd64",
	})
	if err != nil {
		t.Fatalf("ResolveHypervVagrantExecutionSettings returned error: %v", err)
	}

	wantDir := filepath.Join(dirs.ProjectDir, "deployments", "vagrant", "linux-ubuntu-hyperv")
	if settings.VagrantDir != wantDir {
		t.Fatalf("expected VagrantDir %q, got %q", wantDir, settings.VagrantDir)
	}

	assertEnvContains(t, settings.VagrantEnv, "VAGRANT_BOX_NAME=linux-ubuntu-desktop-packer")
	assertEnvContains(t, settings.VagrantEnv, "VAGRANT_VM_NAME=linux-ubuntu-desktop-packer")
	assertEnvContains(t, settings.VagrantEnv, "VAGRANT_DOTFILE_PATH="+path.Join(".vagrant", "linux-ubuntu-desktop-packer"))
}

func assertEnvContains(t *testing.T, env []string, want string) {
	t.Helper()

	for _, entry := range env {
		if entry == want {
			return
		}
	}

	t.Fatalf("expected environment to contain %q, got %v", want, env)
}
