package oci

import (
	"os"
	"path/filepath"
	"testing"
)

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
