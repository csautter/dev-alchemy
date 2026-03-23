package deploy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseBackgroundPID(t *testing.T) {
	pid, err := parseBackgroundPID("12345\n")
	if err != nil {
		t.Fatalf("expected PID parsing to succeed, got error: %v", err)
	}
	if pid != 12345 {
		t.Fatalf("expected pid 12345, got %d", pid)
	}
}

func TestTartListIncludesLocalVM_UsesNamedColumns(t *testing.T) {
	output := `
NAME                    STATUS    SOURCE
sonoma-base-alchemy     stopped   remote
tahoe-base-alchemy      running   local
`

	if !tartListIncludesLocalVM(output, "tahoe-base-alchemy") {
		t.Fatal("expected Tart list parser to find local VM by named columns")
	}
}

func TestTartListLocalVMState_TracksRunningStatusFromNamedColumns(t *testing.T) {
	output := `
NAME                    STATUS    SOURCE
sonoma-base-alchemy     stopped   remote
tahoe-base-alchemy      running   local
`

	state := tartListLocalVMState(output, "tahoe-base-alchemy")
	if !state.exists {
		t.Fatal("expected Tart list parser to find local VM")
	}
	if !state.running {
		t.Fatal("expected Tart list parser to mark running VM as running")
	}
}

func TestTartListIncludesLocalVM_DoesNotMatchSubstringNames(t *testing.T) {
	output := `
local tahoe-base-alchemy-old running
local tahoe-base-alchemy-copy stopped
`

	if tartListIncludesLocalVM(output, "tahoe-base-alchemy") {
		t.Fatal("did not expect Tart list parser to match substring VM names")
	}
}

func TestTartListIncludesLocalVM_FallsBackToCurrentTwoFieldLayout(t *testing.T) {
	output := `
local tahoe-base-alchemy running
remote sonoma-base-alchemy stopped
`

	if !tartListIncludesLocalVM(output, "tahoe-base-alchemy") {
		t.Fatal("expected Tart list parser to support the current Tart list layout")
	}
}

func TestTartListLocalVMState_FallsBackToCurrentLayoutStatus(t *testing.T) {
	output := `
local tahoe-base-alchemy stopped
remote sonoma-base-alchemy running
`

	state := tartListLocalVMState(output, "tahoe-base-alchemy")
	if !state.exists {
		t.Fatal("expected Tart list parser to find local VM in fallback layout")
	}
	if state.running {
		t.Fatal("expected stopped fallback-layout VM to not be marked as running")
	}
}

func TestWaitForTartVMToBecomeReachableWithOptions_ReturnsEarlyLogFailure(t *testing.T) {
	ip, err := waitForTartVMToBecomeReachableWithOptions(
		t.TempDir(),
		"tahoe-base-alchemy",
		tartDetachedRun{
			pid:         4242,
			logDir:      "/tmp",
			logFileName: "tahoe-base-alchemy.log",
		},
		tartReachabilityWaitOptions{
			detectIPv4: func() (string, error) {
				return "", errors.New("ip not assigned yet")
			},
			isProcessRunning: func(pid int) (bool, error) {
				if pid != 4242 {
					t.Fatalf("expected pid 4242, got %d", pid)
				}
				return false, nil
			},
			readLogSummary: func(logDir string, logFileName string) string {
				if logDir != "/tmp" {
					t.Fatalf("unexpected log dir %q", logDir)
				}
				if logFileName != "tahoe-base-alchemy.log" {
					t.Fatalf("unexpected log file name %q", logFileName)
				}
				return "The number of VMs exceeds the system limit"
			},
			sleep:         func(time.Duration) {},
			retryInterval: time.Millisecond,
			maxAttempts:   2,
		},
	)
	if err == nil {
		t.Fatal("expected early tart run failure to return an error")
	}
	if ip != "" {
		t.Fatalf("expected empty IP on failure, got %q", ip)
	}
	if !strings.Contains(err.Error(), "exited early") {
		t.Fatalf("expected early-exit error, got %v", err)
	}
	if !strings.Contains(err.Error(), "system limit") {
		t.Fatalf("expected tart log summary in error, got %v", err)
	}
}

func TestTartRunLogFileName_SanitizesVMName(t *testing.T) {
	got := tartRunLogFileName("../Tahoe Base/Alchemy")

	if got != "Tahoe_Base_Alchemy.log" {
		t.Fatalf("expected sanitized log filename, got %q", got)
	}
}

func TestReadTartRunLogSummary_ReadsScopedLogFile(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "vm.log")
	if err := os.WriteFile(logPath, []byte("line1\nline2\nline3\n"), 0o600); err != nil {
		t.Fatalf("failed to write test log: %v", err)
	}

	got := readTartRunLogSummary(logDir, "vm.log")
	if got != "line1 | line2 | line3" {
		t.Fatalf("expected joined log summary, got %q", got)
	}
}

func TestReadTartRunLogSummary_RejectsTraversal(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "logs")
	if err := os.Mkdir(logDir, 0o750); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	outsideLogPath := filepath.Join(tempDir, "outside.log")
	if err := os.WriteFile(outsideLogPath, []byte("outside"), 0o600); err != nil {
		t.Fatalf("failed to write outside log: %v", err)
	}

	got := readTartRunLogSummary(logDir, "../outside.log")
	if got != "" {
		t.Fatalf("expected traversal attempt to be rejected, got %q", got)
	}
}
