package cmd

import (
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestRunDeployReturnsErrorForUnsupportedEngine(t *testing.T) {
	vm := alchemy_build.VirtualMachineConfig{
		OS:                   "windows11",
		Arch:                 "amd64",
		VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
	}

	err := runDeploy(vm)
	if err == nil {
		t.Fatalf("expected error for unsupported engine %q, got nil", vm.VirtualizationEngine)
	}
	if !strings.Contains(err.Error(), string(alchemy_build.VirtualizationEngineVirtualBox)) {
		t.Fatalf("expected error to mention engine %q, got %q", vm.VirtualizationEngine, err.Error())
	}
}
