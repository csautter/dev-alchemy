package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"

	"github.com/spf13/cobra"
)

// BuildRunner is a function that executes a single VM build.
// It receives a context that is cancelled when the process is interrupted (SIGINT/SIGTERM).
// Implementations should honour ctx.Done() so they can abort early.
type buildRunner func(ctx context.Context, vm alchemy_build.VirtualMachineConfig) error

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

var (
	arch     string
	parallel int
	headless bool
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
			available_virtual_machines := alchemy_build.AvailableVirtualMachineConfigsForCurrentHostOS()

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
				if vm.OS == "ubuntu" {
					return alchemy_build.RunQemuUbuntuBuildOnMacOS(vm)
				}
				if vm.OS == "windows11" {
					return alchemy_build.RunQemuWindowsBuildOnMacOS(vm)
				}
				return fmt.Errorf("unknown OS: %s", vm.OS)
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
			fmt.Printf("❌ Invalid combination: OS=%s, Type=%s, Arch=%s\n", osName, osType, arch)
			return
		}

		port := 5900 + (rand.Intn(100) + 1)

		VirtualMachineConfig.VncPort = port
		VirtualMachineConfig.Headless = headless

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
	buildCmd.Flags().IntVarP(&parallel, "parallel", "p", 1, "Number of parallel builds to run when building all VMs")
	buildCmd.Flags().BoolVar(&headless, "headless", false, "Run QEMU in headless mode (no GUI, VNC only)")
}
