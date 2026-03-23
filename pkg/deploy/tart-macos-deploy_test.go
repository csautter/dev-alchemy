package deploy

import (
	"errors"
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

func TestWaitForTartVMToBecomeReachableWithOptions_ReturnsEarlyLogFailure(t *testing.T) {
	ip, err := waitForTartVMToBecomeReachableWithOptions(
		t.TempDir(),
		"tahoe-base-alchemy",
		tartDetachedRun{
			pid:     4242,
			logPath: "/tmp/tahoe-base-alchemy.log",
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
			readLogSummary: func(path string) string {
				if path != "/tmp/tahoe-base-alchemy.log" {
					t.Fatalf("unexpected log path %q", path)
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
