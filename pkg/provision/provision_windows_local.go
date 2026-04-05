package provision

import "fmt"

func runLocalWindowsProvision(projectDir string, options ProvisionOptions) error {
	switch normalizeLocalWindowsProvisionProtocol(options.LocalWindowsProtocol) {
	case LocalWindowsProvisionProtocolSSH:
		return runLocalWindowsSSHProvision(projectDir, options)
	case LocalWindowsProvisionProtocolWinRM:
		return runLocalWindowsWinRMProvision(projectDir, options)
	default:
		return fmt.Errorf("unsupported local windows provision protocol: %s", options.LocalWindowsProtocol)
	}
}
