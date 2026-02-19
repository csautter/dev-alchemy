//go:build !windows
// +build !windows

package build

// getSystemTotalMemoryMB returns 0 on non-Windows platforms.
// Memory calculation will fall back to the minimum default.
func getSystemTotalMemoryMB() (uint64, error) {
	return 0, nil
}
