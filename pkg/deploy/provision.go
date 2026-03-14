package deploy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

var (
	windowsIPv4Regex   = regexp.MustCompile(`(?mi)IPv4 Address[^:]*:\s*((?:\d{1,3}\.){3}\d{1,3})`)
	ansibleHostRegex   = regexp.MustCompile(`(?m)^(\s*ansible_host:\s*).*$`)
	loopbackAddressSet = map[string]struct{}{
		"127.0.0.1": {},
		"0.0.0.0":   {},
	}
)

func RunProvision(vm alchemy_build.VirtualMachineConfig, check bool) error {
	if isHypervWindows11Amd64ProvisionTarget(vm) {
		return runHypervWindows11Provision(vm, check)
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

func runHypervWindows11Provision(vm alchemy_build.VirtualMachineConfig, check bool) error {
	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	vagrantDir := filepath.Join(projectDir, "deployments", "vagrant", "ansible-windows")
	inventoryPath := filepath.Join(projectDir, "inventory", "hyperv_windows_winrm.yml")

	ip, err := discoverWindowsVagrantIPv4(vagrantDir)
	if err != nil {
		return fmt.Errorf("failed to determine vagrant VM IPv4 address: %w", err)
	}

	if err := upsertWindowsHypervInventory(inventoryPath, ip); err != nil {
		return fmt.Errorf("failed to update inventory file %q: %w", inventoryPath, err)
	}

	args := []string{
		"./playbooks/setup.yml",
		"-i",
		"./inventory/hyperv_windows_winrm.yml",
		"-l",
		"windows_host",
		"-vvv",
	}
	if check {
		args = append(args, "--check")
	}

	if runtime.GOOS == "windows" {
		if err := runAnsibleViaCygwinBash(projectDir, args, 90*time.Minute, fmt.Sprintf("%s:%s:provision", vm.OS, vm.Arch)); err != nil {
			return fmt.Errorf("ansible provisioning failed for %s:%s: %w", vm.OS, vm.Arch, err)
		}
		return nil
	}

	if err := runCommandWithStreamingLogsWithEnv(
		projectDir,
		90*time.Minute,
		"ansible-playbook",
		args,
		ansibleColorEnv(),
		fmt.Sprintf("%s:%s:provision", vm.OS, vm.Arch),
	); err != nil {
		return fmt.Errorf("ansible provisioning failed for %s:%s: %w", vm.OS, vm.Arch, err)
	}

	return nil
}

func discoverWindowsVagrantIPv4(vagrantDir string) (string, error) {
	output, err := runCommandWithCombinedOutput(
		vagrantDir,
		3*time.Minute,
		"vagrant",
		[]string{"winrm", "-c", "ipconfig"},
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

func upsertWindowsHypervInventory(inventoryPath string, ip string) error {
	content, err := os.ReadFile(inventoryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return os.WriteFile(inventoryPath, []byte(defaultHypervWindowsInventory(ip)), 0o644)
		}
		return err
	}

	updated := ansibleHostRegex.ReplaceAllString(string(content), "${1}"+ip)
	if updated == string(content) {
		updated = defaultHypervWindowsInventory(ip)
	}

	return os.WriteFile(inventoryPath, []byte(updated), 0o644)
}

func defaultHypervWindowsInventory(ip string) string {
	return fmt.Sprintf(`all:
    children:
        windows:
            hosts:
                windows_host:
                    ansible_host: %s
                    ansible_user: Administrator
                    ansible_password: P@ssw0rd!
                    ansible_connection: winrm
                    ansible_winrm_transport: basic
                    ansible_port: 5985
`, ip)
}

func runAnsibleViaCygwinBash(workingDir string, ansibleArgs []string, timeout time.Duration, logPrefix string) error {
	cygwinWorkingDir, err := windowsPathToCygwinPath(workingDir)
	if err != nil {
		return fmt.Errorf("failed to convert working directory to cygwin path: %w", err)
	}

	quotedArgs := make([]string, 0, len(ansibleArgs))
	for _, arg := range ansibleArgs {
		quotedArgs = append(quotedArgs, bashSingleQuote(arg))
	}

	bashCommand := "cd " + bashSingleQuote(cygwinWorkingDir) + " && ansible-playbook " + strings.Join(quotedArgs, " ")

	return runCommandWithStreamingLogsWithEnv(
		workingDir,
		timeout,
		getCygwinBashExecutable(),
		[]string{"-l", "-c", bashCommand},
		ansibleColorEnv(),
		logPrefix,
	)
}

func getCygwinBashExecutable() string {
	if configuredPath := strings.TrimSpace(os.Getenv("CYGWIN_BASH_PATH")); configuredPath != "" {
		return resolveCygwinBashPath(configuredPath)
	}
	if configuredTerminalPath := strings.TrimSpace(os.Getenv("CYGWIN_TERMINAL_PATH")); configuredTerminalPath != "" {
		return resolveCygwinBashPath(configuredTerminalPath)
	}

	candidates := []string{
		`C:\tools\cygwin\bin\bash.exe`,
		`C:\cygwin64\bin\bash.exe`,
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return "bash"
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
