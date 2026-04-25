package deploy

import (
	"path/filepath"
	"runtime"
	"strings"
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

func TestLinuxLibvirtHostArch(t *testing.T) {
	got, err := linuxLibvirtHostArch()
	if err != nil {
		t.Fatalf("expected supported host arch for test runtime, got error: %v", err)
	}

	switch runtime.GOARCH {
	case "amd64", "arm64":
		if got != runtime.GOARCH {
			t.Fatalf("expected host arch %q, got %q", runtime.GOARCH, got)
		}
	default:
		t.Fatalf("unexpected test runtime architecture %q", runtime.GOARCH)
	}
}

func TestEnsureLinuxLibvirtNativeArch(t *testing.T) {
	hostArch, err := linuxLibvirtHostArch()
	if err != nil {
		t.Fatalf("expected supported host arch for test runtime, got error: %v", err)
	}

	if err := ensureLinuxLibvirtNativeArch(alchemy_build.VirtualMachineConfig{Arch: hostArch}); err != nil {
		t.Fatalf("expected matching arch to succeed, got error: %v", err)
	}

	guestArch := "amd64"
	if hostArch == "amd64" {
		guestArch = "arm64"
	}

	err = ensureLinuxLibvirtNativeArch(alchemy_build.VirtualMachineConfig{Arch: guestArch})
	if err == nil {
		t.Fatal("expected mismatched host and guest architectures to fail")
	}
	if !strings.Contains(err.Error(), "native virtualization") {
		t.Fatalf("expected native virtualization error, got: %v", err)
	}
}

func TestLinuxLibvirtVirtInstallArgsIncludeNativeCPUAndSpiceAgentDevices(t *testing.T) {
	args := linuxLibvirtVirtInstallArgs(alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "desktop",
		Arch:       "amd64",
		Cpus:       4,
		MemoryMB:   4096,
	}, "qemu:///session", "/tmp/test.qcow2")

	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--cpu host-passthrough",
		"--graphics spice,clipboard.copypaste=on",
		"--video model.type=qxl",
		"--controller type=usb,model=qemu-xhci",
		"--input tablet,bus=usb",
		"--input keyboard,bus=usb",
		"--channel spicevmc,target.type=virtio,target.name=com.redhat.spice.0",
		"--channel unix,target.type=virtio,name=org.qemu.guest_agent.0",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected virt-install args to contain %q, got %q", want, joined)
		}
	}
}

func TestLinuxLibvirtVideoArg(t *testing.T) {
	if got := linuxLibvirtVideoArg(alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "desktop",
		Arch:       "amd64",
	}); got != "model.type=qxl" {
		t.Fatalf("expected amd64 desktop guests to use qxl video, got %q", got)
	}

	if got := linuxLibvirtVideoArg(alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "desktop",
		Arch:       "arm64",
	}); got != "model.type=virtio" {
		t.Fatalf("expected arm64 desktop guests to keep virtio video, got %q", got)
	}

	if got := linuxLibvirtVideoArg(alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "server",
		Arch:       "amd64",
	}); got != "model.type=virtio" {
		t.Fatalf("expected server guests to keep virtio video, got %q", got)
	}
}
