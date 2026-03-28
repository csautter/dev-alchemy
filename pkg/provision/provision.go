package provision

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	alchemy_deploy "github.com/csautter/dev-alchemy/pkg/deploy"
)

const (
	hypervWindowsAnsibleUserEnvVar           = "HYPERV_WINDOWS_ANSIBLE_USER"
	hypervWindowsAnsiblePasswordEnvVar       = "HYPERV_WINDOWS_ANSIBLE_PASSWORD"
	hypervWindowsAnsibleConnectionEnvVar     = "HYPERV_WINDOWS_ANSIBLE_CONNECTION"
	hypervWindowsAnsibleWinrmTransportEnvVar = "HYPERV_WINDOWS_ANSIBLE_WINRM_TRANSPORT"
	hypervWindowsAnsiblePortEnvVar           = "HYPERV_WINDOWS_ANSIBLE_PORT"

	utmWindowsAnsibleUserEnvVar           = "UTM_WINDOWS_ANSIBLE_USER"
	utmWindowsAnsiblePasswordEnvVar       = "UTM_WINDOWS_ANSIBLE_PASSWORD" // #nosec G101 -- environment variable name, not an embedded credential.
	utmWindowsAnsibleConnectionEnvVar     = "UTM_WINDOWS_ANSIBLE_CONNECTION"
	utmWindowsAnsibleWinrmTransportEnvVar = "UTM_WINDOWS_ANSIBLE_WINRM_TRANSPORT"
	utmWindowsAnsiblePortEnvVar           = "UTM_WINDOWS_ANSIBLE_PORT"

	hypervUbuntuAnsibleUserEnvVar           = "HYPERV_UBUNTU_ANSIBLE_USER"
	hypervUbuntuAnsiblePasswordEnvVar       = "HYPERV_UBUNTU_ANSIBLE_PASSWORD"        // #nosec G101 -- environment variable name, not an embedded credential.
	hypervUbuntuAnsibleBecomePasswordEnvVar = "HYPERV_UBUNTU_ANSIBLE_BECOME_PASSWORD" // #nosec G101 -- environment variable name, not an embedded credential.
	hypervUbuntuAnsibleConnectionEnvVar     = "HYPERV_UBUNTU_ANSIBLE_CONNECTION"
	hypervUbuntuAnsibleSshCommonArgsEnvVar  = "HYPERV_UBUNTU_ANSIBLE_SSH_COMMON_ARGS"
	hypervUbuntuAnsibleSshTimeoutEnvVar     = "HYPERV_UBUNTU_ANSIBLE_SSH_TIMEOUT"
	hypervUbuntuAnsibleSshRetriesEnvVar     = "HYPERV_UBUNTU_ANSIBLE_SSH_RETRIES"

	utmUbuntuAnsibleUserEnvVar           = "UTM_UBUNTU_ANSIBLE_USER"
	utmUbuntuAnsiblePasswordEnvVar       = "UTM_UBUNTU_ANSIBLE_PASSWORD"        // #nosec G101 -- environment variable name, not an embedded credential.
	utmUbuntuAnsibleBecomePasswordEnvVar = "UTM_UBUNTU_ANSIBLE_BECOME_PASSWORD" // #nosec G101 -- environment variable name, not an embedded credential.
	utmUbuntuAnsibleConnectionEnvVar     = "UTM_UBUNTU_ANSIBLE_CONNECTION"
	utmUbuntuAnsibleSshCommonArgsEnvVar  = "UTM_UBUNTU_ANSIBLE_SSH_COMMON_ARGS"
	utmUbuntuAnsibleSshTimeoutEnvVar     = "UTM_UBUNTU_ANSIBLE_SSH_TIMEOUT"
	utmUbuntuAnsibleSshRetriesEnvVar     = "UTM_UBUNTU_ANSIBLE_SSH_RETRIES"

	tartMacOSAnsibleUserEnvVar           = "TART_MACOS_ANSIBLE_USER"
	tartMacOSAnsiblePasswordEnvVar       = "TART_MACOS_ANSIBLE_PASSWORD"        // #nosec G101 -- environment variable name, not an embedded credential.
	tartMacOSAnsibleBecomePasswordEnvVar = "TART_MACOS_ANSIBLE_BECOME_PASSWORD" // #nosec G101 -- environment variable name, not an embedded credential.
	tartMacOSAnsibleConnectionEnvVar     = "TART_MACOS_ANSIBLE_CONNECTION"
	tartMacOSAnsibleSshCommonArgsEnvVar  = "TART_MACOS_ANSIBLE_SSH_COMMON_ARGS"
	tartMacOSAnsibleSshTimeoutEnvVar     = "TART_MACOS_ANSIBLE_SSH_TIMEOUT"
	tartMacOSAnsibleSshRetriesEnvVar     = "TART_MACOS_ANSIBLE_SSH_RETRIES"
	tartMacOSVMNameEnvVar                = "TART_MACOS_VM_NAME"
	tartMacOSDefaultVMName               = "tahoe-base-alchemy"
	tartMacOSIPv4DiscoveryRetryWindow    = 5 * time.Minute
	tartMacOSIPv4DiscoveryRetryInterval  = 2 * time.Second
	tartMacOSCommandTimeout              = time.Minute

	utmIPv4DiscoveryRetryWindow     = 45 * time.Second
	utmIPv4DiscoveryRetryInterval   = 3 * time.Second
	utmIPv4ARPCommandTimeout        = time.Minute
	utmIPv4ProbeDialTimeout         = 250 * time.Millisecond
	utmIPv4ProbeWorkerCount         = 32
	utmIPv4ProbePortHTTP            = 5985
	utmIPv4ProbePortHTTPS           = 5986
	utmIPv4ProbeSubnetPrefixMinimum = 24

	defaultAnsibleSSHCommonArgs = "-o StrictHostKeyChecking=no -o ServerAliveInterval=10 -o ServerAliveCountMax=3 -o ControlMaster=no -o ControlPersist=no"
)

var (
	windowsIPv4Regex   = regexp.MustCompile(`(?mi)IPv4 Address[^:]*:\s*((?:\d{1,3}\.){3}\d{1,3})`)
	linuxIPv4Regex     = regexp.MustCompile(`(?m)\b((?:\d{1,3}\.){3}\d{1,3})\b`)
	utmMacAddressRegex = regexp.MustCompile(`(?s)<key>MacAddress</key>\s*<string>([^<]+)</string>`)
	arpEntryRegex      = regexp.MustCompile(`(?i)\((\d{1,3}(?:\.\d{1,3}){3})\)\s+at\s+([0-9a-f:-]+|<incomplete>)`)
	loopbackAddressSet = map[string]struct{}{
		"127.0.0.1": {},
		"0.0.0.0":   {},
	}
)

type windowsAnsibleConnectionConfig struct {
	User           string
	Password       string
	Connection     string
	WinrmTransport string
	Port           string
}

type windowsAnsibleConnectionEnvVars struct {
	User           string
	Password       string
	Connection     string
	WinrmTransport string
	Port           string
}

type ubuntuAnsibleConnectionConfig struct {
	User           string
	Password       string
	BecomePassword string
	Connection     string
	SshCommonArgs  string
	SshTimeout     string
	SshRetries     string
}

type ubuntuAnsibleConnectionEnvVars struct {
	User           string
	Password       string
	BecomePassword string
	Connection     string
	SshCommonArgs  string
	SshTimeout     string
	SshRetries     string
}

type utmIPv4DiscoveryOptions struct {
	readFile        func(string) ([]byte, error)
	runCommand      func(string, time.Duration, string, []string) (string, error)
	sleep           func(time.Duration)
	primeARPCache   func() error
	retryInterval   time.Duration
	maxAttempts     int
	arpCommandTimer time.Duration
}

type tartIPv4DiscoveryOptions struct {
	runCommand     func(string, time.Duration, string, []string) (string, error)
	sleep          func(time.Duration)
	retryInterval  time.Duration
	maxAttempts    int
	commandTimeout time.Duration
}

type tartProvisionAvailabilityOptions struct {
	localVMExists func(string, string) (bool, error)
	discoverIPv4  func(string, string) (string, error)
}

var inspectProvisionTarget = alchemy_deploy.InspectStartTarget
var runProvisionCommandWithCombinedOutputWithEnv = runCommandWithCombinedOutputWithEnv

func RunProvision(vm alchemy_build.VirtualMachineConfig, check bool) error {
	if isHypervWindows11Amd64ProvisionTarget(vm) {
		return runHypervWindows11Provision(vm, check)
	}
	if isUtmWindows11ProvisionTarget(vm) {
		return runUtmWindows11Provision(vm, check)
	}
	if isUtmUbuntuProvisionTarget(vm) {
		return runUtmUbuntuProvision(vm, check)
	}
	if isTartMacOSProvisionTarget(vm) {
		return runTartMacOSProvision(vm, check)
	}
	if isHypervUbuntuAmd64ProvisionTarget(vm) {
		return runHypervUbuntuProvision(vm, check)
	}

	return fmt.Errorf(
		"provision is not implemented for OS=%s type=%s arch=%s host_os=%s virtualization_engine=%s",
		vm.OS,
		vm.UbuntuType,
		vm.Arch,
		vm.HostOs,
		vm.VirtualizationEngine,
	)
}

func isHypervWindows11Amd64ProvisionTarget(vm alchemy_build.VirtualMachineConfig) bool {
	return vm.OS == "windows11" &&
		vm.Arch == "amd64" &&
		vm.HostOs == alchemy_build.HostOsWindows &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineHyperv
}

func isHypervUbuntuAmd64ProvisionTarget(vm alchemy_build.VirtualMachineConfig) bool {
	return vm.OS == "ubuntu" &&
		vm.Arch == "amd64" &&
		vm.HostOs == alchemy_build.HostOsWindows &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineHyperv
}

func isUtmUbuntuProvisionTarget(vm alchemy_build.VirtualMachineConfig) bool {
	return vm.OS == "ubuntu" &&
		(vm.Arch == "amd64" || vm.Arch == "arm64") &&
		vm.HostOs == alchemy_build.HostOsDarwin &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineUtm
}

func isUtmWindows11ProvisionTarget(vm alchemy_build.VirtualMachineConfig) bool {
	return vm.OS == "windows11" &&
		(vm.Arch == "amd64" || vm.Arch == "arm64") &&
		vm.HostOs == alchemy_build.HostOsDarwin &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineUtm
}

func isTartMacOSProvisionTarget(vm alchemy_build.VirtualMachineConfig) bool {
	return vm.OS == "macos" &&
		vm.Arch == "arm64" &&
		vm.HostOs == alchemy_build.HostOsDarwin &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineTart
}

func runHypervWindows11Provision(vm alchemy_build.VirtualMachineConfig, check bool) error {
	if err := ensureProvisionTargetRunning(vm); err != nil {
		return err
	}

	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	vagrantSettings, err := alchemy_deploy.ResolveHypervVagrantExecutionSettings(vm)
	if err != nil {
		return fmt.Errorf("failed to resolve Hyper-V Vagrant settings: %w", err)
	}

	ip, err := discoverWindowsVagrantIPv4(vagrantSettings.VagrantDir, vagrantSettings.VagrantEnv)
	if err != nil {
		return fmt.Errorf("failed to determine vagrant VM IPv4 address: %w", err)
	}

	connectionConfig, err := loadWindowsHypervAnsibleConnectionConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load hyper-v windows ansible configuration: %w", err)
	}

	args, cleanupExtraVarsFile, err := buildWindowsProvisionArgs(projectDir, ip, connectionConfig, check)
	if err != nil {
		return fmt.Errorf("failed to build ansible arguments for discovered host %q: %w", ip, err)
	}

	runErr := runAnsibleProvisionCommand(projectDir, args, 90*time.Minute, fmt.Sprintf("%s:%s:provision", vm.OS, vm.Arch))

	cleanupErr := cleanupExtraVarsFile()
	if runErr != nil {
		if cleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for %s:%s: %w (also failed to remove ansible extra-vars temp file: %v)", vm.OS, vm.Arch, runErr, cleanupErr)
		}
		return fmt.Errorf("ansible provisioning failed for %s:%s: %w", vm.OS, vm.Arch, runErr)
	}
	if cleanupErr != nil {
		return fmt.Errorf("failed to remove ansible extra-vars temp file for %s:%s: %w", vm.OS, vm.Arch, cleanupErr)
	}

	return nil
}

func runUtmWindows11Provision(vm alchemy_build.VirtualMachineConfig, check bool) error {
	if err := ensureProvisionTargetRunning(vm); err != nil {
		return err
	}

	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir

	ip, err := discoverUtmVMIPv4(projectDir, vm)
	if err != nil {
		return fmt.Errorf("failed to determine UTM VM IPv4 address: %w", err)
	}

	connectionConfig, err := loadWindowsUtmAnsibleConnectionConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load UTM windows ansible configuration: %w", err)
	}

	args, cleanupExtraVarsFile, err := buildWindowsProvisionArgs(projectDir, ip, connectionConfig, check)
	if err != nil {
		return fmt.Errorf("failed to build ansible arguments for discovered host %q: %w", ip, err)
	}

	runErr := runAnsibleProvisionCommand(projectDir, args, 90*time.Minute, fmt.Sprintf("%s:%s:provision", vm.OS, vm.Arch))

	cleanupErr := cleanupExtraVarsFile()
	if runErr != nil {
		if cleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for %s:%s: %w (also failed to remove ansible extra-vars temp file: %v)", vm.OS, vm.Arch, runErr, cleanupErr)
		}
		return fmt.Errorf("ansible provisioning failed for %s:%s: %w", vm.OS, vm.Arch, runErr)
	}
	if cleanupErr != nil {
		return fmt.Errorf("failed to remove ansible extra-vars temp file for %s:%s: %w", vm.OS, vm.Arch, cleanupErr)
	}

	return nil
}

func runHypervUbuntuProvision(vm alchemy_build.VirtualMachineConfig, check bool) error {
	if err := ensureProvisionTargetRunning(vm); err != nil {
		return err
	}

	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	vagrantSettings, err := alchemy_deploy.ResolveHypervVagrantExecutionSettings(vm)
	if err != nil {
		return fmt.Errorf("failed to resolve Hyper-V Vagrant settings: %w", err)
	}

	ip, err := discoverLinuxVagrantIPv4(vagrantSettings.VagrantDir, vagrantSettings.VagrantEnv)
	if err != nil {
		return fmt.Errorf("failed to determine vagrant VM IPv4 address: %w", err)
	}

	connectionConfig, err := loadUbuntuHypervAnsibleConnectionConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load hyper-v ubuntu ansible configuration: %w", err)
	}

	args, cleanupExtraVarsFile, err := buildSSHProvisionArgs(projectDir, ip, connectionConfig, check)
	if err != nil {
		return fmt.Errorf("failed to build ansible arguments for discovered host %q: %w", ip, err)
	}

	runErr := runAnsibleProvisionCommand(
		projectDir,
		args,
		90*time.Minute,
		fmt.Sprintf("%s:%s:%s:provision", vm.OS, vm.UbuntuType, vm.Arch),
	)

	cleanupErr := cleanupExtraVarsFile()
	if runErr != nil {
		if cleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for %s:%s:%s: %w (also failed to remove ansible extra-vars temp file: %v)", vm.OS, vm.UbuntuType, vm.Arch, runErr, cleanupErr)
		}
		return fmt.Errorf("ansible provisioning failed for %s:%s:%s: %w", vm.OS, vm.UbuntuType, vm.Arch, runErr)
	}
	if cleanupErr != nil {
		return fmt.Errorf("failed to remove ansible extra-vars temp file for %s:%s:%s: %w", vm.OS, vm.UbuntuType, vm.Arch, cleanupErr)
	}

	return nil
}

func runUtmUbuntuProvision(vm alchemy_build.VirtualMachineConfig, check bool) error {
	if err := ensureProvisionTargetRunning(vm); err != nil {
		return err
	}

	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir

	ip, err := discoverUtmVMIPv4(projectDir, vm)
	if err != nil {
		return fmt.Errorf("failed to determine UTM VM IPv4 address: %w", err)
	}

	connectionConfig, err := loadUbuntuUtmAnsibleConnectionConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load UTM ubuntu ansible configuration: %w", err)
	}

	args, cleanupExtraVarsFile, err := buildSSHProvisionArgs(projectDir, ip, connectionConfig, check)
	if err != nil {
		return fmt.Errorf("failed to build ansible arguments for discovered host %q: %w", ip, err)
	}

	runErr := runAnsibleProvisionCommand(
		projectDir,
		args,
		90*time.Minute,
		fmt.Sprintf("%s:%s:%s:provision", vm.OS, vm.UbuntuType, vm.Arch),
	)

	cleanupErr := cleanupExtraVarsFile()
	if runErr != nil {
		if cleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for %s:%s:%s: %w (also failed to remove ansible extra-vars temp file: %v)", vm.OS, vm.UbuntuType, vm.Arch, runErr, cleanupErr)
		}
		return fmt.Errorf("ansible provisioning failed for %s:%s:%s: %w", vm.OS, vm.UbuntuType, vm.Arch, runErr)
	}
	if cleanupErr != nil {
		return fmt.Errorf("failed to remove ansible extra-vars temp file for %s:%s:%s: %w", vm.OS, vm.UbuntuType, vm.Arch, cleanupErr)
	}

	return nil
}

func runTartMacOSProvision(vm alchemy_build.VirtualMachineConfig, check bool) error {
	if err := ensureProvisionTargetRunning(vm); err != nil {
		return err
	}

	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	vmName := tartMacOSVMName(vm)

	ip, err := ensureTartVMReadyForProvision(projectDir, vmName, tartProvisionAvailabilityOptions{})
	if err != nil {
		return err
	}

	if err := waitForSSHPort(ip); err != nil {
		return fmt.Errorf("Tart VM %q is not ready for SSH: %w", vmName, err)
	}

	connectionConfig, err := loadMacOSTartAnsibleConnectionConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load Tart macOS ansible configuration: %w", err)
	}

	args, cleanupExtraVarsFile, err := buildSSHProvisionArgs(projectDir, ip, connectionConfig, check)
	if err != nil {
		return fmt.Errorf("failed to build ansible arguments for discovered host %q: %w", ip, err)
	}

	runErr := runAnsibleProvisionCommand(
		projectDir,
		args,
		90*time.Minute,
		fmt.Sprintf("%s:%s:provision", vm.OS, vm.Arch),
	)

	cleanupErr := cleanupExtraVarsFile()
	if runErr != nil {
		if cleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for %s:%s: %w (also failed to remove ansible extra-vars temp file: %v)", vm.OS, vm.Arch, runErr, cleanupErr)
		}
		return fmt.Errorf("ansible provisioning failed for %s:%s: %w", vm.OS, vm.Arch, runErr)
	}
	if cleanupErr != nil {
		return fmt.Errorf("failed to remove ansible extra-vars temp file for %s:%s: %w", vm.OS, vm.Arch, cleanupErr)
	}

	return nil
}

func ensureProvisionTargetRunning(vm alchemy_build.VirtualMachineConfig) error {
	state, err := inspectProvisionTarget(vm)
	if err != nil {
		return fmt.Errorf("failed to inspect VM state before provisioning: %w", err)
	}

	if !state.Exists {
		return fmt.Errorf(
			"VM for OS=%s, type=%s, arch=%s does not exist. Run `alchemy create %s` first",
			vm.OS,
			vm.UbuntuType,
			vm.Arch,
			provisionCommandTarget(vm),
		)
	}
	if !state.Running {
		currentState := state.State
		if currentState == "" {
			currentState = "stopped"
		}
		return fmt.Errorf(
			"VM for OS=%s, type=%s, arch=%s is not running (state=%s). Run `alchemy start %s` first",
			vm.OS,
			vm.UbuntuType,
			vm.Arch,
			currentState,
			provisionCommandTarget(vm),
		)
	}

	return nil
}

func provisionCommandTarget(vm alchemy_build.VirtualMachineConfig) string {
	parts := []string{vm.OS}
	if vm.UbuntuType != "" {
		parts = append(parts, "--type", vm.UbuntuType)
	}
	if vm.Arch != "" {
		parts = append(parts, "--arch", vm.Arch)
	}
	return strings.Join(parts, " ")
}

func ensureTartVMReadyForProvision(projectDir string, vmName string, options tartProvisionAvailabilityOptions) (string, error) {
	options = withDefaultTartProvisionAvailabilityOptions(options)

	exists, err := options.localVMExists(projectDir, vmName)
	if err != nil {
		return "", fmt.Errorf("failed to determine whether Tart VM %q exists: %w", vmName, err)
	}
	if !exists {
		return "", fmt.Errorf("Tart VM %q does not exist. Run `alchemy create macos --arch arm64` first", vmName)
	}

	ip, err := options.discoverIPv4(projectDir, vmName)
	if err != nil {
		return "", fmt.Errorf("Tart VM %q exists but is not running or has no IPv4 address yet. Start it with `alchemy start macos --arch arm64`: %w", vmName, err)
	}

	return ip, nil
}

func withDefaultTartProvisionAvailabilityOptions(options tartProvisionAvailabilityOptions) tartProvisionAvailabilityOptions {
	if options.localVMExists == nil {
		options.localVMExists = localTartVMExists
	}
	if options.discoverIPv4 == nil {
		options.discoverIPv4 = func(projectDir string, vmName string) (string, error) {
			return discoverTartVMIPv4WithOptions(projectDir, vmName, tartIPv4DiscoveryOptions{maxAttempts: 1})
		}
	}

	return options
}

func discoverWindowsVagrantIPv4(vagrantDir string, env []string) (string, error) {
	output, err := runProvisionCommandWithCombinedOutputWithEnv(
		vagrantDir,
		3*time.Minute,
		"vagrant",
		[]string{"winrm", "-c", "ipconfig"},
		env,
	)
	if err != nil {
		return "", fmt.Errorf("vagrant winrm call failed: %w; output: %s", err, strings.TrimSpace(output))
	}

	ip, parseErr := extractWindowsIPv4FromIPConfig(output)
	if parseErr != nil {
		return "", fmt.Errorf("could not parse IPv4 address from vagrant output: %w", parseErr)
	}
	return ip, nil
}

func discoverLinuxVagrantIPv4(vagrantDir string, env []string) (string, error) {
	output, err := runProvisionCommandWithCombinedOutputWithEnv(
		vagrantDir,
		3*time.Minute,
		"vagrant",
		[]string{"ssh", "-c", "hostname -I"},
		env,
	)
	if err != nil {
		return "", fmt.Errorf("vagrant ssh call failed: %w; output: %s", err, strings.TrimSpace(output))
	}

	ip, parseErr := extractLinuxIPv4FromHostOutput(output)
	if parseErr != nil {
		return "", fmt.Errorf("could not parse IPv4 address from vagrant output: %w", parseErr)
	}
	return ip, nil
}

func extractWindowsIPv4FromIPConfig(output string) (string, error) {
	matches := windowsIPv4Regex.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return "", errors.New("no IPv4 address found in command output")
	}

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		ip := strings.TrimSpace(match[1])
		if _, isLoopback := loopbackAddressSet[ip]; isLoopback {
			continue
		}
		return ip, nil
	}

	return "", errors.New("only loopback or invalid IPv4 candidates found in command output")
}

func extractLinuxIPv4FromHostOutput(output string) (string, error) {
	matches := linuxIPv4Regex.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return "", errors.New("no IPv4 address found in command output")
	}

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		ip := strings.TrimSpace(match[1])
		if _, isLoopback := loopbackAddressSet[ip]; isLoopback {
			continue
		}
		return ip, nil
	}

	return "", errors.New("only loopback or invalid IPv4 candidates found in command output")
}

func discoverUtmVMIPv4(projectDir string, vm alchemy_build.VirtualMachineConfig) (string, error) {
	return discoverUtmVMIPv4WithOptions(projectDir, vm, utmIPv4DiscoveryOptions{})
}

func discoverUtmVMIPv4WithOptions(projectDir string, vm alchemy_build.VirtualMachineConfig, options utmIPv4DiscoveryOptions) (string, error) {
	options = withDefaultUtmIPv4DiscoveryOptions(options)

	configPath, err := utmConfigPlistPath(vm)
	if err != nil {
		return "", err
	}

	content, err := options.readFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read UTM config %q: %w", configPath, err)
	}

	macAddress, err := extractUtmMacAddressFromConfig(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to extract UTM MAC address from %q: %w", configPath, err)
	}

	var lastLookupErr error
	var primeErr error

	for attempt := 1; attempt <= options.maxAttempts; attempt++ {
		output, lookupErr := options.runCommand(projectDir, options.arpCommandTimer, "arp", []string{"-a"})
		if lookupErr != nil {
			lastLookupErr = fmt.Errorf("arp lookup failed: %w; output: %s", lookupErr, strings.TrimSpace(output))
		} else {
			ip, parseErr := extractIPv4ForMacAddress(output, macAddress)
			if parseErr == nil {
				return ip, nil
			}
			lastLookupErr = fmt.Errorf("could not determine IPv4 address for UTM MAC %q from arp output: %w", macAddress, parseErr)
		}

		if attempt < options.maxAttempts {
			primeErr = options.primeARPCache()
			options.sleep(options.retryInterval)
		}
	}

	if primeErr != nil {
		return "", fmt.Errorf(
			"could not determine IPv4 address for UTM MAC %q after %d attempts over %s (ARP cache probe failed: %v): %w",
			macAddress,
			options.maxAttempts,
			time.Duration(options.maxAttempts-1)*options.retryInterval,
			primeErr,
			lastLookupErr,
		)
	}

	return "", fmt.Errorf(
		"could not determine IPv4 address for UTM MAC %q after %d attempts over %s: %w",
		macAddress,
		options.maxAttempts,
		time.Duration(options.maxAttempts-1)*options.retryInterval,
		lastLookupErr,
	)
}

func withDefaultUtmIPv4DiscoveryOptions(options utmIPv4DiscoveryOptions) utmIPv4DiscoveryOptions {
	if options.readFile == nil {
		options.readFile = os.ReadFile
	}
	if options.runCommand == nil {
		options.runCommand = runCommandWithCombinedOutput
	}
	if options.sleep == nil {
		options.sleep = time.Sleep
	}
	if options.primeARPCache == nil {
		options.primeARPCache = primeUtmARPCache
	}
	if options.retryInterval <= 0 {
		options.retryInterval = utmIPv4DiscoveryRetryInterval
	}
	if options.maxAttempts <= 0 {
		options.maxAttempts = int(utmIPv4DiscoveryRetryWindow/options.retryInterval) + 1
	}
	if options.arpCommandTimer <= 0 {
		options.arpCommandTimer = utmIPv4ARPCommandTimeout
	}

	return options
}

func utmConfigPlistPath(vm alchemy_build.VirtualMachineConfig) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user home directory: %w", err)
	}

	vmName := alchemy_build.GetVirtualMachineNameWithType(vm)
	return filepath.Join(
		homeDir,
		"Library",
		"Containers",
		"com.utmapp.UTM",
		"Data",
		"Documents",
		fmt.Sprintf("%s-%s-dev-alchemy.utm", vmName, vm.Arch),
		"config.plist",
	), nil
}

func extractUtmMacAddressFromConfig(content string) (string, error) {
	match := utmMacAddressRegex.FindStringSubmatch(content)
	if len(match) < 2 {
		return "", errors.New("no MacAddress entry found in config.plist")
	}

	return normalizeMACAddress(match[1])
}

func extractIPv4ForMacAddress(output string, targetMacAddress string) (string, error) {
	normalizedTargetMAC, err := normalizeMACAddress(targetMacAddress)
	if err != nil {
		return "", fmt.Errorf("invalid target MAC address %q: %w", targetMacAddress, err)
	}

	matches := arpEntryRegex.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return "", errors.New("no ARP entries found in command output")
	}

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		ip := strings.TrimSpace(match[1])
		if _, isLoopback := loopbackAddressSet[ip]; isLoopback {
			continue
		}

		normalizedCandidateMAC, normalizeErr := normalizeMACAddress(match[2])
		if normalizeErr != nil {
			continue
		}
		if normalizedCandidateMAC == normalizedTargetMAC {
			return ip, nil
		}
	}

	return "", errors.New("no IPv4 address found for MAC address in arp output")
}

func normalizeMACAddress(value string) (string, error) {
	cleaned := strings.TrimSpace(strings.Trim(value, "<>"))
	if cleaned == "" {
		return "", errors.New("MAC address is empty")
	}

	separator := ":"
	switch {
	case strings.Contains(cleaned, ":"):
		separator = ":"
	case strings.Contains(cleaned, "-"):
		separator = "-"
	default:
		return "", fmt.Errorf("unsupported MAC address format %q", value)
	}

	parts := strings.Split(cleaned, separator)
	if len(parts) != 6 {
		return "", fmt.Errorf("expected 6 MAC address segments, got %d", len(parts))
	}

	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) == 0 || len(part) > 2 {
			return "", fmt.Errorf("invalid MAC address segment %q", part)
		}
		octet, err := strconv.ParseUint(part, 16, 8)
		if err != nil {
			return "", fmt.Errorf("invalid MAC address segment %q: %w", part, err)
		}
		normalized = append(normalized, fmt.Sprintf("%02x", octet))
	}

	return strings.Join(normalized, ":"), nil
}

func primeUtmARPCache() error {
	candidateIPs, err := utmProbeCandidateIPs()
	if err != nil {
		return err
	}
	if len(candidateIPs) == 0 {
		return errors.New("no private IPv4 probe candidates found on active host interfaces")
	}

	type probeTarget struct {
		ip   string
		port int
	}

	targets := make(chan probeTarget)
	var workers sync.WaitGroup

	for workerIndex := 0; workerIndex < utmIPv4ProbeWorkerCount; workerIndex++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for target := range targets {
				address := net.JoinHostPort(target.ip, strconv.Itoa(target.port))
				conn, err := net.DialTimeout("tcp4", address, utmIPv4ProbeDialTimeout)
				if err == nil {
					_ = conn.Close()
				}
			}
		}()
	}

	for _, candidateIP := range candidateIPs {
		targets <- probeTarget{ip: candidateIP, port: utmIPv4ProbePortHTTP}
		targets <- probeTarget{ip: candidateIP, port: utmIPv4ProbePortHTTPS}
	}
	close(targets)
	workers.Wait()

	return nil
}

func utmProbeCandidateIPs() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate host network interfaces: %w", err)
	}

	candidateSet := make(map[string]struct{})
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			hostIP := ipNet.IP.To4()
			if hostIP == nil || !isPrivateIPv4(hostIP) {
				continue
			}

			for _, candidateIP := range probeCandidatesForHostNetwork(hostIP, ipNet.Mask) {
				candidateSet[candidateIP] = struct{}{}
			}
		}
	}

	candidates := make([]string, 0, len(candidateSet))
	for candidateIP := range candidateSet {
		candidates = append(candidates, candidateIP)
	}
	sort.Strings(candidates)

	return candidates, nil
}

func probeCandidatesForHostNetwork(hostIP net.IP, mask net.IPMask) []string {
	ipv4 := hostIP.To4()
	if ipv4 == nil {
		return nil
	}

	ones, bits := mask.Size()
	if bits != net.IPv4len*8 || ones <= 0 {
		return nil
	}

	if ones < utmIPv4ProbeSubnetPrefixMinimum {
		mask = net.CIDRMask(utmIPv4ProbeSubnetPrefixMinimum, net.IPv4len*8)
	}

	networkAddress := ipv4.Mask(mask)
	broadcastAddress := make(net.IP, len(networkAddress))
	copy(broadcastAddress, networkAddress)
	for index := range broadcastAddress {
		broadcastAddress[index] |= ^mask[index]
	}

	start := ipv4ToUint32(networkAddress) + 1
	end := ipv4ToUint32(broadcastAddress) - 1
	hostValue := ipv4ToUint32(ipv4)

	if end < start {
		return nil
	}

	candidates := make([]string, 0, int(end-start+1))
	for candidateValue := start; candidateValue <= end; candidateValue++ {
		if candidateValue == hostValue {
			continue
		}
		candidates = append(candidates, uint32ToIPv4(candidateValue).String())
	}

	return candidates
}

func isPrivateIPv4(ip net.IP) bool {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return false
	}

	switch {
	case ipv4[0] == 10:
		return true
	case ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31:
		return true
	case ipv4[0] == 192 && ipv4[1] == 168:
		return true
	default:
		return false
	}
}

func ipv4ToUint32(ip net.IP) uint32 {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0
	}

	return uint32(ipv4[0])<<24 | uint32(ipv4[1])<<16 | uint32(ipv4[2])<<8 | uint32(ipv4[3])
}

func uint32ToIPv4(value uint32) net.IP {
	return net.IPv4(
		byte(value>>24), // #nosec G115 -- IPv4 octet extraction intentionally truncates to the low 8 bits.
		byte(value>>16), // #nosec G115 -- IPv4 octet extraction intentionally truncates to the low 8 bits.
		byte(value>>8),  // #nosec G115 -- IPv4 octet extraction intentionally truncates to the low 8 bits.
		byte(value),     // #nosec G115 -- IPv4 octet extraction intentionally truncates to the low 8 bits.
	).To4()
}

func buildWindowsProvisionArgs(projectDir string, ip string, connectionConfig windowsAnsibleConnectionConfig, check bool) ([]string, func() error, error) {
	extraVars, err := json.Marshal(map[string]string{
		"ansible_user":            connectionConfig.User,
		"ansible_password":        connectionConfig.Password,
		"ansible_connection":      connectionConfig.Connection,
		"ansible_winrm_transport": connectionConfig.WinrmTransport,
		"ansible_port":            connectionConfig.Port,
	})
	if err != nil {
		return nil, nil, err
	}

	return buildAnsibleProvisionArgs(projectDir, ip, extraVars, check)
}

func buildSSHProvisionArgs(projectDir string, ip string, connectionConfig ubuntuAnsibleConnectionConfig, check bool) ([]string, func() error, error) {
	extraVars, err := json.Marshal(map[string]string{
		"ansible_user":            connectionConfig.User,
		"ansible_password":        connectionConfig.Password,
		"ansible_become_password": connectionConfig.BecomePassword,
		"ansible_connection":      connectionConfig.Connection,
		"ansible_ssh_common_args": connectionConfig.SshCommonArgs,
		"ansible_ssh_timeout":     connectionConfig.SshTimeout,
		"ansible_ssh_retries":     connectionConfig.SshRetries,
	})
	if err != nil {
		return nil, nil, err
	}

	return buildAnsibleProvisionArgs(projectDir, ip, extraVars, check)
}

func buildAnsibleProvisionArgs(projectDir string, ip string, extraVars []byte, check bool) ([]string, func() error, error) {

	extraVarsFile, err := os.CreateTemp(projectDir, ".ansible-extra-vars-*.json")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ansible extra-vars temp file: %w", err)
	}
	extraVarsFilePath := extraVarsFile.Name()
	if _, err := extraVarsFile.Write(extraVars); err != nil {
		_ = extraVarsFile.Close()
		_ = os.Remove(extraVarsFilePath)
		return nil, nil, fmt.Errorf("failed to write ansible extra-vars temp file: %w", err)
	}
	if err := extraVarsFile.Close(); err != nil {
		_ = os.Remove(extraVarsFilePath)
		return nil, nil, fmt.Errorf("failed to close ansible extra-vars temp file: %w", err)
	}

	cleanup := func() error {
		err := os.Remove(extraVarsFilePath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}

	args := []string{
		"./playbooks/setup.yml",
		"-i",
		ip + ",",
		"-l",
		ip,
		"--extra-vars",
		"@" + filepath.Base(extraVarsFilePath),
		"-vvv",
	}
	if check {
		args = append(args, "--check")
	}

	return args, cleanup, nil
}

func loadWindowsHypervAnsibleConnectionConfig(projectDir string) (windowsAnsibleConnectionConfig, error) {
	return loadWindowsAnsibleConnectionConfig(projectDir, windowsAnsibleConnectionEnvVars{
		User:           hypervWindowsAnsibleUserEnvVar,
		Password:       hypervWindowsAnsiblePasswordEnvVar,
		Connection:     hypervWindowsAnsibleConnectionEnvVar,
		WinrmTransport: hypervWindowsAnsibleWinrmTransportEnvVar,
		Port:           hypervWindowsAnsiblePortEnvVar,
	})
}

func loadWindowsUtmAnsibleConnectionConfig(projectDir string) (windowsAnsibleConnectionConfig, error) {
	return loadWindowsAnsibleConnectionConfig(projectDir, windowsAnsibleConnectionEnvVars{
		User:           utmWindowsAnsibleUserEnvVar,
		Password:       utmWindowsAnsiblePasswordEnvVar,
		Connection:     utmWindowsAnsibleConnectionEnvVar,
		WinrmTransport: utmWindowsAnsibleWinrmTransportEnvVar,
		Port:           utmWindowsAnsiblePortEnvVar,
	})
}

func loadWindowsAnsibleConnectionConfig(projectDir string, envVars windowsAnsibleConnectionEnvVars) (windowsAnsibleConnectionConfig, error) {
	envFilePath := filepath.Join(projectDir, ".env")
	valuesFromFile, err := parseDotEnvFile(envFilePath)
	if err != nil {
		return windowsAnsibleConnectionConfig{}, err
	}

	connectionConfig := windowsAnsibleConnectionConfig{
		User:           resolveEnvValue(envVars.User, valuesFromFile),
		Password:       resolveEnvValue(envVars.Password, valuesFromFile),
		Connection:     defaultIfEmpty(resolveEnvValue(envVars.Connection, valuesFromFile), "winrm"),
		WinrmTransport: defaultIfEmpty(resolveEnvValue(envVars.WinrmTransport, valuesFromFile), "basic"),
		Port:           defaultIfEmpty(resolveEnvValue(envVars.Port, valuesFromFile), "5985"),
	}

	missing := make([]string, 0, 2)
	if connectionConfig.User == "" {
		missing = append(missing, envVars.User)
	}
	if connectionConfig.Password == "" {
		missing = append(missing, envVars.Password)
	}
	if len(missing) > 0 {
		return windowsAnsibleConnectionConfig{}, fmt.Errorf(
			"missing required values %s; define them in process environment or in %q",
			strings.Join(missing, ", "),
			envFilePath,
		)
	}

	return connectionConfig, nil
}

func loadUbuntuHypervAnsibleConnectionConfig(projectDir string) (ubuntuAnsibleConnectionConfig, error) {
	return loadUbuntuAnsibleConnectionConfig(projectDir, ubuntuAnsibleConnectionEnvVars{
		User:           hypervUbuntuAnsibleUserEnvVar,
		Password:       hypervUbuntuAnsiblePasswordEnvVar,
		BecomePassword: hypervUbuntuAnsibleBecomePasswordEnvVar,
		Connection:     hypervUbuntuAnsibleConnectionEnvVar,
		SshCommonArgs:  hypervUbuntuAnsibleSshCommonArgsEnvVar,
		SshTimeout:     hypervUbuntuAnsibleSshTimeoutEnvVar,
		SshRetries:     hypervUbuntuAnsibleSshRetriesEnvVar,
	})
}

func loadUbuntuUtmAnsibleConnectionConfig(projectDir string) (ubuntuAnsibleConnectionConfig, error) {
	return loadUbuntuAnsibleConnectionConfig(projectDir, ubuntuAnsibleConnectionEnvVars{
		User:           utmUbuntuAnsibleUserEnvVar,
		Password:       utmUbuntuAnsiblePasswordEnvVar,
		BecomePassword: utmUbuntuAnsibleBecomePasswordEnvVar,
		Connection:     utmUbuntuAnsibleConnectionEnvVar,
		SshCommonArgs:  utmUbuntuAnsibleSshCommonArgsEnvVar,
		SshTimeout:     utmUbuntuAnsibleSshTimeoutEnvVar,
		SshRetries:     utmUbuntuAnsibleSshRetriesEnvVar,
	})
}

func loadUbuntuAnsibleConnectionConfig(projectDir string, envVars ubuntuAnsibleConnectionEnvVars) (ubuntuAnsibleConnectionConfig, error) {
	envFilePath := filepath.Join(projectDir, ".env")
	valuesFromFile, err := parseDotEnvFile(envFilePath)
	if err != nil {
		return ubuntuAnsibleConnectionConfig{}, err
	}

	password := defaultIfEmpty(resolveEnvValue(envVars.Password, valuesFromFile), "P@ssw0rd!")
	connectionConfig := ubuntuAnsibleConnectionConfig{
		User:           defaultIfEmpty(resolveEnvValue(envVars.User, valuesFromFile), "packer"),
		Password:       password,
		BecomePassword: defaultIfEmpty(resolveEnvValue(envVars.BecomePassword, valuesFromFile), password),
		Connection:     defaultIfEmpty(resolveEnvValue(envVars.Connection, valuesFromFile), "ssh"),
		SshCommonArgs: defaultIfEmpty(
			resolveEnvValue(envVars.SshCommonArgs, valuesFromFile),
			defaultAnsibleSSHCommonArgs,
		),
		SshTimeout: defaultIfEmpty(resolveEnvValue(envVars.SshTimeout, valuesFromFile), "120"),
		SshRetries: defaultIfEmpty(resolveEnvValue(envVars.SshRetries, valuesFromFile), "3"),
	}

	return connectionConfig, nil
}

func loadMacOSTartAnsibleConnectionConfig(projectDir string) (ubuntuAnsibleConnectionConfig, error) {
	envFilePath := filepath.Join(projectDir, ".env")
	valuesFromFile, err := parseDotEnvFile(envFilePath)
	if err != nil {
		return ubuntuAnsibleConnectionConfig{}, err
	}

	password := defaultIfEmpty(
		resolveEnvValue(tartMacOSAnsiblePasswordEnvVar, valuesFromFile),
		"admin", // Default Tart guest credential; override in .env via TART_MACOS_ANSIBLE_PASSWORD as documented in README.md "Local tests for macOS (on macos)".
	)
	connectionConfig := ubuntuAnsibleConnectionConfig{
		User: defaultIfEmpty(
			resolveEnvValue(tartMacOSAnsibleUserEnvVar, valuesFromFile),
			"admin", // Default Tart guest credential; override in .env via TART_MACOS_ANSIBLE_USER as documented in README.md "Local tests for macOS (on macos)".
		),
		Password:       password,
		BecomePassword: defaultIfEmpty(resolveEnvValue(tartMacOSAnsibleBecomePasswordEnvVar, valuesFromFile), password),
		Connection:     defaultIfEmpty(resolveEnvValue(tartMacOSAnsibleConnectionEnvVar, valuesFromFile), "ssh"),
		SshCommonArgs: defaultIfEmpty(
			resolveEnvValue(tartMacOSAnsibleSshCommonArgsEnvVar, valuesFromFile),
			defaultAnsibleSSHCommonArgs,
		),
		SshTimeout: defaultIfEmpty(resolveEnvValue(tartMacOSAnsibleSshTimeoutEnvVar, valuesFromFile), "120"),
		SshRetries: defaultIfEmpty(resolveEnvValue(tartMacOSAnsibleSshRetriesEnvVar, valuesFromFile), "3"),
	}

	return connectionConfig, nil
}

func parseDotEnvFile(dotEnvPath string) (map[string]string, error) {
	// #nosec G304 -- dotEnvPath is derived from the repository project directory, not arbitrary user input.
	content, err := os.ReadFile(dotEnvPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("failed to read %q: %w", dotEnvPath, err)
	}

	values := make(map[string]string)
	lines := strings.Split(string(content), "\n")
	for lineNumber, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, hasSeparator := strings.Cut(line, "=")
		if !hasSeparator {
			return nil, fmt.Errorf("invalid .env format at %s:%d: expected KEY=VALUE", dotEnvPath, lineNumber+1)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("invalid .env format at %s:%d: key must not be empty", dotEnvPath, lineNumber+1)
		}

		parsedValue, parseErr := parseDotEnvValue(value)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid .env value for %q at %s:%d: %w", key, dotEnvPath, lineNumber+1, parseErr)
		}
		values[key] = parsedValue
	}

	return values, nil
}

func parseDotEnvValue(value string) (string, error) {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", err
		}
		return unquoted, nil
	}
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1], nil
	}
	return value, nil
}

func resolveEnvValue(key string, valuesFromFile map[string]string) string {
	if envValue, exists := os.LookupEnv(key); exists {
		trimmed := strings.TrimSpace(envValue)
		if trimmed != "" {
			return trimmed
		}
	}
	return strings.TrimSpace(valuesFromFile[key])
}

func defaultIfEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func runAnsibleProvisionCommand(projectDir string, args []string, timeout time.Duration, logPrefix string) error {
	if runtime.GOOS == "windows" {
		return runAnsibleViaCygwinBash(projectDir, args, timeout, logPrefix)
	}

	return runCommandWithStreamingLogsWithEnv(
		projectDir,
		timeout,
		"ansible-playbook",
		args,
		ansibleRuntimeEnv(),
		logPrefix,
	)
}

func runAnsibleViaCygwinBash(workingDir string, ansibleArgs []string, timeout time.Duration, logPrefix string) error {
	cygwinWorkingDir, err := windowsPathToCygwinPath(workingDir)
	if err != nil {
		return fmt.Errorf("failed to convert working directory to cygwin path: %w", err)
	}
	cygwinBashExecutable, err := getCygwinBashExecutable()
	if err != nil {
		return fmt.Errorf("failed to locate cygwin bash executable: %w", err)
	}

	quotedArgs := make([]string, 0, len(ansibleArgs))
	for _, arg := range ansibleArgs {
		quotedArgs = append(quotedArgs, bashSingleQuote(arg))
	}

	bashCommand := "cd " + bashSingleQuote(cygwinWorkingDir) + " && ansible-playbook " + strings.Join(quotedArgs, " ")

	return runCommandWithStreamingLogsWithEnv(
		workingDir,
		timeout,
		cygwinBashExecutable,
		[]string{"-l", "-c", bashCommand},
		ansibleRuntimeEnv(),
		logPrefix,
	)
}

func getCygwinBashExecutable() (string, error) {
	if configuredPath := strings.TrimSpace(os.Getenv("CYGWIN_BASH_PATH")); configuredPath != "" {
		resolvedPath := resolveCygwinBashPath(configuredPath)
		if err := validateCygwinBashExecutable(resolvedPath); err != nil {
			return "", fmt.Errorf("CYGWIN_BASH_PATH is set to %q (resolved to %q), but cygwin bash is unavailable: %w", configuredPath, resolvedPath, err)
		}
		return resolvedPath, nil
	}
	if configuredTerminalPath := strings.TrimSpace(os.Getenv("CYGWIN_TERMINAL_PATH")); configuredTerminalPath != "" {
		resolvedPath := resolveCygwinBashPath(configuredTerminalPath)
		if err := validateCygwinBashExecutable(resolvedPath); err != nil {
			return "", fmt.Errorf("CYGWIN_TERMINAL_PATH is set to %q (resolved to %q), but cygwin bash is unavailable: %w", configuredTerminalPath, resolvedPath, err)
		}
		return resolvedPath, nil
	}

	candidates := []string{
		`C:\tools\cygwin\bin\bash.exe`,
		`C:\cygwin64\bin\bash.exe`,
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("cygwin bash executable not found; checked %q and %q. Install Cygwin or set CYGWIN_BASH_PATH (or CYGWIN_TERMINAL_PATH) to your cygwin bash.exe", candidates[0], candidates[1])
}

func validateCygwinBashExecutable(path string) error {
	// #nosec G703 -- this validates an operator-configured executable path before use.
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("expected executable file but found directory")
	}
	return nil
}

func resolveCygwinBashPath(path string) string {
	cleaned := strings.TrimSpace(path)
	lastSeparatorIndex := strings.LastIndexAny(cleaned, `/\`)
	baseName := cleaned
	dirName := ""
	if lastSeparatorIndex >= 0 {
		dirName = cleaned[:lastSeparatorIndex]
		baseName = cleaned[lastSeparatorIndex+1:]
	}

	if strings.EqualFold(baseName, "mintty.exe") {
		if dirName == "" {
			return "bash.exe"
		}
		separator := `\`
		if strings.Contains(cleaned, "/") && !strings.Contains(cleaned, `\`) {
			separator = "/"
		}
		return strings.TrimRight(dirName, `/\`) + separator + "bash.exe"
	}
	return cleaned
}

func windowsPathToCygwinPath(windowsPath string) (string, error) {
	normalized := strings.TrimSpace(windowsPath)
	normalized = strings.ReplaceAll(normalized, "/", `\`)
	if len(normalized) < 2 || normalized[1] != ':' {
		return "", fmt.Errorf("expected drive-letter windows path, got %q", windowsPath)
	}

	drive := strings.ToLower(normalized[:1])
	rest := normalized[2:]
	rest = strings.TrimPrefix(rest, `\`)
	rest = strings.ReplaceAll(rest, `\`, "/")
	if rest == "" {
		return "/cygdrive/" + drive, nil
	}

	return "/cygdrive/" + drive + "/" + rest, nil
}

func bashSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func ansibleColorEnv() []string {
	return []string{
		"ANSIBLE_FORCE_COLOR=true",
		"PY_COLORS=1",
		"TERM=xterm-256color",
	}
}

func ansibleRuntimeEnv() []string {
	env := ansibleColorEnv()
	if runtime.GOOS == "darwin" {
		env = append(env, "OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES")
	}
	return env
}
