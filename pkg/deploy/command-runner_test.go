package deploy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunCommandWithStreamingLogs_PropagatesCommandFailure(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	err := runCommandWithStreamingLogs(
		t.TempDir(),
		5*time.Second,
		os.Args[0],
		[]string{"-test.run=TestCommandRunnerHelperProcess", "--", "emit-and-fail", "23"},
		"command-runner-test",
	)
	if err == nil {
		t.Fatal("expected runCommandWithStreamingLogs to return an error, got nil")
	}

	if !strings.Contains(err.Error(), "command failed") {
		t.Fatalf("expected wrapped command failure message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "exit status 23") {
		t.Fatalf("expected exit status to be preserved in error, got: %v", err)
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected wrapped *exec.ExitError, got: %T (%v)", err, err)
	}
}

func TestCommandRunnerHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	separatorIndex := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separatorIndex = i
			break
		}
	}
	if separatorIndex < 0 || len(os.Args) <= separatorIndex+2 {
		fmt.Fprintln(os.Stderr, "helper process missing action/exit code")
		os.Exit(2)
	}

	action := os.Args[separatorIndex+1]
	switch action {
	case "emit-and-fail":
		exitCode, err := strconv.Atoi(os.Args[separatorIndex+2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid exit code: %v\n", err)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stdout, "helper stdout line")
		fmt.Fprintln(os.Stderr, "helper stderr line")
		os.Exit(exitCode)
	default:
		fmt.Fprintf(os.Stderr, "unknown helper action: %s\n", action)
		os.Exit(2)
	}
}
