package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

const (
	linuxLibvirtURIEnvVar              = "DEV_ALCHEMY_LIBVIRT_URI"
	linuxLibvirtImageDirEnvVar         = "DEV_ALCHEMY_LIBVIRT_IMAGE_DIR"
	linuxLibvirtDefaultURI             = "qemu:///session"
	linuxLibvirtSystemImageDir         = "/var/lib/libvirt/images/dev-alchemy"
	linuxLibvirtCreateTimeout          = 20 * time.Minute
	linuxLibvirtCommandTimeout         = 2 * time.Minute
	linuxLibvirtStopSettleTimeout      = 45 * time.Second
	linuxLibvirtStopPollInterval       = 2 * time.Second
	linuxLibvirtDiskCloneTimeout       = 30 * time.Minute
	linuxLibvirtManagedDomainDirectory = "libvirt/images"
)

var (
	runLinuxLibvirtCommandWithStreamingLogs = runCommandWithStreamingLogs
	runLinuxLibvirtCommandWithCombinedOut   = runCommandWithCombinedOutput
	linuxLibvirtStopTimeout                 = linuxLibvirtStopSettleTimeout
	linuxLibvirtStopPollEvery               = linuxLibvirtStopPollInterval
)

func isLinuxLibvirtTarget(config alchemy_build.VirtualMachineConfig) bool {
	return config.HostOs == alchemy_build.HostOsLinux &&
		config.VirtualizationEngine == alchemy_build.VirtualizationEngineQemu &&
		config.OS == "ubuntu" &&
		(config.UbuntuType == "server" || config.UbuntuType == "desktop") &&
		(config.Arch == "amd64" || config.Arch == "arm64")
}

func RunLinuxQemuDeployOnLinux(config alchemy_build.VirtualMachineConfig) error {
	if !isLinuxLibvirtTarget(config) {
		return fmt.Errorf("linux libvirt deploy is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}
	if err := ensureLinuxLibvirtNativeArch(config); err != nil {
		return err
	}

	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	artifactPath := linuxQemuArtifactPath(config)
	if _, err := os.Stat(artifactPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("required QCOW2 build artifact is missing at %q; run `alchemy build %s` first", artifactPath, startCommandArguments(config))
		}
		return fmt.Errorf("failed to inspect QCOW2 build artifact %q: %w", artifactPath, err)
	}

	imageDir := linuxLibvirtImageDir()
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return fmt.Errorf(
			"failed to create libvirt image directory %q: %w; override with %s or use a writable libvirt URI via %s",
			imageDir,
			err,
			linuxLibvirtImageDirEnvVar,
			linuxLibvirtURIEnvVar,
		)
	}

	diskPath := linuxLibvirtDiskPath(config)
	if _, err := os.Stat(diskPath); err == nil {
		return fmt.Errorf("managed libvirt disk %q already exists; destroy the existing VM first", diskPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect managed libvirt disk %q: %w", diskPath, err)
	}

	if err := runLinuxLibvirtCommandWithStreamingLogs(
		projectDir,
		linuxLibvirtDiskCloneTimeout,
		"qemu-img",
		[]string{"convert", "-p", "-f", "qcow2", "-O", "qcow2", artifactPath, diskPath},
		fmt.Sprintf("%s:%s:%s:qemu-img-convert", config.OS, config.UbuntuType, config.Arch),
	); err != nil {
		return fmt.Errorf("failed to clone QCOW2 artifact into managed libvirt disk %q: %w", diskPath, err)
	}

	xml, err := runLinuxLibvirtCommandWithCombinedOut(
		projectDir,
		linuxLibvirtCommandTimeout,
		"virt-install",
		linuxLibvirtVirtInstallArgs(config, linuxLibvirtURI(), diskPath),
	)
	if err != nil {
		_ = os.Remove(diskPath)
		return fmt.Errorf("failed to generate libvirt domain XML for %s: %w", linuxLibvirtDomainName(config), err)
	}
	if strings.TrimSpace(xml) == "" {
		_ = os.Remove(diskPath)
		return fmt.Errorf("virt-install generated empty libvirt domain XML for %s", linuxLibvirtDomainName(config))
	}

	xmlFile, err := os.CreateTemp("", "dev-alchemy-libvirt-*.xml")
	if err != nil {
		_ = os.Remove(diskPath)
		return fmt.Errorf("failed to create temporary libvirt XML file: %w", err)
	}
	xmlPath := xmlFile.Name()
	defer os.Remove(xmlPath)

	if _, err := xmlFile.WriteString(xml); err != nil {
		xmlFile.Close()
		_ = os.Remove(diskPath)
		return fmt.Errorf("failed to write libvirt domain XML to %q: %w", xmlPath, err)
	}
	if err := xmlFile.Close(); err != nil {
		_ = os.Remove(diskPath)
		return fmt.Errorf("failed to close libvirt domain XML file %q: %w", xmlPath, err)
	}

	if err := runLinuxLibvirtCommandWithStreamingLogs(
		projectDir,
		linuxLibvirtCreateTimeout,
		"virsh",
		[]string{"--connect", linuxLibvirtURI(), "define", xmlPath},
		fmt.Sprintf("%s:%s:%s:virsh-define", config.OS, config.UbuntuType, config.Arch),
	); err != nil {
		_ = os.Remove(diskPath)
		return fmt.Errorf("failed to define libvirt domain %q: %w", linuxLibvirtDomainName(config), err)
	}

	return nil
}

func RunLinuxQemuStartOnLinux(config alchemy_build.VirtualMachineConfig) error {
	if !isLinuxLibvirtTarget(config) {
		return fmt.Errorf("linux libvirt start is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}
	if err := ensureLinuxLibvirtNativeArch(config); err != nil {
		return err
	}

	state, err := inspectLinuxLibvirtStartTarget(config)
	if err != nil {
		return err
	}
	if !state.Exists {
		return fmt.Errorf("libvirt VM %q does not exist. Run `alchemy create %s` first", linuxLibvirtDomainName(config), startCommandArguments(config))
	}
	if state.Running {
		return nil
	}

	if err := runLinuxLibvirtCommandWithStreamingLogs(
		alchemy_build.GetDirectoriesInstance().ProjectDir,
		linuxLibvirtCommandTimeout,
		"virsh",
		[]string{"--connect", linuxLibvirtURI(), "start", linuxLibvirtDomainName(config)},
		fmt.Sprintf("%s:%s:%s:virsh-start", config.OS, config.UbuntuType, config.Arch),
	); err != nil {
		return fmt.Errorf("failed to start libvirt VM %q: %w", linuxLibvirtDomainName(config), err)
	}

	return nil
}

func RunLinuxQemuStopOnLinux(config alchemy_build.VirtualMachineConfig) error {
	if !isLinuxLibvirtTarget(config) {
		return fmt.Errorf("linux libvirt stop is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}

	state, err := inspectLinuxLibvirtStartTarget(config)
	if err != nil {
		return err
	}
	if !state.Exists || !state.Running {
		return nil
	}

	projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
	domainName := linuxLibvirtDomainName(config)
	if err := runLinuxLibvirtCommandWithStreamingLogs(
		projectDir,
		linuxLibvirtCommandTimeout,
		"virsh",
		[]string{"--connect", linuxLibvirtURI(), "shutdown", domainName},
		fmt.Sprintf("%s:%s:%s:virsh-shutdown", config.OS, config.UbuntuType, config.Arch),
	); err == nil {
		stopped, waitErr := waitForLinuxLibvirtStop(config, linuxLibvirtStopTimeout)
		if waitErr == nil && stopped {
			return nil
		}
	}

	if err := runLinuxLibvirtCommandWithStreamingLogs(
		projectDir,
		linuxLibvirtCommandTimeout,
		"virsh",
		[]string{"--connect", linuxLibvirtURI(), "destroy", domainName},
		fmt.Sprintf("%s:%s:%s:virsh-destroy", config.OS, config.UbuntuType, config.Arch),
	); err != nil {
		return fmt.Errorf("failed to force stop libvirt VM %q after graceful shutdown attempt: %w", domainName, err)
	}

	stopped, err := waitForLinuxLibvirtStop(config, linuxLibvirtStopTimeout)
	if err != nil {
		return fmt.Errorf("failed verifying stopped state for libvirt VM %q: %w", domainName, err)
	}
	if !stopped {
		return fmt.Errorf("libvirt VM %q is still running after force stop", domainName)
	}

	return nil
}

func RunLinuxQemuDestroyOnLinux(config alchemy_build.VirtualMachineConfig) error {
	if !isLinuxLibvirtTarget(config) {
		return fmt.Errorf("linux libvirt destroy is not implemented for OS=%s type=%s arch=%s", config.OS, config.UbuntuType, config.Arch)
	}

	state, err := inspectLinuxLibvirtStartTarget(config)
	if err != nil {
		return err
	}
	if state.Exists {
		if state.Running {
			if err := RunLinuxQemuStopOnLinux(config); err != nil {
				return err
			}
		}

		projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir
		domainName := linuxLibvirtDomainName(config)
		err = runLinuxLibvirtCommandWithStreamingLogs(
			projectDir,
			linuxLibvirtCommandTimeout,
			"virsh",
			[]string{"--connect", linuxLibvirtURI(), "undefine", domainName, "--nvram", "--managed-save", "--snapshots-metadata", "--checkpoints-metadata"},
			fmt.Sprintf("%s:%s:%s:virsh-undefine", config.OS, config.UbuntuType, config.Arch),
		)
		if err != nil {
			err = runLinuxLibvirtCommandWithStreamingLogs(
				projectDir,
				linuxLibvirtCommandTimeout,
				"virsh",
				[]string{"--connect", linuxLibvirtURI(), "undefine", domainName},
				fmt.Sprintf("%s:%s:%s:virsh-undefine", config.OS, config.UbuntuType, config.Arch),
			)
			if err != nil {
				return fmt.Errorf("failed to undefine libvirt VM %q: %w", domainName, err)
			}
		}
	}

	diskPath := linuxLibvirtDiskPath(config)
	if err := os.Remove(diskPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove managed libvirt disk %q: %w", diskPath, err)
	}

	return nil
}

func inspectLinuxLibvirtStartTarget(config alchemy_build.VirtualMachineConfig) (StartTargetState, error) {
	state, err := linuxLibvirtDomainState(config)
	if err != nil {
		return StartTargetState{}, err
	}
	if state == "missing" {
		return StartTargetState{State: "missing"}, nil
	}

	return StartTargetState{
		Exists:  true,
		Running: linuxLibvirtStateIndicatesRunning(state),
		State:   state,
	}, nil
}

func linuxLibvirtDomainState(config alchemy_build.VirtualMachineConfig) (string, error) {
	output, err := runLinuxLibvirtCommandWithCombinedOut(
		alchemy_build.GetDirectoriesInstance().ProjectDir,
		linuxLibvirtCommandTimeout,
		"virsh",
		[]string{"--connect", linuxLibvirtURI(), "domstate", linuxLibvirtDomainName(config)},
	)
	if err != nil {
		if linuxLibvirtOutputIndicatesMissingDomain(output) {
			return "missing", nil
		}
		return "", fmt.Errorf("failed to inspect libvirt VM %q state: %w; output: %s", linuxLibvirtDomainName(config), err, strings.TrimSpace(output))
	}

	normalized := normalizeLinuxLibvirtState(output)
	if normalized == "" {
		return "unknown", nil
	}
	return normalized, nil
}

func waitForLinuxLibvirtStop(config alchemy_build.VirtualMachineConfig, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state, err := inspectLinuxLibvirtStartTarget(config)
		if err != nil {
			return false, err
		}
		if !state.Exists || !state.Running {
			return true, nil
		}
		time.Sleep(linuxLibvirtStopPollEvery)
	}

	state, err := inspectLinuxLibvirtStartTarget(config)
	if err != nil {
		return false, err
	}
	return !state.Exists || !state.Running, nil
}

func linuxLibvirtDomainName(config alchemy_build.VirtualMachineConfig) string {
	return fmt.Sprintf("%s-%s-dev-alchemy", alchemy_build.GetVirtualMachineNameWithType(config), config.Arch)
}

func linuxQemuArtifactPath(config alchemy_build.VirtualMachineConfig) string {
	if len(config.ExpectedBuildArtifacts) > 0 && strings.TrimSpace(config.ExpectedBuildArtifacts[0]) != "" {
		return config.ExpectedBuildArtifacts[0]
	}

	return alchemy_build.GetDirectoriesInstance().CachePath(
		"ubuntu",
		fmt.Sprintf("qemu-ubuntu-%s-packer-%s.qcow2", config.UbuntuType, config.Arch),
	)
}

func linuxLibvirtDiskPath(config alchemy_build.VirtualMachineConfig) string {
	return filepath.Join(linuxLibvirtImageDir(), linuxLibvirtDomainName(config)+".qcow2")
}

func linuxLibvirtURI() string {
	if override := strings.TrimSpace(os.Getenv(linuxLibvirtURIEnvVar)); override != "" {
		return override
	}
	return linuxLibvirtDefaultURI
}

func linuxLibvirtImageDir() string {
	if override := strings.TrimSpace(os.Getenv(linuxLibvirtImageDirEnvVar)); override != "" {
		return filepath.Clean(override)
	}
	if linuxLibvirtUsesSystemConnection(linuxLibvirtURI()) {
		return linuxLibvirtSystemImageDir
	}
	return filepath.Join(alchemy_build.GetDirectoriesInstance().AppDataDir, linuxLibvirtManagedDomainDirectory)
}

func linuxLibvirtUsesSystemConnection(uri string) bool {
	return strings.EqualFold(strings.TrimSpace(uri), "qemu:///system")
}

func linuxLibvirtVirtInstallArgs(config alchemy_build.VirtualMachineConfig, uri string, diskPath string) []string {
	args := []string{
		"--connect", uri,
		"--name", linuxLibvirtDomainName(config),
		"--memory", fmt.Sprintf("%d", alchemy_build.GetVmMemoryMB(config)),
		"--vcpus", fmt.Sprintf("%d", alchemy_build.GetVmCpuCount(config)),
		"--cpu", "host-passthrough",
		"--import",
		"--disk", fmt.Sprintf("path=%s,format=qcow2,bus=virtio", diskPath),
		"--network", linuxLibvirtNetworkArg(uri),
		"--graphics", "spice,clipboard.copypaste=on",
		"--video", linuxLibvirtVideoArg(config),
		"--controller", "type=usb,model=qemu-xhci",
		"--input", "tablet,bus=usb",
		"--input", "keyboard,bus=usb",
		"--channel", "unix,target.type=virtio,name=org.qemu.guest_agent.0",
		"--channel", "spicevmc,target.type=virtio,target.name=com.redhat.spice.0",
		"--rng", "/dev/urandom",
		"--os-variant", "generic",
		"--noautoconsole",
		"--print-xml",
	}

	switch config.Arch {
	case "arm64":
		args = append(args,
			"--arch", "aarch64",
			"--machine", "virt",
			"--boot", "uefi,menu=on",
		)
	default:
		args = append(args,
			"--arch", "x86_64",
			"--machine", "q35",
			"--boot", "hd,menu=on",
		)
	}

	return args
}

func linuxLibvirtNetworkArg(uri string) string {
	if linuxLibvirtUsesSystemConnection(uri) {
		return "network=default,model=virtio"
	}
	return "user,model=virtio"
}

func linuxLibvirtVideoArg(config alchemy_build.VirtualMachineConfig) string {
	if config.OS == "ubuntu" && config.UbuntuType == "desktop" && config.Arch == "amd64" {
		return "model.type=qxl"
	}
	return "model.type=virtio"
}

func ensureLinuxLibvirtNativeArch(config alchemy_build.VirtualMachineConfig) error {
	hostArch, err := linuxLibvirtHostArch()
	if err != nil {
		return err
	}
	if config.Arch == hostArch {
		return nil
	}

	return fmt.Errorf(
		"linux libvirt deploy requires native virtualization on matching host and guest architectures; host=%s guest=%s. Build or run the %s image on a %s host instead",
		hostArch,
		config.Arch,
		config.Arch,
		config.Arch,
	)
}

func linuxLibvirtHostArch() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("linux libvirt deploy does not support host architecture %q", runtime.GOARCH)
	}
}

func linuxLibvirtStateIndicatesRunning(state string) bool {
	switch normalizeLinuxLibvirtState(state) {
	case "", "missing", "shut off", "shutoff", "no state":
		return false
	default:
		return true
	}
}

func normalizeLinuxLibvirtState(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}

func linuxLibvirtOutputIndicatesMissingDomain(output string) bool {
	normalized := strings.ToLower(output)
	return strings.Contains(normalized, "failed to get domain") ||
		strings.Contains(normalized, "domain not found") ||
		strings.Contains(normalized, "failed to get domain '") ||
		strings.Contains(normalized, "domain not found:")
}
