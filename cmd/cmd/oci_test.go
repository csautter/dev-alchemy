package cmd

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestLocalOCIArtifactStateDetectsMissingPartialAndExistingArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	artifactA := filepath.Join(tempDir, "artifact-a.qcow2")
	artifactB := filepath.Join(tempDir, "artifact-b.qcow2")
	vm := alchemy_build.VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifactA, artifactB},
	}

	state, err := localOCIArtifactState(vm)
	if err != nil {
		t.Fatalf("expected missing artifact state to inspect cleanly: %v", err)
	}
	if state != "missing" {
		t.Fatalf("expected missing state, got %q", state)
	}

	if err := os.WriteFile(artifactA, []byte("a"), 0o600); err != nil {
		t.Fatalf("failed to write test artifact: %v", err)
	}
	state, err = localOCIArtifactState(vm)
	if err != nil {
		t.Fatalf("expected partial artifact state to inspect cleanly: %v", err)
	}
	if state != "partial" {
		t.Fatalf("expected partial state, got %q", state)
	}

	if err := os.WriteFile(artifactB, []byte("b"), 0o600); err != nil {
		t.Fatalf("failed to write test artifact: %v", err)
	}
	state, err = localOCIArtifactState(vm)
	if err != nil {
		t.Fatalf("expected existing artifact state to inspect cleanly: %v", err)
	}
	if state != "exists" {
		t.Fatalf("expected exists state, got %q", state)
	}
}

func TestOCIListRowsIncludePushAndPullReadiness(t *testing.T) {
	previousInspector := inspectOCIArtifactState
	t.Cleanup(func() {
		inspectOCIArtifactState = previousInspector
	})

	artifactStates := map[string]string{
		"windows11/":     "exists",
		"ubuntu/server":  "missing",
		"ubuntu/desktop": "partial",
	}
	inspectOCIArtifactState = func(vm alchemy_build.VirtualMachineConfig) (string, error) {
		return artifactStates[vm.OS+"/"+vm.UbuntuType], nil
	}

	vms := []alchemy_build.VirtualMachineConfig{
		{
			OS:                   "windows11",
			Arch:                 "amd64",
			HostOs:               alchemy_build.HostOsLinux,
			VirtualizationEngine: alchemy_build.VirtualizationEngineQemu,
		},
		{
			OS:                   "ubuntu",
			UbuntuType:           "server",
			Arch:                 "amd64",
			HostOs:               alchemy_build.HostOsLinux,
			VirtualizationEngine: alchemy_build.VirtualizationEngineQemu,
		},
		{
			OS:                   "ubuntu",
			UbuntuType:           "desktop",
			Arch:                 "amd64",
			HostOs:               alchemy_build.HostOsLinux,
			VirtualizationEngine: alchemy_build.VirtualizationEngineQemu,
		},
	}

	var pushOutput bytes.Buffer
	if err := printVirtualMachineCombinationTable(
		&pushOutput,
		"Available push combinations for host OS: debian",
		"No push combinations are available for host OS debian.",
		vms,
		[]string{"OS", "Type", "Arch", "Artifact", "Push"},
		pushListRow,
	); err != nil {
		t.Fatalf("expected push list table to print: %v", err)
	}
	push := pushOutput.String()
	for _, want := range []string{"Push", "ready to push", "build required", "incomplete"} {
		if !strings.Contains(push, want) {
			t.Fatalf("expected push list output to contain %q, got %q", want, push)
		}
	}

	var pullOutput bytes.Buffer
	if err := printVirtualMachineCombinationTable(
		&pullOutput,
		"Available pull combinations for host OS: debian",
		"No pull combinations are available for host OS debian.",
		vms,
		[]string{"OS", "Type", "Arch", "Artifact", "Pull"},
		pullListRow,
	); err != nil {
		t.Fatalf("expected pull list table to print: %v", err)
	}
	pull := pullOutput.String()
	for _, want := range []string{"Pull", "will replace", "ready to pull", "will replace partial"} {
		if !strings.Contains(pull, want) {
			t.Fatalf("expected pull list output to contain %q, got %q", want, pull)
		}
	}
}

func TestOCICommandsIncludeListSubcommands(t *testing.T) {
	for _, command := range []*cobra.Command{pushCmd, pullCmd} {
		found := false
		for _, child := range command.Commands() {
			if child.Name() == "list" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %s command to include list subcommand", command.Name())
		}
	}
}
