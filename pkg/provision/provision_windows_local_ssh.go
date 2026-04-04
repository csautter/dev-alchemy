package provision

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	localWindowsSSHInventoryPath   = "./inventory/localhost_windows_ssh.yml"
	localWindowsSSHInventoryTarget = "windows_host"
	localWindowsSSHLoopbackIP      = "127.0.0.1"
	localWindowsSSHDefaultShell    = `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`

	localWindowsProvisionSSHPublicKeyEnvVar = "DEV_ALCHEMY_LOCAL_WINDOWS_ANSIBLE_SSH_PUBLIC_KEY"
	localWindowsForceSSHUninstallEnvVar     = "DEV_ALCHEMY_LOCAL_WINDOWS_FORCE_SSH_UNINSTALL"
	localWindowsSSHBootstrapLogPrefix       = "local:windows:ssh:bootstrap"
	localWindowsSSHCleanupLogPrefix         = "local:windows:ssh:cleanup"
	localWindowsSSHBootstrapScriptPath      = "scripts/windows/local-windows-provision-ssh-bootstrap.ps1"
	localWindowsSSHCleanupScriptPath        = "scripts/windows/local-windows-provision-ssh-cleanup.ps1"
	localWindowsSSHCommonArgs               = defaultAnsibleSSHCommonArgs + " -o UserKnownHostsFile=/dev/null -o IdentitiesOnly=yes"
)

var setupLocalWindowsSSHProvisionSessionFunc = setupLocalWindowsSSHProvisionSession
var cleanupLocalWindowsSSHProvisionSessionFunc = cleanupLocalWindowsSSHProvisionSession
var localWindowsSSHProvisionBootstrapPowerShell = mustLoadLocalWindowsPowerShellAsset(localWindowsSSHBootstrapScriptPath)
var localWindowsSSHProvisionCleanupPowerShell = mustLoadLocalWindowsPowerShellAsset(localWindowsSSHCleanupScriptPath)

type localWindowsSSHProvisionSession struct {
	ConnectionConfig sshAnsibleConnectionConfig
	StatePath        string
	PrivateKeyPath   string
}

func runLocalWindowsSSHProvision(projectDir string, options ProvisionOptions) error {
	session, err := setupLocalWindowsSSHProvisionSessionFunc(projectDir, options)
	if err != nil {
		return err
	}

	inventoryPath, inventoryTarget := resolveStaticInventoryPathAndTarget(
		localWindowsSSHInventoryPath,
		localWindowsSSHInventoryTarget,
		options,
	)

	args, argsCleanup, err := buildSSHStaticInventoryProvisionArgs(
		projectDir,
		inventoryPath,
		inventoryTarget,
		session.ConnectionConfig,
		options,
	)
	if err != nil {
		cleanupErr := cleanupLocalWindowsSSHProvisionSessionFunc(projectDir, session, options)
		if cleanupErr != nil {
			return fmt.Errorf("failed to build ansible arguments for secure local windows SSH provision: %w (also failed to restore secure SSH state: %v)", err, cleanupErr)
		}
		return fmt.Errorf("failed to build ansible arguments for secure local windows SSH provision: %w", err)
	}

	runErr := runAnsibleProvisionCommandFunc(projectDir, args, 90*time.Minute, "local:windows:ssh:provision")
	argsCleanupErr := argsCleanup()
	cleanupErr := cleanupLocalWindowsSSHProvisionSessionFunc(projectDir, session, options)

	if runErr != nil {
		if argsCleanupErr != nil && cleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for local host windows via ssh: %w (also failed to clean ansible temp files: %v; cleanup failed: %v)", runErr, argsCleanupErr, cleanupErr)
		}
		if argsCleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for local host windows via ssh: %w (also failed to clean ansible temp files: %v)", runErr, argsCleanupErr)
		}
		if cleanupErr != nil {
			return fmt.Errorf("ansible provisioning failed for local host windows via ssh: %w (also failed to restore secure SSH state: %v)", runErr, cleanupErr)
		}
		return fmt.Errorf("ansible provisioning failed for local host windows via ssh: %w", runErr)
	}
	if argsCleanupErr != nil && cleanupErr != nil {
		return fmt.Errorf("failed to clean ansible temp files: %w (also failed to restore secure SSH state: %v)", argsCleanupErr, cleanupErr)
	}
	if argsCleanupErr != nil {
		return fmt.Errorf("failed to clean ansible temp files: %w", argsCleanupErr)
	}
	if cleanupErr != nil {
		return fmt.Errorf("failed to restore secure SSH state after local host windows provision: %w", cleanupErr)
	}

	return nil
}

func buildSSHStaticInventoryProvisionArgs(projectDir string, inventoryPath string, inventoryTarget string, connectionConfig sshAnsibleConnectionConfig, options ProvisionOptions) ([]string, func() error, error) {
	extraVars, err := buildSSHProvisionExtraVars(connectionConfig)
	if err != nil {
		return nil, nil, err
	}

	return buildStaticInventoryProvisionArgsWithExtraVars(projectDir, inventoryPath, inventoryTarget, extraVars, options)
}

func buildLocalWindowsSSHProvisionScriptEnv(statePath string, password string, publicKey string, options ProvisionOptions) []string {
	env := []string{
		localWindowsProvisionStatePathEnvVar + "=" + statePath,
		localWindowsProvisionUserEnvVar + "=" + localWindowsProvisionUserName,
		localWindowsForceSSHUninstallEnvVar + "=" + fmt.Sprintf("%t", options.LocalWindowsForceSSHUninstall),
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
		buildLocalWindowsSSHProvisionScriptEnv(statePath, password, publicAuthorizedKey, options),
		localWindowsBootstrapTimeout,
		localWindowsSSHBootstrapLogPrefix,
	)
	if runErr != nil {
		cleanupErr := cleanupLocalWindowsSSHProvisionSession(projectDir, session, options)
		if cleanupErr != nil {
			return localWindowsSSHProvisionSession{}, fmt.Errorf("failed to securely bootstrap local SSH access: %w; output: %s (also failed to restore secure SSH state: %v)", runErr, strings.TrimSpace(output), cleanupErr)
		}
		return localWindowsSSHProvisionSession{}, fmt.Errorf("failed to securely bootstrap local SSH access: %w; output: %s", runErr, strings.TrimSpace(output))
	}

	if err := waitForSSHPort(localWindowsSSHLoopbackIP); err != nil {
		cleanupErr := cleanupLocalWindowsSSHProvisionSession(projectDir, session, options)
		if cleanupErr != nil {
			return localWindowsSSHProvisionSession{}, fmt.Errorf("local Windows SSH bootstrap completed but sshd did not become reachable: %w (also failed to restore secure SSH state: %v)", err, cleanupErr)
		}
		return localWindowsSSHProvisionSession{}, fmt.Errorf("local Windows SSH bootstrap completed but sshd did not become reachable: %w", err)
	}

	return session, nil
}

func cleanupLocalWindowsSSHProvisionSession(projectDir string, session localWindowsSSHProvisionSession, options ProvisionOptions) error {
	var output string
	var runErr error
	if session.StatePath != "" {
		output, runErr = runLocalWindowsPowerShellScript(
			projectDir,
			localWindowsSSHProvisionCleanupPowerShell,
			buildLocalWindowsSSHProvisionScriptEnv(session.StatePath, "", "", options),
			localWindowsCleanupTimeout,
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

	return privateKeyPEM, marshalRSAPublicAuthorizedKey(&privateKey.PublicKey), nil
}

func marshalRSAPublicAuthorizedKey(publicKey *rsa.PublicKey) string {
	var blob bytes.Buffer
	writeSSHString(&blob, "ssh-rsa")
	writeSSHMPI(&blob, big.NewInt(int64(publicKey.E)).Bytes())
	writeSSHMPI(&blob, publicKey.N.Bytes())

	return "ssh-rsa " + base64.StdEncoding.EncodeToString(blob.Bytes())
}

func writeSSHString(buffer *bytes.Buffer, value string) {
	writeSSHBytes(buffer, []byte(value))
}

func writeSSHMPI(buffer *bytes.Buffer, value []byte) {
	trimmed := bytes.TrimLeft(value, "\x00")
	if len(trimmed) == 0 {
		writeSSHBytes(buffer, []byte{0})
		return
	}
	if trimmed[0]&0x80 != 0 {
		trimmed = append([]byte{0}, trimmed...)
	}

	writeSSHBytes(buffer, trimmed)
}

func writeSSHBytes(buffer *bytes.Buffer, value []byte) {
	var lengthBytes [4]byte
	binary.BigEndian.PutUint32(lengthBytes[:], uint32(len(value)))
	buffer.Write(lengthBytes[:])
	buffer.Write(value)
}
