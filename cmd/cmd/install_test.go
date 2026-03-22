package cmd

import (
	"path/filepath"
	"reflect"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestInstallCommandForHostDarwin(t *testing.T) {
	projectDir := filepath.Join("tmp", "dev-alchemy")

	spec, err := installCommandForHost(alchemy_build.HostOsDarwin, projectDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedScriptPath := filepath.Join(projectDir, "scripts", "macos", "dev-alchemy-install-dependencies.sh")
	if spec.executable != "bash" {
		t.Fatalf("expected bash, got %q", spec.executable)
	}
	if spec.scriptPath != expectedScriptPath {
		t.Fatalf("expected script path %q, got %q", expectedScriptPath, spec.scriptPath)
	}
	if !reflect.DeepEqual(spec.args, []string{expectedScriptPath}) {
		t.Fatalf("expected args %v, got %v", []string{expectedScriptPath}, spec.args)
	}
}

func TestInstallCommandForHostWindows(t *testing.T) {
	projectDir := filepath.Join("tmp", "dev-alchemy")

	spec, err := installCommandForHost(alchemy_build.HostOsWindows, projectDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedScriptPath := filepath.Join(projectDir, "scripts", "windows", "dev-alchemy-self-setup.ps1")
	expectedArgs := []string{"-ExecutionPolicy", "Bypass", "-File", expectedScriptPath}
	if spec.executable != "powershell" {
		t.Fatalf("expected powershell, got %q", spec.executable)
	}
	if spec.scriptPath != expectedScriptPath {
		t.Fatalf("expected script path %q, got %q", expectedScriptPath, spec.scriptPath)
	}
	if !reflect.DeepEqual(spec.args, expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, spec.args)
	}
}

func TestInstallCommandForUnsupportedHost(t *testing.T) {
	_, err := installCommandForHost(alchemy_build.HostOsLinux, "/tmp/dev-alchemy")
	if err == nil {
		t.Fatalf("expected unsupported host error")
	}
}
