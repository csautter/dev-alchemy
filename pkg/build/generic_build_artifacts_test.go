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

func TestPrepareBuildArtifactsForBuildRemovesIncompleteArtifactOnFailure(t *testing.T) {
	tempDir := t.TempDir()
	artifact := filepath.Join(tempDir, "artifact-a")

	skip, cleanup, err := prepareBuildArtifactsForBuild(VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifact},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if skip {
		t.Fatal("expected build to proceed when artifact does not exist")
	}

	if err := os.WriteFile(artifact, []byte("partial"), 0644); err != nil {
		t.Fatalf("failed to create partial artifact: %v", err)
	}

	cleanup(false)

	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Fatalf("expected incomplete artifact to be removed, got err=%v", err)
	}
}

func TestPrepareBuildArtifactsForBuildRestoresExistingArtifactsOnFailureWithoutNoCache(t *testing.T) {
	tempDir := t.TempDir()
	existingArtifact := filepath.Join(tempDir, "artifact-a")
	newArtifact := filepath.Join(tempDir, "artifact-b")

	if err := os.WriteFile(existingArtifact, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create existing artifact: %v", err)
	}

	skip, cleanup, err := prepareBuildArtifactsForBuild(VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{existingArtifact, newArtifact},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if skip {
		t.Fatal("expected build to proceed when at least one artifact is missing")
	}

	if _, err := os.Stat(existingArtifact); !os.IsNotExist(err) {
		t.Fatalf("expected existing artifact to be moved aside before build, got err=%v", err)
	}

	if err := os.WriteFile(existingArtifact, []byte("replacement"), 0644); err != nil {
		t.Fatalf("failed to create replacement artifact: %v", err)
	}
	if err := os.WriteFile(newArtifact, []byte("partial"), 0644); err != nil {
		t.Fatalf("failed to create new artifact: %v", err)
	}

	cleanup(false)

	content, err := os.ReadFile(existingArtifact)
	if err != nil {
		t.Fatalf("failed to read restored artifact: %v", err)
	}
	if string(content) != "existing" {
		t.Fatalf("expected original artifact to be restored, got %q", string(content))
	}

	if _, err := os.Stat(newArtifact); !os.IsNotExist(err) {
		t.Fatalf("expected incomplete new artifact to be removed, got err=%v", err)
	}

	backups, err := filepath.Glob(existingArtifact + ".dev-alchemy-backup-*")
	if err != nil {
		t.Fatalf("failed to list backup artifacts: %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("expected backup artifacts to be removed, found %v", backups)
	}
}

func TestPrepareBuildArtifactsForBuildKeepsRebuiltArtifactsOnSuccessWithoutNoCache(t *testing.T) {
	tempDir := t.TempDir()
	existingArtifact := filepath.Join(tempDir, "artifact-a")
	newArtifact := filepath.Join(tempDir, "artifact-b")

	if err := os.WriteFile(existingArtifact, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create existing artifact: %v", err)
	}

	skip, cleanup, err := prepareBuildArtifactsForBuild(VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{existingArtifact, newArtifact},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if skip {
		t.Fatal("expected build to proceed when at least one artifact is missing")
	}

	if err := os.WriteFile(existingArtifact, []byte("rebuilt"), 0644); err != nil {
		t.Fatalf("failed to create rebuilt artifact: %v", err)
	}
	if err := os.WriteFile(newArtifact, []byte("new"), 0644); err != nil {
		t.Fatalf("failed to create new artifact: %v", err)
	}

	cleanup(true)

	existingContent, err := os.ReadFile(existingArtifact)
	if err != nil {
		t.Fatalf("failed to read rebuilt existing artifact: %v", err)
	}
	if string(existingContent) != "rebuilt" {
		t.Fatalf("expected rebuilt artifact to remain in place, got %q", string(existingContent))
	}

	newContent, err := os.ReadFile(newArtifact)
	if err != nil {
		t.Fatalf("failed to read new artifact: %v", err)
	}
	if string(newContent) != "new" {
		t.Fatalf("expected new artifact to remain in place, got %q", string(newContent))
	}

	backups, err := filepath.Glob(existingArtifact + ".dev-alchemy-backup-*")
	if err != nil {
		t.Fatalf("failed to list backup artifacts: %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("expected backup artifacts to be removed, found %v", backups)
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

func TestPrepareBuildArtifactsForBuildPromotesStagedArtifactOnSuccess(t *testing.T) {
	tempDir := t.TempDir()
	artifact := filepath.Join(tempDir, "artifact-a.qcow2")
	stagedArtifact := filepath.Join(tempDir, "artifact-a.dev-alchemy-build-1.qcow2")

	if err := os.WriteFile(artifact, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	skip, cleanup, err := prepareBuildArtifactsForBuild(VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifact},
		StagedBuildArtifacts:   []string{stagedArtifact},
		NoCache:                true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if skip {
		t.Fatal("expected staged no-cache build to proceed")
	}

	content, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatalf("failed to read original artifact during build: %v", err)
	}
	if string(content) != "existing" {
		t.Fatalf("expected original artifact to remain available during build, got %q", string(content))
	}

	if err := os.WriteFile(stagedArtifact, []byte("rebuilt"), 0644); err != nil {
		t.Fatalf("failed to create staged artifact: %v", err)
	}

	if err := cleanup(true); err != nil {
		t.Fatalf("expected cleanup to promote staged artifact, got %v", err)
	}

	content, err = os.ReadFile(artifact)
	if err != nil {
		t.Fatalf("failed to read promoted artifact: %v", err)
	}
	if string(content) != "rebuilt" {
		t.Fatalf("expected staged artifact to replace original on success, got %q", string(content))
	}
	if _, err := os.Stat(stagedArtifact); !os.IsNotExist(err) {
		t.Fatalf("expected staged artifact to be moved into final location, got err=%v", err)
	}
}

func TestPrepareBuildArtifactsForBuildLeavesOriginalArtifactOnStagedFailure(t *testing.T) {
	tempDir := t.TempDir()
	artifact := filepath.Join(tempDir, "artifact-a.qcow2")
	stagedArtifact := filepath.Join(tempDir, "artifact-a.dev-alchemy-build-1.qcow2")

	if err := os.WriteFile(artifact, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	skip, cleanup, err := prepareBuildArtifactsForBuild(VirtualMachineConfig{
		ExpectedBuildArtifacts: []string{artifact},
		StagedBuildArtifacts:   []string{stagedArtifact},
		NoCache:                true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if skip {
		t.Fatal("expected staged no-cache build to proceed")
	}

	if err := os.WriteFile(stagedArtifact, []byte("partial"), 0644); err != nil {
		t.Fatalf("failed to create staged artifact: %v", err)
	}

	if err := cleanup(false); err != nil {
		t.Fatalf("expected cleanup to remove staged artifact, got %v", err)
	}

	content, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatalf("failed to read original artifact after failed build cleanup: %v", err)
	}
	if string(content) != "existing" {
		t.Fatalf("expected original artifact to remain after failed staged build, got %q", string(content))
	}
	if _, err := os.Stat(stagedArtifact); !os.IsNotExist(err) {
		t.Fatalf("expected failed staged artifact to be removed, got err=%v", err)
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
