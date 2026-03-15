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

type hypervVagrantDeploySettings struct {
	BoxName    string
	BoxPath    string
	VagrantDir string
	VagrantEnv []string
}

func RunHypervVagrantDeployOnWindows(config alchemy_build.VirtualMachineConfig) error {
	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	settings, err := resolveHypervVagrantDeploySettings(config, projectDir)
	if err != nil {
		return err
	}

	if err := runCommandWithStreamingLogs(
		projectDir,
		45*time.Minute,
		"vagrant",
		[]string{"box", "add", settings.BoxName, settings.BoxPath, "--provider", "hyperv", "--force"},
		fmt.Sprintf("%s:%s:%s:box-add", config.OS, config.UbuntuType, config.Arch),
	); err != nil {
		return fmt.Errorf("failed to add Vagrant box for %s:%s:%s: %w", config.OS, config.UbuntuType, config.Arch, err)
	}

	if err := runCommandWithStreamingLogsWithEnv(
		settings.VagrantDir,
		45*time.Minute,
		"vagrant",
		[]string{"up", "--provider", "hyperv"},
		settings.VagrantEnv,
		fmt.Sprintf("%s:%s:%s:vagrant-up", config.OS, config.UbuntuType, config.Arch),
	); err != nil {
		return fmt.Errorf("failed to start Vagrant VM for %s:%s:%s: %w", config.OS, config.UbuntuType, config.Arch, err)
	}

	return nil
}

func resolveHypervVagrantDeploySettings(config alchemy_build.VirtualMachineConfig, projectDir string) (hypervVagrantDeploySettings, error) {
	switch config.OS {
	case "windows11":
		return hypervVagrantDeploySettings{
			BoxName:    windowsHypervVagrantBoxName,
			BoxPath:    getHypervWindowsBoxPath(config),
			VagrantDir: filepath.Join(projectDir, "deployments", "vagrant", "ansible-windows"),
		}, nil
	case "ubuntu":
		ubuntuType := config.UbuntuType
		if ubuntuType == "" {
			ubuntuType = "server"
		}
		boxName := fmt.Sprintf("linux-ubuntu-%s-packer", ubuntuType)
		return hypervVagrantDeploySettings{
			BoxName:    boxName,
			BoxPath:    getHypervUbuntuBoxPath(config),
			VagrantDir: filepath.Join(projectDir, "deployments", "vagrant", "linux-ubuntu-hyperv"),
			VagrantEnv: []string{
				"VAGRANT_BOX_NAME=" + boxName,
				"VAGRANT_VM_NAME=" + boxName,
			},
		}, nil
	default:
		return hypervVagrantDeploySettings{}, fmt.Errorf(
			"hyper-v vagrant deploy is not implemented for OS=%s type=%s arch=%s",
			config.OS,
			config.UbuntuType,
			config.Arch,
		)
	}
}

func getHypervWindowsBoxPath(config alchemy_build.VirtualMachineConfig) string {
	if len(config.ExpectedBuildArtifacts) > 0 && config.ExpectedBuildArtifacts[0] != "" {
		return config.ExpectedBuildArtifacts[0]
	}
	return filepath.Join(alchemy_build.GetDirectoriesInstance().CacheDir, "windows11", "hyperv-windows11-amd64.box")
}

func getHypervUbuntuBoxPath(config alchemy_build.VirtualMachineConfig) string {
	if len(config.ExpectedBuildArtifacts) > 0 && config.ExpectedBuildArtifacts[0] != "" {
		return config.ExpectedBuildArtifacts[0]
	}

	ubuntuType := config.UbuntuType
	if ubuntuType == "" {
		ubuntuType = "server"
	}
	return filepath.Join(alchemy_build.GetDirectoriesInstance().CacheDir, "ubuntu", fmt.Sprintf("hyperv-ubuntu-%s-amd64.box", ubuntuType))
}
