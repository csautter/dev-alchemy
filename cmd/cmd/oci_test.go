package cmd

import (
	"bytes"
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	"github.com/spf13/cobra"
)

func TestResolveOCIVirtualMachineRequiresEngineForAmbiguousTarget(t *testing.T) {
	_, err := resolveOCIVirtualMachine("windows", "windows11", "", "amd64", "")
	if err == nil {
		t.Fatal("expected ambiguous Windows artifact target to require --engine")
	}
	if !strings.Contains(err.Error(), "--engine") {
		t.Fatalf("expected error to mention --engine, got %q", err.Error())
	}
}

func TestResolveOCIVirtualMachineSelectsRequestedEngine(t *testing.T) {
	vm, err := resolveOCIVirtualMachine("windows", "windows11", "", "amd64", "virtualbox")
	if err != nil {
		t.Fatalf("expected virtualbox target to resolve: %v", err)
	}
	if vm.VirtualizationEngine != alchemy_build.VirtualizationEngineVirtualBox {
		t.Fatalf("expected virtualbox engine, got %q", vm.VirtualizationEngine)
	}
}

func TestResolveOCIVirtualMachineRequiresOS(t *testing.T) {
	_, err := resolveOCIVirtualMachine("linux", "", "server", "amd64", "")
	if err == nil {
		t.Fatal("expected missing OS to fail")
	}
	if !strings.Contains(err.Error(), "--os") {
		t.Fatalf("expected error to mention --os, got %q", err.Error())
	}
}

func TestOCIRegistryOptionsReadsPasswordStdin(t *testing.T) {
	previousPassword := ociPassword
	previousPasswordStdin := ociPasswordStdin
	previousUsername := ociUsername
	t.Cleanup(func() {
		ociPassword = previousPassword
		ociPasswordStdin = previousPasswordStdin
		ociUsername = previousUsername
	})

	ociUsername = "user"
	ociPassword = ""
	ociPasswordStdin = true

	command := &cobra.Command{}
	command.SetIn(bytes.NewBufferString("secret\n"))

	options, err := ociRegistryOptions(command)
	if err != nil {
		t.Fatalf("expected password stdin to parse: %v", err)
	}
	if options.Username != "user" {
		t.Fatalf("expected username, got %q", options.Username)
	}
	if options.Password != "secret" {
		t.Fatalf("expected trimmed password, got %q", options.Password)
	}
}
