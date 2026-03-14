package cmd

import (
	"errors"
	"fmt"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	alchemy_deploy "github.com/csautter/dev-alchemy/pkg/deploy"
	"github.com/spf13/cobra"
)

var (
	check bool
)

var provisionCmd = &cobra.Command{
	Use:   "provision <osname>",
	Short: "Provision and test Ansible configuration against a VM",
	Long: `Runs Ansible provisioning against VM targets.
Use "all" to provision all VM configurations available for the current host OS.

Examples:
  alchemy provision windows11 --arch amd64 --check
  alchemy provision all --check
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		osName := args[0]

		if osName != "ubuntu" {
			osType = ""
		}

		if osName == "all" {
			fmt.Printf("🔧 Provisioning all available VM configurations (check=%t)\n", check)
			var errs []error
			for _, vm := range alchemy_build.AvailableVirtualMachineConfigsForCurrentHostOS() {
				fmt.Printf("➡️ Provisioning VM for OS: %s, Type: %s, Architecture: %s\n", vm.OS, vm.UbuntuType, vm.Arch)
				if err := runProvision(vm, check); err != nil {
					errs = append(errs, fmt.Errorf("%s/%s/%s: %w", vm.OS, vm.UbuntuType, vm.Arch, err))
					fmt.Printf("⚠️ Provisioning skipped/failed for OS: %s, Type: %s, Architecture: %s — %v\n", vm.OS, vm.UbuntuType, vm.Arch, err)
				}
			}

			if len(errs) > 0 {
				return errors.Join(errs...)
			}
			return nil
		}

		availableVirtualMachines := alchemy_build.AvailableVirtualMachineConfigsForCurrentHostOS()
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

func runProvision(vm alchemy_build.VirtualMachineConfig, check bool) error {
	return alchemy_deploy.RunProvision(vm, check)
}

func init() {
	rootCmd.AddCommand(provisionCmd)

	provisionCmd.Flags().StringVarP(&arch, "arch", "a", "amd64", "Target architecture (e.g., amd64, arm64)")
	provisionCmd.Flags().StringVarP(&osType, "type", "t", "server", "Type of OS (e.g., server, desktop)")
	provisionCmd.Flags().BoolVar(&check, "check", false, "Run ansible with --check (dry-run)")
}
