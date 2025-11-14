package build

import (
	"testing"
)

func TestIntegrationDependencyReconciliation(t *testing.T) {
	tests := []VirtualMachineConfig{
		{
			OS:   "windows11",
			Arch: "amd64",
		},
		{
			OS:   "windows11",
			Arch: "arm64",
		},
		{
			OS:         "ubuntu",
			Arch:       "amd64",
			UbuntuType: "desktop",
		},
		{
			OS:         "ubuntu",
			Arch:       "arm64",
			UbuntuType: "desktop",
		},
		{
			OS:         "ubuntu",
			Arch:       "amd64",
			UbuntuType: "server",
		},
		{
			OS:         "ubuntu",
			Arch:       "arm64",
			UbuntuType: "server",
		},
	}

	for _, vmconfig := range tests {
		DependencyReconciliation(vmconfig)
	}
}
