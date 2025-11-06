package build

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type VirtualMachineConfig struct {
	OS         string
	Arch       string
	UbuntuType string
	VncPort    int
}

func RunExternalProcess(executablePath string, args ...string) {
	// Set a timeout for the script execution (adjust as needed)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Minute)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(sigs)

	cmd := exec.CommandContext(ctx, executablePath, args...)
	cmd.Dir = "../../"

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to get stderr: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start command: %v", err)
	}

	done := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Printf("%s:%s stdout:  %s", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Fatalf("%s", scanner.Text())
		}
	}()

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Fatalf("Script failed: %v", err)
		}
		log.Printf("Script finished successfully.")
	case <-ctx.Done():
		// Kill the process if context is done (timeout or cancellation)
		_ = cmd.Process.Kill()
		log.Fatalf("Script terminated due to timeout or interruption: %v", ctx.Err())
	case sig := <-sigs:
		_ = cmd.Process.Kill()
		log.Fatalf("Script terminated due to signal: %v", sig)
	}
}

func RunBashScript(scriptPath string, args ...string) {
	// Set a timeout for the script execution (adjust as needed)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Minute)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(sigs)

	cmd := exec.CommandContext(ctx, "bash", append([]string{scriptPath}, args...)...)
	cmd.Dir = "../../"

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to get stderr: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start command: %v", err)
	}

	done := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Printf("%s:%s stdout:  %s", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Fatalf("%s", scanner.Text())
		}
	}()

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Fatalf("Script failed: %v", err)
		}
		log.Printf("Script finished successfully.")
	case <-ctx.Done():
		// Kill the process if context is done (timeout or cancellation)
		_ = cmd.Process.Kill()
		log.Fatalf("Script terminated due to timeout or interruption: %v", ctx.Err())
	case sig := <-sigs:
		_ = cmd.Process.Kill()
		log.Fatalf("Script terminated due to signal: %v", sig)
	}
}
