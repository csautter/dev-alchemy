package cmd

import (
	"fmt"

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
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&arch, "arch", "a", "amd64", "Target architecture (e.g., amd64, arm64)")
	createCmd.Flags().StringVarP(&osType, "type", "t", "server", "Type of OS (e.g., server, desktop)")
}
