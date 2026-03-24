package cmd

import (
	"bytes"
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	alchemy_deploy "github.com/csautter/dev-alchemy/pkg/deploy"
)

func TestRunStartReturnsErrorForUnsupportedEngine(t *testing.T) {
	vm := alchemy_build.VirtualMachineConfig{
		OS:                   "windows11",
		Arch:                 "amd64",
		VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
	}

	err := runStart(vm)
	if err == nil {
		t.Fatal("expected error for unsupported start configuration")
	}
	if !strings.Contains(err.Error(), string(alchemy_build.VirtualizationEngineVirtualBox)) {
		t.Fatalf("expected error to mention engine %q, got %q", vm.VirtualizationEngine, err.Error())
	}
}

func TestIsStartSupported(t *testing.T) {
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
		if got := isStartSupported(tt.vm); got != tt.want {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestAvailableStartVirtualMachinesOnlyReturnsSupportedConfigs(t *testing.T) {
	for _, vm := range availableStartVirtualMachines() {
		if !isStartSupported(vm) {
			t.Fatalf("expected only supported start configs, got engine %q", vm.VirtualizationEngine)
		}
	}
}

func TestPrintAvailableStartCombinationsIncludesStartReadiness(t *testing.T) {
	previousInspector := inspectStartTarget
	t.Cleanup(func() {
		inspectStartTarget = previousInspector
	})

	inspectStartTarget = func(vm alchemy_build.VirtualMachineConfig) (alchemy_deploy.StartTargetState, error) {
		switch vm.OS {
		case "macos":
			return alchemy_deploy.StartTargetState{Exists: true, Running: true, State: "running"}, nil
		case "ubuntu":
			return alchemy_deploy.StartTargetState{Exists: true, State: "stopped"}, nil
		default:
			return alchemy_deploy.StartTargetState{State: "missing"}, nil
		}
	}

	var buf bytes.Buffer
	err := printVirtualMachineCombinationTable(
		&buf,
		"Available start combinations for host OS: darwin",
		"No start combinations are available for the current host OS.",
		[]alchemy_build.VirtualMachineConfig{
			{
				OS:                   "macos",
				Arch:                 "arm64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineTart,
			},
			{
				OS:                   "ubuntu",
				UbuntuType:           "server",
				Arch:                 "arm64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineUtm,
			},
			{
				OS:                   "windows11",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineUtm,
			},
		},
		[]string{"OS", "Type", "Arch", "State", "Start"},
		func(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
			state, err := inspectStartTarget(vm)
			if err != nil {
				return nil, err
			}

			startState := "ready to start"
			displayState := state.State
			switch {
			case !state.Exists:
				displayState = "missing"
				startState = "create required"
			case state.Running:
				startState = "already running"
			case displayState == "":
				displayState = "stopped"
			}

			return []string{vm.OS, displayVirtualMachineType(vm), vm.Arch, displayState, startState}, nil
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "State") || !strings.Contains(output, "Start") {
		t.Fatalf("expected start readiness headers, got %q", output)
	}
	if !strings.Contains(output, "macos") || !strings.Contains(output, "running") || !strings.Contains(output, "already running") {
		t.Fatalf("expected running macos start target row, got %q", output)
	}
	if !strings.Contains(output, "ubuntu") || !strings.Contains(output, "ready to start") {
		t.Fatalf("expected stopped ubuntu start target row, got %q", output)
	}
	if !strings.Contains(output, "windows11") || !strings.Contains(output, "create required") {
		t.Fatalf("expected missing windows start target row, got %q", output)
	}
}
