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
