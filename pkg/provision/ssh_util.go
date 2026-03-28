package provision

import (
	"fmt"
	"net"
	"time"
)

const (
	sshPortWaitWindow   = 5 * time.Minute
	sshPortWaitInterval = 2 * time.Second
	sshPort             = 22
)

func waitForSSHPort(ip string) error {
	deadline := time.Now().Add(sshPortWaitWindow)
	var lastErr error

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, fmt.Sprintf("%d", sshPort)), 5*time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err
		time.Sleep(sshPortWaitInterval)
	}

	return fmt.Errorf("SSH on %s:%d did not become reachable within %s: %w", ip, sshPort, sshPortWaitWindow, lastErr)
}
