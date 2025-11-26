package build

import (
	"path"
	"strings"
)

type HostOsType string

const (
	HostOsLinux   HostOsType = "debian"
	HostOsWindows HostOsType = "windows"
	HostOsDarwin  HostOsType = "darwin"
)

type VirtualMachineConfig struct {
	OS                     string
	Arch                   string
	UbuntuType             string
	VncPort                int
	Slug                   string
	ExpectedBuildArtifacts []string
	HostOs                 HostOsType
}

func AvailableVirtualMachineConfigs() []VirtualMachineConfig {
	return []VirtualMachineConfig{
		{
			OS:         "ubuntu",
			Arch:       "arm64",
			UbuntuType: "server",
			VncPort:    5901,
			ExpectedBuildArtifacts: []string{
				path.Join(GetDirectoriesInstance().CacheDir, "ubuntu/qemu-ubuntu-server-packer-arm64.qcow2"),
			},
			HostOs: HostOsDarwin,
		},
		{
			OS:         "ubuntu",
			Arch:       "amd64",
			UbuntuType: "server",
			VncPort:    5902,
			ExpectedBuildArtifacts: []string{
				path.Join(GetDirectoriesInstance().CacheDir, "ubuntu/qemu-ubuntu-server-packer-amd64.qcow2"),
			},
			HostOs: HostOsDarwin,
		},
		{
			OS:         "ubuntu",
			Arch:       "arm64",
			UbuntuType: "desktop",
			VncPort:    5903,
			ExpectedBuildArtifacts: []string{
				path.Join(GetDirectoriesInstance().CacheDir, "ubuntu/qemu-ubuntu-desktop-packer-arm64.qcow2"),
			},
			HostOs: HostOsDarwin,
		},
		{
			OS:         "ubuntu",
			Arch:       "amd64",
			UbuntuType: "desktop",
			VncPort:    5904,
			ExpectedBuildArtifacts: []string{
				path.Join(GetDirectoriesInstance().CacheDir, "ubuntu/qemu-ubuntu-desktop-packer-amd64.qcow2"),
			},
			HostOs: HostOsDarwin,
		},
		{
			OS:      "windows11",
			Arch:    "arm64",
			VncPort: 5911,
			ExpectedBuildArtifacts: []string{
				path.Join(GetDirectoriesInstance().CacheDir, "windows11/qemu-windows11-arm64.qcow2"),
			},
			HostOs: HostOsDarwin,
		},
		{
			OS:      "windows11",
			Arch:    "amd64",
			VncPort: 5912,
			ExpectedBuildArtifacts: []string{
				path.Join(GetDirectoriesInstance().CacheDir, "windows11/qemu-windows11-amd64.qcow2"),
			},
			HostOs: HostOsDarwin,
		},
		// Host OS Windows builds
		{
			OS:      "windows11",
			Arch:    "amd64",
			VncPort: 5912,

			ExpectedBuildArtifacts: []string{
				path.Join(GetDirectoriesInstance().CacheDir, "windows11/hyperv-windows11-amd64.box"),
			},
			HostOs: HostOsWindows,
		},
	}
}

func GenerateVirtualMachineSlug(config *VirtualMachineConfig) string {
	if config.Slug != "" {
		return config.Slug
	}

	slug := strings.ToLower(config.OS)
	if config.UbuntuType != "" {
		slug += "-" + strings.ToLower(config.UbuntuType)
	}
	slug += "-" + strings.ToLower(config.Arch)
	config.Slug = slug
	return slug
}

func GetVirtualMachineNameWithType(config VirtualMachineConfig) string {
	switch config.OS {
	case "ubuntu":
		if config.UbuntuType != "" {
			return config.OS + "-" + config.UbuntuType
		}
		return config.OS
	case "windows11":
		return config.OS
	default:
		return "Unknown OS"
	}
}
