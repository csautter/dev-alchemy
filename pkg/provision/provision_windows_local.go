package provision

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	localWindowsWinRMInventoryPath   = "./inventory/localhost_windows_winrm.yml"
	localWindowsWinRMInventoryTarget = "windows_host"
	localWindowsWinRMHTTPSPort       = "5986"
	localWindowsProvisionUserName    = "devalchemy_ansible"
	localWindowsBootstrapTimeout     = 2 * time.Minute
	localWindowsCleanupTimeout       = 2 * time.Minute

	localWindowsProvisionStatePathEnvVar  = "DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH"
	localWindowsProvisionUserEnvVar       = "DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_USER"
	localWindowsProvisionPasswordEnvVar   = "DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_PASSWORD" // #nosec G101 -- environment variable name, not an embedded credential.
	localWindowsForceWinRMUninstallEnvVar = "DEV_ALCHEMY_LOCAL_WINDOWS_FORCE_WINRM_UNINSTALL"
	localWindowsFirewallRuleName          = "DevAlchemyLocalWinRMHTTPS"
)

var setupLocalWindowsProvisionSessionFunc = setupLocalWindowsProvisionSession
var cleanupLocalWindowsProvisionSessionFunc = cleanupLocalWindowsProvisionSession

type localWindowsProvisionSession struct {
	ConnectionConfig windowsAnsibleConnectionConfig
	StatePath        string
}

func runLocalWindowsProvision(projectDir string, options ProvisionOptions) error {
	session, err := setupLocalWindowsProvisionSessionFunc(projectDir)
	if err != nil {
		return err
	}

	inventoryPath, inventoryTarget := resolveStaticInventoryPathAndTarget(
		localWindowsWinRMInventoryPath,
		localWindowsWinRMInventoryTarget,
		options,
	)

	args, argsCleanup, err := buildWindowsStaticInventoryProvisionArgs(
		projectDir,
		inventoryPath,
		inventoryTarget,
		session.ConnectionConfig,
		options,
	)
	if err != nil {
		cleanupErr := cleanupLocalWindowsProvisionSessionFunc(projectDir, session)
		if cleanupErr != nil {
			return fmt.Errorf("failed to build ansible arguments for secure local windows provision: %w (also failed to restore secure WinRM state: %v)", err, cleanupErr)
		}
		return fmt.Errorf("failed to build ansible arguments for secure local windows provision: %w", err)
	}

	runErr := runAnsibleProvisionCommandFunc(projectDir, args, 90*time.Minute, "local:windows:provision")
	argsCleanupErr := argsCleanup()
	cleanupErr := cleanupLocalWindowsProvisionSessionFunc(projectDir, session)

	if runErr != nil {
		if argsCleanupErr != nil && cleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for local host windows: %w (also failed to clean ansible temp files: %v; cleanup failed: %v)", runErr, argsCleanupErr, cleanupErr)
		}
		if argsCleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for local host windows: %w (also failed to clean ansible temp files: %v)", runErr, argsCleanupErr)
		}
		if cleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for local host windows: %w (also failed to restore secure WinRM state: %v)", runErr, cleanupErr)
		}
		return fmt.Errorf("ansible provisioning failed for local host windows: %w", runErr)
	}
	if argsCleanupErr != nil && cleanupErr != nil {
		return fmt.Errorf("failed to clean ansible temp files: %w (also failed to restore secure WinRM state: %v)", argsCleanupErr, cleanupErr)
	}
	if argsCleanupErr != nil {
		return fmt.Errorf("failed to clean ansible temp files: %w", argsCleanupErr)
	}
	if cleanupErr != nil {
		return fmt.Errorf("failed to restore secure WinRM state after local host windows provision: %w", cleanupErr)
	}

	return nil
}

func buildWindowsStaticInventoryProvisionArgs(projectDir string, inventoryPath string, inventoryTarget string, connectionConfig windowsAnsibleConnectionConfig, options ProvisionOptions) ([]string, func() error, error) {
	extraVars, err := buildWindowsProvisionExtraVars(connectionConfig)
	if err != nil {
		return nil, nil, err
	}

	return buildStaticInventoryProvisionArgsWithExtraVars(projectDir, inventoryPath, inventoryTarget, extraVars, options)
}

func buildLocalWindowsProvisionScriptEnv(statePath string, password string) []string {
	env := []string{
		localWindowsProvisionStatePathEnvVar + "=" + statePath,
		localWindowsForceWinRMUninstallEnvVar + "=" + fmt.Sprintf("%t", localWindowsForceWinRMUninstall),
	}
	if password == "" {
		return env
	}

	return append(env,
		localWindowsProvisionUserEnvVar+"="+localWindowsProvisionUserName,
		localWindowsProvisionPasswordEnvVar+"="+password,
	)
}

func setupLocalWindowsProvisionSession(projectDir string) (localWindowsProvisionSession, error) {
	password, err := generateSecureLocalWindowsProvisionPassword()
	if err != nil {
		return localWindowsProvisionSession{}, fmt.Errorf("failed to generate secure local windows provision password: %w", err)
	}

	stateFile, err := os.CreateTemp(projectDir, ".local-windows-provision-state-*.json")
	if err != nil {
		return localWindowsProvisionSession{}, fmt.Errorf("failed to create secure local windows provision state file: %w", err)
	}
	statePath := stateFile.Name()
	if err := stateFile.Close(); err != nil {
		_ = os.Remove(statePath)
		return localWindowsProvisionSession{}, fmt.Errorf("failed to close secure local windows provision state file: %w", err)
	}

	output, runErr := runLocalWindowsPowerShellScript(
		projectDir,
		localWindowsProvisionBootstrapPowerShell,
		buildLocalWindowsProvisionScriptEnv(statePath, password),
		localWindowsBootstrapTimeout,
	)
	if runErr != nil {
		cleanupErr := cleanupLocalWindowsProvisionSession(projectDir, localWindowsProvisionSession{StatePath: statePath})
		if cleanupErr != nil {
			return localWindowsProvisionSession{}, fmt.Errorf("failed to securely bootstrap local WinRM access: %w; output: %s (also failed to restore secure WinRM state: %v)", runErr, strings.TrimSpace(output), cleanupErr)
		}
		return localWindowsProvisionSession{}, fmt.Errorf("failed to securely bootstrap local WinRM access: %w; output: %s", runErr, strings.TrimSpace(output))
	}

	return localWindowsProvisionSession{
		ConnectionConfig: windowsAnsibleConnectionConfig{
			User:                 localWindowsProvisionUserName,
			Password:             password,
			Connection:           "winrm",
			WinrmTransport:       "basic",
			Port:                 localWindowsWinRMHTTPSPort,
			WinrmScheme:          "https",
			ServerCertValidation: "ignore",
		},
		StatePath: statePath,
	}, nil
}

func cleanupLocalWindowsProvisionSession(projectDir string, session localWindowsProvisionSession) error {
	if session.StatePath == "" {
		return nil
	}

	output, runErr := runLocalWindowsPowerShellScript(
		projectDir,
		localWindowsProvisionCleanupPowerShell,
		buildLocalWindowsProvisionScriptEnv(session.StatePath, ""),
		localWindowsCleanupTimeout,
	)
	removeErr := os.Remove(session.StatePath)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		if runErr != nil {
			return fmt.Errorf("failed to restore secure WinRM state: %w; output: %s (also failed to remove state file %q: %v)", runErr, strings.TrimSpace(output), session.StatePath, removeErr)
		}
		return fmt.Errorf("failed to remove secure local windows provision state file %q: %w", session.StatePath, removeErr)
	}
	if runErr != nil {
		return fmt.Errorf("failed to restore secure WinRM state: %w; output: %s", runErr, strings.TrimSpace(output))
	}

	return nil
}

func runLocalWindowsPowerShellScript(projectDir string, script string, extraEnv []string, timeout time.Duration) (string, error) {
	elevatedScriptPath, outputPath, cleanupFiles, err := writeElevatedLocalWindowsPowerShellArtifacts(projectDir, script, extraEnv)
	if err != nil {
		return "", err
	}
	defer cleanupFiles()

	stopStreaming := make(chan struct{})
	streamingDone := make(chan struct{})
	go streamLocalWindowsPowerShellOutput(outputPath, os.Stdout, stopStreaming, streamingDone)

	output, runErr := runProvisionCommandWithCombinedOutputWithEnv(
		projectDir,
		timeout,
		"powershell.exe",
		[]string{
			"-NoLogo",
			"-NoProfile",
			"-NonInteractive",
			"-ExecutionPolicy",
			"Bypass",
			"-Command",
			buildLocalWindowsElevationLauncherPowerShell(elevatedScriptPath, outputPath),
		},
		nil,
	)
	close(stopStreaming)
	<-streamingDone

	scriptOutputBytes, readErr := os.ReadFile(outputPath)
	scriptOutput := strings.TrimSpace(decodeLocalWindowsPowerShellOutput(scriptOutputBytes))
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		if runErr != nil {
			return "", fmt.Errorf("failed to read elevated local windows provisioning output from %q: %w (launcher output: %s; launcher error: %v)", outputPath, readErr, strings.TrimSpace(output), runErr)
		}
		return "", fmt.Errorf("failed to read elevated local windows provisioning output from %q: %w", outputPath, readErr)
	}

	if runErr != nil {
		combinedOutput := strings.TrimSpace(strings.Join(filterNonEmptyStrings(scriptOutput, strings.TrimSpace(output)), "\n"))
		if combinedOutput == "" {
			combinedOutput = "no output captured; the Windows UAC prompt may have been cancelled"
		}
		return combinedOutput, runErr
	}

	return scriptOutput, nil
}

func writeElevatedLocalWindowsPowerShellArtifacts(projectDir string, script string, extraEnv []string) (string, string, func(), error) {
	elevatedScriptFile, err := os.CreateTemp(projectDir, ".local-windows-provision-elevated-*.ps1")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to create elevated local windows provisioning script: %w", err)
	}
	elevatedScriptPath := elevatedScriptFile.Name()
	if err := elevatedScriptFile.Close(); err != nil {
		_ = os.Remove(elevatedScriptPath)
		return "", "", nil, fmt.Errorf("failed to close elevated local windows provisioning script %q: %w", elevatedScriptPath, err)
	}

	outputFile, err := os.CreateTemp(projectDir, ".local-windows-provision-output-*.log")
	if err != nil {
		_ = os.Remove(elevatedScriptPath)
		return "", "", nil, fmt.Errorf("failed to create elevated local windows provisioning output file: %w", err)
	}
	outputPath := outputFile.Name()
	if err := outputFile.Close(); err != nil {
		_ = os.Remove(elevatedScriptPath)
		_ = os.Remove(outputPath)
		return "", "", nil, fmt.Errorf("failed to close elevated local windows provisioning output file %q: %w", outputPath, err)
	}

	scriptContent := buildElevatedLocalWindowsPowerShellScript(script, extraEnv, outputPath)
	if err := os.WriteFile(elevatedScriptPath, []byte(scriptContent), 0o600); err != nil {
		_ = os.Remove(elevatedScriptPath)
		_ = os.Remove(outputPath)
		return "", "", nil, fmt.Errorf("failed to write elevated local windows provisioning script %q: %w", elevatedScriptPath, err)
	}

	cleanupFiles := func() {
		_ = os.Remove(elevatedScriptPath)
		_ = os.Remove(outputPath)
	}

	return elevatedScriptPath, outputPath, cleanupFiles, nil
}

func buildElevatedLocalWindowsPowerShellScript(script string, extraEnv []string, outputPath string) string {
	var builder strings.Builder
	builder.WriteString("$ErrorActionPreference = 'Stop'\n\n")
	builder.WriteString(fmt.Sprintf("$outputPath = '%s'\n", escapePowerShellSingleQuotedString(outputPath)))
	builder.WriteString("if (Test-Path -Path $outputPath) {\n")
	builder.WriteString("    Remove-Item -Path $outputPath -Force\n")
	builder.WriteString("}\n\n")
	for _, entry := range extraEnv {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		builder.WriteString(fmt.Sprintf("$env:%s = '%s'\n", key, escapePowerShellSingleQuotedString(value)))
	}
	if len(extraEnv) > 0 {
		builder.WriteString("\n")
	}
	builder.WriteString("try {\n")
	builder.WriteString("    & {\n")
	builder.WriteString(script)
	builder.WriteString("\n    } *>> $outputPath\n")
	builder.WriteString("    exit $LASTEXITCODE\n")
	builder.WriteString("} catch {\n")
	builder.WriteString("    ($_ | Out-String) | Add-Content -Path $outputPath -Encoding Ascii\n")
	builder.WriteString("    exit 1\n")
	builder.WriteString("}\n")

	return builder.String()
}

func buildLocalWindowsElevationLauncherPowerShell(elevatedScriptPath string, outputPath string) string {
	return fmt.Sprintf(`
$ErrorActionPreference = 'Stop'

$elevatedScriptPath = '%s'
$outputPath = '%s'

try {
    $argumentList = '-NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "%s"'
    $process = Start-Process -FilePath 'powershell.exe' -ArgumentList $argumentList -Verb RunAs -WindowStyle Hidden -Wait -PassThru
    exit $process.ExitCode
} catch {
    ($_ | Out-String) | Add-Content -Path $outputPath -Encoding UTF8
    exit 1
}
`, escapePowerShellSingleQuotedString(elevatedScriptPath), escapePowerShellSingleQuotedString(outputPath), escapePowerShellDoubleQuotedArgument(elevatedScriptPath))
}

func escapePowerShellSingleQuotedString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func escapePowerShellDoubleQuotedArgument(value string) string {
	return strings.ReplaceAll(filepath.Clean(value), `"`, `""`)
}

func filterNonEmptyStrings(values ...string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered
}

func streamLocalWindowsPowerShellOutput(outputPath string, writer io.Writer, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	printedLength := 0
	flush := func() {
		content, err := os.ReadFile(outputPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return
			}
			return
		}

		decoded := decodeLocalWindowsPowerShellOutput(content)
		if len(decoded) <= printedLength {
			return
		}

		_, _ = io.WriteString(writer, decoded[printedLength:])
		printedLength = len(decoded)
	}

	for {
		select {
		case <-stop:
			flush()
			return
		case <-ticker.C:
			flush()
		}
	}
}

func decodeLocalWindowsPowerShellOutput(content []byte) string {
	if len(content) == 0 {
		return ""
	}

	if len(content) >= 2 {
		switch {
		case content[0] == 0xFF && content[1] == 0xFE:
			return decodeUTF16LittleEndian(content[2:])
		case content[0] == 0xFE && content[1] == 0xFF:
			return decodeUTF16BigEndian(content[2:])
		}
	}
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		return string(content[3:])
	}

	if looksLikeUTF16LittleEndian(content) {
		return decodeUTF16LittleEndian(content)
	}
	if looksLikeUTF16BigEndian(content) {
		return decodeUTF16BigEndian(content)
	}

	return string(content)
}

func looksLikeUTF16LittleEndian(content []byte) bool {
	if len(content) < 4 || len(content)%2 != 0 {
		return false
	}

	zeroCount := 0
	sampleCount := 0
	for index := 1; index < len(content) && sampleCount < 16; index += 2 {
		sampleCount++
		if content[index] == 0 {
			zeroCount++
		}
	}

	return sampleCount > 0 && zeroCount*2 >= sampleCount
}

func looksLikeUTF16BigEndian(content []byte) bool {
	if len(content) < 4 || len(content)%2 != 0 {
		return false
	}

	zeroCount := 0
	sampleCount := 0
	for index := 0; index < len(content) && sampleCount < 16; index += 2 {
		sampleCount++
		if content[index] == 0 {
			zeroCount++
		}
	}

	return sampleCount > 0 && zeroCount*2 >= sampleCount
}

func decodeUTF16LittleEndian(content []byte) string {
	if len(content)%2 != 0 {
		content = content[:len(content)-1]
	}
	runes := make([]rune, 0, len(content)/2)
	for index := 0; index+1 < len(content); index += 2 {
		runes = append(runes, rune(uint16(content[index])|uint16(content[index+1])<<8))
	}
	return string(runes)
}

func decodeUTF16BigEndian(content []byte) string {
	if len(content)%2 != 0 {
		content = content[:len(content)-1]
	}
	runes := make([]rune, 0, len(content)/2)
	for index := 0; index+1 < len(content); index += 2 {
		runes = append(runes, rune(uint16(content[index])<<8|uint16(content[index+1])))
	}
	return string(runes)
}

func generateSecureLocalWindowsProvisionPassword() (string, error) {
	const lowercase = "abcdefghijklmnopqrstuvwxyz"
	const uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const digits = "0123456789"
	const special = "!@#$%^&*()-_=+[]{}"
	const all = lowercase + uppercase + digits + special

	requiredSets := []string{lowercase, uppercase, digits, special}
	passwordRunes := make([]byte, 0, 24)

	for _, charset := range requiredSets {
		index, err := randomInt(len(charset))
		if err != nil {
			return "", err
		}
		passwordRunes = append(passwordRunes, charset[index])
	}
	for len(passwordRunes) < 24 {
		index, err := randomInt(len(all))
		if err != nil {
			return "", err
		}
		passwordRunes = append(passwordRunes, all[index])
	}
	for index := len(passwordRunes) - 1; index > 0; index-- {
		swapIndex, err := randomInt(index + 1)
		if err != nil {
			return "", err
		}
		passwordRunes[index], passwordRunes[swapIndex] = passwordRunes[swapIndex], passwordRunes[index]
	}

	return string(passwordRunes), nil
}

func randomInt(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be greater than zero")
	}

	var randomByte [1]byte
	limit := 256 - (256 % max)
	for {
		if _, err := rand.Read(randomByte[:]); err != nil {
			return 0, err
		}
		if int(randomByte[0]) >= limit {
			continue
		}
		return int(randomByte[0]) % max, nil
	}
}

const localWindowsProvisionBootstrapPowerShell = `
$ErrorActionPreference = 'Stop'

$statePath = $env:DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH
$userName = $env:DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_USER
$passwordPlain = $env:DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_PASSWORD
$forceWinRMUninstall = [System.Convert]::ToBoolean($env:DEV_ALCHEMY_LOCAL_WINDOWS_FORCE_WINRM_UNINSTALL)

if ([string]::IsNullOrWhiteSpace($statePath)) {
    throw 'DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH is required.'
}
if ([string]::IsNullOrWhiteSpace($userName)) {
    throw 'DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_USER is required.'
}
if ([string]::IsNullOrWhiteSpace($passwordPlain)) {
    throw 'DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_PASSWORD is required.'
}

function Save-State($state) {
    $state | ConvertTo-Json -Compress | Set-Content -Path $statePath -Encoding Ascii
}

function Get-WsmanBoolState([string]$path) {
    if (-not (Test-Path -Path $path)) {
        return @{
            Exists = $false
            Value = $false
        }
    }

    return @{
        Exists = $true
        Value = [System.Convert]::ToBoolean((Get-Item -Path $path).Value)
    }
}

function Assert-WsmanPathExists([string]$path) {
    if (-not (Test-Path -Path $path)) {
        throw ("Required WSMan path was not found after preparing WinRM: " + $path)
    }
}

function Get-WinRMServiceState() {
    $service = Get-CimInstance -ClassName Win32_Service -Filter "Name='WinRM'" -ErrorAction Stop
    if ($null -eq $service) {
        throw 'WinRM service is not installed on this host.'
    }

    return @{
        WasRunning = ([string]$service.State -eq 'Running')
        StartMode = [string]$service.StartMode
    }
}

function Get-RegistryDWORDState([string]$path, [string]$name) {
    if (-not (Test-Path -Path $path)) {
        return @{
            Exists = $false
            Value = 0
        }
    }

    $property = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
    if ($null -eq $property) {
        return @{
            Exists = $false
            Value = 0
        }
    }

    return @{
        Exists = $true
        Value = [int]($property.$name)
    }
}

function Set-RegistryDWORDValue([string]$path, [string]$name, [int]$value) {
    if (-not (Test-Path -Path $path)) {
        New-Item -Path $path -Force | Out-Null
    }

    if ($null -eq (Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue)) {
        New-ItemProperty -Path $path -Name $name -Value $value -PropertyType DWord -Force | Out-Null
    } else {
        Set-ItemProperty -Path $path -Name $name -Value $value
    }
}

function Get-NetFirewallRuleState([string]$name) {
    $rule = Get-NetFirewallRule -Name $name -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -eq $rule) {
        return @{
            Exists = $false
            Enabled = $false
        }
    }

    return @{
        Exists = $true
        Enabled = ($rule.Enabled -eq 'True')
    }
}

function Get-LocalAdministratorsGroupName() {
    $group = Get-LocalGroup | Where-Object { $null -ne $_.SID -and $_.SID.Value -eq 'S-1-5-32-544' } | Select-Object -First 1
    if ($null -eq $group) {
        throw 'The built-in local Administrators group was not found.'
    }

    return [string]$group.Name
}

function Restore-WinRMServiceState([bool]$wasRunning, [string]$startMode) {
    $service = Get-Service -Name 'WinRM' -ErrorAction SilentlyContinue
    if ($null -eq $service) {
        return
    }

    if ($wasRunning) {
        if ($service.Status -ne 'Running') {
            Start-Service -Name 'WinRM'
            $service = Get-Service -Name 'WinRM'
        }
    } elseif ($service.Status -eq 'Running') {
        Stop-Service -Name 'WinRM' -Force
        $service = Get-Service -Name 'WinRM'
    }

    switch ($startMode.ToLowerInvariant()) {
        'auto' {
            Set-Service -Name 'WinRM' -StartupType Automatic
        }
        'manual' {
            Set-Service -Name 'WinRM' -StartupType Manual
        }
        'disabled' {
            if ($service.Status -eq 'Running') {
                Stop-Service -Name 'WinRM' -Force
            }
            Set-Service -Name 'WinRM' -StartupType Disabled
        }
    }
}

$listenerKeys = @()
try {
    $listenerKeys = @(Get-ChildItem -Path WSMan:\localhost\Listener -ErrorAction Stop | ForEach-Object { $_.Keys })
} catch {
    $listenerKeys = @()
}

$winRMServiceState = Get-WinRMServiceState
$basicAuthState = Get-WsmanBoolState 'WSMan:\localhost\Service\Auth\Basic'
$allowUnencryptedState = Get-WsmanBoolState 'WSMan:\localhost\Service\AllowUnencrypted'
$localAccountTokenFilterPolicyState = Get-RegistryDWORDState 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' 'LocalAccountTokenFilterPolicy'
$firewallRuleState = Get-NetFirewallRuleState 'DevAlchemyLocalWinRMHTTPS'

$state = @{
    UserName = $userName
    ForceWinRMUninstall = [bool]$forceWinRMUninstall
    ListenerKeys = @($listenerKeys)
    HadListeners = ($listenerKeys.Count -gt 0)
    WinRMServiceWasRunning = [bool]$winRMServiceState.WasRunning
    WinRMServiceStartMode = [string]$winRMServiceState.StartMode
    BasicAuthPathExisted = [bool]$basicAuthState.Exists
    BasicAuthEnabled = [bool]$basicAuthState.Value
    AllowUnencryptedPathExisted = [bool]$allowUnencryptedState.Exists
    AllowUnencryptedEnabled = [bool]$allowUnencryptedState.Value
    LocalAccountTokenFilterPolicyExisted = [bool]$localAccountTokenFilterPolicyState.Exists
    LocalAccountTokenFilterPolicyValue = [int]$localAccountTokenFilterPolicyState.Value
    FirewallRuleExisted = [bool]$firewallRuleState.Exists
    FirewallRuleEnabled = [bool]$firewallRuleState.Enabled
    CreatedHttpsListener = $false
    CreatedCertificateThumbprint = ''
}
Save-State $state

if ([string]$state.WinRMServiceStartMode -eq 'Disabled') {
    Set-Service -Name 'WinRM' -StartupType Manual
}
if (-not $state.WinRMServiceWasRunning) {
    Start-Service -Name 'WinRM'
}

Assert-WsmanPathExists 'WSMan:\localhost\Service\AllowUnencrypted'
Assert-WsmanPathExists 'WSMan:\localhost\Service\Auth\Basic'
Set-Item -Path 'WSMan:\localhost\Service\AllowUnencrypted' -Value $false
Set-Item -Path 'WSMan:\localhost\Service\Auth\Basic' -Value $true
Set-RegistryDWORDValue 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' 'LocalAccountTokenFilterPolicy' 1

$httpsListener = @(
    Get-ChildItem -Path WSMan:\localhost\Listener -ErrorAction SilentlyContinue |
        Where-Object { $_.Keys -match 'Transport=HTTPS' -and $_.Keys -match 'Port=5986' }
)
if ($httpsListener.Count -eq 0) {
    $certificate = New-SelfSignedCertificate -DnsName 'localhost' -CertStoreLocation 'Cert:\LocalMachine\My' -FriendlyName 'Dev Alchemy Local WinRM HTTPS'
    New-Item -Path WSMan:\localhost\Listener -Transport HTTPS -Address * -CertificateThumbprint $certificate.Thumbprint -Port 5986 -Force | Out-Null
    $state.CreatedHttpsListener = $true
    $state.CreatedCertificateThumbprint = $certificate.Thumbprint
    Save-State $state
}

if (-not [bool]$state.FirewallRuleExisted) {
    New-NetFirewallRule -Name 'DevAlchemyLocalWinRMHTTPS' -DisplayName 'Dev Alchemy Local WinRM HTTPS' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 5986 -Profile Any | Out-Null
} elseif (-not [bool]$state.FirewallRuleEnabled) {
    Enable-NetFirewallRule -Name 'DevAlchemyLocalWinRMHTTPS' | Out-Null
}

$securePassword = ConvertTo-SecureString -String $passwordPlain -AsPlainText -Force
$localUser = Get-LocalUser -Name $userName -ErrorAction SilentlyContinue
if ($null -eq $localUser) {
    $localUser = New-LocalUser -Name $userName -Password $securePassword -PasswordNeverExpires -Description 'Dev Alchemy Ansible acct'
} else {
    Set-LocalUser -Name $userName -Password $securePassword -Description 'Dev Alchemy Ansible acct'
    Enable-LocalUser -Name $userName
    $localUser = Get-LocalUser -Name $userName
}

$administratorsGroupName = Get-LocalAdministratorsGroupName
$isAdministrator = @(
    Get-LocalGroupMember -Group $administratorsGroupName -ErrorAction SilentlyContinue |
        Where-Object { $null -ne $_.SID -and $_.SID.Value -eq $localUser.SID.Value }
).Count -gt 0
if (-not $isAdministrator) {
    Add-LocalGroupMember -Group $administratorsGroupName -Member $userName
}
`

const localWindowsProvisionCleanupPowerShell = `
$ErrorActionPreference = 'Stop'

$statePath = $env:DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH
$forceWinRMUninstall = [System.Convert]::ToBoolean($env:DEV_ALCHEMY_LOCAL_WINDOWS_FORCE_WINRM_UNINSTALL)
if ([string]::IsNullOrWhiteSpace($statePath) -or -not (Test-Path -Path $statePath)) {
    return
}

$state = Get-Content -Path $statePath -Raw | ConvertFrom-Json

function Restore-WinRMServiceState([bool]$wasRunning, [string]$startMode) {
    $service = Get-Service -Name 'WinRM' -ErrorAction SilentlyContinue
    if ($null -eq $service) {
        return
    }

    if ($wasRunning) {
        if ($service.Status -ne 'Running') {
            Start-Service -Name 'WinRM'
            $service = Get-Service -Name 'WinRM'
        }
    } elseif ($service.Status -eq 'Running') {
        Stop-Service -Name 'WinRM' -Force
        $service = Get-Service -Name 'WinRM'
    }

    switch ($startMode.ToLowerInvariant()) {
        'auto' {
            Set-Service -Name 'WinRM' -StartupType Automatic
        }
        'manual' {
            Set-Service -Name 'WinRM' -StartupType Manual
        }
        'disabled' {
            if ($service.Status -eq 'Running') {
                Stop-Service -Name 'WinRM' -Force
            }
            Set-Service -Name 'WinRM' -StartupType Disabled
        }
    }
}

function Restore-RegistryDWORDState([string]$path, [string]$name, [bool]$existed, [int]$value) {
    if ($existed) {
        if (-not (Test-Path -Path $path)) {
            New-Item -Path $path -Force | Out-Null
        }
        if ($null -eq (Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue)) {
            New-ItemProperty -Path $path -Name $name -Value $value -PropertyType DWord -Force | Out-Null
        } else {
            Set-ItemProperty -Path $path -Name $name -Value $value
        }
        return
    }

    if (Test-Path -Path $path) {
        Remove-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
    }
}

function Disable-WinRMFirewallRules() {
    $rules = Get-NetFirewallRule -DisplayGroup 'Windows Remote Management' -ErrorAction SilentlyContinue
    if ($null -ne $rules) {
        $rules | Disable-NetFirewallRule | Out-Null
    }
}

function Restore-FirewallRuleState([string]$name, [bool]$existed, [bool]$enabled) {
    $rule = Get-NetFirewallRule -Name $name -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $existed) {
        if ($null -ne $rule) {
            $rule | Remove-NetFirewallRule | Out-Null
        }
        return
    }

    if ($null -eq $rule) {
        return
    }

    if ($enabled) {
        $rule | Enable-NetFirewallRule | Out-Null
    } else {
        $rule | Disable-NetFirewallRule | Out-Null
    }
}

$userName = [string]$state.UserName
if (-not [string]::IsNullOrWhiteSpace($userName)) {
    $localUser = Get-LocalUser -Name $userName -ErrorAction SilentlyContinue
    if ($null -ne $localUser) {
        Disable-LocalUser -Name $userName
    }
}

if ([bool]$state.CreatedHttpsListener) {
    Get-ChildItem -Path WSMan:\localhost\Listener -ErrorAction SilentlyContinue |
        Where-Object { $_.Keys -match 'Transport=HTTPS' -and $_.Keys -match 'Port=5986' } |
        ForEach-Object { Remove-Item -Path $_.PSPath -Recurse -Force }
}

$originalListenerKeys = @()
if ($null -ne $state.ListenerKeys) {
    $originalListenerKeys = @($state.ListenerKeys | ForEach-Object { [string]$_ })
}
Get-ChildItem -Path WSMan:\localhost\Listener -ErrorAction SilentlyContinue |
    Where-Object { $originalListenerKeys -notcontains [string]$_.Keys } |
    ForEach-Object { Remove-Item -Path $_.PSPath -Recurse -Force }

if ($forceWinRMUninstall -and (Test-Path -Path 'WSMan:\localhost\Service\Auth\Basic')) {
    Set-Item -Path 'WSMan:\localhost\Service\Auth\Basic' -Value $false
} elseif ([bool]$state.BasicAuthPathExisted -and (Test-Path -Path 'WSMan:\localhost\Service\Auth\Basic')) {
    Set-Item -Path 'WSMan:\localhost\Service\Auth\Basic' -Value ([bool]$state.BasicAuthEnabled)
}
if ($forceWinRMUninstall -and (Test-Path -Path 'WSMan:\localhost\Service\AllowUnencrypted')) {
    Set-Item -Path 'WSMan:\localhost\Service\AllowUnencrypted' -Value $false
} elseif ([bool]$state.AllowUnencryptedPathExisted -and (Test-Path -Path 'WSMan:\localhost\Service\AllowUnencrypted')) {
    Set-Item -Path 'WSMan:\localhost\Service\AllowUnencrypted' -Value ([bool]$state.AllowUnencryptedEnabled)
}
Restore-RegistryDWORDState 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' 'LocalAccountTokenFilterPolicy' ([bool]$state.LocalAccountTokenFilterPolicyExisted) ([int]$state.LocalAccountTokenFilterPolicyValue)
Restore-FirewallRuleState 'DevAlchemyLocalWinRMHTTPS' ([bool]$state.FirewallRuleExisted) ([bool]$state.FirewallRuleEnabled)

$certificateThumbprint = [string]$state.CreatedCertificateThumbprint
if (-not [string]::IsNullOrWhiteSpace($certificateThumbprint)) {
    $certificatePath = 'Cert:\LocalMachine\My\' + $certificateThumbprint
    if (Test-Path -Path $certificatePath) {
        Remove-Item -Path $certificatePath -Force
    }
}

if ($forceWinRMUninstall) {
    Get-ChildItem -Path WSMan:\localhost\Listener -ErrorAction SilentlyContinue |
        ForEach-Object { Remove-Item -Path $_.PSPath -Recurse -Force }
    Get-NetFirewallRule -Name 'DevAlchemyLocalWinRMHTTPS' -ErrorAction SilentlyContinue | Remove-NetFirewallRule | Out-Null
    Disable-WinRMFirewallRules
    Restore-WinRMServiceState $false 'Disabled'
} else {
    Restore-WinRMServiceState ([bool]$state.WinRMServiceWasRunning) ([string]$state.WinRMServiceStartMode)
}
`
