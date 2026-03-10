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
			panic(fmt.Sprintf("Command failed (%s %v): %s", executable, args, err.Error()))
		}
		log.Printf("Command finished successfully: %s %v", executable, args)
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		streamsWG.Wait()
		panic("Command terminated due to timeout or interruption: " + ctx.Err().Error())
	}
}
