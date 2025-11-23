package build

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/KarpelesLab/vncpasswd"
)

type VncRecordingConfig struct {
	OutputFile   string
	OutputFolder string
	Password     string
}

func RunVncSnapshotProcess(vm_config VirtualMachineConfig, ctx context.Context, process_config RunProcessConfig, recording_config *VncRecordingConfig) context.Context {
	// if running on windows, skip vnc snapshot
	if runtime.GOOS == "windows" {
		log.Printf("Skipping vnc snapshot on Windows host.")
		return ctx
	}
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
	// if running on windows, skip ffmpeg video generation
	if runtime.GOOS == "windows" {
		log.Printf("Skipping ffmpeg video generation on Windows host.")
		return ctx
	}
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

func RunQemuUbuntuBuildOnMacOS(config VirtualMachineConfig) error {
	scriptPath := filepath.Join(GetDirectoriesInstance().GetDirectories().ProjectDir, "build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh")
	args := []string{"--project-root", GetDirectoriesInstance().GetDirectories().ProjectDir, "--arch", config.Arch, "--ubuntu-type", config.UbuntuType, "--vnc-port", fmt.Sprintf("%d", config.VncPort), "--headless"}
	return RunBuildScript(config, scriptPath, args)
}

func RunQemuWindowsBuildOnMacOS(config VirtualMachineConfig) error {
	scriptPath := filepath.Join(GetDirectoriesInstance().GetDirectories().ProjectDir, "build/packer/windows/windows11-on-macos.sh")
	args := []string{"--project-root", GetDirectoriesInstance().GetDirectories().ProjectDir, "--arch", config.Arch, "--vnc-port", fmt.Sprintf("%d", config.VncPort), "--headless"}
	return RunBuildScript(config, scriptPath, args)
}
