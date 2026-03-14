package deploy

import (
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

func TestUpsertWindowsHypervInventory_ReplacesExistingHostIP(t *testing.T) {
	tempDir := t.TempDir()
	inventoryPath := filepath.Join(tempDir, "hyperv_windows_winrm.yml")

	initial := `all:
    children:
        windows:
            hosts:
                windows_host:
                    ansible_host: 10.0.0.5
                    ansible_user: Administrator
`

	if err := os.WriteFile(inventoryPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("failed to write initial inventory: %v", err)
	}

	if err := upsertWindowsHypervInventory(inventoryPath, "172.25.125.159"); err != nil {
		t.Fatalf("upsertWindowsHypervInventory returned error: %v", err)
	}

	content, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("failed to read updated inventory: %v", err)
	}

	updated := string(content)
	if !strings.Contains(updated, "ansible_host: 172.25.125.159") {
		t.Fatalf("expected updated inventory to contain new ansible_host, content: %s", updated)
	}
	if strings.Contains(updated, "ansible_host: 10.0.0.5") {
		t.Fatalf("expected old ansible_host value to be replaced, content: %s", updated)
	}
}

func TestUpsertWindowsHypervInventory_CreatesDefaultWhenMissing(t *testing.T) {
	tempDir := t.TempDir()
	inventoryPath := filepath.Join(tempDir, "hyperv_windows_winrm.yml")

	if err := upsertWindowsHypervInventory(inventoryPath, "172.25.125.159"); err != nil {
		t.Fatalf("upsertWindowsHypervInventory returned error: %v", err)
	}

	content, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("failed to read created inventory: %v", err)
	}

	created := string(content)
	if !strings.Contains(created, "windows_host:") {
		t.Fatalf("expected default inventory to define windows_host, content: %s", created)
	}
	if !strings.Contains(created, "ansible_host: 172.25.125.159") {
		t.Fatalf("expected default inventory to contain injected ip, content: %s", created)
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
