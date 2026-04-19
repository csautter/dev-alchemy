//go:build linux
// +build linux

package build

import (
	"os"
	"testing"
)

func linuxQemuUbuntuServerConfig(arch string, vncPort int) VirtualMachineConfig {
	memoryMB := 4096
	if os.Getenv("GITHUB_ACTIONS") != "" {
		memoryMB = 0
	}

	return VirtualMachineConfig{
		OS:                   "ubuntu",
		Arch:                 arch,
		UbuntuType:           "server",
		VncPort:              vncPort,
		HostOs:               HostOsLinux,
		VirtualizationEngine: VirtualizationEngineQemu,
		Cpus:                 4,
		MemoryMB:             memoryMB,
		Headless:             true,
	}
}

func TestIntegrationDependencyReconciliationQemuUbuntuAmd64OnLinux(t *testing.T) {
	requireIntegrationTests(t)

	DependencyReconciliation(linuxQemuUbuntuServerConfig("amd64", 5922))
}

func TestIntegrationDependencyReconciliationQemuUbuntuArm64OnLinux(t *testing.T) {
	requireIntegrationTests(t)

	DependencyReconciliation(linuxQemuUbuntuServerConfig("arm64", 5921))
}

func TestBuildQemuUbuntuServerAmd64OnLinux(t *testing.T) {
	requireIntegrationTests(t)
	t.Parallel()

	if err := RunQemuUbuntuBuildOnLinux(linuxQemuUbuntuServerConfig("amd64", 5922)); err != nil {
		t.Fatalf("Failed to build QEMU Ubuntu Server Amd64 on Linux: %v", err)
	}
}

func TestBuildQemuUbuntuServerArm64OnLinux(t *testing.T) {
	requireIntegrationTests(t)
	t.Parallel()

	if err := RunQemuUbuntuBuildOnLinux(linuxQemuUbuntuServerConfig("arm64", 5921)); err != nil {
		t.Fatalf("Failed to build QEMU Ubuntu Server Arm64 on Linux: %v", err)
	}
}
