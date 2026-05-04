package cmd

import (
	"fmt"
	"runtime"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

var currentHostArchitectureFunc = func() string {
	return runtime.GOARCH
}

type virtualMachineSupportPredicate func(alchemy_build.VirtualMachineConfig) bool

func availableVirtualMachinesForHostOS(hostOs alchemy_build.HostOsType, isSupported virtualMachineSupportPredicate) []alchemy_build.VirtualMachineConfig {
	var supported []alchemy_build.VirtualMachineConfig
	for _, vm := range alchemy_build.AvailableVirtualMachineConfigsForHostOS(hostOs) {
		if isSupported(vm) {
			supported = append(supported, vm)
		}
	}
	return supported
}

func stableVirtualMachines(vms []alchemy_build.VirtualMachineConfig) []alchemy_build.VirtualMachineConfig {
	var stable []alchemy_build.VirtualMachineConfig
	for _, vm := range vms {
		if !isVirtualMachineTargetUnstable(vm) {
			stable = append(stable, vm)
		}
	}
	return stable
}

func unstableVirtualMachines(vms []alchemy_build.VirtualMachineConfig) []alchemy_build.VirtualMachineConfig {
	var unstable []alchemy_build.VirtualMachineConfig
	for _, vm := range vms {
		if isVirtualMachineTargetUnstable(vm) {
			unstable = append(unstable, vm)
		}
	}
	return unstable
}

func virtualMachineTargetStatus(vm alchemy_build.VirtualMachineConfig) string {
	if isVirtualMachineTargetUnstable(vm) {
		return "unstable"
	}
	return "stable"
}

func isVirtualMachineTargetUnstable(vm alchemy_build.VirtualMachineConfig) bool {
	return alchemy_build.IsVirtualizationEngineUnstable(vm.VirtualizationEngine) ||
		isCrossArchitectureEmulationTarget(vm)
}

func isCrossArchitectureEmulationTarget(vm alchemy_build.VirtualMachineConfig) bool {
	if vm.Arch == "" || vm.Arch == "-" {
		return false
	}
	hostArch := currentHostArchitectureFunc()
	if vm.Arch == hostArch {
		return false
	}

	switch {
	case vm.HostOs == alchemy_build.HostOsDarwin &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineUtm:
		return isEmulatableArchitecture(vm.Arch) && isEmulatableArchitecture(hostArch)
	case vm.HostOs == alchemy_build.HostOsLinux &&
		vm.VirtualizationEngine == alchemy_build.VirtualizationEngineQemu:
		return isEmulatableArchitecture(vm.Arch) && isEmulatableArchitecture(hostArch)
	default:
		return false
	}
}

func isEmulatableArchitecture(arch string) bool {
	switch arch {
	case "amd64", "arm64":
		return true
	default:
		return false
	}
}

func printSkippedUnstableTargets(action string, vms []alchemy_build.VirtualMachineConfig) {
	skipped := unstableVirtualMachines(vms)
	if len(skipped) == 0 {
		return
	}

	fmt.Printf("⚠️ Skipping %d unstable %s target(s). Run a target explicitly to use cross-architecture emulation.\n", len(skipped), action)
}

func printUnstableTargetWarning(vm alchemy_build.VirtualMachineConfig) {
	if isCrossArchitectureEmulationTarget(vm) {
		fmt.Printf("⚠️ Cross-architecture emulation for OS=%s, Type=%s, Arch=%s is marked unstable and may be slow.\n", vm.OS, displayVirtualMachineType(vm), vm.Arch)
		return
	}

	if alchemy_build.IsVirtualizationEngineUnstable(vm.VirtualizationEngine) {
		fmt.Printf("⚠️ Virtualization engine %s is marked unstable for OS=%s, Type=%s, Arch=%s.\n", vm.VirtualizationEngine, vm.OS, displayVirtualMachineType(vm), vm.Arch)
	}
}
