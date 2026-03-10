package cmd

import (
	"fmt"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	alchemy_deploy "github.com/csautter/dev-alchemy/pkg/deploy"
	"github.com/spf13/cobra"
)

var (
	osType string
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create <osname>",
	Short: "Creates a new VM on your system with the defined OS",
	Long: `Creates a new VM on your system with the defined OS.
Use "all" to create all available VM configurations.

Example:
  alchemy create ubuntu --type server --arch amd64
  alchemy create windows11 --arch arm64
  alchemy create all
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		osName := args[0]

		if osName != "ubuntu" {
			osType = ""
		}

		if osName == "all" {
			fmt.Println("🔧 Creating all available VM configurations")
			for _, vm := range alchemy_build.AvailableVirtualMachineConfigsForCurrentHostOS() {
				fmt.Printf("➡️ Creating VM for OS: %s, Type: %s, Architecture: %s\n", vm.OS, vm.UbuntuType, vm.Arch)
				if err := runDeploy(vm); err != nil {
					return fmt.Errorf("failed creating VM for OS=%s, type=%s, arch=%s: %w", vm.OS, vm.UbuntuType, vm.Arch, err)
				}
			}
			return nil
		}

		available_virtual_machines := alchemy_build.AvailableVirtualMachineConfigsForCurrentHostOS()
		var VirtualMachineConfig alchemy_build.VirtualMachineConfig
		valid := false
		for _, vm := range available_virtual_machines {
			if vm.OS == osName && vm.UbuntuType == osType && vm.Arch == arch {
				valid = true
				VirtualMachineConfig = vm
				break
			}
		}
		if !valid {
			return fmt.Errorf("❌ Invalid combination: OS=%s, Type=%s, Arch=%s", osName, osType, arch)
		}

		fmt.Printf("🔧 Creating VM for OS: %s, Architecture: %s, Type: %s\n", osName, arch, osType)
		if err := runDeploy(VirtualMachineConfig); err != nil {
			return fmt.Errorf("failed creating VM for OS=%s, type=%s, arch=%s: %w", osName, osType, arch, err)
		}
		return nil
	},
}

func runDeploy(vm alchemy_build.VirtualMachineConfig) error {
	switch vm.VirtualizationEngine {
	case alchemy_build.VirtualizationEngineUtm:
		alchemy_deploy.RunUtmDeployOnMacOS(vm)
		return nil
	case alchemy_build.VirtualizationEngineHyperv:
		alchemy_deploy.RunHypervVagrantDeployOnWindows(vm)
		return nil
	default:
		return fmt.Errorf("❌ deploy is not implemented for virtualization engine: %s", vm.VirtualizationEngine)
	}
}

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&arch, "arch", "a", "amd64", "Target architecture (e.g., amd64, arm64)")
	createCmd.Flags().StringVarP(&osType, "type", "t", "server", "Type of OS (e.g., server, desktop)")
}
