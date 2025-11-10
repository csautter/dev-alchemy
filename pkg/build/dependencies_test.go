package build

import (
	"testing"
)

func TestIntegrationDependencyReconciliation(t *testing.T) {
	vmconfig := VirtualMachineConfig{
		OS:   "windows11",
		Arch: "amd64",
	}

	DependencyReconciliation(vmconfig)
}
