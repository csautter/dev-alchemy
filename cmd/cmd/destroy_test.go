package cmd

import (
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestRunDestroyReturnsErrorForUnsupportedEngine(t *testing.T) {
	vm := alchemy_build.VirtualMachineConfig{
		OS:                   "windows11",
		Arch:                 "amd64",
		VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
	}

	err := runDestroy(vm)
	if err == nil {
		t.Fatal("expected error for unsupported destroy configuration")
	}
	if !strings.Contains(err.Error(), string(alchemy_build.VirtualizationEngineVirtualBox)) {
		t.Fatalf("expected error to mention engine %q, got %q", vm.VirtualizationEngine, err.Error())
	}
}

func TestIsDestroySupported(t *testing.T) {
	tests := []struct {
		name string
		vm   alchemy_build.VirtualMachineConfig
		want bool
	}{
		{
			name: "utm supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "ubuntu",
				Arch:                 "arm64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineUtm,
			},
			want: true,
		},
		{
			name: "hyperv supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "windows11",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsWindows,
				VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
			},
			want: true,
		},
		{
			name: "tart supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "macos",
				Arch:                 "arm64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineTart,
			},
			want: true,
		},
		{
			name: "virtualbox unsupported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "windows11",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsWindows,
				VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		if got := isDestroySupported(tt.vm); got != tt.want {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestAvailableDestroyVirtualMachinesOnlyReturnsSupportedConfigs(t *testing.T) {
	for _, vm := range availableDestroyVirtualMachines() {
		if !isDestroySupported(vm) {
			t.Fatalf("expected only supported destroy configs, got engine %q", vm.VirtualizationEngine)
		}
	}
}

func TestEveryCreateSupportedVirtualMachineAlsoSupportsDestroy(t *testing.T) {
	for _, vm := range alchemy_build.AvailableVirtualMachineConfigs() {
		if !isCreateSupported(vm) {
			continue
		}
		if !isDestroySupported(vm) {
			t.Fatalf(
				"expected create-supported config to support destroy: OS=%s type=%s arch=%s host=%s engine=%s",
				vm.OS,
				vm.UbuntuType,
				vm.Arch,
				vm.HostOs,
				vm.VirtualizationEngine,
			)
		}
	}
}
