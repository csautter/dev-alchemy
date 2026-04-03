package provision

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
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
		[]string{
			"DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH=" + statePath,
			"DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_USER=" + localWindowsProvisionUserName,
			"DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_PASSWORD=" + password,
		},
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
		[]string{"DEV_ALCHEMY_LOCAL_WINDOWS_PROVISION_STATE_PATH=" + session.StatePath},
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
	return runProvisionCommandWithCombinedOutputWithEnv(
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
			script,
		},
		extraEnv,
	)
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

function Get-WsmanBoolValue([string]$path) {
    return [System.Convert]::ToBoolean((Get-Item -Path $path).Value)
}

$listenerKeys = @()
try {
    $listenerKeys = @(Get-ChildItem -Path WSMan:\localhost\Listener -ErrorAction Stop | ForEach-Object { $_.Keys })
} catch {
    $listenerKeys = @()
}

$state = @{
    UserName = $userName
    HadListeners = ($listenerKeys.Count -gt 0)
    WinRMServiceWasRunning = ((Get-Service -Name 'WinRM').Status -eq 'Running')
    BasicAuthEnabled = (Get-WsmanBoolValue 'WSMan:\localhost\Service\Auth\Basic')
    AllowUnencryptedEnabled = (Get-WsmanBoolValue 'WSMan:\localhost\Service\AllowUnencrypted')
    CreatedHttpsListener = $false
    CreatedCertificateThumbprint = ''
}
Save-State $state

if (-not $state.HadListeners) {
    Enable-PSRemoting -Force -SkipNetworkProfileCheck
} elseif (-not $state.WinRMServiceWasRunning) {
    Start-Service -Name 'WinRM'
}

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
    New-LocalUser -Name $userName -Password $securePassword -PasswordNeverExpires -Description 'Temporary Dev Alchemy Ansible provisioning account' | Out-Null
} else {
    Set-LocalUser -Name $userName -Password $securePassword -Description 'Temporary Dev Alchemy Ansible provisioning account'
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
if ([string]::IsNullOrWhiteSpace($statePath) -or -not (Test-Path -Path $statePath)) {
    return
}

$state = Get-Content -Path $statePath -Raw | ConvertFrom-Json

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

Set-Item -Path 'WSMan:\localhost\Service\Auth\Basic' -Value ([bool]$state.BasicAuthEnabled)
Set-Item -Path 'WSMan:\localhost\Service\AllowUnencrypted' -Value ([bool]$state.AllowUnencryptedEnabled)

$certificateThumbprint = [string]$state.CreatedCertificateThumbprint
if (-not [string]::IsNullOrWhiteSpace($certificateThumbprint)) {
    $certificatePath = 'Cert:\LocalMachine\My\' + $certificateThumbprint
    if (Test-Path -Path $certificatePath) {
        Remove-Item -Path $certificatePath -Force
    }
}

if (-not [bool]$state.HadListeners) {
    Disable-PSRemoting -Force
} elseif (-not [bool]$state.WinRMServiceWasRunning) {
    Stop-Service -Name 'WinRM' -Force
}
`
