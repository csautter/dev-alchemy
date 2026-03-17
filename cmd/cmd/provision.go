package cmd

import (
	"fmt"
	"sort"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	alchemy_deploy "github.com/csautter/dev-alchemy/pkg/deploy"
	"github.com/spf13/cobra"
)

var (
	check bool
)

func isProvisionSupported(vm alchemy_build.VirtualMachineConfig) bool {
	return vm.HostOs == alchemy_build.HostOsWindows &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineHyperv &&
		vm.Arch == "amd64" &&
		(vm.OS == "windows11" || vm.OS == "ubuntu")
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

func provisionVirtualizationEngines(vms []alchemy_build.VirtualMachineConfig) []alchemy_build.VirtualizationEngine {
	engineSet := make(map[alchemy_build.VirtualizationEngine]struct{})
	for _, vm := range vms {
		engineSet[vm.VirtualizationEngine] = struct{}{}
	}

	engines := make([]alchemy_build.VirtualizationEngine, 0, len(engineSet))
	for engine := range engineSet {
		engines = append(engines, engine)
	}
	sort.Slice(engines, func(i, j int) bool {
		return engines[i] < engines[j]
	})
	return engines
}

func printAvailableProvisionCombinations() {
	vms := availableProvisionVirtualMachines()
	fmt.Printf("Available provision combinations for host OS: %s\n", alchemy_build.GetCurrentHostOs())
	if len(vms) == 0 {
		fmt.Printf("No provision combinations are available for the current host OS.\n")
		return
	}

	grouped := make(map[alchemy_build.VirtualizationEngine][]alchemy_build.VirtualMachineConfig)
	for _, vm := range vms {
		grouped[vm.VirtualizationEngine] = append(grouped[vm.VirtualizationEngine], vm)
	}

	for _, engine := range provisionVirtualizationEngines(vms) {
		fmt.Printf("\nVirtualization engine: %s\n", engine)
		fmt.Printf("%-12s %-10s %-8s\n", "OS", "Type", "Arch")
		for _, vm := range grouped[engine] {
			vmType := vm.UbuntuType
			if vmType == "" {
				vmType = "-"
			}
			fmt.Printf("%-12s %-10s %-8s\n", vm.OS, vmType, vm.Arch)
		}
	}
}

var provisionCmd = &cobra.Command{
	Use:   "provision <osname>",
	Short: "Provision and test Ansible configuration against a VM",
	Long: `Runs Ansible provisioning against VM targets.

Examples:
  alchemy provision windows11 --arch amd64 --check
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
	Run: func(cmd *cobra.Command, args []string) {
		printAvailableProvisionCombinations()
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
