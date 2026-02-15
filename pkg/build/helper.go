package build

import (
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
