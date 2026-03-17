package build

import (
	"fmt"
)

const (
	ubuntuHypervPackerFile = "build/packer/linux/ubuntu/linux-ubuntu-hyperv.pkr.hcl"
	ubuntuServerISOPath    = "./cache/linux/ubuntu-24.04.3-live-server-amd64.iso"
)

// RunHypervUbuntuBuildOnWindows builds an Ubuntu VM using Hyper-V on Windows.
func RunHypervUbuntuBuildOnWindows(config VirtualMachineConfig) error {
	if err := initializePacker(ubuntuHypervPackerFile); err != nil {
		return fmt.Errorf("failed to initialize packer: %w", err)
	}

	args := []string{
		"build",
		"-var", fmt.Sprintf("iso_url=%s", ubuntuServerISOPath),
		"-var", fmt.Sprintf("ubuntu_type=%s", defaultUbuntuType(config.UbuntuType)),
		"-var", fmt.Sprintf("cpus=%s", getVmCpuCountString(config)),
		"-var", fmt.Sprintf("memory=%d", getVmMemoryMB(config)),
		ubuntuHypervPackerFile,
	}

	return RunBuildScript(config, packerExecutable, args)
}

func defaultUbuntuType(ubuntuType string) string {
	if ubuntuType == "" {
		return "server"
	}
	return ubuntuType
}
