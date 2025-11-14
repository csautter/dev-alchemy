package build

type VirtualMachineConfig struct {
	OS         string
	Arch       string
	UbuntuType string
	VncPort    int
	Slug       string
}

func AvailableVirtualMachineConfigs() []VirtualMachineConfig {
	return []VirtualMachineConfig{
		{
			OS:         "ubuntu",
			Arch:       "arm64",
			UbuntuType: "server",
			VncPort:    5901,
		},
		{
			OS:         "ubuntu",
			Arch:       "amd64",
			UbuntuType: "server",
			VncPort:    5902,
		},
		{
			OS:         "ubuntu",
			Arch:       "arm64",
			UbuntuType: "desktop",
			VncPort:    5903,
		},
		{
			OS:         "ubuntu",
			Arch:       "amd64",
			UbuntuType: "desktop",
			VncPort:    5904,
		},
		{
			OS:      "windows11",
			Arch:    "arm64",
			VncPort: 5911,
		},
		{
			OS:      "windows11",
			Arch:    "amd64",
			VncPort: 5912,
		},
	}
}
