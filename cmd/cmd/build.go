package cmd

import (
	"fmt"
	"math/rand"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"

	"github.com/spf13/cobra"
)

var (
	arch string
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build <osname>",
	Short: "Build the VM for the given operating system",
	Long: `Builds the VM for a specified operating system.
You can specify the OS name, type, and architecture.
Use "all" to build all available VM configurations.

Example:
  alchemy build ubuntu --type server --arch amd64
  alchemy build windows11 --arch arm64
  alchemy build all
`,
	Args: cobra.ExactArgs(1), // Enforce exactly one positional argument
	Run: func(cmd *cobra.Command, args []string) {
		osName := args[0]

		if osName == "all" {
			available_virtual_machines := alchemy_build.AvailableVirtualMachineConfigs()
			for _, vm := range available_virtual_machines {
				fmt.Printf("➡️  Building VM for OS: %s, Type: %s, Architecture: %s\n", vm.OS, vm.UbuntuType, vm.Arch)
				if vm.OS == "ubuntu" {
					alchemy_build.RunQemuUbuntuBuildOnMacOS(vm)
				}
				if vm.OS == "windows11" {
					alchemy_build.RunQemuWindowsBuildOnMacOS(vm)
				}
			}
			return
		}

		fmt.Printf("🔧 Building VM for OS: %s, Type: %s, Architecture: %s\n", osName, osType, arch)
		available_virtual_machines := alchemy_build.AvailableVirtualMachineConfigs()
		valid := false
		for _, vm := range available_virtual_machines {
			if vm.OS == osName && vm.UbuntuType == osType && vm.Arch == arch {
				valid = true
				break
			}
		}
		if !valid {
			fmt.Printf("❌ Invalid combination: OS=%s, Type=%s, Arch=%s\n", osName, osType, arch)
			return
		}

		port := 5900 + (rand.Intn(100) + 1)

		VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
			OS:         osName,
			Arch:       arch,
			UbuntuType: osType,
			VncPort:    port,
		}
		if osName == "ubuntu" {
			alchemy_build.RunQemuUbuntuBuildOnMacOS(VirtualMachineConfig)
		}
		if osName == "windows11" {
			alchemy_build.RunQemuWindowsBuildOnMacOS(VirtualMachineConfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&arch, "arch", "a", "amd64", "Target architecture (e.g., amd64, arm64)")
	buildCmd.Flags().StringVarP(&osType, "type", "t", "server", "Type of OS (e.g., server, desktop)")
}
