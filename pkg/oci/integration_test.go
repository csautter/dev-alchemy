package oci

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestOCIRegistryPushPullIntegration(t *testing.T) {
	reference := os.Getenv("DEV_ALCHEMY_OCI_INTEGRATION_REF")
	if reference == "" {
		t.Skip("set DEV_ALCHEMY_OCI_INTEGRATION_REF to run against a live OCI registry, for example localhost:5000/dev-alchemy/test:oci")
	}

	root := t.TempDir()
	artifactPath := filepath.Join(root, "cache", "ubuntu", "integration.qcow2")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o700); err != nil {
		t.Fatalf("failed to create artifact dir: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("dev-alchemy-oci-integration"), 0o600); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	vm := alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "server",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsLinux,
		VirtualizationEngine: alchemy_build.VirtualizationEngineQemu,
		ExpectedBuildArtifacts: []string{
			artifactPath,
		},
	}
	options := RegistryOptions{
		PlainHTTP: parseBoolEnv("DEV_ALCHEMY_OCI_INTEGRATION_PLAIN_HTTP"),
	}

	ctx := context.Background()
	if _, err := Push(ctx, vm, reference, PushOptions{RegistryOptions: options}); err != nil {
		t.Fatalf("failed to push integration artifact: %v", err)
	}

	if err := os.Remove(artifactPath); err != nil {
		t.Fatalf("failed to remove local artifact before pull: %v", err)
	}
	if _, err := Pull(ctx, vm, reference, PullOptions{RegistryOptions: options}); err != nil {
		t.Fatalf("failed to pull integration artifact: %v", err)
	}

	content, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("failed to read pulled artifact: %v", err)
	}
	if string(content) != "dev-alchemy-oci-integration" {
		t.Fatalf("unexpected pulled artifact content %q", string(content))
	}
}

func parseBoolEnv(name string) bool {
	switch strings.ToLower(os.Getenv(name)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
