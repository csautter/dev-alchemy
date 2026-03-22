package build

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildArtifactsExistReturnsTrueWhenAllArtifactsExist(t *testing.T) {
	tempDir := t.TempDir()
	artifactA := filepath.Join(tempDir, "artifact-a")
	artifactB := filepath.Join(tempDir, "artifact-b")

	if err := os.WriteFile(artifactA, []byte("a"), 0644); err != nil {
		t.Fatalf("failed to create artifact A: %v", err)
	}
	if err := os.WriteFile(artifactB, []byte("b"), 0644); err != nil {
		t.Fatalf("failed to create artifact B: %v", err)
	}

	exists, err := BuildArtifactsExist(VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifactA, artifactB},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatal("expected artifacts to exist")
	}
}

func TestBuildArtifactsExistReturnsFalseWhenArtifactIsMissing(t *testing.T) {
	tempDir := t.TempDir()
	artifactA := filepath.Join(tempDir, "artifact-a")
	missingArtifact := filepath.Join(tempDir, "missing-artifact")

	if err := os.WriteFile(artifactA, []byte("a"), 0644); err != nil {
		t.Fatalf("failed to create artifact A: %v", err)
	}

	exists, err := BuildArtifactsExist(VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifactA, missingArtifact},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatal("expected artifacts to be reported missing")
	}
}

func TestBuildArtifactsExistReturnsErrorWhenNoArtifactsAreDefined(t *testing.T) {
	_, err := BuildArtifactsExist(VirtualMachineConfig{
		OS:   "does-not-exist",
		Arch: "amd64",
	})
	if err == nil {
		t.Fatal("expected error when no build artifacts are defined")
	}
}

func TestBuildArtifactsExistQuietDoesNotLog(t *testing.T) {
	tempDir := t.TempDir()
	artifact := filepath.Join(tempDir, "artifact-a")

	if err := os.WriteFile(artifact, []byte("a"), 0644); err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	var buf bytes.Buffer
	originalWriter := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(originalWriter)

	exists, err := BuildArtifactsExistQuiet(VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifact},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatal("expected artifact to exist")
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no log output, got %q", buf.String())
	}
}

func TestPrepareBuildArtifactsForBuildSkipsWhenArtifactsExistAndNoCacheDisabled(t *testing.T) {
	tempDir := t.TempDir()
	artifact := filepath.Join(tempDir, "artifact-a")

	if err := os.WriteFile(artifact, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	skip, cleanup, err := prepareBuildArtifactsForBuild(VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifact},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !skip {
		t.Fatal("expected build to be skipped when artifact exists and no-cache is disabled")
	}

	cleanup(false)

	content, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatalf("failed to read artifact after cleanup: %v", err)
	}
	if string(content) != "existing" {
		t.Fatalf("expected artifact content to remain unchanged, got %q", string(content))
	}
}

func TestPrepareBuildArtifactsForBuildNoCacheRestoresOriginalArtifactOnFailure(t *testing.T) {
	tempDir := t.TempDir()
	artifact := filepath.Join(tempDir, "artifact-a")

	if err := os.WriteFile(artifact, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	skip, cleanup, err := prepareBuildArtifactsForBuild(VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifact},
		NoCache:                true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if skip {
		t.Fatal("expected no-cache build to proceed")
	}

	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Fatalf("expected original artifact to be moved aside before build, got err=%v", err)
	}

	if err := os.WriteFile(artifact, []byte("partial"), 0644); err != nil {
		t.Fatalf("failed to create replacement artifact: %v", err)
	}

	cleanup(false)

	content, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatalf("failed to read restored artifact: %v", err)
	}
	if string(content) != "existing" {
		t.Fatalf("expected original artifact to be restored, got %q", string(content))
	}

	backups, err := filepath.Glob(artifact + ".dev-alchemy-backup-*")
	if err != nil {
		t.Fatalf("failed to list backup artifacts: %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("expected backup artifacts to be removed, found %v", backups)
	}
}

func TestPrepareBuildArtifactsForBuildNoCacheRemovesBackupOnSuccess(t *testing.T) {
	tempDir := t.TempDir()
	artifact := filepath.Join(tempDir, "artifact-a")

	if err := os.WriteFile(artifact, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	skip, cleanup, err := prepareBuildArtifactsForBuild(VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifact},
		NoCache:                true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if skip {
		t.Fatal("expected no-cache build to proceed")
	}

	if err := os.WriteFile(artifact, []byte("rebuilt"), 0644); err != nil {
		t.Fatalf("failed to create rebuilt artifact: %v", err)
	}

	cleanup(true)

	content, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatalf("failed to read rebuilt artifact: %v", err)
	}
	if string(content) != "rebuilt" {
		t.Fatalf("expected rebuilt artifact to remain in place, got %q", string(content))
	}

	backups, err := filepath.Glob(artifact + ".dev-alchemy-backup-*")
	if err != nil {
		t.Fatalf("failed to list backup artifacts: %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("expected backup artifacts to be removed, found %v", backups)
	}
}

func TestRemoveBuildArtifactsForConfigUsesResolvedArtifacts(t *testing.T) {
	tempDir := t.TempDir()

	config := VirtualMachineConfig{
		OS:                   "windows11",
		Arch:                 "arm64",
		HostOs:               HostOsDarwin,
		VirtualizationEngine: VirtualizationEngineUtm,
	}
	config.ExpectedBuildArtifacts = nil

	dirs := GetDirectoriesInstance()
	originalCacheDir := dirs.CacheDir
	dirs.CacheDir = tempDir
	defer func() {
		dirs.CacheDir = originalCacheDir
	}()

	// Sanity check that the resolved artifact path uses the test cache directory.
	resolved, err := resolveExpectedBuildArtifacts(config)
	if err != nil {
		t.Fatalf("failed to resolve artifacts: %v", err)
	}
	if len(resolved) != 1 || !strings.HasPrefix(resolved[0], tempDir) {
		t.Fatalf("expected resolved artifact inside temp dir, got %v", resolved)
	}

	if err := os.MkdirAll(filepath.Dir(resolved[0]), 0755); err != nil {
		t.Fatalf("failed to create resolved artifact directory: %v", err)
	}

	if err := os.WriteFile(resolved[0], []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create resolved artifact: %v", err)
	}

	RemoveBuildArtifactsForConfig(config)

	if _, err := os.Stat(resolved[0]); !os.IsNotExist(err) {
		t.Fatalf("expected resolved artifact to be removed, got err=%v", err)
	}
}
