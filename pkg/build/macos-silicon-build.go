package build

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/KarpelesLab/vncpasswd"
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
	DelayBeforeStart   time.Duration
	Context            context.Context
	FailOnError        bool
	Retries            int
	RetryInterval      time.Duration
	InterruptRetryChan chan bool
}

type VncRecordingConfig struct {
	OutputFile   string
	OutputFolder string
	Password     string
}

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

	if config.DelayBeforeStart > 0 {
		log.Printf("Delaying start of process %s by %s", config.ExecutablePath, config.DelayBeforeStart)
		select {
		case <-time.After(config.DelayBeforeStart):
			// Reset DelayBeforeStart to 0 after the delay
			config.DelayBeforeStart = 0
			// continue
		case sig := <-sigs:
			log.Printf("Process %s start interrupted by signal: %v", config.ExecutablePath, sig)
			return ctx
		case <-ctx.Done():
			log.Printf("Process %s start cancelled due to timeout or interruption: %v", config.ExecutablePath, ctx.Err())
			return ctx
		}
	}

	log.Printf("Starting process: %s %s", config.ExecutablePath, strings.Join(config.Args, " "))

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

	snapshot_dir := filepath.Join(GetDirectoriesInstance().GetDirectories().CacheDir, vm_config.OS, "qemu-out-"+GenerateVirtualMachineSlug(&vm_config)+"-vncsnapshot")
	recording_config.OutputFolder = snapshot_dir

	if err := os.RemoveAll(snapshot_dir); err != nil {
		log.Fatalf("Failed to remove snapshot directory: %v", err)
	}

	if err := os.MkdirAll(snapshot_dir, 0755); err != nil {
		log.Fatalf("Failed to create snapshot directory: %v", err)
	}
	snapshot_file := filepath.Join(snapshot_dir, "qemu.vnc.jpg")
	recording_config.OutputFile = snapshot_file

	vnc_passwd_file := filepath.Join(snapshot_dir, ".packer-qemu.vnc.pass")
	if _, err := os.Stat(vnc_passwd_file); err == nil {
		if err := os.Remove(vnc_passwd_file); err != nil {
			log.Fatalf("Failed to remove existing VNC password file: %v", err)
		}
	}

	config := RunProcessConfig{
		ExecutablePath:   "vncsnapshot",
		Args:             []string{"-quiet", "-passwd", vnc_passwd_file, "-compresslevel", "9", "-count", "21600", "-fps", "1", "localhost:" + vnc_display, snapshot_file},
		WorkingDir:       GetDirectoriesInstance().GetDirectories().ProjectDir,
		Timeout:          10 * time.Minute,
		Context:          ctx,
		Retries:          5,
		RetryInterval:    time.Minute,
		DelayBeforeStart: time.Minute,
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
	if process_config.DelayBeforeStart != 0 {
		config.DelayBeforeStart = process_config.DelayBeforeStart
	}

	// Write VNC password file
	vnc_password := "packer"
	if recording_config.Password != "" {
		vnc_password = recording_config.Password
	}

	encrypted := vncpasswd.Crypt(vnc_password)
	if err := os.WriteFile(vnc_passwd_file, encrypted, 0600); err != nil {
		log.Fatalf("Failed to write VNC password file: %v", err)
	}

	ctx = RunExternalProcessWithRetries(config)

	// Remove VNC password file
	if err := os.Remove(vnc_passwd_file); err != nil {
		log.Printf("Failed to remove VNC password file: %v", err)
	}

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
		WorkingDir:    GetDirectoriesInstance().GetDirectories().ProjectDir,
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

	// remove the snapshot images after video generation
	removePattern := recording_config.OutputFile
	if len(removePattern) > 4 && removePattern[len(removePattern)-4:] == ".jpg" {
		removePattern = removePattern[:len(removePattern)-4] + "*.jpg"
	}

	log.Printf("Removing vncsnapshot images with pattern: %s", removePattern)

	matches, err := filepath.Glob(removePattern)
	if err != nil {
		log.Fatalf("Failed to glob snapshot images for removal: %v", err)
	}
	for _, path := range matches {
		err := os.RemoveAll(path)
		if err != nil {
			log.Printf("Failed to remove snapshot image %v: %v", path, err)
		}
	}

	return ctx
}

func RunQemuUbuntuBuildOnMacOS(config VirtualMachineConfig) {
	scriptPath := filepath.Join(GetDirectoriesInstance().GetDirectories().ProjectDir, "build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh")
	args := []string{"--project-root", GetDirectoriesInstance().GetDirectories().ProjectDir, "--arch", config.Arch, "--ubuntu-type", config.UbuntuType, "--vnc-port", fmt.Sprintf("%d", config.VncPort), "--headless"}
	RunBuildScript(config, scriptPath, args)
}

func RunQemuWindowsBuildOnMacOS(config VirtualMachineConfig) {
	scriptPath := filepath.Join(GetDirectoriesInstance().GetDirectories().ProjectDir, "build/packer/windows/windows11-on-macos.sh")
	args := []string{"--arch", config.Arch, "--vnc-port", fmt.Sprintf("%d", config.VncPort), "--headless"}
	RunBuildScript(config, scriptPath, args)
}

func RunBuildScript(config VirtualMachineConfig, scriptPath string, args []string) {
	// Set a timeout for the script execution (adjust as needed)
	timeout := 120 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(sigs)

	// print current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	log.Printf("Current working directory: %s", cwd)
	cmd := exec.CommandContext(ctx, "bash", append([]string{scriptPath}, args...)...)
	cmd.Dir = GetDirectoriesInstance().GetDirectories().ProjectDir

	// VNC integration:
	// - Opening a VNC viewer (Screen Sharing) is useful for observing the VM build process in real time.
	// - VNC recording enables capturing the build process for later review or debugging.
	vnc_recording_config := VncRecordingConfig{Password: "packer"}
	go func() {
		config := RunProcessConfig{
			ExecutablePath:   "open",
			Args:             []string{"-a", "Screen Sharing", fmt.Sprintf("vnc://:%s@localhost:%d", vnc_recording_config.Password, config.VncPort)},
			Timeout:          5 * time.Minute,
			WorkingDir:       "",
			Context:          ctx,
			FailOnError:      false,
			Retries:          5,
			RetryInterval:    time.Minute,
			DelayBeforeStart: time.Minute,
		}
		RunExternalProcessWithRetries(config)
	}()

	// Start Screen Capture to record the VM build process
	var vnc_snapshot_ctx context.Context
	vnc_snapshot_done := make(chan struct{})
	vnc_interrupt_retry_chan := make(chan bool)
	go func() {
		vnc_snapshot_ctx = RunVncSnapshotProcess(config, ctx, RunProcessConfig{Timeout: timeout, InterruptRetryChan: vnc_interrupt_retry_chan}, &vnc_recording_config)
		if vnc_snapshot_ctx != nil {
			<-vnc_snapshot_ctx.Done()
		}
		close(vnc_snapshot_done)
	}()

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

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Printf("%s:%s stdout:  %s", config.UbuntuType, config.Arch, scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("%s:%s stderr:  %s", config.UbuntuType, config.Arch, scanner.Text())
		}
	}()

	done := make(chan error, 1)

	go func() {
		err := cmd.Wait()
		if vnc_snapshot_ctx != nil {
			vnc_snapshot_ctx.Done()
		}
		done <- err
	}()

	// FFmpeg integration:
	// - FFmpeg is useful for generating a video from the VNC recording, allowing playback and sharing of the build process.
	var ffmpeg_run = func() {
		// Wait for vnc_snapshot to finish
		_, ok := <-vnc_snapshot_done
		if !ok {
			// Channel is closed, proceed
		} else {
			// Channel not closed, wait for it
			<-vnc_snapshot_done
		}
		// Always run ffmpeg after vnc_snapshot is done
		RunFfmpegVideoGenerationProcess(config, ctx, RunProcessConfig{Timeout: 10 * time.Minute}, &vnc_recording_config)
	}

	select {
	case err := <-done:
		vnc_interrupt_retry_chan <- true
		if err != nil {
			ffmpeg_run()
			log.Fatalf("Script failed: %v", err)
		}
		ffmpeg_run()
		log.Printf("Script finished successfully.")
	case <-ctx.Done():
		// Kill the process if context is done (timeout or cancellation)
		_ = cmd.Process.Kill()
		vnc_interrupt_retry_chan <- true
		ffmpeg_run()
		log.Fatalf("Script terminated due to timeout or interruption: %v", ctx.Err())
	case sig := <-sigs:
		_ = cmd.Process.Kill()
		vnc_interrupt_retry_chan <- true
		ffmpeg_run()
		log.Fatalf("Script terminated due to signal: %v", sig)
	}

	// TODO: check for vnc recording files and video generation success
}
