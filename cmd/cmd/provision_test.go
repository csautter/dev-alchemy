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
