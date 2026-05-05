package build

import (
	"fmt"
	"path/filepath"
)

func RunQemuUbuntuBuildOnMacOS(config VirtualMachineConfig) error {
	return runQemuUbuntuBuild(config, "build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh")
}

func RunQemuWindowsBuildOnMacOS(config VirtualMachineConfig) error {
	return runQemuWindowsBuild(config, "build/packer/windows/windows11-on-macos.sh")
}

func runQemuWindowsBuild(config VirtualMachineConfig, relativeScriptPath string) error {
	config, err := withStagedBuildArtifactsForNoCache(config)
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(GetDirectoriesInstance().GetDirectories().ProjectDir, relativeScriptPath)
	args := []string{
		scriptPath,
		"--project-root", GetDirectoriesInstance().GetDirectories().ProjectDir,
		"--build-output-dir", getQemuBuildOutputDir(config),
		"--arch", config.Arch,
		"--vnc-port", fmt.Sprintf("%d", config.VncPort),
		"--cpus", getVmCpuCountString(config),
		"--memory", fmt.Sprintf("%d", getVmMemoryMB(config)),
	}
	if stagedArtifact, ok := firstStagedBuildArtifact(config); ok {
		args = append(args, "--artifact-output-path", stagedArtifact)
	}
	if config.Headless {
		args = append(args, "--headless")
	}
	if config.Verbose {
		args = append(args, "--verbose")
	}
	return RunBuildScript(config, "bash", args)
}
