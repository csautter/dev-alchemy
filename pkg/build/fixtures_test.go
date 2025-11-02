package build

import (
	"log"
	"os"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
    log.Println("Ensure macos-silicon tests only run on macOS Silicon")
	skipIfNotMacOSSilicon(&testing.T{})
    exitVal := m.Run()

    os.Exit(exitVal)
}

func skipIfNotMacOSSilicon(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("Tests only run on macOS Silicon (darwin/arm64)")
	}
}