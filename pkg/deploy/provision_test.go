package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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

func TestBuildWindowsProvisionArgs(t *testing.T) {
	projectDir := t.TempDir()
	config := windowsAnsibleConnectionConfig{
		User:           "Administrator",
		Password:       "Top$ecret!",
		Connection:     "winrm",
		WinrmTransport: "basic",
		Port:           "5985",
	}

	args, cleanup, err := buildWindowsProvisionArgs(projectDir, "172.25.125.159", config, true)
	if err != nil {
		t.Fatalf("buildWindowsProvisionArgs returned error: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Fatalf("failed to clean up extra vars file: %v", cleanupErr)
		}
	})

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
		if arg == "--extra-vars" {
			extraVarsIndex = index + 1
			break
		}
	}
	if extraVarsIndex <= 0 || extraVarsIndex >= len(args) {
		t.Fatalf("expected --extra-vars with @temp file reference, args: %v", args)
	}
	if strings.Contains(strings.Join(args, " "), "Top$ecret!") {
		t.Fatalf("did not expect password in process arguments, args: %v", args)
	}
	if !strings.HasPrefix(args[extraVarsIndex], "@") {
		t.Fatalf("expected @<file> notation for --extra-vars, got: %q", args[extraVarsIndex])
	}

	extraVarsFilePath := filepath.Join(projectDir, strings.TrimPrefix(args[extraVarsIndex], "@"))
	content, readErr := os.ReadFile(extraVarsFilePath)
	if readErr != nil {
		t.Fatalf("failed to read extra vars file %q: %v", extraVarsFilePath, readErr)
	}

	extraVars := map[string]string{}
	if err := json.Unmarshal(content, &extraVars); err != nil {
		t.Fatalf("expected extra vars file to contain valid JSON, got error: %v", err)
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

	if cleanupErr := cleanup(); cleanupErr != nil {
		t.Fatalf("cleanup failed: %v", cleanupErr)
	}
	if _, statErr := os.Stat(extraVarsFilePath); !os.IsNotExist(statErr) {
		t.Fatalf("expected extra vars temp file to be deleted, stat error: %v", statErr)
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

func TestLoadWindowsUtmAnsibleConnectionConfig_UsesDotEnvValues(t *testing.T) {
	projectDir := t.TempDir()
	dotEnvPath := filepath.Join(projectDir, ".env")

	content := strings.Join([]string{
		utmWindowsAnsibleUserEnvVar + "=Administrator",
		utmWindowsAnsiblePasswordEnvVar + "='P@ssw0rd! with spaces'",
		utmWindowsAnsibleConnectionEnvVar + "=winrm",
		utmWindowsAnsibleWinrmTransportEnvVar + "=basic",
		utmWindowsAnsiblePortEnvVar + "=5985",
		"",
	}, "\n")
	if err := os.WriteFile(dotEnvPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create .env fixture: %v", err)
	}

	connectionConfig, err := loadWindowsUtmAnsibleConnectionConfig(projectDir)
	if err != nil {
		t.Fatalf("loadWindowsUtmAnsibleConnectionConfig returned error: %v", err)
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

func TestLoadWindowsUtmAnsibleConnectionConfig_ReturnsErrorWhenRequiredValuesMissing(t *testing.T) {
	projectDir := t.TempDir()

	_, err := loadWindowsUtmAnsibleConnectionConfig(projectDir)
	if err == nil {
		t.Fatal("expected missing configuration to return an error")
	}
	if !strings.Contains(err.Error(), utmWindowsAnsibleUserEnvVar) {
		t.Fatalf("expected missing user env var name in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), utmWindowsAnsiblePasswordEnvVar) {
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

func TestGetCygwinBashExecutable_UsesConfiguredBashPath(t *testing.T) {
	tempDir := t.TempDir()
	bashPath := filepath.Join(tempDir, "bash.exe")
	if err := os.WriteFile(bashPath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("failed to create fake bash executable: %v", err)
	}

	t.Setenv("CYGWIN_BASH_PATH", bashPath)
	t.Setenv("CYGWIN_TERMINAL_PATH", "")

	got, err := getCygwinBashExecutable()
	if err != nil {
		t.Fatalf("expected configured cygwin bash path to succeed, got error: %v", err)
	}
	if got != bashPath {
		t.Fatalf("expected %q, got %q", bashPath, got)
	}
}

func TestGetCygwinBashExecutable_UsesConfiguredTerminalPath(t *testing.T) {
	tempDir := t.TempDir()
	terminalPath := filepath.Join(tempDir, "mintty.exe")
	bashPath := filepath.Join(tempDir, "bash.exe")
	if err := os.WriteFile(terminalPath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("failed to create fake terminal executable: %v", err)
	}
	if err := os.WriteFile(bashPath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("failed to create fake bash executable: %v", err)
	}

	t.Setenv("CYGWIN_BASH_PATH", "")
	t.Setenv("CYGWIN_TERMINAL_PATH", terminalPath)

	got, err := getCygwinBashExecutable()
	if err != nil {
		t.Fatalf("expected configured cygwin terminal path to succeed, got error: %v", err)
	}
	if got != bashPath {
		t.Fatalf("expected %q, got %q", bashPath, got)
	}
}

func TestGetCygwinBashExecutable_ReturnsErrorForInvalidConfiguredPath(t *testing.T) {
	t.Setenv("CYGWIN_BASH_PATH", filepath.Join(t.TempDir(), "missing-bash.exe"))
	t.Setenv("CYGWIN_TERMINAL_PATH", "")

	_, err := getCygwinBashExecutable()
	if err == nil {
		t.Fatal("expected getCygwinBashExecutable to fail for invalid configured path")
	}
	if !strings.Contains(err.Error(), "CYGWIN_BASH_PATH") {
		t.Fatalf("expected CYGWIN_BASH_PATH in error, got: %v", err)
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

func TestAnsibleRuntimeEnv(t *testing.T) {
	entries := ansibleRuntimeEnv()
	combined := strings.Join(entries, ";")

	for _, required := range []string{"ANSIBLE_FORCE_COLOR=true", "PY_COLORS=1", "TERM=xterm-256color"} {
		if !strings.Contains(combined, required) {
			t.Fatalf("expected env %q in %q", required, combined)
		}
	}

	if runtime.GOOS == "darwin" && !strings.Contains(combined, "OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES") {
		t.Fatalf("expected macOS ansible runtime env to disable fork safety, got %q", combined)
	}
	if runtime.GOOS != "darwin" && strings.Contains(combined, "OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES") {
		t.Fatalf("did not expect macOS-specific env on %s, got %q", runtime.GOOS, combined)
	}
}

func TestExtractLinuxIPv4FromHostOutput(t *testing.T) {
	output := `
default:
  127.0.0.1
  172.24.78.254 172.24.78.255
`

	ip, err := extractLinuxIPv4FromHostOutput(output)
	if err != nil {
		t.Fatalf("expected IP extraction to succeed, got error: %v", err)
	}
	if ip != "172.24.78.254" {
		t.Fatalf("expected 172.24.78.254, got %s", ip)
	}
}

func TestExtractUtmMacAddressFromConfig(t *testing.T) {
	content := `
<dict>
  <key>Network</key>
  <array>
    <dict>
      <key>MacAddress</key>
      <string>A6:1:B:0C:0d:EF</string>
    </dict>
  </array>
</dict>
`

	macAddress, err := extractUtmMacAddressFromConfig(content)
	if err != nil {
		t.Fatalf("expected UTM MAC extraction to succeed, got error: %v", err)
	}
	if macAddress != "a6:01:0b:0c:0d:ef" {
		t.Fatalf("expected normalized UTM MAC address, got %q", macAddress)
	}
}

func TestExtractIPv4ForMacAddress(t *testing.T) {
	output := `
? (127.0.0.1) at 00:00:00:00:00:00 on lo0 ifscope [loopback]
? (192.168.64.21) at a6:1:b:c:d:ef on en0 ifscope [ethernet]
`

	ip, err := extractIPv4ForMacAddress(output, "A6:01:0B:0C:0D:EF")
	if err != nil {
		t.Fatalf("expected ARP IP extraction to succeed, got error: %v", err)
	}
	if ip != "192.168.64.21" {
		t.Fatalf("expected 192.168.64.21, got %s", ip)
	}
}

func TestUtmConfigPlistPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	path, err := utmConfigPlistPath(alchemy_build.VirtualMachineConfig{
		OS:   "windows11",
		Arch: "amd64",
	})
	if err != nil {
		t.Fatalf("utmConfigPlistPath returned error: %v", err)
	}

	expected := filepath.Join(homeDir, "Library", "Containers", "com.utmapp.UTM", "Data", "Documents", "windows11-amd64-dev-alchemy.utm", "config.plist")
	if path != expected {
		t.Fatalf("expected %q, got %q", expected, path)
	}
}

func TestDiscoverUtmVMIPv4_RetriesAfterPrimingArpCache(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	vm := alchemy_build.VirtualMachineConfig{
		OS:   "windows11",
		Arch: "amd64",
	}

	configPath, err := utmConfigPlistPath(vm)
	if err != nil {
		t.Fatalf("utmConfigPlistPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configContent := `
<dict>
  <key>Network</key>
  <array>
    <dict>
      <key>MacAddress</key>
      <string>A6:1:B:0C:0d:EF</string>
    </dict>
  </array>
</dict>
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to create UTM config fixture: %v", err)
	}

	arpOutputs := []string{
		"? (192.168.64.1) at 00:11:22:33:44:55 on en0 ifscope [ethernet]",
		"? (192.168.64.21) at a6:1:b:c:d:ef on en0 ifscope [ethernet]",
	}

	var arpCalls int
	var primeCalls int

	ip, err := discoverUtmVMIPv4WithOptions(t.TempDir(), vm, utmIPv4DiscoveryOptions{
		readFile: os.ReadFile,
		runCommand: func(_ string, _ time.Duration, executable string, args []string) (string, error) {
			if executable != "arp" {
				t.Fatalf("expected arp lookup, got executable %q", executable)
			}
			if len(args) != 1 || args[0] != "-a" {
				t.Fatalf("expected arp -a invocation, got args %v", args)
			}
			if arpCalls >= len(arpOutputs) {
				return arpOutputs[len(arpOutputs)-1], nil
			}
			output := arpOutputs[arpCalls]
			arpCalls++
			return output, nil
		},
		primeARPCache: func() error {
			primeCalls++
			return nil
		},
		sleep:         func(time.Duration) {},
		retryInterval: time.Millisecond,
		maxAttempts:   3,
	})
	if err != nil {
		t.Fatalf("expected UTM IPv4 discovery to succeed after retry, got error: %v", err)
	}
	if ip != "192.168.64.21" {
		t.Fatalf("expected discovered IP 192.168.64.21, got %q", ip)
	}
	if arpCalls != 2 {
		t.Fatalf("expected 2 arp lookups before success, got %d", arpCalls)
	}
	if primeCalls != 1 {
		t.Fatalf("expected ARP cache probe to run once after the first miss, got %d", primeCalls)
	}
}

func TestDiscoverUtmVMIPv4_RePrimesArpCacheUntilMacAddressAppears(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	vm := alchemy_build.VirtualMachineConfig{
		OS:   "windows11",
		Arch: "amd64",
	}

	configPath, err := utmConfigPlistPath(vm)
	if err != nil {
		t.Fatalf("utmConfigPlistPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configContent := `
<dict>
  <key>Network</key>
  <array>
    <dict>
      <key>MacAddress</key>
      <string>A6:1:B:0C:0d:EF</string>
    </dict>
  </array>
</dict>
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to create UTM config fixture: %v", err)
	}

	arpOutputs := []string{
		"? (192.168.64.1) at 00:11:22:33:44:55 on en0 ifscope [ethernet]",
		"? (192.168.64.1) at 00:11:22:33:44:55 on en0 ifscope [ethernet]",
		"? (192.168.64.21) at a6:1:b:c:d:ef on en0 ifscope [ethernet]",
	}

	var arpCalls int
	var primeCalls int

	ip, err := discoverUtmVMIPv4WithOptions(t.TempDir(), vm, utmIPv4DiscoveryOptions{
		readFile: os.ReadFile,
		runCommand: func(_ string, _ time.Duration, executable string, args []string) (string, error) {
			if executable != "arp" {
				t.Fatalf("expected arp lookup, got executable %q", executable)
			}
			if len(args) != 1 || args[0] != "-a" {
				t.Fatalf("expected arp -a invocation, got args %v", args)
			}
			if arpCalls >= len(arpOutputs) {
				return arpOutputs[len(arpOutputs)-1], nil
			}
			output := arpOutputs[arpCalls]
			arpCalls++
			return output, nil
		},
		primeARPCache: func() error {
			primeCalls++
			return nil
		},
		sleep:         func(time.Duration) {},
		retryInterval: time.Millisecond,
		maxAttempts:   4,
	})
	if err != nil {
		t.Fatalf("expected UTM IPv4 discovery to succeed after repeated ARP cache priming, got error: %v", err)
	}
	if ip != "192.168.64.21" {
		t.Fatalf("expected discovered IP 192.168.64.21, got %q", ip)
	}
	if arpCalls != 3 {
		t.Fatalf("expected 3 arp lookups before success, got %d", arpCalls)
	}
	if primeCalls != 2 {
		t.Fatalf("expected ARP cache probe to run after each miss before success, got %d", primeCalls)
	}
}

func TestBuildUbuntuProvisionArgs(t *testing.T) {
	projectDir := t.TempDir()
	config := ubuntuAnsibleConnectionConfig{
		User:           "packer",
		Password:       "P@ssw0rd!",
		BecomePassword: "P@ssw0rd!",
		Connection:     "ssh",
		SshCommonArgs:  "-o StrictHostKeyChecking=no",
		SshTimeout:     "120",
		SshRetries:     "3",
	}

	args, cleanup, err := buildUbuntuProvisionArgs(projectDir, "172.24.78.254", config, true)
	if err != nil {
		t.Fatalf("buildUbuntuProvisionArgs returned error: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Fatalf("failed to clean up extra vars file: %v", cleanupErr)
		}
	})

	if !strings.Contains(strings.Join(args, " "), "-i 172.24.78.254,") {
		t.Fatalf("expected inline inventory with discovered host IP, args: %v", args)
	}
	if !strings.Contains(strings.Join(args, " "), "-l 172.24.78.254") {
		t.Fatalf("expected limit to discovered host IP, args: %v", args)
	}
	if args[len(args)-1] != "--check" {
		t.Fatalf("expected --check to be passed through when requested, args: %v", args)
	}

	extraVarsIndex := -1
	for index, arg := range args {
		if arg == "--extra-vars" {
			extraVarsIndex = index + 1
			break
		}
	}
	if extraVarsIndex <= 0 || extraVarsIndex >= len(args) {
		t.Fatalf("expected --extra-vars with @temp file reference, args: %v", args)
	}
	if strings.Contains(strings.Join(args, " "), "P@ssw0rd!") {
		t.Fatalf("did not expect password in process arguments, args: %v", args)
	}
	if !strings.HasPrefix(args[extraVarsIndex], "@") {
		t.Fatalf("expected @<file> notation for --extra-vars, got: %q", args[extraVarsIndex])
	}

	extraVarsFilePath := filepath.Join(projectDir, strings.TrimPrefix(args[extraVarsIndex], "@"))
	content, readErr := os.ReadFile(extraVarsFilePath)
	if readErr != nil {
		t.Fatalf("failed to read extra vars file %q: %v", extraVarsFilePath, readErr)
	}

	extraVars := map[string]string{}
	if err := json.Unmarshal(content, &extraVars); err != nil {
		t.Fatalf("expected extra vars file to contain valid JSON, got error: %v", err)
	}

	for key, expected := range map[string]string{
		"ansible_user":            "packer",
		"ansible_password":        "P@ssw0rd!",
		"ansible_become_password": "P@ssw0rd!",
		"ansible_connection":      "ssh",
		"ansible_ssh_common_args": "-o StrictHostKeyChecking=no",
		"ansible_ssh_timeout":     "120",
		"ansible_ssh_retries":     "3",
	} {
		if extraVars[key] != expected {
			t.Fatalf("expected %s=%q in extra vars, got: %v", key, expected, extraVars)
		}
	}
}

func TestLoadUbuntuHypervAnsibleConnectionConfig_UsesDefaults(t *testing.T) {
	projectDir := t.TempDir()

	connectionConfig, err := loadUbuntuHypervAnsibleConnectionConfig(projectDir)
	if err != nil {
		t.Fatalf("loadUbuntuHypervAnsibleConnectionConfig returned error: %v", err)
	}

	if connectionConfig.User != "packer" {
		t.Fatalf("expected default user packer, got %q", connectionConfig.User)
	}
	if connectionConfig.Password != "P@ssw0rd!" {
		t.Fatalf("expected default password, got %q", connectionConfig.Password)
	}
	if connectionConfig.BecomePassword != "P@ssw0rd!" {
		t.Fatalf("expected default become password, got %q", connectionConfig.BecomePassword)
	}
	if connectionConfig.Connection != "ssh" {
		t.Fatalf("expected default connection ssh, got %q", connectionConfig.Connection)
	}
}

func TestLoadUbuntuHypervAnsibleConnectionConfig_EnvOverridesDotEnv(t *testing.T) {
	projectDir := t.TempDir()
	dotEnvPath := filepath.Join(projectDir, ".env")

	content := strings.Join([]string{
		hypervUbuntuAnsibleUserEnvVar + "=file-user",
		hypervUbuntuAnsiblePasswordEnvVar + "=file-pass",
		hypervUbuntuAnsibleBecomePasswordEnvVar + "=file-become",
		"",
	}, "\n")
	if err := os.WriteFile(dotEnvPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create .env fixture: %v", err)
	}

	t.Setenv(hypervUbuntuAnsibleUserEnvVar, "env-user")
	t.Setenv(hypervUbuntuAnsiblePasswordEnvVar, "env-pass")
	t.Setenv(hypervUbuntuAnsibleBecomePasswordEnvVar, "env-become")

	connectionConfig, err := loadUbuntuHypervAnsibleConnectionConfig(projectDir)
	if err != nil {
		t.Fatalf("loadUbuntuHypervAnsibleConnectionConfig returned error: %v", err)
	}

	if connectionConfig.User != "env-user" {
		t.Fatalf("expected environment user to override .env, got %q", connectionConfig.User)
	}
	if connectionConfig.Password != "env-pass" {
		t.Fatalf("expected environment password to override .env, got %q", connectionConfig.Password)
	}
	if connectionConfig.BecomePassword != "env-become" {
		t.Fatalf("expected environment become password to override .env, got %q", connectionConfig.BecomePassword)
	}
}

func TestLoadUbuntuUtmAnsibleConnectionConfig_UsesDefaults(t *testing.T) {
	projectDir := t.TempDir()

	connectionConfig, err := loadUbuntuUtmAnsibleConnectionConfig(projectDir)
	if err != nil {
		t.Fatalf("loadUbuntuUtmAnsibleConnectionConfig returned error: %v", err)
	}

	if connectionConfig.User != "packer" {
		t.Fatalf("expected default user packer, got %q", connectionConfig.User)
	}
	if connectionConfig.Password != "P@ssw0rd!" {
		t.Fatalf("expected default password, got %q", connectionConfig.Password)
	}
	if connectionConfig.BecomePassword != "P@ssw0rd!" {
		t.Fatalf("expected default become password, got %q", connectionConfig.BecomePassword)
	}
	if connectionConfig.Connection != "ssh" {
		t.Fatalf("expected default connection ssh, got %q", connectionConfig.Connection)
	}
}

func TestLoadUbuntuUtmAnsibleConnectionConfig_EnvOverridesDotEnv(t *testing.T) {
	projectDir := t.TempDir()
	dotEnvPath := filepath.Join(projectDir, ".env")

	content := strings.Join([]string{
		utmUbuntuAnsibleUserEnvVar + "=file-user",
		utmUbuntuAnsiblePasswordEnvVar + "=file-pass",
		utmUbuntuAnsibleBecomePasswordEnvVar + "=file-become",
		"",
	}, "\n")
	if err := os.WriteFile(dotEnvPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create .env fixture: %v", err)
	}

	t.Setenv(utmUbuntuAnsibleUserEnvVar, "env-user")
	t.Setenv(utmUbuntuAnsiblePasswordEnvVar, "env-pass")
	t.Setenv(utmUbuntuAnsibleBecomePasswordEnvVar, "env-become")

	connectionConfig, err := loadUbuntuUtmAnsibleConnectionConfig(projectDir)
	if err != nil {
		t.Fatalf("loadUbuntuUtmAnsibleConnectionConfig returned error: %v", err)
	}

	if connectionConfig.User != "env-user" {
		t.Fatalf("expected environment user to override .env, got %q", connectionConfig.User)
	}
	if connectionConfig.Password != "env-pass" {
		t.Fatalf("expected environment password to override .env, got %q", connectionConfig.Password)
	}
	if connectionConfig.BecomePassword != "env-become" {
		t.Fatalf("expected environment become password to override .env, got %q", connectionConfig.BecomePassword)
	}
}
