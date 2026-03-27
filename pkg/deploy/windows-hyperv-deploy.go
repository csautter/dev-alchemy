package deploy

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

const (
	windowsHypervVagrantBoxName = "win11-packer"
	hypervVagrantBoxNameEnvVar  = "VAGRANT_BOX_NAME"
	hypervVagrantVMNameEnvVar   = "VAGRANT_VM_NAME"
	hypervVagrantDotfileEnvVar  = "VAGRANT_DOTFILE_PATH"
	hypervVagrantCpuEnvVar      = "VAGRANT_VM_CPUS"
	hypervVagrantMemoryEnvVar   = "VAGRANT_VM_MEMORY_MB"
	hypervVagrantUpTimeout      = 60 * time.Minute
)

var (
	runHypervVagrantCommandWithEnv     = runCommandWithStreamingLogsWithEnv
	inspectHypervVagrantStartCmdTarget = inspectHypervVagrantStartTarget
	inspectHypervVagrantStopTarget     = inspectHypervVagrantStartTarget
	hypervVagrantMachineExistsChecker  = hypervVagrantMachineExists
	hypervVagrantBoxInstalledChecker   = hypervVagrantBoxInstalled
	hypervVagrantStopSettleTimeout     = 15 * time.Second
	hypervVagrantStopPollInterval      = 500 * time.Millisecond
)

type hypervVagrantDeploySettings struct {
	BoxName    string
	BoxPath    string
	VagrantDir string
	VagrantEnv []string
}

type HypervVagrantExecutionSettings struct {
	VagrantDir string
	VagrantEnv []string
}

func isHypervVagrantTarget(config alchemy_build.VirtualMachineConfig) bool {
	return config.HostOs == alchemy_build.HostOsWindows &&
		config.VirtualizationEngine == alchemy_build.VirtualizationEngineHyperv &&
		config.Arch == "amd64" &&
		(config.OS == "windows11" || config.OS == "ubuntu")
}

func RunHypervVagrantDeployOnWindows(config alchemy_build.VirtualMachineConfig) error {
	if !isHypervVagrantTarget(config) {
		return fmt.Errorf("hyper-v vagrant deploy is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}

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

	if err := runHypervVagrantCommandWithEnv(
		settings.VagrantDir,
		hypervVagrantUpTimeout,
		"vagrant",
		[]string{"up", "--provider", "hyperv"},
		settings.VagrantEnv,
		fmt.Sprintf("%s:%s:%s:vagrant-up", config.OS, config.UbuntuType, config.Arch),
	); err != nil {
		return fmt.Errorf("failed to start Vagrant VM for %s:%s:%s: %w", config.OS, config.UbuntuType, config.Arch, err)
	}

	return nil
}

func RunHypervVagrantDestroyOnWindows(config alchemy_build.VirtualMachineConfig) error {
	if !isHypervVagrantTarget(config) {
		return fmt.Errorf("hyper-v vagrant destroy is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}

	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	settings, err := resolveHypervVagrantDeploySettings(config, projectDir)
	if err != nil {
		return err
	}

	exists, err := hypervVagrantMachineExistsChecker(settings.VagrantDir, settings.VagrantEnv)
	if err != nil {
		return err
	}
	if exists {
		if err := runHypervVagrantCommandWithEnv(
			settings.VagrantDir,
			20*time.Minute,
			"vagrant",
			[]string{"destroy", "-f"},
			settings.VagrantEnv,
			fmt.Sprintf("%s:%s:%s:vagrant-destroy", config.OS, config.UbuntuType, config.Arch),
		); err != nil {
			return fmt.Errorf("failed to destroy Vagrant VM for %s:%s:%s: %w", config.OS, config.UbuntuType, config.Arch, err)
		}
	}

	boxInstalled, err := hypervVagrantBoxInstalledChecker(projectDir, settings.BoxName)
	if err != nil {
		return err
	}
	if boxInstalled {
		if err := runCommandWithStreamingLogs(
			projectDir,
			5*time.Minute,
			"vagrant",
			[]string{"box", "remove", settings.BoxName, "--provider", "hyperv", "--force"},
			fmt.Sprintf("%s:%s:%s:box-remove", config.OS, config.UbuntuType, config.Arch),
		); err != nil {
			return fmt.Errorf("failed to remove Vagrant box for %s:%s:%s: %w", config.OS, config.UbuntuType, config.Arch, err)
		}
	}

	return nil
}

func RunHypervVagrantStartOnWindows(config alchemy_build.VirtualMachineConfig) error {
	if !isHypervVagrantTarget(config) {
		return fmt.Errorf("hyper-v vagrant start is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}

	state, err := inspectHypervVagrantStartCmdTarget(config)
	if err != nil {
		return err
	}
	if !state.Exists {
		return fmt.Errorf("Hyper-V VM for %s does not exist. Run `alchemy create %s` first", startCommandArguments(config), startCommandArguments(config))
	}
	if state.Running {
		return nil
	}

	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	settings, err := resolveHypervVagrantDeploySettings(config, projectDir)
	if err != nil {
		return err
	}

	if err := runHypervVagrantCommandWithEnv(
		settings.VagrantDir,
		hypervVagrantUpTimeout,
		"vagrant",
		[]string{"up", "--provider", "hyperv"},
		settings.VagrantEnv,
		fmt.Sprintf("%s:%s:%s:vagrant-up", config.OS, config.UbuntuType, config.Arch),
	); err != nil {
		return fmt.Errorf("failed to start Vagrant VM for %s:%s:%s: %w", config.OS, config.UbuntuType, config.Arch, err)
	}

	return nil
}

func RunHypervVagrantStopOnWindows(config alchemy_build.VirtualMachineConfig) error {
	if !isHypervVagrantTarget(config) {
		return fmt.Errorf("hyper-v vagrant stop is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}

	state, err := inspectHypervVagrantStopTarget(config)
	if err != nil {
		return err
	}
	if !state.Exists {
		return nil
	}
	if !state.Running {
		return nil
	}

	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	settings, err := resolveHypervVagrantDeploySettings(config, projectDir)
	if err != nil {
		return err
	}

	if err := runHypervVagrantCommandWithEnv(
		settings.VagrantDir,
		20*time.Minute,
		"vagrant",
		[]string{"halt"},
		settings.VagrantEnv,
		fmt.Sprintf("%s:%s:%s:vagrant-halt", config.OS, config.UbuntuType, config.Arch),
	); err != nil {
		stopped, waitErr := waitForHypervVagrantStop(config, hypervVagrantStopSettleTimeout)
		if waitErr == nil && stopped {
			return nil
		}

		forceErr := runHypervVagrantCommandWithEnv(
			settings.VagrantDir,
			20*time.Minute,
			"vagrant",
			[]string{"halt", "--force"},
			settings.VagrantEnv,
			fmt.Sprintf("%s:%s:%s:vagrant-halt-force", config.OS, config.UbuntuType, config.Arch),
		)
		if forceErr == nil {
			stopped, waitErr = waitForHypervVagrantStop(config, hypervVagrantStopSettleTimeout)
			if waitErr == nil && stopped {
				return nil
			}
		}

		if waitErr != nil {
			return fmt.Errorf(
				"failed to stop Vagrant VM for %s:%s:%s: graceful halt failed: %v; forced halt failed: %v; failed verifying final state: %w",
				config.OS,
				config.UbuntuType,
				config.Arch,
				err,
				forceErr,
				waitErr,
			)
		}
		if forceErr != nil {
			return fmt.Errorf(
				"failed to stop Vagrant VM for %s:%s:%s: graceful halt failed: %v; forced halt failed: %w",
				config.OS,
				config.UbuntuType,
				config.Arch,
				err,
				forceErr,
			)
		}
		return fmt.Errorf(
			"failed to stop Vagrant VM for %s:%s:%s: graceful halt failed: %v; forced halt completed but VM is still running",
			config.OS,
			config.UbuntuType,
			config.Arch,
			err,
		)
	}

	return nil
}

func waitForHypervVagrantStop(config alchemy_build.VirtualMachineConfig, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	for {
		state, err := inspectHypervVagrantStopTarget(config)
		if err != nil {
			return false, err
		}
		if !state.Exists || !state.Running {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		time.Sleep(hypervVagrantStopPollInterval)
	}
}

func resolveHypervVagrantDeploySettings(config alchemy_build.VirtualMachineConfig, projectDir string) (hypervVagrantDeploySettings, error) {
	switch config.OS {
	case "windows11":
		boxName := windowsHypervVagrantBoxName
		vmName := boxName
		return hypervVagrantDeploySettings{
			BoxName:    boxName,
			BoxPath:    getHypervWindowsBoxPath(config),
			VagrantDir: filepath.Join(projectDir, "deployments", "vagrant", "ansible-windows"),
			VagrantEnv: append([]string{
				hypervVagrantBoxNameEnvVar + "=" + boxName,
				hypervVagrantVMNameEnvVar + "=" + vmName,
				hypervVagrantDotfileEnvVar + "=" + hypervVagrantDotfilePath(vmName),
			}, buildHypervVagrantResourceEnv(config)...),
		}, nil
	case "ubuntu":
		ubuntuType := config.UbuntuType
		if ubuntuType == "" {
			ubuntuType = "server"
		}
		boxName := fmt.Sprintf("linux-ubuntu-%s-packer", ubuntuType)
		vmName := boxName
		return hypervVagrantDeploySettings{
			BoxName:    boxName,
			BoxPath:    getHypervUbuntuBoxPath(config),
			VagrantDir: filepath.Join(projectDir, "deployments", "vagrant", "linux-ubuntu-hyperv"),
			VagrantEnv: append([]string{
				hypervVagrantBoxNameEnvVar + "=" + boxName,
				hypervVagrantVMNameEnvVar + "=" + vmName,
				hypervVagrantDotfileEnvVar + "=" + hypervVagrantDotfilePath(vmName),
			}, buildHypervVagrantResourceEnv(config)...),
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

func ResolveHypervVagrantExecutionSettings(config alchemy_build.VirtualMachineConfig) (HypervVagrantExecutionSettings, error) {
	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	settings, err := resolveHypervVagrantDeploySettings(config, projectDir)
	if err != nil {
		return HypervVagrantExecutionSettings{}, err
	}

	return HypervVagrantExecutionSettings{
		VagrantDir: settings.VagrantDir,
		VagrantEnv: append([]string(nil), settings.VagrantEnv...),
	}, nil
}

func hypervVagrantDotfilePath(vmName string) string {
	return path.Join(".vagrant", vmName)
}

func buildHypervVagrantResourceEnv(config alchemy_build.VirtualMachineConfig) []string {
	return []string{
		hypervVagrantCpuEnvVar + "=" + strconv.Itoa(alchemy_build.GetVmCpuCount(config)),
		hypervVagrantMemoryEnvVar + "=" + strconv.Itoa(alchemy_build.GetVmMemoryMB(config)),
	}
}

func hypervVagrantMachineExists(vagrantDir string, env []string) (bool, error) {
	state, err := hypervVMState(vagrantDir, env)
	if err != nil {
		return false, err
	}

	return state != "missing", nil
}

func inspectHypervVagrantStartTarget(config alchemy_build.VirtualMachineConfig) (StartTargetState, error) {
	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	settings, err := resolveHypervVagrantDeploySettings(config, projectDir)
	if err != nil {
		return StartTargetState{}, err
	}

	state, err := hypervVMState(settings.VagrantDir, settings.VagrantEnv)
	if err != nil {
		return StartTargetState{}, err
	}

	return hypervStartTargetStateFromVMState(state), nil
}

func hypervVMState(workingDir string, env []string) (string, error) {
	vmName, err := hypervVagrantVMName(env)
	if err != nil {
		return "", err
	}

	output, err := runCommandWithCombinedOutput(
		workingDir,
		time.Minute,
		"powershell",
		[]string{
			"-NoProfile",
			"-NonInteractive",
			"-Command",
			fmt.Sprintf(
				"$vm = Get-VM -Name %s -ErrorAction SilentlyContinue; if ($null -eq $vm) { Write-Output missing } else { Write-Output $vm.State }",
				powershellSingleQuote(vmName),
			),
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to inspect Hyper-V VM %q in %q: %w; output: %s", vmName, workingDir, err, strings.TrimSpace(output))
	}

	return hypervVMStateFromOutput(output), nil
}

func hypervVagrantVMName(env []string) (string, error) {
	const prefix = hypervVagrantVMNameEnvVar + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			vmName := strings.TrimSpace(strings.TrimPrefix(entry, prefix))
			if vmName == "" {
				break
			}
			return vmName, nil
		}
	}

	return "", fmt.Errorf("missing %s in Vagrant environment", hypervVagrantVMNameEnvVar)
}

func hypervVMStateFromOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "missing"
	}

	lines := strings.Split(trimmed, "\n")
	return strings.ToLower(strings.TrimSpace(lines[len(lines)-1]))
}

func hypervStartTargetStateFromVMState(state string) StartTargetState {
	switch state {
	case "", "missing":
		return StartTargetState{State: "missing"}
	case "running":
		return StartTargetState{Exists: true, Running: true, State: state}
	default:
		return StartTargetState{Exists: true, State: state}
	}
}

func hypervVagrantBoxInstalled(projectDir string, boxName string) (bool, error) {
	output, err := runCommandWithCombinedOutput(projectDir, time.Minute, "vagrant", []string{"box", "list"})
	if err != nil {
		return false, fmt.Errorf("failed to list Vagrant boxes: %w; output: %s", err, strings.TrimSpace(output))
	}

	return vagrantBoxListIncludes(output, boxName, "hyperv"), nil
}

func vagrantMachineExistsInStatusOutput(output string) bool {
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Split(line, ",")
		if len(fields) < 4 {
			continue
		}
		if fields[2] != "state" {
			continue
		}
		state := strings.TrimSpace(fields[3])
		if state != "" && state != "not_created" {
			return true
		}
	}

	return false
}

func startTargetStateFromVagrantStatusOutput(output string) StartTargetState {
	state := vagrantMachineStateFromStatusOutput(output)
	switch state {
	case "", "not_created":
		return StartTargetState{State: "missing"}
	case "running":
		return StartTargetState{Exists: true, Running: true, State: state}
	default:
		return StartTargetState{Exists: true, State: state}
	}
}

func vagrantMachineStateFromStatusOutput(output string) string {
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Split(line, ",")
		if len(fields) < 4 {
			continue
		}
		if fields[2] != "state" {
			continue
		}
		return strings.TrimSpace(fields[3])
	}

	return ""
}

func vagrantBoxListIncludes(output string, boxName string, provider string) bool {
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		name, remainder, ok := strings.Cut(line, " (")
		if !ok || name != boxName {
			continue
		}

		providerValue, _, ok := strings.Cut(remainder, ",")
		if !ok {
			continue
		}
		if strings.TrimSpace(providerValue) == provider {
			return true
		}
	}

	return false
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
