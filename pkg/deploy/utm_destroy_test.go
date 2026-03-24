package deploy

import (
	"path/filepath"
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestIsUtmDeployTarget(t *testing.T) {
	config := alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "server",
		Arch:                 "arm64",
		HostOs:               alchemy_build.HostOsDarwin,
		VirtualizationEngine: alchemy_build.VirtualizationEngineUtm,
	}

	if !isUtmDeployTarget(config) {
		t.Fatal("expected darwin utm ubuntu config to be supported")
	}
}

func TestUtmVirtualMachineNameIncludesTypeAndArch(t *testing.T) {
	config := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "desktop",
		Arch:       "amd64",
	}

	if got := utmVirtualMachineName(config); got != "ubuntu-desktop-amd64-dev-alchemy" {
		t.Fatalf("unexpected UTM VM name %q", got)
	}
}

func TestUtmVirtualMachinePathUsesUtmDocumentsBundle(t *testing.T) {
	config := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "server",
		Arch:       "arm64",
	}

	got, err := utmVirtualMachinePath(config)
	if err != nil {
		t.Fatalf("utmVirtualMachinePath returned error: %v", err)
	}

	wantSuffix := filepath.Join(
		"Library",
		"Containers",
		"com.utmapp.UTM",
		"Data",
		"Documents",
		"ubuntu-server-arm64-dev-alchemy.utm",
	)
	if !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("expected UTM VM path %q to end with %q", got, wantSuffix)
	}
}
