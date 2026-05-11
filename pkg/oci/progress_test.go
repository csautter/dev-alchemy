package oci

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
)

type recordingProgress struct {
	started int64
	added   int64
	done    atomic.Bool
	success atomic.Bool
}

func (p *recordingProgress) Start(totalBytes int64) {
	atomic.StoreInt64(&p.started, totalBytes)
}

func (p *recordingProgress) Add(bytes int64) {
	atomic.AddInt64(&p.added, bytes)
}

func (p *recordingProgress) Done(success bool) {
	p.success.Store(success)
	p.done.Store(true)
}

func TestCopyArtifactReportsByteProgress(t *testing.T) {
	ctx := context.Background()
	src, err := file.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create source store: %v", err)
	}
	defer src.Close()
	layerBytes := []byte("dev-alchemy-progress")
	layerPath := filepath.Join(t.TempDir(), "artifact.bin")
	if err := os.WriteFile(layerPath, layerBytes, 0o600); err != nil {
		t.Fatalf("failed to write test layer: %v", err)
	}
	layerDesc, err := src.Add(ctx, "artifact.bin", MediaTypeArtifact, layerPath)
	if err != nil {
		t.Fatalf("failed to add test layer: %v", err)
	}
	manifestDesc, err := oras.PackManifest(ctx, src, oras.PackManifestVersion1_1, ArtifactType, oras.PackManifestOptions{
		Layers: []ocispec.Descriptor{layerDesc},
	})
	if err != nil {
		t.Fatalf("failed to pack test manifest: %v", err)
	}
	if err := src.Tag(ctx, manifestDesc, "progress-test"); err != nil {
		t.Fatalf("failed to tag test manifest: %v", err)
	}

	dst, err := file.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create destination store: %v", err)
	}
	defer dst.Close()
	progress := &recordingProgress{}
	total := descriptorTotal(layerDesc, manifestDesc)
	got, err := copyArtifact(ctx, src, "progress-test", dst, "progress-test", total, progress)
	if err != nil {
		t.Fatalf("expected copy to succeed: %v", err)
	}
	if got.Digest != manifestDesc.Digest {
		t.Fatalf("expected copied manifest digest %s, got %s", manifestDesc.Digest, got.Digest)
	}
	if atomic.LoadInt64(&progress.started) != total {
		t.Fatalf("expected progress total %d, got %d", total, atomic.LoadInt64(&progress.started))
	}
	if atomic.LoadInt64(&progress.added) != total {
		t.Fatalf("expected %d progress bytes, got %d", total, atomic.LoadInt64(&progress.added))
	}
	if !progress.done.Load() || !progress.success.Load() {
		t.Fatal("expected progress to be marked done successfully")
	}
}

type failingTarget struct {
	oras.Target
}

func (t failingTarget) Push(context.Context, ocispec.Descriptor, io.Reader) error {
	return errors.New("push failed")
}

func TestCopyArtifactMarksProgressFailed(t *testing.T) {
	ctx := context.Background()
	src, err := file.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create source store: %v", err)
	}
	defer src.Close()
	layerBytes := []byte("dev-alchemy-progress")
	layerPath := filepath.Join(t.TempDir(), "artifact.bin")
	if err := os.WriteFile(layerPath, layerBytes, 0o600); err != nil {
		t.Fatalf("failed to write test layer: %v", err)
	}
	layerDesc, err := src.Add(ctx, "artifact.bin", MediaTypeArtifact, layerPath)
	if err != nil {
		t.Fatalf("failed to add test layer: %v", err)
	}
	if err := src.Tag(ctx, layerDesc, "progress-test"); err != nil {
		t.Fatalf("failed to tag test layer: %v", err)
	}

	dst, err := file.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create destination store: %v", err)
	}
	defer dst.Close()
	progress := &recordingProgress{}
	_, err = copyArtifact(ctx, src, "progress-test", failingTarget{Target: dst}, "progress-test", descriptorTotal(layerDesc), progress)
	if err == nil {
		t.Fatal("expected copy to fail")
	}
	if !progress.done.Load() {
		t.Fatal("expected progress to be marked done")
	}
	if progress.success.Load() {
		t.Fatal("expected failed progress completion")
	}
}
