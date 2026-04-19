//go:build windows

package build

import (
	"os/exec"
	"time"
)

func configureCommandForCleanup(cmd *exec.Cmd) {}

func commandProcessGroupID(cmd *exec.Cmd) int {
	return 0
}

func terminateProcessGroup(processGroupID int, gracePeriod time.Duration) {}

func restoreInteractiveTerminal() {}
