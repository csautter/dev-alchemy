package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUbuntuPackerTemplatesQuoteShellLocalExportPaths(t *testing.T) {
	t.Parallel()

	for _, templatePath := range []string{
		"build/packer/linux/ubuntu/linux-ubuntu-on-macos.pkr.hcl",
		"build/packer/linux/ubuntu/linux-ubuntu-qemu.pkr.hcl",
	} {
		templatePath := templatePath
		t.Run(filepath.Base(templatePath), func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(repoPath(t, templatePath))
			if err != nil {
				t.Fatalf("failed to read template %q: %v", templatePath, err)
			}

			got := string(content)
			wantQuotedMkdir := "\"mkdir -p \\\"${local.cache_directory}/ubuntu\\\"\""
			wantQuotedCp := "\"cp \\\"${local.output_directory}\\\"/linux-ubuntu-${var.ubuntu_type}-packer-* \\\"${local.cache_directory}/ubuntu/qemu-ubuntu-${var.ubuntu_type}-packer-${var.arch}.qcow2\\\"\""
			oldUnquotedCp := "\"cp ${local.output_directory}/linux-ubuntu-${var.ubuntu_type}-packer-* ${local.cache_directory}/ubuntu/qemu-ubuntu-${var.ubuntu_type}-packer-${var.arch}.qcow2\""

			if !strings.Contains(got, wantQuotedMkdir) {
				t.Fatalf("expected template %q to quote the export directory creation command", templatePath)
			}
			if !strings.Contains(got, wantQuotedCp) {
				t.Fatalf("expected template %q to quote the QCOW2 export command", templatePath)
			}
			if strings.Contains(got, oldUnquotedCp) {
				t.Fatalf("template %q still contains the unquoted QCOW2 export command", templatePath)
			}
		})
	}
}

func TestQemuAutoinstallUsesBootCommandWithoutSubiquityRestart(t *testing.T) {
	t.Parallel()

	userDataPath := "build/packer/linux/ubuntu/cloud-init/qemu-server/user-data"
	userData, err := os.ReadFile(repoPath(t, userDataPath))
	if err != nil {
		t.Fatalf("failed to read %q: %v", userDataPath, err)
	}

	gotUserData := string(userData)
	for _, oldHack := range []string{
		"runcmd:",
		"/proc/cmdline",
		"snap restart subiquity",
	} {
		if strings.Contains(gotUserData, oldHack) {
			t.Fatalf("qemu server user-data still contains Subiquity restart hack %q", oldHack)
		}
	}

	for _, templatePath := range []string{
		"build/packer/linux/ubuntu/linux-ubuntu-on-macos.pkr.hcl",
		"build/packer/linux/ubuntu/linux-ubuntu-qemu.pkr.hcl",
	} {
		templatePath := templatePath
		t.Run(filepath.Base(templatePath), func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(repoPath(t, templatePath))
			if err != nil {
				t.Fatalf("failed to read template %q: %v", templatePath, err)
			}
			if !strings.Contains(string(content), "autoinstall ds=nocloud") {
				t.Fatalf("expected template %q to pass autoinstall on the kernel command line", templatePath)
			}
		})
	}
}

func TestQemuCloudInitLeavesInstallerNetworkingToSubiquityDefaults(t *testing.T) {
	t.Parallel()

	for _, userDataPath := range []string{
		"build/packer/linux/ubuntu/cloud-init/qemu-server/user-data",
		"build/packer/linux/ubuntu/cloud-init/qemu-desktop/user-data",
	} {
		userDataPath := userDataPath
		t.Run(filepath.Base(filepath.Dir(userDataPath)), func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(repoPath(t, userDataPath))
			if err != nil {
				t.Fatalf("failed to read %q: %v", userDataPath, err)
			}
			if strings.Contains(string(content), "\n  network:\n") {
				t.Fatalf("expected %q to use Subiquity's default DHCP network config", userDataPath)
			}
		})
	}
}

func TestQemuCloudInitExtendsInstallerBusctlTimeout(t *testing.T) {
	t.Parallel()

	for _, userDataPath := range []string{
		"build/packer/linux/ubuntu/cloud-init/qemu-server/user-data",
		"build/packer/linux/ubuntu/cloud-init/qemu-desktop/user-data",
	} {
		userDataPath := userDataPath
		t.Run(filepath.Base(filepath.Dir(userDataPath)), func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(repoPath(t, userDataPath))
			if err != nil {
				t.Fatalf("failed to read %q: %v", userDataPath, err)
			}

			got := string(content)
			for _, want := range []string{
				"early-commands:",
				"/usr/bin/busctl.dev-alchemy-original --timeout=120",
			} {
				if !strings.Contains(got, want) {
					t.Fatalf("expected %q to contain %q", userDataPath, want)
				}
			}
		})
	}
}

func TestArm64QemuBootOrderPrefersInstalledDiskAfterInstall(t *testing.T) {
	t.Parallel()

	for _, templatePath := range []string{
		"build/packer/linux/ubuntu/linux-ubuntu-on-macos.pkr.hcl",
		"build/packer/linux/ubuntu/linux-ubuntu-qemu.pkr.hcl",
	} {
		templatePath := templatePath
		t.Run(filepath.Base(templatePath), func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(repoPath(t, templatePath))
			if err != nil {
				t.Fatalf("failed to read template %q: %v", templatePath, err)
			}

			got := string(content)
			if !strings.Contains(got, "drive=disk,serial=deadbeef,bootindex=0") {
				t.Fatalf("expected template %q to prefer the installed ARM64 disk once it becomes bootable", templatePath)
			}
			if !strings.Contains(got, "drive=cdrom,bootindex=1") {
				t.Fatalf("expected template %q to keep the ARM64 installer ISO as the blank-disk fallback", templatePath)
			}
			if !strings.Contains(got, "efi_boot") {
				t.Fatalf("expected template %q to enable Packer EFI mode for ARM64 so the qemu builder does not inject -boot", templatePath)
			}
			if !strings.Contains(got, "AAVMF_CODE.no-secboot.fd") {
				t.Fatalf("expected template %q to attach ARM64 AAVMF code firmware", templatePath)
			}
			if !strings.Contains(got, "file={{ .OutputDir }}/efivars.fd,if=pflash,unit=1,format=raw") {
				t.Fatalf("expected template %q to attach Packer's writable ARM64 efivars copy", templatePath)
			}
			if strings.Contains(got, "[\"-boot\",") {
				t.Fatalf("template %q uses unsupported ARM64 QEMU boot ordering through -boot", templatePath)
			}
		})
	}
}

func repoPath(t *testing.T, relPath string) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine test file path")
	}

	return filepath.Join(filepath.Dir(currentFile), "..", "..", relPath)
}
