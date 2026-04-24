package deploy

import (
	"path/filepath"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestIsLinuxLibvirtTarget(t *testing.T) {
	if !isLinuxLibvirtTarget(alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "server",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsLinux,
		VirtualizationEngine: alchemy_build.VirtualizationEngineQemu,
	}) {
		t.Fatal("expected linux ubuntu qemu config to be a supported libvirt target")
	}

	if isLinuxLibvirtTarget(alchemy_build.VirtualMachineConfig{
		OS:                   "windows11",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsLinux,
		VirtualizationEngine: alchemy_build.VirtualizationEngineQemu,
	}) {
		t.Fatal("did not expect non-Ubuntu qemu config to be a libvirt target")
	}
}

func TestLinuxLibvirtDomainName(t *testing.T) {
	got := linuxLibvirtDomainName(alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "desktop",
		Arch:       "arm64",
	})

	if got != "ubuntu-desktop-arm64-dev-alchemy" {
		t.Fatalf("unexpected domain name %q", got)
	}
}

func TestLinuxQemuArtifactPathFallsBackToCache(t *testing.T) {
	dirs := alchemy_build.GetDirectoriesInstance()
	originalCacheDir := dirs.CacheDir
	dirs.CacheDir = t.TempDir()
	t.Cleanup(func() {
		dirs.CacheDir = originalCacheDir
	})

	got := linuxQemuArtifactPath(alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "server",
		Arch:       "amd64",
	})

	want := filepath.Join(dirs.CacheDir, "ubuntu", "qemu-ubuntu-server-packer-amd64.qcow2")
	if got != want {
		t.Fatalf("expected artifact path %q, got %q", want, got)
	}
}

func TestLinuxLibvirtImageDirDefaultsToManagedAppDataForSession(t *testing.T) {
	dirs := alchemy_build.GetDirectoriesInstance()
	originalAppDataDir := dirs.AppDataDir
	dirs.AppDataDir = t.TempDir()
	t.Setenv(linuxLibvirtURIEnvVar, "")
	t.Setenv(linuxLibvirtImageDirEnvVar, "")
	t.Cleanup(func() {
		dirs.AppDataDir = originalAppDataDir
	})

	got := linuxLibvirtImageDir()
	want := filepath.Join(dirs.AppDataDir, "libvirt", "images")
	if got != want {
		t.Fatalf("expected session image dir %q, got %q", want, got)
	}
}

func TestLinuxLibvirtImageDirUsesSystemDefaultWhenConfigured(t *testing.T) {
	t.Setenv(linuxLibvirtURIEnvVar, "qemu:///system")
	t.Setenv(linuxLibvirtImageDirEnvVar, "")

	if got := linuxLibvirtImageDir(); got != linuxLibvirtSystemImageDir {
		t.Fatalf("expected system image dir %q, got %q", linuxLibvirtSystemImageDir, got)
	}
}

func TestLinuxLibvirtNetworkArg(t *testing.T) {
	if got := linuxLibvirtNetworkArg("qemu:///system"); got != "network=default,model=virtio" {
		t.Fatalf("unexpected system network arg %q", got)
	}
	if got := linuxLibvirtNetworkArg("qemu:///session"); got != "user,model=virtio" {
		t.Fatalf("unexpected session network arg %q", got)
	}
}

func TestLinuxLibvirtStateIndicatesRunning(t *testing.T) {
	if linuxLibvirtStateIndicatesRunning("shut off") {
		t.Fatal("expected shut off VM to be treated as stopped")
	}
	if !linuxLibvirtStateIndicatesRunning("running") {
		t.Fatal("expected running VM to be treated as active")
	}
	if !linuxLibvirtStateIndicatesRunning("paused") {
		t.Fatal("expected paused VM to still be treated as active")
	}
}
