package deploy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"
)

const (
	scannerInitialBufferSize = 64 * 1024
	scannerMaxBufferSize     = 1024 * 1024
)

func runCommandWithStreamingLogs(workingDir string, timeout time.Duration, executable string, args []string, logPrefix string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = workingDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout for %q: %w", executable, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr for %q: %w", executable, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command %q %v: %w", executable, args, err)
	}

	done := make(chan error, 1)
	var streamsWG sync.WaitGroup

	streamOutput := func(output io.Reader, streamName string) {
		defer streamsWG.Done()
		scanner := bufio.NewScanner(output)
		scanner.Buffer(make([]byte, scannerInitialBufferSize), scannerMaxBufferSize)
		for scanner.Scan() {
			log.Printf("%s %s: %s", logPrefix, streamName, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("%s %s scanner error: %v", logPrefix, streamName, err)
		}
	}

	streamsWG.Add(2)
	go streamOutput(stdout, "stdout")
	go streamOutput(stderr, "stderr")

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		streamsWG.Wait()
		if err != nil {
			return fmt.Errorf("command failed (%s %v): %w", executable, args, err)
		}
		log.Printf("Command finished successfully: %s %v", executable, args)
		return nil
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
		streamsWG.Wait()
		return fmt.Errorf("command terminated due to timeout or interruption (%s %v): %w", executable, args, ctx.Err())
	}
}
