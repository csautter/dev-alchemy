//go:build linux
// +build linux

package build

import (
	"os"
	"testing"
)

func linuxQemuUbuntuConfig(arch string, ubuntuType string, vncPort int) VirtualMachineConfig {
	memoryMB := 4096
	if os.Getenv("GITHUB_ACTIONS") != "" {
		memoryMB = 0
	}

	return VirtualMachineConfig{
		OS:                   "ubuntu",
		Arch:                 arch,
		UbuntuType:           ubuntuType,
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

	DependencyReconciliation(linuxQemuUbuntuConfig("amd64", "server", 5922))
}

func TestIntegrationDependencyReconciliationQemuUbuntuArm64OnLinux(t *testing.T) {
	requireIntegrationTests(t)

	DependencyReconciliation(linuxQemuUbuntuConfig("arm64", "server", 5921))
}

func TestIntegrationDependencyReconciliationQemuUbuntuDesktopAmd64OnLinux(t *testing.T) {
	requireIntegrationTests(t)

	DependencyReconciliation(linuxQemuUbuntuConfig("amd64", "desktop", 5924))
}

func TestIntegrationDependencyReconciliationQemuUbuntuDesktopArm64OnLinux(t *testing.T) {
	requireIntegrationTests(t)

	DependencyReconciliation(linuxQemuUbuntuConfig("arm64", "desktop", 5923))
}

func TestBuildQemuUbuntuServerAmd64OnLinux(t *testing.T) {
	requireIntegrationTests(t)
	t.Parallel()

	if err := RunQemuUbuntuBuildOnLinux(linuxQemuUbuntuConfig("amd64", "server", 5922)); err != nil {
		t.Fatalf("Failed to build QEMU Ubuntu Server Amd64 on Linux: %v", err)
	}
}

func TestBuildQemuUbuntuServerArm64OnLinux(t *testing.T) {
	requireIntegrationTests(t)
	t.Parallel()

	if err := RunQemuUbuntuBuildOnLinux(linuxQemuUbuntuConfig("arm64", "server", 5921)); err != nil {
		t.Fatalf("Failed to build QEMU Ubuntu Server Arm64 on Linux: %v", err)
	}
}

func TestBuildQemuUbuntuDesktopAmd64OnLinux(t *testing.T) {
	requireIntegrationTests(t)
	t.Parallel()

	if err := RunQemuUbuntuBuildOnLinux(linuxQemuUbuntuConfig("amd64", "desktop", 5924)); err != nil {
		t.Fatalf("Failed to build QEMU Ubuntu Desktop Amd64 on Linux: %v", err)
	}
}

func TestBuildQemuUbuntuDesktopArm64OnLinux(t *testing.T) {
	requireIntegrationTests(t)
	t.Parallel()

	if err := RunQemuUbuntuBuildOnLinux(linuxQemuUbuntuConfig("arm64", "desktop", 5923)); err != nil {
		t.Fatalf("Failed to build QEMU Ubuntu Desktop Arm64 on Linux: %v", err)
	}
}
