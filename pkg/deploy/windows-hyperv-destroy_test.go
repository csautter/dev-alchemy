package deploy

import (
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestIsHypervVagrantTarget(t *testing.T) {
	config := alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "server",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsWindows,
		VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
	}

	if !isHypervVagrantTarget(config) {
		t.Fatal("expected windows hyper-v ubuntu config to be supported")
	}
}

func TestVagrantMachineExistsInStatusOutput(t *testing.T) {
	output := "1737600000,default,state,running\n1737600000,default,provider-name,hyperv\n"

	if !vagrantMachineExistsInStatusOutput(output) {
		t.Fatal("expected running machine to be detected")
	}
}

func TestVagrantMachineExistsInStatusOutputTreatsNotCreatedAsAbsent(t *testing.T) {
	output := "1737600000,default,state,not_created\n"

	if vagrantMachineExistsInStatusOutput(output) {
		t.Fatal("expected not_created machine to be absent")
	}
}

func TestVagrantMachineExistsInStatusOutputTreatsAbortedAsPresent(t *testing.T) {
	output := "1737600000,default,state,aborted\n"

	if !vagrantMachineExistsInStatusOutput(output) {
		t.Fatal("expected aborted machine to still require destroy")
	}
}

func TestVagrantMachineStateFromStatusOutput(t *testing.T) {
	output := "1737600000,default,state,poweroff\n1737600000,default,provider-name,hyperv\n"

	if got := vagrantMachineStateFromStatusOutput(output); got != "poweroff" {
		t.Fatalf("expected poweroff state, got %q", got)
	}
}

func TestStartTargetStateFromVagrantStatusOutput(t *testing.T) {
	state := startTargetStateFromVagrantStatusOutput("1737600000,default,state,running\n")
	if !state.Exists || !state.Running || state.State != "running" {
		t.Fatalf("expected running start target state, got %#v", state)
	}

	missing := startTargetStateFromVagrantStatusOutput("1737600000,default,state,not_created\n")
	if missing.Exists || missing.Running || missing.State != "missing" {
		t.Fatalf("expected missing start target state, got %#v", missing)
	}
}

func TestHypervVagrantVMName(t *testing.T) {
	vmName, err := hypervVagrantVMName([]string{
		"VAGRANT_BOX_NAME=linux-ubuntu-server-packer",
		"VAGRANT_VM_NAME=linux-ubuntu-desktop-packer",
	})
	if err != nil {
		t.Fatalf("expected vm name to be resolved, got %v", err)
	}
	if vmName != "linux-ubuntu-desktop-packer" {
		t.Fatalf("expected desktop vm name, got %q", vmName)
	}
}

func TestHypervVagrantVMNameReturnsErrorWhenMissing(t *testing.T) {
	_, err := hypervVagrantVMName([]string{"VAGRANT_BOX_NAME=linux-ubuntu-server-packer"})
	if err == nil {
		t.Fatal("expected missing vm name to return an error")
	}
}

func TestHypervVMStateFromOutput(t *testing.T) {
	if got := hypervVMStateFromOutput("Running\n"); got != "running" {
		t.Fatalf("expected running state, got %q", got)
	}
	if got := hypervVMStateFromOutput("Off\n"); got != "off" {
		t.Fatalf("expected off state, got %q", got)
	}
	if got := hypervVMStateFromOutput("\n"); got != "missing" {
		t.Fatalf("expected blank output to be treated as missing, got %q", got)
	}
}

func TestHypervStartTargetStateFromVMState(t *testing.T) {
	state := hypervStartTargetStateFromVMState("running")
	if !state.Exists || !state.Running || state.State != "running" {
		t.Fatalf("expected running start target state, got %#v", state)
	}

	stopped := hypervStartTargetStateFromVMState("off")
	if !stopped.Exists || stopped.Running || stopped.State != "off" {
		t.Fatalf("expected stopped start target state, got %#v", stopped)
	}

	missing := hypervStartTargetStateFromVMState("missing")
	if missing.Exists || missing.Running || missing.State != "missing" {
		t.Fatalf("expected missing start target state, got %#v", missing)
	}
}

func TestVagrantBoxListIncludesMatchesExactNameAndProvider(t *testing.T) {
	output := "win11-packer (hyperv, 0)\nlinux-ubuntu-server-packer (hyperv, 0)\n"

	if !vagrantBoxListIncludes(output, "win11-packer", "hyperv") {
		t.Fatal("expected hyper-v box to be found")
	}
	if vagrantBoxListIncludes(output, "win11", "hyperv") {
		t.Fatal("did not expect substring box name match")
	}
	if vagrantBoxListIncludes(output, "win11-packer", "virtualbox") {
		t.Fatal("did not expect provider mismatch to match")
	}
}
