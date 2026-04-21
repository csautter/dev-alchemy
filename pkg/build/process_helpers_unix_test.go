//go:build unix

package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestTerminateProcessGroupKillsOrphanedChildren(t *testing.T) {
	t.Parallel()

	childPIDPath := filepath.Join(t.TempDir(), "child.pid")
	cmd := exec.Command("bash", "-lc", `sleep 30 & printf '%s\n' "$!" > "$1"; wait`, "bash", childPIDPath)
	configureCommandForCleanup(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper process: %v", err)
	}

	processGroupID := commandProcessGroupID(cmd)
	childPID := waitForPIDFile(t, childPIDPath, 5*time.Second)

	terminateProcessGroup(processGroupID, 100*time.Millisecond)

	if err := waitForProcessExit(childPID, 5*time.Second); err != nil {
		t.Fatalf("background child process still running after group termination: %v", err)
	}

	if err := cmd.Wait(); err == nil {
		t.Fatal("expected helper process to be terminated by process group cleanup")
	}
}

func TestConfigureCommandForInteractiveTerminalEnablesForegroundTTYAccess(t *testing.T) {
	t.Parallel()

	tty, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("failed to open test tty surrogate: %v", err)
	}
	defer tty.Close()

	cmd := exec.Command("bash", "-lc", "true")

	restore := configureCommandForInteractiveTerminal(cmd, tty, syscall.Getpgrp())
	if restore == nil {
		t.Fatal("expected restore function")
	}

	if cmd.Stdin != tty {
		t.Fatal("expected command stdin to be attached to the interactive terminal")
	}
	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be initialized")
	}
	if !cmd.SysProcAttr.Setpgid {
		t.Fatal("expected Setpgid to remain enabled for process-group cleanup")
	}
	if !cmd.SysProcAttr.Foreground {
		t.Fatal("expected command to run in the foreground terminal process group")
	}
	ttyFD, ok := fileDescriptorAsInt(tty)
	if !ok {
		t.Fatal("expected test tty file descriptor to fit in int")
	}
	if cmd.SysProcAttr.Ctty != ttyFD {
		t.Fatalf("expected Ctty=%d, got %d", ttyFD, cmd.SysProcAttr.Ctty)
	}
}

func waitForPIDFile(t *testing.T, path string, timeout time.Duration) int {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		content, err := os.ReadFile(path)
		if err == nil {
			pid, parseErr := strconv.Atoi(strings.TrimSpace(string(content)))
			if parseErr != nil {
				t.Fatalf("failed to parse child PID %q: %v", strings.TrimSpace(string(content)), parseErr)
			}
			return pid
		}
		if !os.IsNotExist(err) {
			t.Fatalf("failed to read PID file %q: %v", path, err)
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for PID file %q", path)
	return 0
}

func waitForProcessExit(pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, 0)
		if err != nil {
			if errno, ok := err.(syscall.Errno); ok && errno == syscall.ESRCH {
				return nil
			}
			return fmt.Errorf("failed to probe process %d: %w", pid, err)
		}
		time.Sleep(25 * time.Millisecond)
	}

	return fmt.Errorf("process %d did not exit before timeout", pid)
}
