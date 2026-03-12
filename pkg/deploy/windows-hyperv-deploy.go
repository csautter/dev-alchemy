package deploy

import (
	"fmt"
	"path/filepath"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

const (
	windowsHypervVagrantBoxName = "win11-packer"
)

func RunHypervVagrantDeployOnWindows(config alchemy_build.VirtualMachineConfig) error {
	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	vagrantDir := filepath.Join(projectDir, "deployments", "vagrant", "ansible-windows")
	boxPath := getHypervWindowsBoxPath(config)

	if err := runCommandWithStreamingLogs(
		projectDir,
		45*time.Minute,
		"vagrant",
		[]string{"box", "add", windowsHypervVagrantBoxName, boxPath, "--provider", "hyperv", "--force"},
		fmt.Sprintf("%s:%s", config.OS, config.Arch),
	); err != nil {
		return fmt.Errorf("failed to add Vagrant box for %s:%s: %w", config.OS, config.Arch, err)
	}

	if err := runCommandWithStreamingLogs(
		vagrantDir,
		45*time.Minute,
		"vagrant",
		[]string{"up", "--provider", "hyperv"},
		fmt.Sprintf("%s:%s", config.OS, config.Arch),
	); err != nil {
		return fmt.Errorf("failed to start Vagrant VM for %s:%s: %w", config.OS, config.Arch, err)
	}

	return nil
}

func getHypervWindowsBoxPath(config alchemy_build.VirtualMachineConfig) string {
	if len(config.ExpectedBuildArtifacts) > 0 && config.ExpectedBuildArtifacts[0] != "" {
		return config.ExpectedBuildArtifacts[0]
	}
	return filepath.Join(alchemy_build.GetDirectoriesInstance().CacheDir, "windows11", "hyperv-windows11-amd64.box")
}
