package deploy

import (
	"fmt"
	"log"
	"path"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func RunUtmDeployOnMacOS(config alchemy_build.VirtualMachineConfig) error {
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
