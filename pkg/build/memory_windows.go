//go:build windows
// +build windows

package build

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modKernel32         = syscall.NewLazyDLL("kernel32.dll")
	procGlobalMemStatus = modKernel32.NewProc("GlobalMemoryStatusEx")
)

// memoryStatusEx mirrors the Windows MEMORYSTATUSEX structure.
// See: https://learn.microsoft.com/en-us/windows/win32/api/sysinfoapi/ns-sysinfoapi-memorystatusex
type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// getSystemTotalMemoryMB returns the total physical memory of the host in megabytes.
func getSystemTotalMemoryMB() (uint64, error) {
	var mem memoryStatusEx
	mem.dwLength = uint32(unsafe.Sizeof(mem))

	ret, _, err := procGlobalMemStatus.Call(uintptr(unsafe.Pointer(&mem)))
	if ret == 0 {
		return 0, fmt.Errorf("GlobalMemoryStatusEx failed: %w", err)
	}

	return mem.ullTotalPhys / (1024 * 1024), nil
}
