package cmd

import (
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func withCurrentHostArchitecture(t *testing.T, hostArch string) {
	t.Helper()
	previousCurrentHostArchitectureFunc := currentHostArchitectureFunc
	currentHostArchitectureFunc = func() string {
		return hostArch
	}
	t.Cleanup(func() {
		currentHostArchitectureFunc = previousCurrentHostArchitectureFunc
	})
}

func requireVMStatus(t *testing.T, vms []alchemy_build.VirtualMachineConfig, ubuntuType string, arch string, wantStatus string) {
	t.Helper()
	for _, vm := range vms {
		if vm.OS == "ubuntu" && vm.UbuntuType == ubuntuType && vm.Arch == arch {
			if got := virtualMachineTargetStatus(vm); got != wantStatus {
				t.Fatalf("expected %s/%s to be %s, got %s", ubuntuType, arch, wantStatus, got)
			}
			return
		}
	}
	t.Fatalf("expected to find ubuntu/%s/%s in VM list", ubuntuType, arch)
}

func requireProvisionVMStatus(t *testing.T, vms []alchemy_build.VirtualMachineConfig, ubuntuType string, arch string, wantStatus string) {
	t.Helper()
	for _, vm := range vms {
		if vm.OS == "ubuntu" && vm.UbuntuType == ubuntuType && vm.Arch == arch {
			if got := provisionStatus(vm); got != wantStatus {
				t.Fatalf("expected provision %s/%s to be %s, got %s", ubuntuType, arch, wantStatus, got)
			}
			return
		}
	}
	t.Fatalf("expected to find ubuntu/%s/%s in provision VM list", ubuntuType, arch)
}

func requireOnlyArch(t *testing.T, vms []alchemy_build.VirtualMachineConfig, wantArch string) {
	t.Helper()
	if len(vms) == 0 {
		t.Fatalf("expected at least one %s VM", wantArch)
	}
	for _, vm := range vms {
		if vm.Arch != wantArch {
			t.Fatalf("expected only %s VMs, got %+v", wantArch, vm)
		}
	}
}
