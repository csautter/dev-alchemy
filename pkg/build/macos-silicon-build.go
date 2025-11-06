package build

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type VirtualMachineConfig struct {
	OS         string
	Arch       string
	UbuntuType string
	VncPort    int
}

type RunProcessConfig struct {
	ExecutablePath string
	Args           []string
	WorkingDir     string
	Timeout        time.Duration
	Context        context.Context
	FailOnError    bool
	Retries        int
	RetryInterval  time.Duration
}

const vncsnapshot_base_path = "./cache"

func RunExternalProcess(config RunProcessConfig) context.Context {
	var ctx context.Context
	var cancel context.CancelFunc
	if config.Context != nil {
		ctx, cancel = context.WithTimeout(config.Context, config.Timeout)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), config.Timeout)
	}
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(sigs)

	cmd := exec.CommandContext(ctx, config.ExecutablePath, config.Args...)
	cmd.Dir = config.WorkingDir

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
			log.Printf("stdout:  %s", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("stderr: %s", scanner.Text())
		}
	}()

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			if config.FailOnError {
				log.Fatalf("Process %s failed: %v", config.ExecutablePath, err)
			}
			log.Printf("Process %s finished with error: %v", config.ExecutablePath, err)
			return ctx
		}
		log.Printf("Process %s finished successfully.", config.ExecutablePath)
	case <-ctx.Done():
		// Kill the process if context is done (timeout or cancellation)
		_ = cmd.Process.Kill()
		log.Fatalf("Process %s terminated due to timeout or interruption: %v", config.ExecutablePath, ctx.Err())
	case sig := <-sigs:
		_ = cmd.Process.Kill()
		log.Fatalf("Process %s terminated due to signal: %v", config.ExecutablePath, sig)
	}
	return ctx
}

func RunBashScript(config RunProcessConfig) {
	scriptPath := config.ExecutablePath
	config.ExecutablePath = "bash"
	config.Args = append([]string{scriptPath}, config.Args...)
	RunExternalProcess(config)
}

func RunVncSnapshotProcess(vm_config VirtualMachineConfig, ctx context.Context, process_config RunProcessConfig) context.Context {
	vnc_display := strconv.Itoa(vm_config.VncPort - 5900)

	snapshot_dir := vncsnapshot_base_path + "/" + vm_config.OS + "/qemu-" + vm_config.OS + "-out-" + vm_config.Arch + "/vncsnapshot"
	if err := os.MkdirAll("../../"+snapshot_dir, 0755); err != nil {
		log.Fatalf("Failed to create snapshot directory: %v", err)
	}
	snapshot_file := snapshot_dir + "/qemu.vnc.jpg"

	config := RunProcessConfig{
		ExecutablePath: "vncsnapshot",
		Args:           []string{"-quiet", "-passwd", "./build/packer/windows/.build_tmp/packer-qemu.vnc.pass", "-compresslevel", "9", "-count", "21600", "-fps", "1", "localhost:" + vnc_display, snapshot_file},
		WorkingDir:     "../../",
		Timeout:        10 * time.Minute,
		Context:        ctx,
		Retries:        5,
		RetryInterval:  time.Minute,
	}

	// Overwrite config fields with process_config if set (non-zero values)
	if process_config.ExecutablePath != "" {
		config.ExecutablePath = process_config.ExecutablePath
	}
	if len(process_config.Args) > 0 {
		config.Args = process_config.Args
	}
	if process_config.WorkingDir != "" {
		config.WorkingDir = process_config.WorkingDir
	}
	if process_config.Timeout != 0 {
		config.Timeout = process_config.Timeout
	}
	if process_config.Context != nil {
		config.Context = process_config.Context
	}
	if process_config.FailOnError {
		config.FailOnError = process_config.FailOnError
	}
	if process_config.Retries != 0 {
		config.Retries = process_config.Retries
	}
	if process_config.RetryInterval != 0 {
		config.RetryInterval = process_config.RetryInterval
	}
	startTime := time.Now()

	var lastErr error
	for retries := 0; retries < config.Retries && time.Since(startTime) < config.Timeout; retries++ {
		ctx = RunExternalProcess(config)
		// Check if context was done due to error or timeout
		if ctx.Err() == nil {
			// Success
			return ctx
		}
		lastErr = ctx.Err()
		log.Printf("VNC snapshot process failed (attempt %d/%d): %v. Retrying in %v...", retries+1, config.Retries, lastErr, config.RetryInterval)
		time.Sleep(config.RetryInterval)
	}
	log.Printf("VNC snapshot process failed after %d retries or %v: %v", config.Retries, config.Timeout, lastErr)
	return ctx
}
