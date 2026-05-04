package build

import (
	"fmt"
)

const (
	ubuntuHypervPackerFile = "build/packer/linux/ubuntu/linux-ubuntu-hyperv.pkr.hcl"
)

// RunHypervUbuntuBuildOnWindows builds an Ubuntu VM using Hyper-V on Windows.
func RunHypervUbuntuBuildOnWindows(config VirtualMachineConfig) error {
	if err := initializePacker(config, ubuntuHypervPackerFile); err != nil {
		return fmt.Errorf("failed to initialize packer: %w", err)
	}

	args := []string{
		"build",
		"-var", fmt.Sprintf("iso_url=%s", ubuntuLiveServerISOPath("amd64", ubuntuLiveServerAMD64Version)),
		"-var", fmt.Sprintf("iso_checksum=sha256:%s", ubuntuLiveServerAMD64SHA256),
		"-var", fmt.Sprintf("cache_dir=%s", GetDirectoriesInstance().CacheDir),
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
