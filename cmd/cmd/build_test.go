package cmd

import (
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestIsBuildSupported(t *testing.T) {
	tests := []struct {
		name string
		vm   alchemy_build.VirtualMachineConfig
		want bool
	}{
		{
			name: "darwin utm ubuntu supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "ubuntu",
				Arch:                 "arm64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineUtm,
			},
			want: true,
		},
		{
			name: "darwin tart macos unsupported for build",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "macos",
				Arch:                 "arm64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineTart,
			},
			want: false,
		},
		{
			name: "windows hyperv windows11 supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "windows11",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsWindows,
				VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
			},
			want: true,
		},
		{
			name: "windows virtualbox ubuntu unsupported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "ubuntu",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsWindows,
				VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		if got := isBuildSupported(tt.vm); got != tt.want {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestAvailableBuildVirtualMachinesOnlyReturnsSupportedConfigs(t *testing.T) {
	for _, vm := range availableBuildVirtualMachines() {
		if !isBuildSupported(vm) {
			t.Fatalf("expected only supported build configs, got engine %q", vm.VirtualizationEngine)
		}
	}
}
