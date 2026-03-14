package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestExtractWindowsIPv4FromIPConfig(t *testing.T) {
	output := `
Windows IP Configuration

Ethernet adapter Loopback:
   IPv4 Address. . . . . . . . . . . : 127.0.0.1

Ethernet adapter Ethernet:
   IPv4 Address. . . . . . . . . . . : 172.25.125.159
`

	ip, err := extractWindowsIPv4FromIPConfig(output)
	if err != nil {
		t.Fatalf("expected IP extraction to succeed, got error: %v", err)
	}
	if ip != "172.25.125.159" {
		t.Fatalf("expected 172.25.125.159, got %s", ip)
	}
}

func TestBuildWindowsHypervProvisionArgs(t *testing.T) {
	config := windowsHypervAnsibleConnectionConfig{
		User:           "Administrator",
		Password:       "Top$ecret!",
		Connection:     "winrm",
		WinrmTransport: "basic",
		Port:           "5985",
	}

	args, err := buildWindowsHypervProvisionArgs("172.25.125.159", config, true)
	if err != nil {
		t.Fatalf("buildWindowsHypervProvisionArgs returned error: %v", err)
	}

	if !strings.Contains(strings.Join(args, " "), "-i 172.25.125.159,") {
		t.Fatalf("expected inline inventory with discovered host IP, args: %v", args)
	}
	if !strings.Contains(strings.Join(args, " "), "-l 172.25.125.159") {
		t.Fatalf("expected limit to discovered host IP, args: %v", args)
	}
	if args[len(args)-1] != "--check" {
		t.Fatalf("expected --check to be passed through when requested, args: %v", args)
	}

	extraVarsIndex := -1
	for index, arg := range args {
		if arg == "-e" {
			extraVarsIndex = index + 1
			break
		}
	}
	if extraVarsIndex <= 0 || extraVarsIndex >= len(args) {
		t.Fatalf("expected -e with json payload, args: %v", args)
	}

	extraVars := map[string]string{}
	if err := json.Unmarshal([]byte(args[extraVarsIndex]), &extraVars); err != nil {
		t.Fatalf("expected extra vars to be valid JSON, got error: %v", err)
	}

	if extraVars["ansible_user"] != "Administrator" {
		t.Fatalf("expected ansible_user in extra vars, got: %v", extraVars)
	}
	if extraVars["ansible_password"] != "Top$ecret!" {
		t.Fatalf("expected ansible_password in extra vars, got: %v", extraVars)
	}
	if extraVars["ansible_connection"] != "winrm" {
		t.Fatalf("expected ansible_connection in extra vars, got: %v", extraVars)
	}
	if extraVars["ansible_winrm_transport"] != "basic" {
		t.Fatalf("expected ansible_winrm_transport in extra vars, got: %v", extraVars)
	}
	if extraVars["ansible_port"] != "5985" {
		t.Fatalf("expected ansible_port in extra vars, got: %v", extraVars)
	}
}

func TestLoadWindowsHypervAnsibleConnectionConfig_UsesDotEnvValues(t *testing.T) {
	projectDir := t.TempDir()
	dotEnvPath := filepath.Join(projectDir, ".env")

	content := strings.Join([]string{
		hypervWindowsAnsibleUserEnvVar + "=Administrator",
		hypervWindowsAnsiblePasswordEnvVar + "='P@ssw0rd! with spaces'",
		hypervWindowsAnsibleConnectionEnvVar + "=winrm",
		hypervWindowsAnsibleWinrmTransportEnvVar + "=basic",
		hypervWindowsAnsiblePortEnvVar + "=5985",
		"",
	}, "\n")
	if err := os.WriteFile(dotEnvPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create .env fixture: %v", err)
	}

	connectionConfig, err := loadWindowsHypervAnsibleConnectionConfig(projectDir)
	if err != nil {
		t.Fatalf("loadWindowsHypervAnsibleConnectionConfig returned error: %v", err)
	}

	if connectionConfig.User != "Administrator" {
		t.Fatalf("expected user from .env, got %q", connectionConfig.User)
	}
	if connectionConfig.Password != "P@ssw0rd! with spaces" {
		t.Fatalf("expected password from .env, got %q", connectionConfig.Password)
	}
	if connectionConfig.Connection != "winrm" {
		t.Fatalf("expected connection from .env, got %q", connectionConfig.Connection)
	}
	if connectionConfig.WinrmTransport != "basic" {
		t.Fatalf("expected winrm transport from .env, got %q", connectionConfig.WinrmTransport)
	}
	if connectionConfig.Port != "5985" {
		t.Fatalf("expected port from .env, got %q", connectionConfig.Port)
	}
}

func TestLoadWindowsHypervAnsibleConnectionConfig_EnvOverridesDotEnv(t *testing.T) {
	projectDir := t.TempDir()
	dotEnvPath := filepath.Join(projectDir, ".env")

	content := strings.Join([]string{
		hypervWindowsAnsibleUserEnvVar + "=file-user",
		hypervWindowsAnsiblePasswordEnvVar + "=file-pass",
		"",
	}, "\n")
	if err := os.WriteFile(dotEnvPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create .env fixture: %v", err)
	}

	t.Setenv(hypervWindowsAnsibleUserEnvVar, "env-user")
	t.Setenv(hypervWindowsAnsiblePasswordEnvVar, "env-pass")
	t.Setenv(hypervWindowsAnsibleConnectionEnvVar, "winrm")
	t.Setenv(hypervWindowsAnsibleWinrmTransportEnvVar, "basic")
	t.Setenv(hypervWindowsAnsiblePortEnvVar, "5986")

	connectionConfig, err := loadWindowsHypervAnsibleConnectionConfig(projectDir)
	if err != nil {
		t.Fatalf("loadWindowsHypervAnsibleConnectionConfig returned error: %v", err)
	}

	if connectionConfig.User != "env-user" {
		t.Fatalf("expected environment user to override .env, got %q", connectionConfig.User)
	}
	if connectionConfig.Password != "env-pass" {
		t.Fatalf("expected environment password to override .env, got %q", connectionConfig.Password)
	}
	if connectionConfig.Port != "5986" {
		t.Fatalf("expected environment port to override .env, got %q", connectionConfig.Port)
	}
}

func TestLoadWindowsHypervAnsibleConnectionConfig_ReturnsErrorWhenRequiredValuesMissing(t *testing.T) {
	projectDir := t.TempDir()

	_, err := loadWindowsHypervAnsibleConnectionConfig(projectDir)
	if err == nil {
		t.Fatal("expected missing configuration to return an error")
	}
	if !strings.Contains(err.Error(), hypervWindowsAnsibleUserEnvVar) {
		t.Fatalf("expected missing user env var name in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), hypervWindowsAnsiblePasswordEnvVar) {
		t.Fatalf("expected missing password env var name in error, got: %v", err)
	}
}

func TestParseDotEnvFile_ReturnsErrorForInvalidLine(t *testing.T) {
	projectDir := t.TempDir()
	dotEnvPath := filepath.Join(projectDir, ".env")

	if err := os.WriteFile(dotEnvPath, []byte("INVALID_LINE"), 0o644); err != nil {
		t.Fatalf("failed to create .env fixture: %v", err)
	}

	_, err := parseDotEnvFile(dotEnvPath)
	if err == nil {
		t.Fatal("expected parseDotEnvFile to return an error")
	}
	if !strings.Contains(err.Error(), "expected KEY=VALUE") {
		t.Fatalf("unexpected parse error: %v", err)
	}
}

func TestRunProvisionReturnsNotImplementedForUnsupportedConfig(t *testing.T) {
	vm := alchemy_build.VirtualMachineConfig{
		OS:                   "windows11",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsWindows,
		VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
	}

	err := RunProvision(vm, false)
	if err == nil {
		t.Fatal("expected RunProvision to fail for unsupported vm configuration, got nil")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not implemented message, got: %v", err)
	}
}

func TestWindowsPathToCygwinPath(t *testing.T) {
	got, err := windowsPathToCygwinPath(`C:\workspace\dev-alchemy`)
	if err != nil {
		t.Fatalf("windowsPathToCygwinPath returned error: %v", err)
	}

	want := "/cygdrive/c/workspace/dev-alchemy"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestWindowsPathToCygwinPath_ReturnsErrorForInvalidPath(t *testing.T) {
	_, err := windowsPathToCygwinPath(`/workspaces/dev-alchemy`)
	if err == nil {
		t.Fatal("expected windowsPathToCygwinPath to fail for non-windows path")
	}
}

func TestBashSingleQuote_EscapesEmbeddedQuotes(t *testing.T) {
	input := `C:\Users\O'Connor\dev-alchemy`
	got := bashSingleQuote(input)
	want := `'C:\Users\O'"'"'Connor\dev-alchemy'`

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveCygwinBashPath_ConvertsMinttyToBash(t *testing.T) {
	got := resolveCygwinBashPath(`C:\tools\cygwin\bin\mintty.exe`)
	want := `C:\tools\cygwin\bin\bash.exe`

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveCygwinBashPath_LeavesBashPathUntouched(t *testing.T) {
	input := `C:\tools\cygwin\bin\bash.exe`
	got := resolveCygwinBashPath(input)

	if got != input {
		t.Fatalf("expected %q, got %q", input, got)
	}
}

func TestAnsibleColorEnv(t *testing.T) {
	entries := ansibleColorEnv()
	combined := strings.Join(entries, ";")

	for _, required := range []string{"ANSIBLE_FORCE_COLOR=true", "PY_COLORS=1", "TERM=xterm-256color"} {
		if !strings.Contains(combined, required) {
			t.Fatalf("expected env %q in %q", required, combined)
		}
	}
}
