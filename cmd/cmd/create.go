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
	Use:   "create",
	Short: "Creates a new VM on your system with the defined OS",
	Long: `Creates a new VM on your system with the defined OS.

Example:
  alchemy create ubuntu --type server --arch amd64
  alchemy create windows11 --arch arm64
`,
	Run: func(cmd *cobra.Command, args []string) {
		osName := args[0]
		fmt.Printf("🔧 Creating VM for OS: %s, Architecture: %s, Type: %s\n", osName, arch, osType)

		VirtualMachineConfig := alchemy_build.VirtualMachineConfig{
			OS:         osName,
			Arch:       arch,
			UbuntuType: osType,
			VncPort:    5901,
		}
		if osName == "ubuntu" {
			alchemy_deploy.RunUtmDeployOnMacOS(VirtualMachineConfig)
		}
		if osName == "windows11" {
			alchemy_deploy.RunUtmDeployOnMacOS(VirtualMachineConfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&arch, "arch", "a", "amd64", "Target architecture (e.g., amd64, arm64)")
	createCmd.Flags().StringVarP(&osType, "type", "t", "server", "Type of OS (e.g., server, desktop)")
}
