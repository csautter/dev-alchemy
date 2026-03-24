package deploy

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

const (
	utmAutomationCommandTimeout = time.Minute
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

func RunUtmStartOnMacOS(config alchemy_build.VirtualMachineConfig) error {
	if !isUtmDeployTarget(config) {
		return fmt.Errorf("UTM start is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}

	state, err := inspectUtmStartTarget(config)
	if err != nil {
		return err
	}
	if !state.Exists {
		return fmt.Errorf("UTM VM %q does not exist. Run `alchemy create %s` first", utmVirtualMachineName(config), startCommandArguments(config))
	}
	if state.Running {
		log.Printf("UTM VM %q is already %s", utmVirtualMachineName(config), state.State)
		return nil
	}

	if _, err := runUtmAppleScript(
		[]string{
			`tell application "UTM"`,
			fmt.Sprintf(`set targetVM to virtual machine named %q`, utmVirtualMachineName(config)),
			`start targetVM`,
			`end tell`,
		},
	); err != nil {
		return fmt.Errorf("failed to start UTM VM %q: %w", utmVirtualMachineName(config), err)
	}

	log.Printf("UTM VM %q start requested", utmVirtualMachineName(config))
	return nil
}

func inspectUtmStartTarget(config alchemy_build.VirtualMachineConfig) (StartTargetState, error) {
	vmPath, err := utmVirtualMachinePath(config)
	if err != nil {
		return StartTargetState{}, err
	}

	if _, err := os.Stat(vmPath); err != nil {
		if os.IsNotExist(err) {
			return StartTargetState{State: "missing"}, nil
		}
		return StartTargetState{}, fmt.Errorf("failed to stat UTM VM bundle %q: %w", vmPath, err)
	}

	status, err := utmVirtualMachineStatus(config)
	if err != nil {
		return StartTargetState{}, err
	}

	return StartTargetState{
		Exists:  true,
		Running: utmStatusIndicatesRunning(status),
		State:   status,
	}, nil
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

func utmVirtualMachineStatus(config alchemy_build.VirtualMachineConfig) (string, error) {
	output, err := runUtmAppleScript(
		[]string{
			`tell application "UTM"`,
			fmt.Sprintf(`set targetVM to virtual machine named %q`, utmVirtualMachineName(config)),
			`return (status of targetVM) as text`,
			`end tell`,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to query status of UTM VM %q: %w", utmVirtualMachineName(config), err)
	}

	return strings.ToLower(strings.TrimSpace(output)), nil
}

func utmStatusIndicatesRunning(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "started", "starting", "resuming":
		return true
	default:
		return false
	}
}

func runUtmAppleScript(lines []string) (string, error) {
	args := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		args = append(args, "-e", line)
	}

	output, err := runCommandWithCombinedOutput(
		alchemy_build.GetDirectoriesInstance().ProjectDir,
		utmAutomationCommandTimeout,
		"osascript",
		args,
	)
	if err != nil {
		return "", fmt.Errorf("%w; output: %s", err, strings.TrimSpace(output))
	}

	return output, nil
}
