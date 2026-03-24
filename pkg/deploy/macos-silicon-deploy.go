package deploy

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func RunUtmDeployOnMacOS(config alchemy_build.VirtualMachineConfig) error {
	if !isUtmDeployTarget(config) {
		return fmt.Errorf("UTM deploy is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}

	scriptPath := path.Join(alchemy_build.GetDirectoriesInstance().ProjectDir, "deployments/utm/create-utm-vm.sh")
	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir

	vmName := alchemy_build.GetVirtualMachineNameWithType(config)
	args := []string{"--arch", config.Arch, "--os", vmName}

	if err := runCommandWithStreamingLogs(
		projectDir,
		20*time.Minute,
		"bash",
		append([]string{scriptPath}, args...),
		fmt.Sprintf("%s:%s", vmName, config.Arch),
	); err != nil {
		return fmt.Errorf("UTM deploy failed for %s:%s: %w", vmName, config.Arch, err)
	}
	log.Printf("UTM deploy completed for %s:%s", vmName, config.Arch)
	return nil
}

func RunUtmDestroyOnMacOS(config alchemy_build.VirtualMachineConfig) error {
	if !isUtmDeployTarget(config) {
		return fmt.Errorf("UTM destroy is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}

	vmPath, err := utmVirtualMachinePath(config)
	if err != nil {
		return err
	}

	if _, err := os.Stat(vmPath); err != nil {
		if os.IsNotExist(err) {
			log.Printf("UTM VM %q is already absent", vmPath)
			return nil
		}
		return fmt.Errorf("failed to stat UTM VM bundle %q: %w", vmPath, err)
	}

	if err := os.RemoveAll(vmPath); err != nil {
		return fmt.Errorf("failed to remove UTM VM bundle %q: %w", vmPath, err)
	}

	log.Printf("UTM VM bundle removed: %s", vmPath)
	return nil
}

func isUtmDeployTarget(vm alchemy_build.VirtualMachineConfig) bool {
	return vm.HostOs == alchemy_build.HostOsDarwin &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineUtm &&
		(vm.OS == "ubuntu" || vm.OS == "windows11") &&
		(vm.Arch == "amd64" || vm.Arch == "arm64")
}

func utmVirtualMachineName(config alchemy_build.VirtualMachineConfig) string {
	return fmt.Sprintf("%s-%s-dev-alchemy", alchemy_build.GetVirtualMachineNameWithType(config), config.Arch)
}

func utmVirtualMachinePath(config alchemy_build.VirtualMachineConfig) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user home directory for UTM VM path: %w", err)
	}

	return filepath.Join(
		homeDir,
		"Library",
		"Containers",
		"com.utmapp.UTM",
		"Data",
		"Documents",
		utmVirtualMachineName(config)+".utm",
	), nil
}
