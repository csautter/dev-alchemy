package build

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type VirtualMachineConfig struct {
	OS         string
	Arch       string
	UbuntuType string
	VncPort    int
	Slug       string
}

type RunProcessConfig struct {
	ExecutablePath     string
	Args               []string
	WorkingDir         string
	Timeout            time.Duration
	Context            context.Context
	FailOnError        bool
	Retries            int
	RetryInterval      time.Duration
	InterruptRetryChan chan bool
}

type VncRecordingConfig struct {
	OutputFile   string
	OutputFolder string
}

const vncsnapshot_base_path = "./cache"

func GenerateVirtualMachineSlug(config *VirtualMachineConfig) string {
	if config.Slug != "" {
		return config.Slug
	}

	slug := strings.ToLower(config.OS)
	if config.UbuntuType != "" {
		slug += "-" + strings.ToLower(config.UbuntuType)
	}
	slug += "-" + strings.ToLower(config.Arch)
	config.Slug = slug
	return slug
}

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
			log.Printf("stdout:%s:  %s", config.ExecutablePath, scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("stderr:%s:  %s", config.ExecutablePath, scanner.Text())
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
		log.Printf("Process %s terminated due to timeout or interruption: %v", config.ExecutablePath, ctx.Err())
	case sig := <-sigs:
		_ = cmd.Process.Kill()
		log.Printf("Process %s terminated due to signal: %v", config.ExecutablePath, sig)
	}
	return ctx
}

func RunExternalProcessWithRetries(config RunProcessConfig) context.Context {
	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt <= config.Retries && time.Since(startTime) < config.Timeout; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying process %s (attempt %d/%d) after error: %v", config.ExecutablePath, attempt, config.Retries, lastErr)
			time.Sleep(config.RetryInterval)
		}
		ctx := RunExternalProcess(config)
		if ctx.Err() == nil {
			return ctx
		}
		lastErr = ctx.Err()

		// Check if we received an interrupt signal to stop retries
		if config.InterruptRetryChan != nil {
			select {
			case <-config.InterruptRetryChan:
				log.Printf("Received interrupt signal, stopping retries for process %s", config.ExecutablePath)
				return ctx
			default:
				// No interrupt signal, continue
			}
		}
	}
	log.Printf("Process %s failed after %d attempts: %v", config.ExecutablePath, config.Retries+1, lastErr)
	return context.Background()
}

func RunBashScript(config RunProcessConfig) {
	scriptPath := config.ExecutablePath
	config.ExecutablePath = "bash"
	config.Args = append([]string{scriptPath}, config.Args...)
	RunExternalProcess(config)
}

func RunVncSnapshotProcess(vm_config VirtualMachineConfig, ctx context.Context, process_config RunProcessConfig, recording_config *VncRecordingConfig) context.Context {
	vnc_display := strconv.Itoa(vm_config.VncPort - 5900)

	snapshot_dir := vncsnapshot_base_path + "/" + vm_config.OS + "/qemu-" + vm_config.OS + "-out-" + vm_config.Arch + "/vncsnapshot"
	recording_config.OutputFolder = snapshot_dir

	if err := os.RemoveAll("../../" + snapshot_dir); err != nil {
		log.Fatalf("Failed to remove snapshot directory: %v", err)
	}

	if err := os.MkdirAll("../../"+snapshot_dir, 0755); err != nil {
		log.Fatalf("Failed to create snapshot directory: %v", err)
	}
	snapshot_file := snapshot_dir + "/qemu.vnc.jpg"
	recording_config.OutputFile = snapshot_file

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
	if process_config.InterruptRetryChan != nil {
		config.InterruptRetryChan = process_config.InterruptRetryChan
	}

	ctx = RunExternalProcessWithRetries(config)
	return ctx
}

func RunFfmpegVideoGenerationProcess(vm_config VirtualMachineConfig, ctx context.Context, process_config RunProcessConfig, recording_config *VncRecordingConfig) context.Context {
	// Modify the OutputFile to use ffmpeg's sequence pattern (e.g., qemu.vnc%05d.jpg)
	inputPattern := recording_config.OutputFile
	if len(inputPattern) > 4 && inputPattern[len(inputPattern)-4:] == ".jpg" {
		inputPattern = inputPattern[:len(inputPattern)-4] + "%05d.jpg"
	}

	config := RunProcessConfig{
		ExecutablePath: "ffmpeg",
		Args: []string{
			"-framerate", "1",
			"-i", inputPattern, // e.g., qemu.vnc%05d.jpg
			"-c:v", "libx264",
			"-pix_fmt", "yuv420p",
			recording_config.OutputFolder + "/qemu.vnc.mp4",
		},
		WorkingDir:    "../../",
		Timeout:       10 * time.Minute,
		Context:       ctx,
		Retries:       3,
		RetryInterval: 1 * time.Minute,
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

	ctx = RunExternalProcess(config)
	return ctx
}
