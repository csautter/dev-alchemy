package cmd

import (
	"path/filepath"
	"reflect"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestInstallCommandForHostDarwin(t *testing.T) {
	projectDir := filepath.Join("tmp", "dev-alchemy")

	spec, err := installCommandForHost(alchemy_build.HostOsDarwin, projectDir, installCommandOptions{})
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

	spec, err := installCommandForHost(alchemy_build.HostOsWindows, projectDir, installCommandOptions{})
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

func TestInstallCommandForHostLinux(t *testing.T) {
	projectDir := filepath.Join("tmp", "dev-alchemy")

	spec, err := installCommandForHost(alchemy_build.HostOsLinux, projectDir, installCommandOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedScriptPath := filepath.Join(projectDir, "scripts", "linux", "dev-alchemy-install-dependencies.sh")
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

func TestInstallCommandForHostDarwinWithGo(t *testing.T) {
	projectDir := filepath.Join("tmp", "dev-alchemy")

	spec, err := installCommandForHost(alchemy_build.HostOsDarwin, projectDir, installCommandOptions{withGo: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedScriptPath := filepath.Join(projectDir, "scripts", "macos", "dev-alchemy-install-dependencies.sh")
	expectedArgs := []string{expectedScriptPath, "--with-go"}
	if !reflect.DeepEqual(spec.args, expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, spec.args)
	}
}

func TestInstallCommandForHostLinuxWithGo(t *testing.T) {
	projectDir := filepath.Join("tmp", "dev-alchemy")

	spec, err := installCommandForHost(alchemy_build.HostOsLinux, projectDir, installCommandOptions{withGo: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedScriptPath := filepath.Join(projectDir, "scripts", "linux", "dev-alchemy-install-dependencies.sh")
	expectedArgs := []string{expectedScriptPath, "--with-go"}
	if !reflect.DeepEqual(spec.args, expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, spec.args)
	}
}

func TestInstallCommandForHostWindowsWithGoUnsupported(t *testing.T) {
	_, err := installCommandForHost(alchemy_build.HostOsWindows, "/tmp/dev-alchemy", installCommandOptions{withGo: true})
	if err == nil {
		t.Fatal("expected unsupported with-go error")
	}
}

func TestInstallCommandForUnsupportedHost(t *testing.T) {
	_, err := installCommandForHost(alchemy_build.HostOsType("plan9"), "/tmp/dev-alchemy", installCommandOptions{})
	if err == nil {
		t.Fatalf("expected unsupported host error")
	}
}
