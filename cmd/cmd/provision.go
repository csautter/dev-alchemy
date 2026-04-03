package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	alchemy_provision "github.com/csautter/dev-alchemy/pkg/provision"
	"github.com/spf13/cobra"
)

var (
	check               bool
	assumeYes           bool
	forceWinRMUninstall bool
)

const localProvisionVirtualizationEngine = alchemy_build.VirtualizationEngine("local")

var (
	currentHostLocalProvisionVirtualMachineFunc = currentHostLocalProvisionVirtualMachine
	runProvisionFunc                            = alchemy_provision.RunProvision
	configureLocalWindowsProvisionFunc          = alchemy_provision.SetLocalWindowsForceWinRMUninstall
	promptForConfirmationFunc                   = promptForConfirmation
)

func isProvisionSupported(vm alchemy_build.VirtualMachineConfig) bool {
	if vm.HostOs == alchemy_build.HostOsWindows &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineHyperv &&
		vm.Arch == "amd64" &&
		(vm.OS == "windows11" || vm.OS == "ubuntu") {
		return true
	}

	return vm.HostOs == alchemy_build.HostOsDarwin &&
		((vm.VirtualizationEngine == alchemy_build.VirtualizationEngineUtm &&
			(vm.OS == "windows11" || vm.OS == "ubuntu") &&
			(vm.Arch == "amd64" || vm.Arch == "arm64")) ||
			(vm.VirtualizationEngine == alchemy_build.VirtualizationEngineTart &&
				vm.OS == "macos" &&
				vm.Arch == "arm64"))
}

func currentHostLocalProvisionVirtualMachine() (alchemy_build.VirtualMachineConfig, bool) {
	hostOs := alchemy_build.GetCurrentHostOs()
	switch hostOs {
	case alchemy_build.HostOsWindows, alchemy_build.HostOsDarwin, alchemy_build.HostOsLinux:
		return alchemy_build.VirtualMachineConfig{
			OS:                   "local",
			Arch:                 "-",
			HostOs:               hostOs,
			VirtualizationEngine: localProvisionVirtualizationEngine,
		}, true
	default:
		return alchemy_build.VirtualMachineConfig{}, false
	}
}

func isLocalProvisionUnstable(hostOs alchemy_build.HostOsType) bool {
	return hostOs != alchemy_build.HostOsWindows
}

func provisionStatus(vm alchemy_build.VirtualMachineConfig) string {
	if vm.OS == "local" {
		if isLocalProvisionUnstable(vm.HostOs) {
			return "unstable"
		}
		return "stable"
	}
	if alchemy_build.IsVirtualizationEngineUnstable(vm.VirtualizationEngine) {
		return "unstable"
	}
	return "stable"
}

func availableProvisionVirtualMachines() []alchemy_build.VirtualMachineConfig {
	var supported []alchemy_build.VirtualMachineConfig
	for _, vm := range alchemy_build.AvailableVirtualMachineConfigsForCurrentHostOS() {
		if isProvisionSupported(vm) {
			supported = append(supported, vm)
		}
	}
	if localVM, ok := currentHostLocalProvisionVirtualMachine(); ok {
		supported = append(supported, localVM)
	}
	return supported
}

func printAvailableProvisionCombinations() error {
	vms := availableProvisionVirtualMachines()
	return printVirtualMachineCombinationTable(
		os.Stdout,
		fmt.Sprintf("Available provision combinations for host OS: %s", alchemy_build.GetCurrentHostOs()),
		"No provision combinations are available for the current host OS.",
		vms,
		[]string{"OS", "Type", "Arch", "Status"},
		func(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
			return []string{vm.OS, displayVirtualMachineType(vm), vm.Arch, provisionStatus(vm)}, nil
		},
	)
}

var provisionCmd = &cobra.Command{
	Use:   "provision <osname|local>",
	Short: "Provision and test Ansible configuration against a VM or the local host",
	Long: `Runs Ansible provisioning against VM targets or the current host.

Examples:
  alchemy provision local --check
  alchemy provision macos --arch arm64 --check
  alchemy provision windows11 --arch amd64 --check
  alchemy provision windows11 --arch arm64 --check
  alchemy provision ubuntu --type server --arch amd64 --check
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		osName := args[0]

		if osName == "all" {
			return fmt.Errorf("❌ \"all\" is not supported for provision; provide one target, for example: alchemy provision windows11 --arch amd64 --check")
		}

		if osName == "local" {
			if cmd.Flags().Changed("arch") || cmd.Flags().Changed("type") {
				return fmt.Errorf("❌ local provisioning does not accept --arch or --type; use `alchemy provision local [--check]`")
			}

			selectedVM, ok := currentHostLocalProvisionVirtualMachineFunc()
			if !ok {
				return fmt.Errorf("❌ local provisioning is not available for host OS: %s", alchemy_build.GetCurrentHostOs())
			}
			if forceWinRMUninstall && selectedVM.HostOs != alchemy_build.HostOsWindows {
				return fmt.Errorf("❌ --force-winrm-uninstall is only supported for local Windows provisioning")
			}

			if isLocalProvisionUnstable(selectedVM.HostOs) {
				fmt.Printf("⚠️ Local provisioning on host OS %s is currently marked unstable and has not been validated end-to-end yet.\n", selectedVM.HostOs)
			}
			if err := confirmProvisionIntent(cmd, selectedVM); err != nil {
				return err
			}

			fmt.Printf("🔧 Provisioning local host for OS: %s (check=%t)\n", selectedVM.HostOs, check)
			if err := runProvision(selectedVM, check); err != nil {
				return fmt.Errorf("failed provisioning local host for host_os=%s: %w", selectedVM.HostOs, err)
			}

			return nil
		}

		if osName != "ubuntu" {
			osType = ""
		}

		availableVirtualMachines := availableProvisionVirtualMachines()
		var selectedVM alchemy_build.VirtualMachineConfig
		valid := false
		for _, vm := range availableVirtualMachines {
			if vm.OS == osName && vm.UbuntuType == osType && vm.Arch == arch {
				selectedVM = vm
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("❌ invalid combination: OS=%s, Type=%s, Arch=%s", osName, osType, arch)
		}

		fmt.Printf("🔧 Provisioning VM for OS: %s, Type: %s, Architecture: %s (check=%t)\n", osName, osType, arch, check)
		if err := runProvision(selectedVM, check); err != nil {
			return fmt.Errorf("failed provisioning for OS=%s, type=%s, arch=%s: %w", osName, osType, arch, err)
		}

		return nil
	},
}

var provisionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available provision combinations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return printAvailableProvisionCombinations()
	},
}

func runProvision(vm alchemy_build.VirtualMachineConfig, check bool) error {
	restoreLocalWindowsProvision := configureLocalWindowsProvisionFunc(forceWinRMUninstall)
	defer restoreLocalWindowsProvision()

	return runProvisionFunc(vm, check)
}

func init() {
	rootCmd.AddCommand(provisionCmd)
	provisionCmd.AddCommand(provisionListCmd)

	provisionCmd.Flags().StringVarP(&arch, "arch", "a", "amd64", "Target architecture (e.g., amd64, arm64)")
	provisionCmd.Flags().StringVarP(&osType, "type", "t", "server", "Type of OS (e.g., server, desktop)")
	provisionCmd.Flags().BoolVar(&check, "check", false, "Run ansible with --check (dry-run)")
	provisionCmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "Skip confirmation prompts for operations that change local system state")
	provisionCmd.Flags().BoolVar(&forceWinRMUninstall, "force-winrm-uninstall", false, "For local Windows provisioning, force cleanup to disable WinRM and remove transient setup after the run")
}

func confirmProvisionIntent(cmd *cobra.Command, vm alchemy_build.VirtualMachineConfig) error {
	if assumeYes || !requiresProvisionConfirmation(vm) {
		return nil
	}

	message := "Local Windows provisioning will temporarily enable or reconfigure WinRM, create or update a temporary local administrator account for Ansible, and create a self-signed HTTPS listener. Windows will also show a UAC elevation prompt for the setup and cleanup steps."
	if forceWinRMUninstall {
		message += " Because --force-winrm-uninstall is set, cleanup will aggressively disable WinRM and remove transient configuration even if the run fails."
	}

	if !isInteractiveInput(cmd.InOrStdin()) {
		return fmt.Errorf("%s Re-run with --yes to continue non-interactively", message)
	}

	confirmed, err := promptForConfirmationFunc(
		cmd.InOrStdin(),
		cmd.OutOrStdout(),
		message+" Continue? [y/N]: ",
	)
	if err != nil {
		return fmt.Errorf("failed to read confirmation for local Windows provisioning: %w", err)
	}
	if !confirmed {
		return fmt.Errorf("local Windows provisioning cancelled")
	}

	return nil
}

func requiresProvisionConfirmation(vm alchemy_build.VirtualMachineConfig) bool {
	return vm.OS == "local" && vm.HostOs == alchemy_build.HostOsWindows
}

func isInteractiveInput(input io.Reader) bool {
	file, ok := input.(*os.File)
	if !ok {
		return true
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func promptForConfirmation(input io.Reader, output io.Writer, prompt string) (bool, error) {
	if _, err := fmt.Fprint(output, prompt); err != nil {
		return false, err
	}

	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
