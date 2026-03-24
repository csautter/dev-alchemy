package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"

	"github.com/spf13/cobra"
)

// BuildRunner is a function that executes a single VM build.
// It receives a context that is cancelled when the process is interrupted (SIGINT/SIGTERM).
// Implementations should honour ctx.Done() so they can abort early.
type buildRunner func(ctx context.Context, vm alchemy_build.VirtualMachineConfig) error

func isBuildSupported(vm alchemy_build.VirtualMachineConfig) bool {
	switch vm.HostOs {
	case alchemy_build.HostOsDarwin:
		// Tart VMs are pulled from OCI registries via `tart clone`; local packer builds are not applicable.
		return vm.VirtualizationEngine == alchemy_build.VirtualizationEngineUtm &&
			(vm.OS == "ubuntu" || vm.OS == "windows11")
	case alchemy_build.HostOsWindows:
		switch vm.VirtualizationEngine {
		case alchemy_build.VirtualizationEngineHyperv:
			return vm.OS == "ubuntu" || vm.OS == "windows11"
		case alchemy_build.VirtualizationEngineVirtualBox:
			return vm.OS == "windows11"
		default:
			return false
		}
	default:
		return false
	}
}

func availableBuildVirtualMachines() []alchemy_build.VirtualMachineConfig {
	var supported []alchemy_build.VirtualMachineConfig
	for _, vm := range alchemy_build.AvailableVirtualMachineConfigsForCurrentHostOS() {
		if isBuildSupported(vm) {
			supported = append(supported, vm)
		}
	}
	return supported
}

// runParallelBuilds launches up to parallelism concurrent builds for the provided VMs.
// A failing build does NOT stop the remaining ones — all errors are collected and returned.
// When ctx is cancelled (e.g. on SIGINT) no new goroutines are started; already-running
// builds will be interrupted if their buildRunner honours the context.
func runParallelBuilds(ctx context.Context, vms []alchemy_build.VirtualMachineConfig, parallelism int, runner buildRunner) []error {
	var mu sync.Mutex
	var errs []error
	var wg sync.WaitGroup
	sem := make(chan struct{}, parallelism)

	for _, vm := range vms {
		// Acquire a semaphore slot OR bail out if the context is cancelled while waiting.
		// Without this select the loop would deadlock when ctx is cancelled while the
		// semaphore channel is full (i.e. all parallel slots are occupied).
		select {
		case sem <- struct{}{}:
			// acquired a slot, proceed
		case <-ctx.Done():
			mu.Lock()
			errs = append(errs, fmt.Errorf("build cancelled before starting %s/%s/%s: %w", vm.OS, vm.UbuntuType, vm.Arch, ctx.Err()))
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(vm alchemy_build.VirtualMachineConfig) {
			defer wg.Done()
			defer func() { <-sem }()

			fmt.Printf("➡️ Building VM for OS: %s, Type: %s, Architecture: %s\n", vm.OS, vm.UbuntuType, vm.Arch)
			if err := runner(ctx, vm); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s/%s/%s: %w", vm.OS, vm.UbuntuType, vm.Arch, err))
				mu.Unlock()
				fmt.Printf("❌ Build failed for OS: %s, Type: %s, Architecture: %s — %v\n", vm.OS, vm.UbuntuType, vm.Arch, err)
			} else {
				fmt.Printf("✅ Build succeeded for OS: %s, Type: %s, Architecture: %s\n", vm.OS, vm.UbuntuType, vm.Arch)
			}
		}(vm)
	}

	wg.Wait()
	return errs
}

func runBuild(vm alchemy_build.VirtualMachineConfig) error {
	switch vm.HostOs {
	case alchemy_build.HostOsDarwin:
		switch vm.VirtualizationEngine {
		case alchemy_build.VirtualizationEngineUtm:
			if vm.OS == "ubuntu" {
				return alchemy_build.RunQemuUbuntuBuildOnMacOS(vm)
			}
			if vm.OS == "windows11" {
				return alchemy_build.RunQemuWindowsBuildOnMacOS(vm)
			}
		}
	case alchemy_build.HostOsWindows:
		switch vm.VirtualizationEngine {
		case alchemy_build.VirtualizationEngineHyperv:
			if vm.OS == "windows11" {
				return alchemy_build.RunHypervWindowsBuildOnWindows(vm)
			}
			if vm.OS == "ubuntu" {
				return alchemy_build.RunHypervUbuntuBuildOnWindows(vm)
			}
		case alchemy_build.VirtualizationEngineVirtualBox:
			if vm.OS == "windows11" {
				return alchemy_build.RunVirtualBoxWindowsBuildOnWindows(vm)
			}
		}
	}

	return fmt.Errorf(
		"build is not implemented for OS=%s type=%s arch=%s host_os=%s virtualization_engine=%s",
		vm.OS,
		vm.UbuntuType,
		vm.Arch,
		vm.HostOs,
		vm.VirtualizationEngine,
	)
}

var (
	arch     string
	parallel int
	headless bool
	noCache  bool
)

func printAvailableBuildCombinations() error {
	vms := availableBuildVirtualMachines()
	engines := alchemy_build.VirtualizationEnginesForVirtualMachineConfigs(vms)

	if err := printVirtualMachineCombinationTable(
		os.Stdout,
		fmt.Sprintf("Available build combinations for host OS: %s", alchemy_build.GetCurrentHostOs()),
		"No build combinations are available for the current host OS.",
		vms,
		[]string{"OS", "Type", "Arch"},
		func(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
			return []string{vm.OS, displayVirtualMachineType(vm), vm.Arch}, nil
		},
	); err != nil {
		return err
	}

	if len(engines) > 1 {
		engineNames := make([]string, 0, len(engines))
		for _, engine := range engines {
			engineNames = append(engineNames, string(engine))
		}
		fmt.Printf("\nCurrent host supports multiple virtualization engines: %s\n", strings.Join(engineNames, ", "))
	}

	return nil
}

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
  alchemy build all --parallel 4
`,
	Args: cobra.ExactArgs(1), // Enforce exactly one positional argument
	Run: func(cmd *cobra.Command, args []string) {
		osName := args[0]

		if osName != "ubuntu" {
			osType = ""
		}

		if osName == "all" {
			fmt.Printf("🔧 Building all available VM configurations with %d parallel builds\n", parallel)
			available_virtual_machines := availableBuildVirtualMachines()
			for i := range available_virtual_machines {
				available_virtual_machines[i].NoCache = noCache
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			defer signal.Stop(sigs)
			go func() {
				select {
				case <-sigs:
					fmt.Printf("\n⚠️  Interrupted! Cancelling all remaining builds...\n")
					cancel()
				case <-ctx.Done():
				}
			}()

			runner := func(ctx context.Context, vm alchemy_build.VirtualMachineConfig) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				return runBuild(vm)
			}

			errs := runParallelBuilds(ctx, available_virtual_machines, parallel, runner)
			if len(errs) > 0 {
				fmt.Printf("❌ %d build(s) failed:\n", len(errs))
				for _, e := range errs {
					fmt.Printf("  - %v\n", e)
				}
			} else {
				fmt.Printf("✅ All builds completed successfully\n")
			}
			return
		}

		fmt.Printf("🔧 Building VM for OS: %s, Type: %s, Architecture: %s\n", osName, osType, arch)
		available_virtual_machines := availableBuildVirtualMachines()
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
			fmt.Printf("❌ Invalid combination: OS=%s, Type=%s, Arch=%s\n", osName, osType, arch)
			return
		}

		// #nosec G404 -- this random value only spreads local VNC port selection and is not security-sensitive.
		port := 5900 + (rand.Intn(100) + 1)

		VirtualMachineConfig.VncPort = port
		VirtualMachineConfig.Headless = headless
		VirtualMachineConfig.NoCache = noCache

		if err := runBuild(VirtualMachineConfig); err != nil {
			fmt.Printf("❌ Build failed for OS: %s, Type: %s, Architecture: %s — %v\n", osName, osType, arch, err)
		}
	},
}

var buildListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available OS, type, and architecture combinations for build",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return printAvailableBuildCombinations()
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.AddCommand(buildListCmd)

	buildCmd.Flags().StringVarP(&arch, "arch", "a", "amd64", "Target architecture (e.g., amd64, arm64)")
	buildCmd.Flags().StringVarP(&osType, "type", "t", "server", "Type of OS (e.g., server, desktop)")
	buildCmd.Flags().IntVarP(&parallel, "parallel", "p", 1, "Number of parallel builds to run when building all VMs")
	buildCmd.Flags().BoolVar(&headless, "headless", false, "Run QEMU in headless mode (no GUI, VNC only)")
	buildCmd.Flags().BoolVar(&noCache, "no-cache", false, "Force a rebuild even when the build artifact already exists")
}
