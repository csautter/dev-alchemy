package oci

import (
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestMediaTypeForPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "ubuntu.qcow2", want: MediaTypeQCOW2},
		{path: "windows.QCOW2", want: MediaTypeQCOW2},
		{path: "hyperv.box", want: MediaTypeVagrantBox},
		{path: "artifact.img", want: MediaTypeArtifact},
	}

	for _, tt := range tests {
		if got := MediaTypeForPath(tt.path); got != tt.want {
			t.Fatalf("MediaTypeForPath(%q): expected %q, got %q", tt.path, tt.want, got)
		}
	}
}

func TestArtifactFilesResolveExpectedArtifactsFromKnownVMConfig(t *testing.T) {
	files, err := ArtifactFiles(alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "server",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsLinux,
		VirtualizationEngine: alchemy_build.VirtualizationEngineQemu,
	})
	if err != nil {
		t.Fatalf("expected known VM config to resolve artifact files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one artifact file, got %d", len(files))
	}
	if !strings.HasSuffix(files[0].Name, "qemu-ubuntu-server-packer-amd64.qcow2") {
		t.Fatalf("expected qcow2 artifact name, got %q", files[0].Name)
	}
	if files[0].MediaType != MediaTypeQCOW2 {
		t.Fatalf("expected qcow2 media type, got %q", files[0].MediaType)
	}
}
