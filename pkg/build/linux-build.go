package build

import (
	"fmt"
	"path/filepath"
)

func runQemuUbuntuBuild(config VirtualMachineConfig, relativeScriptPath string) error {
	scriptPath := filepath.Join(GetDirectoriesInstance().GetDirectories().ProjectDir, relativeScriptPath)
	args := []string{
		scriptPath,
		"--project-root", GetDirectoriesInstance().GetDirectories().ProjectDir,
		"--build-output-dir", getQemuBuildOutputDir(config),
		"--arch", config.Arch,
		"--ubuntu-type", config.UbuntuType,
		"--vnc-port", fmt.Sprintf("%d", config.VncPort),
		"--cpus", getVmCpuCountString(config),
		"--memory", fmt.Sprintf("%d", getVmMemoryMB(config)),
	}
	if config.Headless {
		args = append(args, "--headless")
	}
	return RunBuildScript(config, "bash", args)
}

func RunQemuUbuntuBuildOnLinux(config VirtualMachineConfig) error {
	return runQemuUbuntuBuild(config, "build/packer/linux/ubuntu/linux-ubuntu-on-linux.sh")
}
