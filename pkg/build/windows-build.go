package build

import "fmt"

const (
	packerExecutable     = "packer"
	windows11ISOPath     = "./cache/windows11/iso/win11_25h2_english_amd64.iso"
	hypervPackerFile     = "build/packer/windows/windows11-on-windows-hyperv.pkr.hcl"
	virtualBoxPackerFile = "build/packer/windows/windows11-on-windows-virtualbox.pkr.hcl"
)

// RunHypervWindowsBuildOnWindows builds a Windows 11 VM using Hyper-V on Windows.
func RunHypervWindowsBuildOnWindows(config VirtualMachineConfig) error {
	if err := createHypervTempDir(GetDirectoriesInstance()); err != nil {
		return fmt.Errorf("failed to create hyperv temp directory: %w", err)
	}
	return runWindowsBuild(config, hypervPackerFile)
}

// RunVirtualBoxWindowsBuildOnWindows builds a Windows 11 VM using VirtualBox on Windows.
func RunVirtualBoxWindowsBuildOnWindows(config VirtualMachineConfig) error {
	return runWindowsBuild(config, virtualBoxPackerFile)
}

// runWindowsBuild executes the Packer build process for Windows VMs.
func runWindowsBuild(config VirtualMachineConfig, packerFile string) error {
	// Initialize Packer with the specified configuration file
	if err := initializePacker(packerFile); err != nil {
		return fmt.Errorf("failed to initialize packer: %w", err)
	}

	// Build the Packer arguments
	args := buildPackerArgs(config, packerFile)

	// Execute the build
	return RunBuildScript(config, packerExecutable, args)
}

// initializePacker runs the packer init command for the given file.
func initializePacker(packerFile string) error {
	RunCliCommand(GetDirectoriesInstance().ProjectDir, packerExecutable, []string{"init", packerFile})
	return nil
}

// buildPackerArgs constructs the command-line arguments for the Packer build command.
func buildPackerArgs(config VirtualMachineConfig, packerFile string) []string {
	args := []string{"build"}

	// Add temp disk path if configured
	if tempDiskPath := getTempDiskPathForHypervBuild(); tempDiskPath != "" {
		args = append(args, "-var", fmt.Sprintf("temp_disk_path=%s", tempDiskPath))
	}

	// Add ISO URL, CPU count, memory, and Packer file
	args = append(args,
		"-var", fmt.Sprintf("iso_url=%s", windows11ISOPath),
		"-var", fmt.Sprintf("cpus=%s", getVmCpuCountString(config)),
		"-var", fmt.Sprintf("memory=%d", getVmMemoryMB(config)),
		packerFile,
	)

	return args
}
