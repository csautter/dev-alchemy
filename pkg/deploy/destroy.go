package deploy

import (
	"fmt"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func SupportsDestroy(config alchemy_build.VirtualMachineConfig) bool {
	switch {
	case isUtmDeployTarget(config):
		return true
	case isTartMacOSDeployTarget(config):
		return true
	case isHypervVagrantTarget(config):
		return true
	default:
		return false
	}
}

func RunDestroy(config alchemy_build.VirtualMachineConfig) error {
	switch {
	case isUtmDeployTarget(config):
		return RunUtmDestroyOnMacOS(config)
	case isTartMacOSDeployTarget(config):
		return RunTartDestroyOnMacOS(config)
	case isHypervVagrantTarget(config):
		return RunHypervVagrantDestroyOnWindows(config)
	default:
		return fmt.Errorf(
			"destroy is not implemented for OS=%s type=%s arch=%s host=%s engine=%s",
			config.OS,
			config.UbuntuType,
			config.Arch,
			config.HostOs,
			config.VirtualizationEngine,
		)
	}
}
