package cmd

import (
	"bytes"
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	alchemy_deploy "github.com/csautter/dev-alchemy/pkg/deploy"
)

func TestRunStopReturnsErrorForUnsupportedEngine(t *testing.T) {
	vm := alchemy_build.VirtualMachineConfig{
		OS:                   "windows11",
		Arch:                 "amd64",
		VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
	}

	err := runStop(vm)
	if err == nil {
		t.Fatal("expected error for unsupported stop configuration")
	}
	if !strings.Contains(err.Error(), string(alchemy_build.VirtualizationEngineVirtualBox)) {
		t.Fatalf("expected error to mention engine %q, got %q", vm.VirtualizationEngine, err.Error())
	}
}

func TestIsStopSupported(t *testing.T) {
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
				UbuntuType:           "server",
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
		if got := isStopSupported(tt.vm); got != tt.want {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestAvailableStopVirtualMachinesOnlyReturnsSupportedConfigs(t *testing.T) {
	for _, vm := range availableStopVirtualMachines() {
		if !isStopSupported(vm) {
			t.Fatalf("expected only supported stop configs, got engine %q", vm.VirtualizationEngine)
		}
	}
}

func TestPrintAvailableStopCombinationsIncludesStopReadiness(t *testing.T) {
	previousInspector := inspectStopTarget
	t.Cleanup(func() {
		inspectStopTarget = previousInspector
	})

	inspectStopTarget = func(vm alchemy_build.VirtualMachineConfig) (alchemy_deploy.VirtualMachineState, error) {
		switch vm.OS {
		case "macos":
			return alchemy_deploy.VirtualMachineState{Exists: true, Running: true, State: "running"}, nil
		case "ubuntu":
			return alchemy_deploy.VirtualMachineState{Exists: true, State: "stopped"}, nil
		default:
			return alchemy_deploy.VirtualMachineState{State: "missing"}, nil
		}
	}

	var buf bytes.Buffer
	err := printVirtualMachineCombinationTable(
		&buf,
		"Available stop combinations for host OS: darwin",
		"No stop combinations are available for the current host OS.",
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
		[]string{"OS", "Type", "Arch", "State", "Stop"},
		func(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
			state, err := inspectStopTarget(vm)
			if err != nil {
				return nil, err
			}

			stopState := "ready to stop"
			displayState := state.State
			switch {
			case !state.Exists:
				displayState = "missing"
				stopState = "already absent"
			case state.Running:
				if displayState == "" {
					displayState = "running"
				}
			default:
				if displayState == "" {
					displayState = "stopped"
				}
				stopState = "already stopped"
			}

			return []string{vm.OS, displayVirtualMachineType(vm), vm.Arch, displayState, stopState}, nil
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "State") || !strings.Contains(output, "Stop") {
		t.Fatalf("expected stop readiness headers, got %q", output)
	}
	if !strings.Contains(output, "macos") || !strings.Contains(output, "ready to stop") {
		t.Fatalf("expected running macos stop target row, got %q", output)
	}
	if !strings.Contains(output, "ubuntu") || !strings.Contains(output, "already stopped") {
		t.Fatalf("expected stopped ubuntu stop target row, got %q", output)
	}
	if !strings.Contains(output, "windows11") || !strings.Contains(output, "already absent") {
		t.Fatalf("expected missing windows stop target row, got %q", output)
	}
}
