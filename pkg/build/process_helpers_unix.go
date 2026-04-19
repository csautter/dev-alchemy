//go:build unix

package build

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func configureCommandForCleanup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func commandProcessGroupID(cmd *exec.Cmd) int {
	if cmd == nil || cmd.Process == nil {
		return 0
	}
	return cmd.Process.Pid
}

func terminateProcessGroup(processGroupID int, gracePeriod time.Duration) {
	if processGroupID <= 0 {
		return
	}
	if !signalProcessGroup(processGroupID, syscall.SIGTERM) {
		return
	}
	if gracePeriod > 0 {
		time.Sleep(gracePeriod)
	}
	_ = signalProcessGroup(processGroupID, syscall.SIGKILL)
}

func signalProcessGroup(processGroupID int, signal syscall.Signal) bool {
	err := syscall.Kill(-processGroupID, signal)
	return err == nil || !errors.Is(err, syscall.ESRCH)
}

func restoreInteractiveTerminal() {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return
	}
	defer tty.Close()

	cmd := exec.Command("stty", "sane", "echo")
	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty
	_ = cmd.Run()
}
