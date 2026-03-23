package deploy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

const (
	tartMacOSImageEnvVar                = "TART_MACOS_IMAGE"
	tartMacOSVMNameEnvVar               = "TART_MACOS_VM_NAME"
	tartMacOSBridgedInterfaceEnvVar     = "TART_MACOS_BRIDGED_INTERFACE"
	tartMacOSDefaultImageReference      = "ghcr.io/cirruslabs/macos-tahoe-base:latest"
	tartMacOSDefaultVMName              = "tahoe-base-alchemy"
	tartMacOSCloneTimeout               = 45 * time.Minute
	tartMacOSStartTimeout               = time.Minute
	tartMacOSIPv4DiscoveryRetryWindow   = 5 * time.Minute
	tartMacOSIPv4DiscoveryRetryInterval = 2 * time.Second
	tartMacOSCommandTimeout             = time.Minute
)

type tartIPv4DiscoveryOptions struct {
	runCommand     func(string, time.Duration, string, []string) (string, error)
	sleep          func(time.Duration)
	retryInterval  time.Duration
	maxAttempts    int
	commandTimeout time.Duration
}

type tartDetachedRun struct {
	logPath string
	pid     int
}

type tartReachabilityWaitOptions struct {
	detectIPv4       func() (string, error)
	isProcessRunning func(int) (bool, error)
	readLogSummary   func(string) string
	sleep            func(time.Duration)
	retryInterval    time.Duration
	maxAttempts      int
}

func RunTartDeployOnMacOS(config alchemy_build.VirtualMachineConfig) error {
	if !isTartMacOSDeployTarget(config) {
		return fmt.Errorf("Tart deploy is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}

	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	vmName := tartMacOSVMName(config)

	if err := ensureLocalTartVM(projectDir, vmName); err != nil {
		return err
	}

	if ip, err := discoverTartVMIPv4WithOptions(projectDir, vmName, tartIPv4DiscoveryOptions{maxAttempts: 1}); err == nil && ip != "" {
		return nil
	}

	runState, err := startTartVMDetached(projectDir, vmName)
	if err != nil {
		return err
	}

	if _, err := waitForTartVMToBecomeReachable(projectDir, vmName, runState); err != nil {
		return err
	}

	return nil
}

func isTartMacOSDeployTarget(vm alchemy_build.VirtualMachineConfig) bool {
	return vm.OS == "macos" &&
		vm.Arch == "arm64" &&
		vm.HostOs == alchemy_build.HostOsDarwin &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineTart
}

func resolveTartMacOSImageReference() string {
	return defaultIfEmpty(strings.TrimSpace(os.Getenv(tartMacOSImageEnvVar)), tartMacOSDefaultImageReference)
}

func tartMacOSVMName(_ alchemy_build.VirtualMachineConfig) string {
	return defaultIfEmpty(strings.TrimSpace(os.Getenv(tartMacOSVMNameEnvVar)), tartMacOSDefaultVMName)
}

func ensureLocalTartVM(projectDir string, vmName string) error {
	exists, err := localTartVMExists(projectDir, vmName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	imageReference := resolveTartMacOSImageReference()
	if err := runCommandWithStreamingLogs(
		projectDir,
		tartMacOSCloneTimeout,
		"tart",
		[]string{"clone", imageReference, vmName},
		fmt.Sprintf("%s:clone", vmName),
	); err != nil {
		return fmt.Errorf("failed to clone Tart image %q into %q: %w", imageReference, vmName, err)
	}

	return nil
}

func localTartVMExists(projectDir string, vmName string) (bool, error) {
	output, err := runCommandWithCombinedOutput(projectDir, tartMacOSCommandTimeout, "tart", []string{"list"})
	if err != nil {
		return false, fmt.Errorf("failed to list Tart VMs: %w; output: %s", err, strings.TrimSpace(output))
	}

	return tartListIncludesLocalVM(output, vmName), nil
}

func tartListIncludesLocalVM(output string, vmName string) bool {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return false
	}

	if nameColumn, sourceColumn, ok := tartListColumnIndexes(lines); ok {
		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) <= nameColumn || len(fields) <= sourceColumn {
				continue
			}
			if fields[sourceColumn] == "local" && fields[nameColumn] == vmName {
				return true
			}
		}

		return false
	}

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == "local" && fields[1] == vmName {
			return true
		}
	}

	return false
}

func tartListColumnIndexes(lines []string) (int, int, bool) {
	for _, line := range lines {
		header := strings.Fields(strings.ToLower(strings.TrimSpace(line)))
		if len(header) == 0 {
			continue
		}

		nameColumn := -1
		sourceColumn := -1
		for index, field := range header {
			switch field {
			case "name":
				nameColumn = index
			case "source":
				sourceColumn = index
			}
		}

		if nameColumn >= 0 && sourceColumn >= 0 {
			return nameColumn, sourceColumn, true
		}

		break
	}

	return 0, 0, false
}

func startTartVMDetached(projectDir string, vmName string) (tartDetachedRun, error) {
	logDir := filepath.Join(projectDir, ".dev-alchemy", "tart")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return tartDetachedRun{}, fmt.Errorf("failed to create Tart log directory %q: %w", logDir, err)
	}

	logPath := filepath.Join(logDir, vmName+".log")
	commandParts := []string{"tart", "run"}
	if bridgedInterface := strings.TrimSpace(os.Getenv(tartMacOSBridgedInterfaceEnvVar)); bridgedInterface != "" {
		commandParts = append(commandParts, fmt.Sprintf("--net-bridged=%s", bridgedInterface))
	}
	commandParts = append(commandParts, vmName)

	quotedArgs := make([]string, 0, len(commandParts)-1)
	for _, arg := range commandParts[1:] {
		quotedArgs = append(quotedArgs, bashSingleQuote(arg))
	}

	shellCommand := "nohup " + commandParts[0] + " " + strings.Join(quotedArgs, " ") + " >" + bashSingleQuote(logPath) + " 2>&1 </dev/null & echo $!"
	output, err := runCommandWithCombinedOutput(projectDir, tartMacOSStartTimeout, "bash", []string{"-lc", shellCommand})
	if err != nil {
		return tartDetachedRun{}, fmt.Errorf("failed to start Tart VM %q: %w; output: %s", vmName, err, strings.TrimSpace(output))
	}

	pid, err := parseBackgroundPID(output)
	if err != nil {
		return tartDetachedRun{}, fmt.Errorf("failed to determine Tart background PID for %q: %w; output: %s", vmName, err, strings.TrimSpace(output))
	}

	return tartDetachedRun{
		logPath: logPath,
		pid:     pid,
	}, nil
}

func discoverTartVMIPv4(projectDir string, vmName string) (string, error) {
	return discoverTartVMIPv4WithOptions(projectDir, vmName, tartIPv4DiscoveryOptions{})
}

func discoverTartVMIPv4WithOptions(projectDir string, vmName string, options tartIPv4DiscoveryOptions) (string, error) {
	options = withDefaultTartIPv4DiscoveryOptions(options)

	var lastErr error
	for attempt := 1; attempt <= options.maxAttempts; attempt++ {
		ip, err := detectTartVMIPv4(projectDir, vmName, options.runCommand, options.commandTimeout)
		if err == nil {
			return ip, nil
		}
		lastErr = err

		if attempt < options.maxAttempts {
			options.sleep(options.retryInterval)
		}
	}

	return "", fmt.Errorf(
		"could not determine IPv4 address for Tart VM %q after %d attempts over %s: %w",
		vmName,
		options.maxAttempts,
		time.Duration(options.maxAttempts-1)*options.retryInterval,
		lastErr,
	)
}

func withDefaultTartIPv4DiscoveryOptions(options tartIPv4DiscoveryOptions) tartIPv4DiscoveryOptions {
	if options.runCommand == nil {
		options.runCommand = runCommandWithCombinedOutput
	}
	if options.sleep == nil {
		options.sleep = time.Sleep
	}
	if options.retryInterval <= 0 {
		options.retryInterval = tartMacOSIPv4DiscoveryRetryInterval
	}
	if options.maxAttempts <= 0 {
		options.maxAttempts = int(tartMacOSIPv4DiscoveryRetryWindow/options.retryInterval) + 1
	}
	if options.commandTimeout <= 0 {
		options.commandTimeout = tartMacOSCommandTimeout
	}

	return options
}

func detectTartVMIPv4(
	projectDir string,
	vmName string,
	runCommand func(string, time.Duration, string, []string) (string, error),
	commandTimeout time.Duration,
) (string, error) {
	type ipCommand struct {
		args  []string
		label string
	}

	commands := []ipCommand{
		{args: []string{"ip", vmName}, label: "default resolver"},
		{args: []string{"ip", "--resolver=arp", vmName}, label: "arp resolver"},
	}

	var failures []string
	for _, command := range commands {
		output, err := runCommand(projectDir, commandTimeout, "tart", command.args)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s failed: %v; output: %s", command.label, err, strings.TrimSpace(output)))
			continue
		}

		ip, parseErr := extractLinuxIPv4FromHostOutput(output)
		if parseErr == nil {
			return ip, nil
		}

		failures = append(failures, fmt.Sprintf("%s returned no IPv4 address: %v; output: %s", command.label, parseErr, strings.TrimSpace(output)))
	}

	return "", errors.New(strings.Join(failures, "; "))
}

func waitForTartVMToBecomeReachable(projectDir string, vmName string, runState tartDetachedRun) (string, error) {
	return waitForTartVMToBecomeReachableWithOptions(projectDir, vmName, runState, tartReachabilityWaitOptions{})
}

func waitForTartVMToBecomeReachableWithOptions(
	projectDir string,
	vmName string,
	runState tartDetachedRun,
	options tartReachabilityWaitOptions,
) (string, error) {
	waitOptions := withDefaultTartReachabilityWaitOptions(projectDir, vmName, options)

	var lastErr error
	for attempt := 1; attempt <= waitOptions.maxAttempts; attempt++ {
		ip, err := waitOptions.detectIPv4()
		if err == nil {
			return ip, nil
		}
		lastErr = err

		if runState.pid > 0 {
			running, runErr := waitOptions.isProcessRunning(runState.pid)
			if runErr != nil {
				return "", fmt.Errorf("failed to inspect Tart process %d for %q: %w", runState.pid, vmName, runErr)
			}
			if !running {
				logSummary := waitOptions.readLogSummary(runState.logPath)
				if logSummary != "" {
					return "", fmt.Errorf("failed to start Tart VM %q: tart run exited early: %s", vmName, logSummary)
				}
				return "", fmt.Errorf("failed to start Tart VM %q: tart run exited early before an IPv4 address was assigned", vmName)
			}
		}

		if attempt < waitOptions.maxAttempts {
			waitOptions.sleep(waitOptions.retryInterval)
		}
	}

	if runState.logPath != "" {
		logSummary := waitOptions.readLogSummary(runState.logPath)
		if logSummary != "" {
			return "", fmt.Errorf("Tart VM %q did not expose an IPv4 address before timeout. Last tart run log output: %s. Last IP discovery error: %w", vmName, logSummary, lastErr)
		}
	}

	return "", fmt.Errorf("Tart VM %q started but no IPv4 address became available: %w", vmName, lastErr)
}

func withDefaultTartReachabilityWaitOptions(
	projectDir string,
	vmName string,
	options tartReachabilityWaitOptions,
) tartReachabilityWaitOptions {
	discoveryOptions := withDefaultTartIPv4DiscoveryOptions(tartIPv4DiscoveryOptions{})

	if options.detectIPv4 == nil {
		options.detectIPv4 = func() (string, error) {
			return detectTartVMIPv4(projectDir, vmName, discoveryOptions.runCommand, discoveryOptions.commandTimeout)
		}
	}
	if options.isProcessRunning == nil {
		options.isProcessRunning = isProcessRunning
	}
	if options.readLogSummary == nil {
		options.readLogSummary = readTartRunLogSummary
	}
	if options.sleep == nil {
		options.sleep = time.Sleep
	}
	if options.retryInterval <= 0 {
		options.retryInterval = discoveryOptions.retryInterval
	}
	if options.maxAttempts <= 0 {
		options.maxAttempts = discoveryOptions.maxAttempts
	}

	return options
}

func parseBackgroundPID(output string) (int, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		line := strings.TrimSpace(lines[index])
		if line == "" {
			continue
		}

		pid, err := strconv.Atoi(line)
		if err == nil && pid > 0 {
			return pid, nil
		}
	}

	return 0, errors.New("no shell background pid found in command output")
}

func isProcessRunning(pid int) (bool, error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, err
	}

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrProcessDone) {
		return false, nil
	}

	errText := strings.ToLower(err.Error())
	if strings.Contains(errText, "process already finished") || strings.Contains(errText, "no such process") {
		return false, nil
	}

	return false, err
}

func readTartRunLogSummary(logPath string) string {
	if strings.TrimSpace(logPath) == "" {
		return ""
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 {
		return ""
	}

	const maxLines = 5
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	summary := strings.TrimSpace(strings.Join(lines, " | "))
	if summary == "" {
		return ""
	}

	return summary
}
