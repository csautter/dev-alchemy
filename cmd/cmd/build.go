package cmd

import (
	"fmt"

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

Example:
  alchemy build ubuntu --type server --arch amd64
  alchemy build windows11 --arch arm64
`,
	Args: cobra.ExactArgs(1), // Enforce exactly one positional argument
	Run: func(cmd *cobra.Command, args []string) {
		osName := args[0]
		fmt.Printf("🔧 Building VM for OS: %s, Type: %s, Architecture: %s\n", osName, osType, arch)

		VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
			OS:         osName,
			Arch:       arch,
			UbuntuType: osType,
			VncPort:    5901,
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
