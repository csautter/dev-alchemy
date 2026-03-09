package deploy

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"
)

func runCommandWithStreamingLogs(workingDir string, timeout time.Duration, executable string, args []string, logPrefix string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = workingDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic("Failed to get stdout: " + err.Error())
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic("Failed to get stderr: " + err.Error())
	}

	if err := cmd.Start(); err != nil {
		panic("Failed to start command: " + err.Error())
	}

	done := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Printf("%s stdout: %s", logPrefix, scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("%s stderr: %s", logPrefix, scanner.Text())
		}
	}()

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			panic(fmt.Sprintf("Command failed (%s %v): %s", executable, args, err.Error()))
		}
		log.Printf("Command finished successfully: %s %v", executable, args)
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		panic("Command terminated due to timeout or interruption: " + ctx.Err().Error())
	}
}
