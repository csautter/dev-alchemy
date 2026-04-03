package provision

import (
	"crypto/rand"
	"errors"
	"fmt"
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
)

var setupLocalWindowsProvisionSessionFunc = setupLocalWindowsProvisionSession
var cleanupLocalWindowsProvisionSessionFunc = cleanupLocalWindowsProvisionSession

type localWindowsProvisionSession struct {
	ConnectionConfig windowsAnsibleConnectionConfig
	StatePath        string
}

func runLocalWindowsProvision(projectDir string, check bool) error {
	session, err := setupLocalWindowsProvisionSessionFunc(projectDir)
	if err != nil {
		return err
	}

	args, argsCleanup, err := buildWindowsStaticInventoryProvisionArgs(
		projectDir,
		localWindowsWinRMInventoryPath,
		localWindowsWinRMInventoryTarget,
		session.ConnectionConfig,
		check,
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

func buildWindowsStaticInventoryProvisionArgs(projectDir string, inventoryPath string, inventoryTarget string, connectionConfig windowsAnsibleConnectionConfig, check bool) ([]string, func() error, error) {
	extraVars, err := buildWindowsProvisionExtraVars(connectionConfig)
	if err != nil {
		return nil, nil, err
	}

	return buildStaticInventoryProvisionArgsWithExtraVars(projectDir, inventoryPath, inventoryTarget, extraVars, check)
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

	scriptOutputBytes, readErr := os.ReadFile(outputPath)
	scriptOutput := strings.TrimSpace(string(scriptOutputBytes))
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
    $process = Start-Process -FilePath 'powershell.exe' -ArgumentList $argumentList -Verb RunAs -Wait -PassThru
    exit $process.ExitCode
} catch {
    ($_ | Out-String) | Add-Content -Path $outputPath -Encoding Ascii
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
    CreatedHttpsListener = $false
    CreatedCertificateThumbprint = ''
}
Save-State $state

if (-not $state.HadListeners) {
    Enable-PSRemoting -Force -SkipNetworkProfileCheck
} elseif (-not $state.WinRMServiceWasRunning) {
    if ([string]$state.WinRMServiceStartMode -eq 'Disabled') {
        Set-Service -Name 'WinRM' -StartupType Manual
    }
    Start-Service -Name 'WinRM'
}

Assert-WsmanPathExists 'WSMan:\localhost\Service\AllowUnencrypted'
Assert-WsmanPathExists 'WSMan:\localhost\Service\Auth\Basic'
Set-Item -Path 'WSMan:\localhost\Service\AllowUnencrypted' -Value $false
Set-Item -Path 'WSMan:\localhost\Service\Auth\Basic' -Value $true

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

$securePassword = ConvertTo-SecureString -String $passwordPlain -AsPlainText -Force
$localUser = Get-LocalUser -Name $userName -ErrorAction SilentlyContinue
if ($null -eq $localUser) {
    New-LocalUser -Name $userName -Password $securePassword -PasswordNeverExpires -Description 'Dev Alchemy Ansible acct' | Out-Null
} else {
    Set-LocalUser -Name $userName -Password $securePassword -Description 'Dev Alchemy Ansible acct'
    Enable-LocalUser -Name $userName
}

$isAdministrator = @(
    Get-LocalGroupMember -Group 'Administrators' -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match ("(^|\\\\)" + [regex]::Escape($userName) + '$') }
).Count -gt 0
if (-not $isAdministrator) {
    Add-LocalGroupMember -Group 'Administrators' -Member $userName
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
    Disable-PSRemoting -Force -ErrorAction SilentlyContinue
    Disable-WinRMFirewallRules
    Restore-WinRMServiceState $false 'Disabled'
} elseif (-not [bool]$state.HadListeners) {
    Disable-PSRemoting -Force -ErrorAction SilentlyContinue
    Restore-WinRMServiceState ([bool]$state.WinRMServiceWasRunning) ([string]$state.WinRMServiceStartMode)
} else {
    Restore-WinRMServiceState ([bool]$state.WinRMServiceWasRunning) ([string]$state.WinRMServiceStartMode)
}
`
