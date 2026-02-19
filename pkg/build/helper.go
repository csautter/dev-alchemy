package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

func getVmCpuCountInt(config VirtualMachineConfig) int {
	desired_cpus := config.Cpus
	system_cpus := runtime.NumCPU()
	if desired_cpus > system_cpus {
		return system_cpus
	}
	return desired_cpus
}

func getVmCpuCountString(config VirtualMachineConfig) string {
	return strconv.Itoa(getVmCpuCountInt(config))
}

func getTempDiskPathForHypervBuild() string {
	// we can use the temp disk like (D:\) if available to speed up the build process
	// determine an env variable to check for the temp disk path, default to empty string if not set
	temp_disk_path := ""
	if tempPath := os.Getenv("PACKER_TEMP_PATH"); tempPath != "" {
		temp_disk_path = tempPath
	}
	return temp_disk_path
}

// createHypervTempDir ensures the temp directory for the Hyper-V packer build exists.
// It mirrors the same logic used in the packer locals block for temp_dir.
func createHypervTempDir(dirs *Directories) error {
	tempPath := getTempDiskPathForHypervBuild()
	if tempPath == "" {
		tempPath = filepath.Join(dirs.CacheDir, "windows11", "hyperv-temp")
	}
	return os.MkdirAll(tempPath, 0755)
}
