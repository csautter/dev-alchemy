package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

const (
	hypervWindowsAnsibleUserEnvVar           = "HYPERV_WINDOWS_ANSIBLE_USER"
	hypervWindowsAnsiblePasswordEnvVar       = "HYPERV_WINDOWS_ANSIBLE_PASSWORD"
	hypervWindowsAnsibleConnectionEnvVar     = "HYPERV_WINDOWS_ANSIBLE_CONNECTION"
	hypervWindowsAnsibleWinrmTransportEnvVar = "HYPERV_WINDOWS_ANSIBLE_WINRM_TRANSPORT"
	hypervWindowsAnsiblePortEnvVar           = "HYPERV_WINDOWS_ANSIBLE_PORT"
)

var (
	windowsIPv4Regex   = regexp.MustCompile(`(?mi)IPv4 Address[^:]*:\s*((?:\d{1,3}\.){3}\d{1,3})`)
	loopbackAddressSet = map[string]struct{}{
		"127.0.0.1": {},
		"0.0.0.0":   {},
	}
)

type windowsHypervAnsibleConnectionConfig struct {
	User           string
	Password       string
	Connection     string
	WinrmTransport string
	Port           string
}

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

	ip, err := discoverWindowsVagrantIPv4(vagrantDir)
	if err != nil {
		return fmt.Errorf("failed to determine vagrant VM IPv4 address: %w", err)
	}

	connectionConfig, err := loadWindowsHypervAnsibleConnectionConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load hyper-v windows ansible configuration: %w", err)
	}

	args, cleanupExtraVarsFile, err := buildWindowsHypervProvisionArgs(projectDir, ip, connectionConfig, check)
	if err != nil {
		return fmt.Errorf("failed to build ansible arguments for discovered host %q: %w", ip, err)
	}

	runErr := error(nil)
	if runtime.GOOS == "windows" {
		runErr = runAnsibleViaCygwinBash(projectDir, args, 90*time.Minute, fmt.Sprintf("%s:%s:provision", vm.OS, vm.Arch))
	} else {
		runErr = runCommandWithStreamingLogsWithEnv(
			projectDir,
			90*time.Minute,
			"ansible-playbook",
			args,
			ansibleColorEnv(),
			fmt.Sprintf("%s:%s:provision", vm.OS, vm.Arch),
		)
	}

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

func buildWindowsHypervProvisionArgs(projectDir string, ip string, connectionConfig windowsHypervAnsibleConnectionConfig, check bool) ([]string, func() error, error) {
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

func loadWindowsHypervAnsibleConnectionConfig(projectDir string) (windowsHypervAnsibleConnectionConfig, error) {
	envFilePath := filepath.Join(projectDir, ".env")
	valuesFromFile, err := parseDotEnvFile(envFilePath)
	if err != nil {
		return windowsHypervAnsibleConnectionConfig{}, err
	}

	connectionConfig := windowsHypervAnsibleConnectionConfig{
		User:           resolveEnvValue(hypervWindowsAnsibleUserEnvVar, valuesFromFile),
		Password:       resolveEnvValue(hypervWindowsAnsiblePasswordEnvVar, valuesFromFile),
		Connection:     defaultIfEmpty(resolveEnvValue(hypervWindowsAnsibleConnectionEnvVar, valuesFromFile), "winrm"),
		WinrmTransport: defaultIfEmpty(resolveEnvValue(hypervWindowsAnsibleWinrmTransportEnvVar, valuesFromFile), "basic"),
		Port:           defaultIfEmpty(resolveEnvValue(hypervWindowsAnsiblePortEnvVar, valuesFromFile), "5985"),
	}

	missing := make([]string, 0, 2)
	if connectionConfig.User == "" {
		missing = append(missing, hypervWindowsAnsibleUserEnvVar)
	}
	if connectionConfig.Password == "" {
		missing = append(missing, hypervWindowsAnsiblePasswordEnvVar)
	}
	if len(missing) > 0 {
		return windowsHypervAnsibleConnectionConfig{}, fmt.Errorf(
			"missing required values %s; define them in process environment or in %q",
			strings.Join(missing, ", "),
			envFilePath,
		)
	}

	return connectionConfig, nil
}

func parseDotEnvFile(dotEnvPath string) (map[string]string, error) {
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
