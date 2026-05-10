package oci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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

func TestParsePushReferenceRejectsDigest(t *testing.T) {
	_, err := parsePushReference("localhost:5000/dev-alchemy/artifact@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err == nil {
		t.Fatal("expected digest push reference to fail")
	}
	if !strings.Contains(err.Error(), "must use a tag") {
		t.Fatalf("expected tag-only error, got %q", err.Error())
	}
}

func TestValidateManifestLayersRejectsUnexpectedLayer(t *testing.T) {
	err := validateManifestLayers(
		ocispec.Manifest{
			Layers: []ocispec.Descriptor{
				{
					MediaType: MediaTypeQCOW2,
					Annotations: map[string]string{
						ocispec.AnnotationTitle: "unexpected.qcow2",
					},
				},
			},
		},
		[]ArtifactFile{{Name: "expected.qcow2", MediaType: MediaTypeQCOW2}},
	)
	if err == nil {
		t.Fatal("expected unexpected layer to fail validation")
	}
	if !strings.Contains(err.Error(), "unexpected layer") {
		t.Fatalf("expected unexpected layer error, got %q", err.Error())
	}
}

func TestPromotePulledArtifactsReplacesExistingArtifact(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	final := filepath.Join(root, "cache", "ubuntu", "artifact.qcow2")
	if err := os.MkdirAll(filepath.Join(staging, "ubuntu"), 0o700); err != nil {
		t.Fatalf("failed to create staging dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(final), 0o700); err != nil {
		t.Fatalf("failed to create final dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staging, "ubuntu", "artifact.qcow2"), []byte("new"), 0o600); err != nil {
		t.Fatalf("failed to write staged artifact: %v", err)
	}
	if err := os.WriteFile(final, []byte("old"), 0o600); err != nil {
		t.Fatalf("failed to write existing artifact: %v", err)
	}

	err := promotePulledArtifacts(staging, []ArtifactFile{{Name: "ubuntu/artifact.qcow2", Path: final}})
	if err != nil {
		t.Fatalf("expected promotion to succeed: %v", err)
	}

	content, err := os.ReadFile(final)
	if err != nil {
		t.Fatalf("failed to read promoted artifact: %v", err)
	}
	if string(content) != "new" {
		t.Fatalf("expected promoted content, got %q", string(content))
	}
	backups, err := filepath.Glob(final + ".dev-alchemy-oci-backup-*")
	if err != nil {
		t.Fatalf("failed to glob backup artifacts: %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("expected backup cleanup, got %v", backups)
	}
}

func TestPromotePulledArtifactsRollsBackEarlierReplacement(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	firstFinal := filepath.Join(root, "cache", "first.qcow2")
	secondFinal := filepath.Join(root, "cache", "second.qcow2")
	if err := os.MkdirAll(staging, 0o700); err != nil {
		t.Fatalf("failed to create staging dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(firstFinal), 0o700); err != nil {
		t.Fatalf("failed to create final dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staging, "first.qcow2"), []byte("new-first"), 0o600); err != nil {
		t.Fatalf("failed to write staged artifact: %v", err)
	}
	if err := os.WriteFile(firstFinal, []byte("old-first"), 0o600); err != nil {
		t.Fatalf("failed to write first artifact: %v", err)
	}
	if err := os.WriteFile(secondFinal, []byte("old-second"), 0o600); err != nil {
		t.Fatalf("failed to write second artifact: %v", err)
	}

	err := promotePulledArtifacts(staging, []ArtifactFile{
		{Name: "first.qcow2", Path: firstFinal},
		{Name: "missing.qcow2", Path: secondFinal},
	})
	if err == nil {
		t.Fatal("expected missing staged artifact to fail promotion")
	}

	firstContent, err := os.ReadFile(firstFinal)
	if err != nil {
		t.Fatalf("failed to read first artifact after rollback: %v", err)
	}
	if string(firstContent) != "old-first" {
		t.Fatalf("expected first artifact rollback, got %q", string(firstContent))
	}
	secondContent, err := os.ReadFile(secondFinal)
	if err != nil {
		t.Fatalf("failed to read second artifact after rollback: %v", err)
	}
	if string(secondContent) != "old-second" {
		t.Fatalf("expected second artifact to remain unchanged, got %q", string(secondContent))
	}
}
