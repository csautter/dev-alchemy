package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var ubuntuQemuTemplatePaths = []string{
	"build/packer/linux/ubuntu/linux-ubuntu-qemu.pkr.hcl",
}

func TestUbuntuPackerTemplatesQuoteShellLocalExportPaths(t *testing.T) {
	t.Parallel()

	for _, templatePath := range ubuntuQemuTemplatePaths {
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

	for _, templatePath := range ubuntuQemuTemplatePaths {
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

func TestQemuCloudInitConfiguresPersistentDHCPNetworking(t *testing.T) {
	t.Parallel()

	const explicitDHCPNetwork = `  network:
    version: 2
    ethernets:
      default:
        dhcp4: true
        dhcp6: true
        match:
          name: e*`

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
			if !strings.Contains(normalizeLineEndings(string(content)), explicitDHCPNetwork) {
				t.Fatalf("expected %q to configure persistent DHCP networking for QEMU interfaces", userDataPath)
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

	for _, templatePath := range ubuntuQemuTemplatePaths {
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

func TestQemuWrapperScriptsUseSharedTemplate(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		scriptPath string
		hostOS     string
	}{
		{
			scriptPath: "build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh",
			hostOS:     "darwin",
		},
		{
			scriptPath: "build/packer/linux/ubuntu/linux-ubuntu-qemu.sh",
			hostOS:     "linux",
		},
	} {
		tc := tc
		t.Run(filepath.Base(tc.scriptPath), func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(repoPath(t, tc.scriptPath))
			if err != nil {
				t.Fatalf("failed to read script %q: %v", tc.scriptPath, err)
			}

			got := string(content)
			if !strings.Contains(got, `packer_file="build/packer/linux/ubuntu/linux-ubuntu-qemu.pkr.hcl"`) {
				t.Fatalf("expected script %q to use the shared QEMU Packer template", tc.scriptPath)
			}
			if !strings.Contains(got, `-var "host_os=`+tc.hostOS+`"`) {
				t.Fatalf("expected script %q to pass host_os=%s", tc.scriptPath, tc.hostOS)
			}
			if !strings.Contains(got, `-var "host_arch=$host_arch"`) {
				t.Fatalf("expected script %q to pass the detected host architecture", tc.scriptPath)
			}
		})
	}
}

func TestMacOSQemuWrapperSupportsPackerStartOnlyProbe(t *testing.T) {
	t.Parallel()

	scriptPath := "build/packer/linux/ubuntu/linux-ubuntu-on-macos.sh"
	content, err := os.ReadFile(repoPath(t, scriptPath))
	if err != nil {
		t.Fatalf("failed to read script %q: %v", scriptPath, err)
	}

	got := string(content)
	for _, want := range []string{
		`DEV_ALCHEMY_PACKER_START_ONLY`,
		`--packer-start-only`,
		`run_packer_build_start_only`,
		`Using isolated cache directory for start-only Packer probe`,
		`mktemp -d "/tmp/da-pc.XXXXXX"`,
		`build_output_dir="$effective_cache_dir/o"`,
		`-var "cache_dir=$effective_cache_dir"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected macOS QEMU wrapper to contain %q", want)
		}
	}

	for _, oldLongPath := range []string{
		`dev-alchemy-packer-start-only-cache.XXXXXX`,
		`build_output_dir="$effective_cache_dir/qemu-out-ubuntu-${ubuntu_type}-${arch}"`,
		`.XXXXXX.log`,
	} {
		if strings.Contains(got, oldLongPath) {
			t.Fatalf("macOS QEMU start-only probe still contains long or non-portable path pattern %q", oldLongPath)
		}
	}
}

func TestMacOSWorkflowUsesStartOnlyProbeForUbuntuQemuBuilds(t *testing.T) {
	t.Parallel()

	workflowPath := ".github/workflows/test-build-macos.yml"
	content, err := os.ReadFile(repoPath(t, workflowPath))
	if err != nil {
		t.Fatalf("failed to read workflow %q: %v", workflowPath, err)
	}

	got := string(content)
	for _, testName := range []string{
		"TestBuildQemuUbuntuServerArm64OnMacos",
		"TestBuildQemuUbuntuServerAmd64OnMacos",
		"TestBuildQemuUbuntuDesktopArm64OnMacos",
		"TestBuildQemuUbuntuDesktopAmd64OnMacos",
	} {
		entry := workflowMatrixEntryForTest(t, got, testName)
		if !strings.Contains(entry, `packer_start_only: "true"`) {
			t.Fatalf("expected workflow matrix entry for %s to run as a start-only Packer probe", testName)
		}
	}

	if !strings.Contains(got, `DEV_ALCHEMY_PACKER_START_ONLY: ${{ matrix.packer_start_only }}`) {
		t.Fatal("expected workflow to pass packer_start_only through to the build step")
	}
	if !strings.Contains(got, `steps.packer_build.outcome == 'success' && matrix.packer_start_only != 'true'`) {
		t.Fatal("expected workflow to skip deploy smoke tests for start-only Packer probes")
	}
}

func TestMacOSWorkflowDefaultsToGitHubHostedRunnersWithTartOptIn(t *testing.T) {
	t.Parallel()

	workflowPath := ".github/workflows/test-build-macos.yml"
	content, err := os.ReadFile(repoPath(t, workflowPath))
	if err != nil {
		t.Fatalf("failed to read workflow %q: %v", workflowPath, err)
	}

	got := string(content)
	runnerExpression := `${{ vars.USE_TART_MACOS_RUNNERS == 'true' && matrix.tart_runs_on || matrix.runs_on }}`
	for _, want := range []string{
		`runs-on: ` + runnerExpression,
		`runner_os ` + runnerExpression,
		`USE_TART_MACOS_RUNNERS: ${{ vars.USE_TART_MACOS_RUNNERS == 'true' && 'true' || 'false' }}`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected macOS workflow to contain %q", want)
		}
	}

	for _, testName := range []string{
		"TestBuildQemuWindows11Arm64OnMacos",
		"TestBuildQemuWindows11Amd64OnMacos",
		"TestBuildQemuUbuntuServerArm64OnMacos",
		"TestBuildQemuUbuntuServerAmd64OnMacos",
		"TestBuildQemuUbuntuDesktopArm64OnMacos",
		"TestBuildQemuUbuntuDesktopAmd64OnMacos",
	} {
		entry := workflowMatrixEntryForTest(t, got, testName)
		if !strings.Contains(entry, "runs_on: macos-26") {
			t.Fatalf("expected workflow matrix entry for %s to default to the GitHub-hosted macos-26 runner", testName)
		}
		if !strings.Contains(entry, "tart_runs_on: macos-26-tart") {
			t.Fatalf("expected workflow matrix entry for %s to keep the Tart runner opt-in label", testName)
		}
	}
}

func workflowMatrixEntryForTest(t *testing.T, workflow string, testName string) string {
	t.Helper()

	testMarker := "go_test_name: " + testName
	start := strings.Index(workflow, testMarker)
	if start == -1 {
		t.Fatalf("failed to find workflow matrix entry for %s", testName)
	}

	entryStart := strings.LastIndex(workflow[:start], "\n          - ")
	if entryStart == -1 {
		t.Fatalf("failed to find start of workflow matrix entry for %s", testName)
	}
	entryStart++

	nextEntry := strings.Index(workflow[start:], "\n          - ")
	if nextEntry == -1 {
		return workflow[entryStart:]
	}
	return workflow[entryStart : start+nextEntry]
}

func normalizeLineEndings(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

func repoPath(t *testing.T, relPath string) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine test file path")
	}

	return filepath.Join(filepath.Dir(currentFile), "..", "..", relPath)
}
