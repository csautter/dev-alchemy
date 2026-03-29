package cmd

import (
	"bytes"
	"strings"
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

func TestIsBuildIncludedByDefault(t *testing.T) {
	tests := []struct {
		name string
		vm   alchemy_build.VirtualMachineConfig
		want bool
	}{
		{
			name: "windows hyperv windows11 included by default",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "windows11",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsWindows,
				VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
			},
			want: true,
		},
		{
			name: "windows virtualbox windows11 excluded by default",
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
		if got := isBuildIncludedByDefault(tt.vm); got != tt.want {
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

func TestDefaultBuildVirtualMachinesExcludesUnstableConfigs(t *testing.T) {
	for _, vm := range defaultBuildVirtualMachines() {
		if !isBuildIncludedByDefault(vm) {
			t.Fatalf("expected only default build configs, got engine %q", vm.VirtualizationEngine)
		}
	}
}

func TestResolveBuildVirtualMachineRequiresEngineForAmbiguousSelection(t *testing.T) {
	vms := []alchemy_build.VirtualMachineConfig{
		{
			OS:                   "windows11",
			Arch:                 "amd64",
			HostOs:               alchemy_build.HostOsWindows,
			VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
		},
		{
			OS:                   "windows11",
			Arch:                 "amd64",
			HostOs:               alchemy_build.HostOsWindows,
			VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
		},
	}

	_, err := resolveBuildVirtualMachine(vms, "windows11", "", "amd64", "")
	if err == nil {
		t.Fatal("expected ambiguous selection error, got nil")
	}
	if !strings.Contains(err.Error(), "--engine") {
		t.Fatalf("expected error to mention --engine, got %q", err.Error())
	}
}

func TestResolveBuildVirtualMachineSelectsRequestedEngine(t *testing.T) {
	vms := []alchemy_build.VirtualMachineConfig{
		{
			OS:                   "windows11",
			Arch:                 "amd64",
			HostOs:               alchemy_build.HostOsWindows,
			VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
		},
		{
			OS:                   "windows11",
			Arch:                 "amd64",
			HostOs:               alchemy_build.HostOsWindows,
			VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
		},
	}

	vm, err := resolveBuildVirtualMachine(vms, "windows11", "", "amd64", "virtualbox")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if vm.VirtualizationEngine != alchemy_build.VirtualizationEngineVirtualBox {
		t.Fatalf("expected virtualbox engine, got %q", vm.VirtualizationEngine)
	}
}

func TestBuildHelpIncludesEngineFlag(t *testing.T) {
	var buf bytes.Buffer
	buildCmd.SetOut(&buf)
	buildCmd.SetErr(&buf)

	if err := buildCmd.Help(); err != nil {
		t.Fatalf("expected no help error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--engine string") {
		t.Fatalf("expected --engine flag in help output, got %q", output)
	}
	if !strings.Contains(output, "build all stable VM configurations") {
		t.Fatalf("expected help output to describe stable all builds, got %q", output)
	}
}
