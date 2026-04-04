package provision

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	localWindowsSSHInventoryPath    = "./inventory/localhost_windows_ssh.yml"
	localWindowsSSHInventoryTarget  = "windows_host"
	localWindowsSSHLoopbackIP       = "127.0.0.1"
	localWindowsSSHDefaultShell     = `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
	localWindowsSSHBootstrapTimeout = 15 * time.Minute
	localWindowsSSHCleanupTimeout   = 10 * time.Minute
	localWindowsSSHPreflightTimeout = 45 * time.Second
	localWindowsSSHPortEnvVar       = "DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_SSH_PORT"

	localWindowsProvisionSSHPublicKeyEnvVar = "DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_SSH_PUBLIC_KEY"
	localWindowsForceSSHUninstallEnvVar     = "DEV_ALCHEMY_LOCAL_WINDOWS_FORCE_SSH_UNINSTALL"
	localWindowsSSHBootstrapLogPrefix       = "local:windows:ssh:bootstrap"
	localWindowsSSHCleanupLogPrefix         = "local:windows:ssh:cleanup"
	localWindowsSSHPreflightLogPrefix       = "local:windows:ssh:preflight"
	localWindowsSSHBootstrapScriptPath      = "scripts/windows/local-windows-provision-ssh-bootstrap.ps1"
	localWindowsSSHCleanupScriptPath        = "scripts/windows/local-windows-provision-ssh-cleanup.ps1"
	localWindowsSSHCommonArgs               = defaultAnsibleSSHCommonArgs + " -o UserKnownHostsFile=/dev/null -o IdentitiesOnly=yes"
)

var setupLocalWindowsSSHProvisionSessionFunc = setupLocalWindowsSSHProvisionSession
var cleanupLocalWindowsSSHProvisionSessionFunc = cleanupLocalWindowsSSHProvisionSession
var runLocalWindowsSSHPreflightFunc = runLocalWindowsSSHPreflight
var localWindowsSSHProvisionBootstrapPowerShell = mustLoadLocalWindowsPowerShellAsset(localWindowsSSHBootstrapScriptPath)
var localWindowsSSHProvisionCleanupPowerShell = mustLoadLocalWindowsPowerShellAsset(localWindowsSSHCleanupScriptPath)

type localWindowsSSHProvisionSession struct {
	ConnectionConfig sshAnsibleConnectionConfig
	StatePath        string
	PrivateKeyPath   string
}

type localWindowsSSHListenerProcess struct {
	ID          int    `json:"Id"`
	ProcessName string `json:"ProcessName"`
}

func runLocalWindowsSSHProvision(projectDir string, options ProvisionOptions) error {
	return runLocalWindowsProvisionSession(projectDir, options, localWindowsProvisionSessionRunner[localWindowsSSHProvisionSession]{
		setup:   setupLocalWindowsSSHProvisionSessionFunc,
		cleanup: cleanupLocalWindowsSSHProvisionSessionFunc,
		afterSetup: func(projectDir string, session localWindowsSSHProvisionSession, _ ProvisionOptions) error {
			return runLocalWindowsSSHPreflightFunc(projectDir, session.ConnectionConfig)
		},
		buildArgs: func(projectDir string, session localWindowsSSHProvisionSession, options ProvisionOptions) ([]string, func() error, error) {
			inventoryPath, inventoryTarget := resolveStaticInventoryPathAndTarget(
				localWindowsSSHInventoryPath,
				localWindowsSSHInventoryTarget,
				options,
			)

			return buildSSHStaticInventoryProvisionArgs(
				projectDir,
				inventoryPath,
				inventoryTarget,
				session.ConnectionConfig,
				options,
			)
		},
		afterSetupError: func(err error, cleanupErr error) error {
			return formatLocalWindowsProvisionStepError(
				"local Windows SSH bootstrap completed but the direct SSH preflight failed",
				err,
				cleanupErr,
				"SSH",
			)
		},
		buildArgsError: func(err error, cleanupErr error) error {
			return formatLocalWindowsProvisionStepError(
				"failed to build ansible arguments for secure local windows SSH provision",
				err,
				cleanupErr,
				"SSH",
			)
		},
		provisionResult: func(runErr error, argsCleanupErr error, cleanupErr error) error {
			return formatLocalWindowsProvisionOutcome("ssh", "SSH", runErr, argsCleanupErr, cleanupErr)
		},
		ansibleLogPrefix: "local:windows:ssh:provision",
		runTimeout:       90 * time.Minute,
	})
}

func buildSSHStaticInventoryProvisionArgs(projectDir string, inventoryPath string, inventoryTarget string, connectionConfig sshAnsibleConnectionConfig, options ProvisionOptions) ([]string, func() error, error) {
	extraVars, err := buildSSHProvisionExtraVars(connectionConfig)
	if err != nil {
		return nil, nil, err
	}

	return buildStaticInventoryProvisionArgsWithExtraVars(projectDir, inventoryPath, inventoryTarget, extraVars, options)
}

func buildLocalWindowsSSHProvisionScriptEnv(statePath string, password string, publicKey string, sshPort string, options ProvisionOptions) []string {
	env := []string{
		localWindowsProvisionStatePathEnvVar + "=" + statePath,
		localWindowsProvisionUserEnvVar + "=" + localWindowsProvisionUserName,
		localWindowsForceSSHUninstallEnvVar + "=" + fmt.Sprintf("%t", options.LocalWindowsForceSSHUninstall),
		localWindowsSSHPortEnvVar + "=" + sshPort,
	}
	if password == "" {
		return env
	}

	return append(env,
		localWindowsProvisionPasswordEnvVar+"="+password,
		localWindowsProvisionSSHPublicKeyEnvVar+"="+publicKey,
	)
}

func setupLocalWindowsSSHProvisionSession(projectDir string, options ProvisionOptions) (localWindowsSSHProvisionSession, error) {
	password, err := generateSecureLocalWindowsProvisionPassword()
	if err != nil {
		return localWindowsSSHProvisionSession{}, fmt.Errorf("failed to generate secure local windows provision password: %w", err)
	}
	sshPort, err := selectLocalWindowsSSHBootstrapPort(projectDir)
	if err != nil {
		return localWindowsSSHProvisionSession{}, fmt.Errorf("failed to select a local windows ssh port for bootstrap: %w", err)
	}

	privateKeyPEM, publicAuthorizedKey, err := generateSecureLocalWindowsProvisionSSHKeyPair()
	if err != nil {
		return localWindowsSSHProvisionSession{}, fmt.Errorf("failed to generate secure local windows ssh key pair: %w", err)
	}

	privateKeyPath, err := writeLocalWindowsProvisionPrivateKey(projectDir, privateKeyPEM)
	if err != nil {
		return localWindowsSSHProvisionSession{}, err
	}

	statePath, err := createLocalWindowsProvisionStateFile(projectDir)
	if err != nil {
		_ = removeLocalWindowsPrivateKeyFile(privateKeyPath)
		return localWindowsSSHProvisionSession{}, err
	}

	session := localWindowsSSHProvisionSession{
		ConnectionConfig: sshAnsibleConnectionConfig{
			User:            localWindowsProvisionUserName,
			Connection:      "ssh",
			Port:            sshPort,
			SshCommonArgs:   localWindowsSSHCommonArgs,
			SshTimeout:      "120",
			SshRetries:      "3",
			PrivateKeyFile:  filepath.Base(privateKeyPath),
			ShellType:       "powershell",
			ShellExecutable: "powershell.exe",
		},
		StatePath:      statePath,
		PrivateKeyPath: privateKeyPath,
	}

	output, runErr := runLocalWindowsPowerShellScript(
		projectDir,
		localWindowsSSHProvisionBootstrapPowerShell,
		buildLocalWindowsSSHProvisionScriptEnv(statePath, password, publicAuthorizedKey, sshPort, options),
		localWindowsSSHBootstrapTimeout,
		localWindowsSSHBootstrapLogPrefix,
	)
	if runErr != nil {
		cleanupErr := cleanupLocalWindowsSSHProvisionSession(projectDir, session, options)
		if cleanupErr != nil {
			return localWindowsSSHProvisionSession{}, fmt.Errorf("failed to securely bootstrap local SSH access: %w; output: %s (also failed to restore secure SSH state: %v)", runErr, strings.TrimSpace(output), cleanupErr)
		}
		return localWindowsSSHProvisionSession{}, fmt.Errorf("failed to securely bootstrap local SSH access: %w; output: %s", runErr, strings.TrimSpace(output))
	}

	portNumber, convErr := parseLocalWindowsSSHPort(sshPort)
	if convErr != nil {
		cleanupErr := cleanupLocalWindowsSSHProvisionSession(projectDir, session, options)
		if cleanupErr != nil {
			return localWindowsSSHProvisionSession{}, fmt.Errorf("local Windows SSH bootstrap completed but the temporary ssh port %q was invalid: %w (also failed to restore secure SSH state: %v)", sshPort, convErr, cleanupErr)
		}
		return localWindowsSSHProvisionSession{}, fmt.Errorf("local Windows SSH bootstrap completed but the temporary ssh port %q was invalid: %w", sshPort, convErr)
	}

	if err := waitForSSHPortOnPort(localWindowsSSHLoopbackIP, portNumber); err != nil {
		cleanupErr := cleanupLocalWindowsSSHProvisionSession(projectDir, session, options)
		if cleanupErr != nil {
			return localWindowsSSHProvisionSession{}, fmt.Errorf("local Windows SSH bootstrap completed but sshd did not become reachable: %w (also failed to restore secure SSH state: %v)", err, cleanupErr)
		}
		return localWindowsSSHProvisionSession{}, fmt.Errorf("local Windows SSH bootstrap completed but sshd did not become reachable: %w", err)
	}

	return session, nil
}

func runLocalWindowsSSHPreflight(projectDir string, connectionConfig sshAnsibleConnectionConfig) error {
	portNumber, err := parseLocalWindowsSSHPort(connectionConfig.Port)
	if err != nil {
		return fmt.Errorf("failed to parse temporary SSH preflight port %q: %w", connectionConfig.Port, err)
	}
	listenerProcesses, err := inspectLocalWindowsSSHListener(projectDir, portNumber)
	if err != nil {
		return fmt.Errorf("failed to inspect the local listener for Windows SSH preflight on port %s: %w", connectionConfig.Port, err)
	}
	log.Printf("%s listener: port %s owners: %s", localWindowsSSHPreflightLogPrefix, connectionConfig.Port, summarizeLocalWindowsSSHListenerProcesses(listenerProcesses))
	if err := validateLocalWindowsSSHListener(connectionConfig.Port, listenerProcesses); err != nil {
		return err
	}

	cygwinWorkingDir, err := windowsPathToCygwinPath(projectDir)
	if err != nil {
		return fmt.Errorf("failed to convert working directory to cygwin path for SSH preflight: %w", err)
	}
	cygwinBashExecutable, err := getCygwinBashExecutable()
	if err != nil {
		return fmt.Errorf("failed to locate cygwin bash executable for SSH preflight: %w", err)
	}

	sshArgs := []string{
		"ssh",
		"-vvv",
		"-o", "BatchMode=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "PubkeyAuthentication=yes",
		"-o", "PasswordAuthentication=no",
		"-o", "KbdInteractiveAuthentication=no",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "IdentitiesOnly=yes",
		"-o", "ConnectTimeout=20",
		"-p", connectionConfig.Port,
		"-i", connectionConfig.PrivateKeyFile,
		connectionConfig.User + "@" + localWindowsSSHLoopbackIP,
		"whoami.exe",
	}

	quotedArgs := make([]string, 0, len(sshArgs))
	for _, arg := range sshArgs {
		quotedArgs = append(quotedArgs, bashSingleQuote(arg))
	}

	bashCommand := "cd " + bashSingleQuote(cygwinWorkingDir) + " && " + strings.Join(quotedArgs, " ")
	output, err := runProvisionCommandWithCombinedOutputWithEnv(
		projectDir,
		localWindowsSSHPreflightTimeout,
		cygwinBashExecutable,
		[]string{"-l", "-c", bashCommand},
		ansibleRuntimeEnv(),
	)
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput != "" {
		for _, line := range strings.Split(trimmedOutput, "\n") {
			logLine := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
			if logLine == "" {
				continue
			}
			log.Printf("%s ssh: %s", localWindowsSSHPreflightLogPrefix, logLine)
		}
	}
	if err := validateLocalWindowsSSHRemoteBanner(connectionConfig.Port, trimmedOutput); err != nil {
		details := trimmedOutput
		if details == "" {
			details = "no SSH diagnostic output captured"
		}
		return fmt.Errorf("%w; output: %s", err, details)
	}
	if err != nil {
		details := trimmedOutput
		if details == "" {
			details = "no SSH diagnostic output captured"
		}
		return fmt.Errorf("temporary SSH authentication test failed before Ansible started: %w; output: %s", err, details)
	}
	return nil
}

func cleanupLocalWindowsSSHProvisionSession(projectDir string, session localWindowsSSHProvisionSession, options ProvisionOptions) error {
	var output string
	var runErr error
	if session.StatePath != "" {
		output, runErr = runLocalWindowsPowerShellScript(
			projectDir,
			localWindowsSSHProvisionCleanupPowerShell,
			buildLocalWindowsSSHProvisionScriptEnv(session.StatePath, "", "", session.ConnectionConfig.Port, options),
			localWindowsSSHCleanupTimeout,
			localWindowsSSHCleanupLogPrefix,
		)
	}

	removePrivateKeyErr := removeLocalWindowsPrivateKeyFile(session.PrivateKeyPath)
	removeStateErr := removeLocalWindowsProvisionStateFile(session.StatePath)

	if runErr != nil {
		extraErrs := filterNonEmptyStrings(
			describeCleanupErr("remove secure local windows ssh private key", removePrivateKeyErr),
			describeCleanupErr("remove secure local windows provision state file", removeStateErr),
		)
		if len(extraErrs) > 0 {
			return fmt.Errorf("failed to restore secure SSH state: %w; output: %s (also failed to %s)", runErr, strings.TrimSpace(output), strings.Join(extraErrs, "; "))
		}
		return fmt.Errorf("failed to restore secure SSH state: %w; output: %s", runErr, strings.TrimSpace(output))
	}
	if removePrivateKeyErr != nil && removeStateErr != nil {
		return fmt.Errorf("failed to remove secure local windows ssh private key: %w (also failed to remove secure local windows provision state file: %v)", removePrivateKeyErr, removeStateErr)
	}
	if removePrivateKeyErr != nil {
		return removePrivateKeyErr
	}
	if removeStateErr != nil {
		return removeStateErr
	}

	return nil
}

func describeCleanupErr(action string, err error) string {
	if err == nil {
		return ""
	}

	return fmt.Sprintf("%s: %v", action, err)
}

func writeLocalWindowsProvisionPrivateKey(projectDir string, privateKeyPEM []byte) (string, error) {
	privateKeyFile, err := os.CreateTemp(projectDir, ".local-windows-provision-key-*.pem")
	if err != nil {
		return "", fmt.Errorf("failed to create secure local windows ssh private key file: %w", err)
	}
	privateKeyPath := privateKeyFile.Name()
	if err := privateKeyFile.Close(); err != nil {
		_ = os.Remove(privateKeyPath)
		return "", fmt.Errorf("failed to close secure local windows ssh private key file %q: %w", privateKeyPath, err)
	}
	if err := os.WriteFile(privateKeyPath, privateKeyPEM, 0o600); err != nil {
		_ = os.Remove(privateKeyPath)
		return "", fmt.Errorf("failed to write secure local windows ssh private key file %q: %w", privateKeyPath, err)
	}

	return privateKeyPath, nil
}

func removeLocalWindowsPrivateKeyFile(privateKeyPath string) error {
	if strings.TrimSpace(privateKeyPath) == "" {
		return nil
	}

	removeErr := os.Remove(privateKeyPath)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return fmt.Errorf("failed to remove secure local windows ssh private key %q: %w", privateKeyPath, removeErr)
	}

	return nil
}

func generateSecureLocalWindowsProvisionSSHKeyPair() ([]byte, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return nil, "", err
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	publicAuthorizedKey, err := marshalRSAPublicAuthorizedKey(&privateKey.PublicKey)
	if err != nil {
		return nil, "", err
	}

	return privateKeyPEM, publicAuthorizedKey, nil
}

func marshalRSAPublicAuthorizedKey(publicKey *rsa.PublicKey) (string, error) {
	var blob bytes.Buffer
	if err := writeSSHString(&blob, "ssh-rsa"); err != nil {
		return "", err
	}
	if err := writeSSHMPI(&blob, big.NewInt(int64(publicKey.E)).Bytes()); err != nil {
		return "", err
	}
	if err := writeSSHMPI(&blob, publicKey.N.Bytes()); err != nil {
		return "", err
	}

	return "ssh-rsa " + base64.StdEncoding.EncodeToString(blob.Bytes()), nil
}

func writeSSHString(buffer *bytes.Buffer, value string) error {
	return writeSSHBytes(buffer, []byte(value))
}

func writeSSHMPI(buffer *bytes.Buffer, value []byte) error {
	trimmed := bytes.TrimLeft(value, "\x00")
	if len(trimmed) == 0 {
		return writeSSHBytes(buffer, []byte{0})
	}
	if trimmed[0]&0x80 != 0 {
		trimmed = append([]byte{0}, trimmed...)
	}

	return writeSSHBytes(buffer, trimmed)
}

func writeSSHBytes(buffer *bytes.Buffer, value []byte) error {
	valueLength, err := sshWireValueLength(len(value))
	if err != nil {
		return err
	}

	var lengthBytes [4]byte
	binary.BigEndian.PutUint32(lengthBytes[:], valueLength)
	buffer.Write(lengthBytes[:])
	buffer.Write(value)

	return nil
}

func sshWireValueLength(length int) (uint32, error) {
	if int64(length) < 0 || uint64(length) > uint64(^uint32(0)) {
		return 0, fmt.Errorf("ssh wire value length %d exceeds uint32 maximum", length)
	}

	return uint32(length), nil
}

func selectLocalWindowsSSHBootstrapPort(projectDir string) (string, error) {
	standardPort := fmt.Sprintf("%d", sshPort)
	listenerProcesses, err := inspectLocalWindowsSSHListener(projectDir, sshPort)
	if err != nil {
		return "", fmt.Errorf("failed to inspect whether standard SSH port %s is available: %w", standardPort, err)
	}

	if canUseStandardLocalWindowsSSHPort(listenerProcesses) {
		if len(listenerProcesses) == 0 {
			log.Printf("%s: standard SSH port %s is free; using it for this provisioning run.", localWindowsSSHBootstrapLogPrefix, standardPort)
		} else {
			log.Printf("%s: standard SSH port %s is already owned only by sshd; using it for this provisioning run.", localWindowsSSHBootstrapLogPrefix, standardPort)
		}
		return standardPort, nil
	}

	selectedPort, err := selectAvailableLocalWindowsSSHPort()
	if err != nil {
		return "", err
	}
	log.Printf("%s: standard SSH port %s is owned by %s; using temporary loopback port %s instead.", localWindowsSSHBootstrapLogPrefix, standardPort, summarizeLocalWindowsSSHListenerProcesses(listenerProcesses), selectedPort)

	return selectedPort, nil
}

func canUseStandardLocalWindowsSSHPort(processes []localWindowsSSHListenerProcess) bool {
	if len(processes) == 0 {
		return true
	}

	for _, process := range processes {
		if !strings.EqualFold(strings.TrimSpace(process.ProcessName), "sshd") {
			return false
		}
	}

	return true
}

func selectAvailableLocalWindowsSSHPort() (string, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(localWindowsSSHLoopbackIP, "0"))
	if err != nil {
		return "", err
	}
	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return "", fmt.Errorf("unexpected listener address type %T", listener.Addr())
	}

	return fmt.Sprintf("%d", tcpAddr.Port), nil
}

func parseLocalWindowsSSHPort(port string) (int, error) {
	portNumber, err := net.LookupPort("tcp", strings.TrimSpace(port))
	if err != nil {
		return 0, err
	}

	return portNumber, nil
}

func inspectLocalWindowsSSHListener(projectDir string, port int) ([]localWindowsSSHListenerProcess, error) {
	powerShellCommand := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$processIds = @(
    Get-NetTCPConnection -State Listen -LocalPort %d -ErrorAction SilentlyContinue |
        Select-Object -ExpandProperty OwningProcess -Unique
)
$processes = @(
    foreach ($processId in $processIds) {
        $process = Get-Process -Id $processId -ErrorAction SilentlyContinue
        if ($null -eq $process) {
            [pscustomobject]@{
                Id = [int]$processId
                ProcessName = ''
            }
            continue
        }

        [pscustomobject]@{
            Id = [int]$process.Id
            ProcessName = [string]$process.ProcessName
        }
    }
)
@($processes | Sort-Object -Property Id -Unique) | ConvertTo-Json -Compress
`, port)

	output, err := runProvisionCommandWithCombinedOutputWithEnv(
		projectDir,
		localWindowsSSHPreflightTimeout,
		"powershell.exe",
		[]string{
			"-NoLogo",
			"-NoProfile",
			"-NonInteractive",
			"-ExecutionPolicy",
			"Bypass",
			"-Command",
			powerShellCommand,
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	return parseLocalWindowsSSHListenerProcesses(strings.TrimSpace(output))
}

func parseLocalWindowsSSHListenerProcesses(output string) ([]localWindowsSSHListenerProcess, error) {
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}

	var processes []localWindowsSSHListenerProcess
	if err := json.Unmarshal([]byte(output), &processes); err == nil {
		return processes, nil
	}

	var singleProcess localWindowsSSHListenerProcess
	if err := json.Unmarshal([]byte(output), &singleProcess); err == nil {
		return []localWindowsSSHListenerProcess{singleProcess}, nil
	}

	return nil, fmt.Errorf("failed to decode listener process inspection output %q", output)
}

func summarizeLocalWindowsSSHListenerProcesses(processes []localWindowsSSHListenerProcess) string {
	if len(processes) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(processes))
	for _, process := range processes {
		name := strings.TrimSpace(process.ProcessName)
		if name == "" {
			name = "unknown"
		}
		parts = append(parts, fmt.Sprintf("%s(pid=%d)", name, process.ID))
	}

	return strings.Join(parts, ", ")
}

func validateLocalWindowsSSHListener(port string, processes []localWindowsSSHListenerProcess) error {
	if len(processes) == 0 {
		return fmt.Errorf("the temporary Windows SSH preflight port %s does not have a listening process after bootstrap completed", port)
	}

	unexpectedProcesses := make([]localWindowsSSHListenerProcess, 0, len(processes))
	for _, process := range processes {
		if strings.EqualFold(strings.TrimSpace(process.ProcessName), "sshd") {
			continue
		}
		unexpectedProcesses = append(unexpectedProcesses, process)
	}
	if len(unexpectedProcesses) == 0 {
		return nil
	}

	details := summarizeLocalWindowsSSHListenerProcesses(processes)
	for _, process := range unexpectedProcesses {
		if strings.EqualFold(strings.TrimSpace(process.ProcessName), "wslrelay") {
			return fmt.Errorf("the temporary Windows SSH preflight port %s is being intercepted by %s; expected only Windows sshd. WSL SSH forwarding appears to be taking over this port", port, details)
		}
	}

	return fmt.Errorf("the temporary Windows SSH preflight port %s is owned by unexpected processes: %s; expected only Windows sshd", port, details)
}

func validateLocalWindowsSSHRemoteBanner(port string, output string) error {
	remoteSoftwareVersion := extractLocalWindowsSSHRemoteSoftwareVersion(output)
	if remoteSoftwareVersion == "" {
		return fmt.Errorf("the temporary Windows SSH preflight on port %s did not capture a remote SSH banner, so the target server identity could not be verified", port)
	}
	if strings.Contains(strings.ToLower(remoteSoftwareVersion), "openssh_for_windows") {
		return nil
	}

	return fmt.Errorf("the temporary Windows SSH preflight on port %s reached %q instead of Windows OpenSSH", port, remoteSoftwareVersion)
}

func extractLocalWindowsSSHRemoteSoftwareVersion(output string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmedLine := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if trimmedLine == "" {
			continue
		}

		lowerLine := strings.ToLower(trimmedLine)
		marker := "remote software version "
		index := strings.Index(lowerLine, marker)
		if index < 0 {
			continue
		}

		return strings.TrimSpace(trimmedLine[index+len(marker):])
	}

	return ""
}
