package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	alchemy_oci "github.com/csautter/dev-alchemy/pkg/oci"
	"github.com/spf13/cobra"
)

type ociTransferRunner func(context.Context, alchemy_build.VirtualMachineConfig, string, alchemy_oci.RegistryOptions) (alchemy_oci.TransferResult, error)

var (
	ociOS                       string
	ociType                     string
	ociArch                     string
	ociEngine                   string
	ociHostOS                   string
	ociPlainHTTP                bool
	ociInsecureSkipTLSVerify    bool
	ociCAFile                   string
	ociUsername                 string
	ociPassword                 string
	ociPasswordStdin            bool
	ociAccessToken              string
	ociRefreshToken             string
	ociDisableDockerCredentials bool

	runOCIPush ociTransferRunner = func(ctx context.Context, vm alchemy_build.VirtualMachineConfig, reference string, opts alchemy_oci.RegistryOptions) (alchemy_oci.TransferResult, error) {
		return alchemy_oci.Push(ctx, vm, reference, alchemy_oci.PushOptions{RegistryOptions: opts})
	}
	runOCIPull ociTransferRunner = func(ctx context.Context, vm alchemy_build.VirtualMachineConfig, reference string, opts alchemy_oci.RegistryOptions) (alchemy_oci.TransferResult, error) {
		return alchemy_oci.Pull(ctx, vm, reference, alchemy_oci.PullOptions{RegistryOptions: opts})
	}
	inspectOCIArtifactState = localOCIArtifactState
)

func isOCISupported(vm alchemy_build.VirtualMachineConfig) bool {
	return len(vm.ExpectedBuildArtifacts) > 0
}

func availableOCIVirtualMachinesForHostOS(hostOs alchemy_build.HostOsType) []alchemy_build.VirtualMachineConfig {
	return availableVirtualMachinesForHostOS(hostOs, isOCISupported)
}

func printAvailableOCICombinations(action string, rowBuilder vmTableRowBuilder) error {
	hostOs, err := parseHostOS(ociHostOS)
	if err != nil {
		return err
	}

	return printVirtualMachineCombinationTable(
		os.Stdout,
		fmt.Sprintf("Available %s combinations for host OS: %s", action, hostOs),
		fmt.Sprintf("No %s combinations are available for host OS %s.", action, hostOs),
		availableOCIVirtualMachinesForHostOS(hostOs),
		[]string{"OS", "Type", "Arch", "Artifact", ociActionColumnTitle(action)},
		rowBuilder,
	)
}

func ociActionColumnTitle(action string) string {
	if action == "" {
		return ""
	}
	return strings.ToUpper(action[:1]) + action[1:]
}

func localOCIArtifactState(vm alchemy_build.VirtualMachineConfig) (string, error) {
	artifacts, err := alchemy_oci.ArtifactFiles(vm)
	if err != nil {
		return "", err
	}
	if len(artifacts) == 0 {
		return "missing", nil
	}

	existingArtifacts := 0
	for _, artifact := range artifacts {
		if _, err := os.Stat(artifact.Path); err == nil {
			existingArtifacts++
			continue
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect artifact %s: %w", artifact.Path, err)
		}
	}

	switch existingArtifacts {
	case len(artifacts):
		return "exists", nil
	case 0:
		return "missing", nil
	default:
		return "partial", nil
	}
}

func pushListRow(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
	artifactState, err := inspectOCIArtifactState(vm)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect push artifacts for OS=%s, type=%s, arch=%s: %w", vm.OS, vm.UbuntuType, vm.Arch, err)
	}

	pushState := "build required"
	switch artifactState {
	case "exists":
		pushState = "ready to push"
	case "partial":
		pushState = "incomplete"
	}

	return []string{vm.OS, displayVirtualMachineType(vm), vm.Arch, artifactState, pushState}, nil
}

func pullListRow(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
	artifactState, err := inspectOCIArtifactState(vm)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect pull destination for OS=%s, type=%s, arch=%s: %w", vm.OS, vm.UbuntuType, vm.Arch, err)
	}

	pullState := "ready to pull"
	switch artifactState {
	case "exists":
		pullState = "will replace"
	case "partial":
		pullState = "will replace partial"
	}

	return []string{vm.OS, displayVirtualMachineType(vm), vm.Arch, artifactState, pullState}, nil
}

func resolveOCIVirtualMachine(hostOsValue string, osName string, osTypeValue string, archValue string, engineValue string) (alchemy_build.VirtualMachineConfig, error) {
	hostOs, err := parseHostOS(hostOsValue)
	if err != nil {
		return alchemy_build.VirtualMachineConfig{}, err
	}
	if osName == "" {
		return alchemy_build.VirtualMachineConfig{}, fmt.Errorf("missing required --os value")
	}
	if osName != "ubuntu" {
		osTypeValue = ""
	}

	var matches []alchemy_build.VirtualMachineConfig
	for _, vm := range availableOCIVirtualMachinesForHostOS(hostOs) {
		if vm.OS == osName && vm.UbuntuType == osTypeValue && vm.Arch == archValue {
			matches = append(matches, vm)
		}
	}
	if len(matches) == 0 {
		return alchemy_build.VirtualMachineConfig{}, fmt.Errorf("invalid OCI artifact target: OS=%s, Type=%s, Arch=%s, HostOS=%s", osName, osTypeValue, archValue, hostOs)
	}

	if engineValue == "" {
		if len(matches) > 1 {
			return alchemy_build.VirtualMachineConfig{}, fmt.Errorf(
				"multiple OCI artifact targets match OS=%s, Type=%s, Arch=%s, HostOS=%s; specify --engine (%s)",
				osName,
				osTypeValue,
				archValue,
				hostOs,
				displayBuildEngines(matches),
			)
		}
		return matches[0], nil
	}

	requestedEngine := alchemy_build.VirtualizationEngine(strings.ToLower(engineValue))
	for _, vm := range matches {
		if vm.VirtualizationEngine == requestedEngine {
			return vm, nil
		}
	}

	return alchemy_build.VirtualMachineConfig{}, fmt.Errorf(
		"invalid OCI artifact engine %q for OS=%s, Type=%s, Arch=%s, HostOS=%s; available engines: %s",
		engineValue,
		osName,
		osTypeValue,
		archValue,
		hostOs,
		displayBuildEngines(matches),
	)
}

func parseHostOS(value string) (alchemy_build.HostOsType, error) {
	if value == "" {
		return alchemy_build.GetCurrentHostOs(), nil
	}

	switch strings.ToLower(value) {
	case "linux", string(alchemy_build.HostOsLinux):
		return alchemy_build.HostOsLinux, nil
	case string(alchemy_build.HostOsWindows):
		return alchemy_build.HostOsWindows, nil
	case "macos", string(alchemy_build.HostOsDarwin):
		return alchemy_build.HostOsDarwin, nil
	default:
		return "", fmt.Errorf("invalid host OS %q; expected linux/debian, windows, or darwin/macos", value)
	}
}

func ociRegistryOptions(cmd *cobra.Command) (alchemy_oci.RegistryOptions, error) {
	password := ociPassword
	if ociPasswordStdin {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return alchemy_oci.RegistryOptions{}, fmt.Errorf("read OCI registry password from stdin: %w", err)
		}
		password = strings.TrimRight(string(data), "\r\n")
	}

	return alchemy_oci.RegistryOptions{
		PlainHTTP:                ociPlainHTTP,
		InsecureSkipTLSVerify:    ociInsecureSkipTLSVerify,
		CAFile:                   ociCAFile,
		Username:                 ociUsername,
		Password:                 password,
		AccessToken:              ociAccessToken,
		RefreshToken:             ociRefreshToken,
		DisableDockerCredentials: ociDisableDockerCredentials,
	}, nil
}

func runOCITransfer(cmd *cobra.Command, reference string, runner ociTransferRunner) (alchemy_oci.TransferResult, error) {
	vm, err := resolveOCIVirtualMachine(ociHostOS, ociOS, ociType, ociArch, ociEngine)
	if err != nil {
		return alchemy_oci.TransferResult{}, err
	}
	options, err := ociRegistryOptions(cmd)
	if err != nil {
		return alchemy_oci.TransferResult{}, err
	}
	return runner(cmd.Context(), vm, reference, options)
}

func printOCITransferResult(action string, result alchemy_oci.TransferResult) {
	fmt.Printf("✅ %s OCI artifact: %s\n", action, result.Reference)
	fmt.Printf("Digest: %s\n", result.Digest)
	for _, artifact := range result.Artifacts {
		fmt.Printf("Artifact: %s -> %s\n", artifact.Name, artifact.Path)
	}
}

var pushCmd = &cobra.Command{
	Use:   "push <registry>/<repository>[:tag]",
	Short: "Push VM build artifacts to an OCI registry",
	Long: `Pushes the final VM build artifact for a selected Dev Alchemy target to an OCI registry.

Examples:
  alchemy push localhost:5000/dev-alchemy/ubuntu-server-amd64:qemu --plain-http --os ubuntu --type server --arch amd64
  alchemy push ghcr.io/example/dev-alchemy/windows11-amd64:hyperv --os windows11 --arch amd64 --engine hyperv --host-os windows
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := runOCITransfer(cmd, args[0], runOCIPush)
		if err != nil {
			return err
		}
		printOCITransferResult("Pushed", result)
		return nil
	},
}

var pushListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available push combinations and local artifact readiness",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return printAvailableOCICombinations("push", pushListRow)
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull <registry>/<repository>[:tag|@digest]",
	Short: "Pull VM build artifacts from an OCI registry",
	Long: `Pulls the final VM build artifact for a selected Dev Alchemy target from an OCI registry.

Examples:
  alchemy pull localhost:5000/dev-alchemy/ubuntu-server-amd64:qemu --plain-http --os ubuntu --type server --arch amd64
  alchemy pull ghcr.io/example/dev-alchemy/windows11-amd64:hyperv --os windows11 --arch amd64 --engine hyperv --host-os windows
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := runOCITransfer(cmd, args[0], runOCIPull)
		if err != nil {
			return err
		}
		printOCITransferResult("Pulled", result)
		return nil
	},
}

var pullListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available pull combinations and local artifact overwrite status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return printAvailableOCICombinations("pull", pullListRow)
	},
}

func addOCIFlags(command *cobra.Command) {
	command.Flags().StringVar(&ociOS, "os", "", "Target operating system for the build artifact (e.g., ubuntu, windows11)")
	command.Flags().StringVarP(&ociType, "type", "t", "server", "Type of OS for Ubuntu artifacts (e.g., server, desktop)")
	command.Flags().StringVarP(&ociArch, "arch", "a", "amd64", "Target architecture (e.g., amd64, arm64)")
	command.Flags().StringVar(&ociEngine, "engine", "", "Virtualization engine for the build artifact (e.g., qemu, utm, hyperv, virtualbox)")
	command.Flags().StringVar(&ociHostOS, "host-os", string(alchemy_build.GetCurrentHostOs()), "Host OS that owns the build artifact shape (linux/debian, windows, darwin/macos)")
	command.Flags().BoolVar(&ociPlainHTTP, "plain-http", false, "Use plain HTTP for the OCI registry")
	command.Flags().BoolVar(&ociInsecureSkipTLSVerify, "insecure-skip-tls-verify", false, "Skip TLS certificate verification for HTTPS OCI registries")
	command.Flags().StringVar(&ociCAFile, "ca-file", "", "Path to a PEM CA certificate bundle to trust for HTTPS OCI registries")
	command.Flags().StringVar(&ociUsername, "username", "", "OCI registry username")
	command.Flags().StringVar(&ociPassword, "password", "", "OCI registry password")
	command.Flags().BoolVar(&ociPasswordStdin, "password-stdin", false, "Read OCI registry password from stdin")
	command.Flags().StringVar(&ociAccessToken, "access-token", "", "OCI registry bearer access token")
	command.Flags().StringVar(&ociRefreshToken, "refresh-token", "", "OCI registry refresh token")
	command.Flags().BoolVar(&ociDisableDockerCredentials, "no-docker-credentials", false, "Do not read credentials from Docker's credential store")
}

func addOCIListFlags(command *cobra.Command) {
	command.Flags().StringVar(&ociHostOS, "host-os", string(alchemy_build.GetCurrentHostOs()), "Host OS that owns the build artifact shape (linux/debian, windows, darwin/macos)")
}

func init() {
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
	pushCmd.AddCommand(pushListCmd)
	pullCmd.AddCommand(pullListCmd)
	addOCIFlags(pushCmd)
	addOCIFlags(pullCmd)
	addOCIListFlags(pushListCmd)
	addOCIListFlags(pullListCmd)
}
