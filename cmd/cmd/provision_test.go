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

func TestProvisionCommandRejectsArchAndTypeForLocalTarget(t *testing.T) {
	previousArch := arch
	previousOsType := osType
	previousCheck := check
	t.Cleanup(func() {
		arch = previousArch
		osType = previousOsType
		check = previousCheck
	})

	arch = "arm64"
	osType = "server"
	check = false

	if err := provisionCmd.Flags().Set("arch", "arm64"); err != nil {
		t.Fatalf("failed to mark arch flag as changed: %v", err)
	}
	if err := provisionCmd.Flags().Set("type", "server"); err != nil {
		t.Fatalf("failed to mark type flag as changed: %v", err)
	}
	t.Cleanup(func() {
		provisionCmd.Flags().Lookup("arch").Changed = false
		provisionCmd.Flags().Lookup("type").Changed = false
	})

	err := provisionCmd.RunE(provisionCmd, []string{"local"})
	if err == nil {
		t.Fatal("expected an error when local target is combined with arch/type flags")
	}
	if !strings.Contains(err.Error(), "does not accept --arch or --type") {
		t.Fatalf("expected explicit local flag validation error, got: %v", err)
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
			name: "darwin utm ubuntu arm64 supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "ubuntu",
				Arch:                 "arm64",
				UbuntuType:           "server",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineUtm,
			},
			want: true,
		},
		{
			name: "darwin utm ubuntu amd64 supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "ubuntu",
				Arch:                 "amd64",
				UbuntuType:           "desktop",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineUtm,
			},
			want: true,
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
		{
			name: "darwin tart macos arm64 supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "macos",
				Arch:                 "arm64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineTart,
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
	var foundLocal bool
	for _, vm := range availableProvisionVirtualMachines() {
		if vm.OS == "local" {
			foundLocal = true
			continue
		}
		if !isProvisionSupported(vm) {
			t.Fatalf("expected only supported provision configs, got engine %q", vm.VirtualizationEngine)
		}
	}
	if !foundLocal {
		t.Fatal("expected local provision target to be included for the current host")
	}
}

func TestCurrentHostLocalProvisionVirtualMachineUsesLocalEngine(t *testing.T) {
	vm, ok := currentHostLocalProvisionVirtualMachine()
	if !ok {
		t.Fatal("expected current host local provision target to be available")
	}
	if vm.OS != "local" {
		t.Fatalf("expected local provision OS, got %q", vm.OS)
	}
	if vm.VirtualizationEngine != localProvisionVirtualizationEngine {
		t.Fatalf("expected local virtualization engine, got %q", vm.VirtualizationEngine)
	}
	if vm.Arch != "-" {
		t.Fatalf("expected local provision arch placeholder, got %q", vm.Arch)
	}
}

func TestProvisionStatusMarksLocalNonWindowsHostsUnstable(t *testing.T) {
	if got := provisionStatus(alchemy_build.VirtualMachineConfig{
		OS:     "local",
		HostOs: alchemy_build.HostOsWindows,
	}); got != "stable" {
		t.Fatalf("expected windows local provision to be stable, got %q", got)
	}

	if got := provisionStatus(alchemy_build.VirtualMachineConfig{
		OS:     "local",
		HostOs: alchemy_build.HostOsDarwin,
	}); got != "unstable" {
		t.Fatalf("expected darwin local provision to be unstable, got %q", got)
	}

	if got := provisionStatus(alchemy_build.VirtualMachineConfig{
		OS:     "local",
		HostOs: alchemy_build.HostOsLinux,
	}); got != "unstable" {
		t.Fatalf("expected linux local provision to be unstable, got %q", got)
	}
}
