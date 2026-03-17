package build

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
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
