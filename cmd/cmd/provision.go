package cmd

import (
	"fmt"
	"os"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	alchemy_deploy "github.com/csautter/dev-alchemy/pkg/deploy"
	"github.com/spf13/cobra"
)

var (
	check bool
)

func isProvisionSupported(vm alchemy_build.VirtualMachineConfig) bool {
	if vm.HostOs == alchemy_build.HostOsWindows &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineHyperv &&
		vm.Arch == "amd64" &&
		(vm.OS == "windows11" || vm.OS == "ubuntu") {
		return true
	}

	return vm.HostOs == alchemy_build.HostOsDarwin &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineUtm &&
		(vm.OS == "windows11" || vm.OS == "ubuntu") &&
		(vm.Arch == "amd64" || vm.Arch == "arm64")
}

func availableProvisionVirtualMachines() []alchemy_build.VirtualMachineConfig {
	var supported []alchemy_build.VirtualMachineConfig
	for _, vm := range alchemy_build.AvailableVirtualMachineConfigsForCurrentHostOS() {
		if isProvisionSupported(vm) {
			supported = append(supported, vm)
		}
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
		[]string{"OS", "Type", "Arch"},
		func(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
			return []string{vm.OS, displayVirtualMachineType(vm), vm.Arch}, nil
		},
	)
}

var provisionCmd = &cobra.Command{
	Use:   "provision <osname>",
	Short: "Provision and test Ansible configuration against a VM",
	Long: `Runs Ansible provisioning against VM targets.

Examples:
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
	return alchemy_deploy.RunProvision(vm, check)
}

func init() {
	rootCmd.AddCommand(provisionCmd)
	provisionCmd.AddCommand(provisionListCmd)

	provisionCmd.Flags().StringVarP(&arch, "arch", "a", "amd64", "Target architecture (e.g., amd64, arm64)")
	provisionCmd.Flags().StringVarP(&osType, "type", "t", "server", "Type of OS (e.g., server, desktop)")
	provisionCmd.Flags().BoolVar(&check, "check", false, "Run ansible with --check (dry-run)")
}
