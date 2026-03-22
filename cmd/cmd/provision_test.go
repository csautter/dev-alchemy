package cmd

import (
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestRunProvisionReturnsErrorForUnsupportedConfig(t *testing.T) {
	vm := alchemy_build.VirtualMachineConfig{
		OS:                   "windows11",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsWindows,
		VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
	}

	err := runProvision(vm, false)
	if err == nil {
		t.Fatal("expected runProvision to return an error for unsupported vm configuration")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected error to mention not implemented, got: %v", err)
	}
}

func TestProvisionCommandRejectsAllTarget(t *testing.T) {
	previousArch := arch
	previousOsType := osType
	previousCheck := check
	t.Cleanup(func() {
		arch = previousArch
		osType = previousOsType
		check = previousCheck
	})

	arch = "amd64"
	osType = "server"
	check = false

	err := provisionCmd.RunE(provisionCmd, []string{"all"})
	if err == nil {
		t.Fatal("expected an error when using provision all")
	}
	if !strings.Contains(err.Error(), "\"all\" is not supported for provision") {
		t.Fatalf("expected explicit unsupported-all error, got: %v", err)
	}
}

func TestIsProvisionSupported(t *testing.T) {
	tests := []struct {
		name string
		vm   alchemy_build.VirtualMachineConfig
		want bool
	}{
		{
			name: "windows hyperv windows11 amd64 supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "windows11",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsWindows,
				VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
			},
			want: true,
		},
		{
			name: "windows hyperv ubuntu amd64 supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "ubuntu",
				Arch:                 "amd64",
				UbuntuType:           "server",
				HostOs:               alchemy_build.HostOsWindows,
				VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
			},
			want: true,
		},
		{
			name: "windows virtualbox unsupported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "windows11",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsWindows,
				VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
			},
			want: false,
		},
		{
			name: "darwin utm unsupported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "ubuntu",
				Arch:                 "arm64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineUtm,
			},
			want: false,
		},
		{
			name: "darwin utm windows11 arm64 supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "windows11",
				Arch:                 "arm64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineUtm,
			},
			want: true,
		},
		{
			name: "darwin utm windows11 amd64 supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "windows11",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineUtm,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		if got := isProvisionSupported(tt.vm); got != tt.want {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestAvailableProvisionVirtualMachinesOnlyReturnsSupportedConfigs(t *testing.T) {
	for _, vm := range availableProvisionVirtualMachines() {
		if !isProvisionSupported(vm) {
			t.Fatalf("expected only supported provision configs, got engine %q", vm.VirtualizationEngine)
		}
	}
}
