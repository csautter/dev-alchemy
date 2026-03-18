package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestPrintVirtualMachineCombinationTableGroupsBySortedEngine(t *testing.T) {
	vms := []alchemy_build.VirtualMachineConfig{
		{
			OS:                   "windows11",
			Arch:                 "amd64",
			VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
		},
		{
			OS:                   "ubuntu",
			Arch:                 "amd64",
			UbuntuType:           "server",
			VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
		},
		{
			OS:                   "windows11",
			Arch:                 "arm64",
			VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
		},
	}

	var buf bytes.Buffer
	err := printVirtualMachineCombinationTable(
		&buf,
		"Available test combinations for host OS: windows",
		"No test combinations are available for the current host OS.",
		vms,
		[]string{"OS", "Type", "Arch"},
		func(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
			return []string{vm.OS, displayVirtualMachineType(vm), vm.Arch}, nil
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Available test combinations for host OS: windows") {
		t.Fatalf("expected title in output, got %q", output)
	}

	hypervIndex := strings.Index(output, "Virtualization engine: hyperv")
	virtualboxIndex := strings.Index(output, "Virtualization engine: virtualbox")
	if hypervIndex == -1 || virtualboxIndex == -1 {
		t.Fatalf("expected both virtualization engines in output, got %q", output)
	}
	if hypervIndex > virtualboxIndex {
		t.Fatalf("expected sorted engine sections, got %q", output)
	}

	headerCount := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "OS") && strings.Contains(line, "Type") && strings.Contains(line, "Arch") {
			headerCount++
		}
	}
	if headerCount != 2 {
		t.Fatalf("expected headers for each engine section, got %q", output)
	}
	if !strings.Contains(output, "ubuntu     server  amd64") {
		t.Fatalf("expected ubuntu row in output, got %q", output)
	}
	if !strings.Contains(output, "windows11  -       arm64") {
		t.Fatalf("expected windows row with fallback type in output, got %q", output)
	}
}

func TestPrintVirtualMachineCombinationTableWritesEmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	err := printVirtualMachineCombinationTable(
		&buf,
		"Available test combinations for host OS: windows",
		"No test combinations are available for the current host OS.",
		nil,
		[]string{"OS", "Type", "Arch"},
		func(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
			return []string{vm.OS, displayVirtualMachineType(vm), vm.Arch}, nil
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No test combinations are available for the current host OS.") {
		t.Fatalf("expected empty message in output, got %q", output)
	}
}

func TestPrintVirtualMachineCombinationTableReturnsRowBuilderError(t *testing.T) {
	wantErr := errors.New("row failed")

	err := printVirtualMachineCombinationTable(
		&bytes.Buffer{},
		"Available test combinations for host OS: windows",
		"No test combinations are available for the current host OS.",
		[]alchemy_build.VirtualMachineConfig{
			{
				OS:                   "windows11",
				Arch:                 "amd64",
				VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
			},
		},
		[]string{"OS", "Type", "Arch"},
		func(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
			return nil, wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}
