package cmd

import (
	"fmt"
	"os"
	"strings"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	alchemy_deploy "github.com/csautter/dev-alchemy/pkg/deploy"
	"github.com/spf13/cobra"
)

var (
	osType string
)

var inspectCreateTargetExists = alchemy_deploy.CreateTargetExists
var inspectCreateArtifactExists = alchemy_build.BuildArtifactsExistQuiet

func isCreateSupported(vm alchemy_build.VirtualMachineConfig) bool {
	switch vm.VirtualizationEngine {
	case alchemy_build.VirtualizationEngineUtm, alchemy_build.VirtualizationEngineHyperv, alchemy_build.VirtualizationEngineTart:
		return true
	default:
		return false
	}
}

func availableCreateVirtualMachines() []alchemy_build.VirtualMachineConfig {
	var supported []alchemy_build.VirtualMachineConfig
	for _, vm := range alchemy_build.AvailableVirtualMachineConfigsForCurrentHostOS() {
		if isCreateSupported(vm) {
			supported = append(supported, vm)
		}
	}
	return supported
}

func printAvailableCreateCombinations() error {
	vms := availableCreateVirtualMachines()
	return printVirtualMachineCombinationTable(
		os.Stdout,
		fmt.Sprintf("Available create combinations for host OS: %s", alchemy_build.GetCurrentHostOs()),
		"No create combinations are available for the current host OS.",
		vms,
		[]string{"OS", "Type", "Arch", "Artifact", "Create"},
		func(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
			targetExists, err := inspectCreateTargetExists(vm)
			if err != nil {
				return nil, fmt.Errorf("failed to inspect create target for OS=%s, type=%s, arch=%s: %w", vm.OS, vm.UbuntuType, vm.Arch, err)
			}

			if vm.VirtualizationEngine == alchemy_build.VirtualizationEngineTart {
				createState := "ready to create"
				if targetExists {
					createState = "already created"
				}
				return []string{vm.OS, displayVirtualMachineType(vm), vm.Arch, "public image", createState}, nil
			}

			artifactsExist, err := inspectCreateArtifactExists(vm)
			if err != nil {
				return nil, fmt.Errorf("failed to check build artifacts for OS=%s, type=%s, arch=%s: %w", vm.OS, vm.UbuntuType, vm.Arch, err)
			}

			artifactState := "missing"
			createState := "build required"
			if targetExists {
				createState = "already created"
			}
			if artifactsExist {
				artifactState = "exists"
				if !targetExists {
					createState = "ready to create"
				}
			}

			return []string{vm.OS, displayVirtualMachineType(vm), vm.Arch, artifactState, createState}, nil
		},
	)
}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create <osname>",
	Short: "Creates a new VM on your system with the defined OS",
	Long: `Creates a new VM on your system with the defined OS.
Use "all" to create all available VM configurations.

Example:
  alchemy create ubuntu --type server --arch amd64
  alchemy create macos --arch arm64
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
			for _, vm := range availableCreateVirtualMachines() {
				fmt.Printf("➡️ Creating VM for OS: %s, Type: %s, Architecture: %s\n", vm.OS, vm.UbuntuType, vm.Arch)
				if err := runDeploy(vm); err != nil {
					return fmt.Errorf("failed creating VM for OS=%s, type=%s, arch=%s: %w", vm.OS, vm.UbuntuType, vm.Arch, err)
				}
			}
			return nil
		}

		available_virtual_machines := availableCreateVirtualMachines()
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

var createListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available create combinations and create readiness",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return printAvailableCreateCombinations()
	},
}

func runDeploy(vm alchemy_build.VirtualMachineConfig) error {
	targetExists, err := inspectCreateTargetExists(vm)
	if err != nil {
		return fmt.Errorf("failed to inspect create target for OS=%s, type=%s, arch=%s: %w", vm.OS, vm.UbuntuType, vm.Arch, err)
	}
	if targetExists {
		return fmt.Errorf(
			"VM for %s already exists. Use `alchemy start %s` to reuse it or `alchemy destroy %s` first",
			createCommandArguments(vm),
			createCommandArguments(vm),
			createCommandArguments(vm),
		)
	}

	switch vm.VirtualizationEngine {
	case alchemy_build.VirtualizationEngineUtm:
		return alchemy_deploy.RunUtmDeployOnMacOS(vm)
	case alchemy_build.VirtualizationEngineTart:
		return alchemy_deploy.RunTartDeployOnMacOS(vm)
	case alchemy_build.VirtualizationEngineHyperv:
		return alchemy_deploy.RunHypervVagrantDeployOnWindows(vm)
	default:
		return fmt.Errorf("❌ deploy is not implemented for virtualization engine: %s", vm.VirtualizationEngine)
	}
}

func createCommandArguments(vm alchemy_build.VirtualMachineConfig) string {
	args := []string{vm.OS}
	if vm.UbuntuType != "" {
		args = append(args, "--type", vm.UbuntuType)
	}
	if vm.Arch != "" {
		args = append(args, "--arch", vm.Arch)
	}
	return strings.Join(args, " ")
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createListCmd)

	createCmd.Flags().StringVarP(&arch, "arch", "a", "amd64", "Target architecture (e.g., amd64, arm64)")
	createCmd.Flags().StringVarP(&osType, "type", "t", "server", "Type of OS (e.g., server, desktop)")
}
