package build

func RunHypervWindowsBuildOnWindows(config VirtualMachineConfig) error {
	packer_file := "build/packer/windows/windows11-on-windows-hyperv.pkr.hcl"

	RunCliCommand(GetDirectoriesInstance().ProjectDir, "packer", []string{"init", packer_file})
	args := []string{"build", "-var", "iso_url=./vendor/windows/win11_25h2_english_amd64.iso", packer_file}
	return RunBuildScript(config, "packer", args)
}
