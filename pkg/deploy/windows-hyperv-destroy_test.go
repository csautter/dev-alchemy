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
