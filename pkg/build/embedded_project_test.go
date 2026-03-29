package build

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureProjectDir_PrefersGitCheckout(t *testing.T) {
	repoRoot := t.TempDir()
	workingDir := filepath.Join(repoRoot, "nested", "dir")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("failed to create fake .git directory: %v", err)
	}
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("failed to create working directory: %v", err)
	}

	projectDir, err := ensureProjectDir(workingDir, t.TempDir())
	if err != nil {
		t.Fatalf("ensureProjectDir returned error: %v", err)
	}
	if projectDir != repoRoot {
		t.Fatalf("expected repo root %q, got %q", repoRoot, projectDir)
	}
}

func TestEnsureProjectDir_FallsBackToEmbeddedProject(t *testing.T) {
	appDataDir := t.TempDir()

	projectDir, err := ensureProjectDir(t.TempDir(), appDataDir)
	if err != nil {
		t.Fatalf("ensureProjectDir returned error: %v", err)
	}

	wantProjectDir := filepath.Join(appDataDir, embeddedProjectDirName)
	if projectDir != wantProjectDir {
		t.Fatalf("expected embedded project dir %q, got %q", wantProjectDir, projectDir)
	}

	for _, relPath := range []string{
		"ansible.cfg",
		filepath.Join("playbooks", "setup.yml"),
		filepath.Join("scripts", "macos", "dev-alchemy-install-dependencies.sh"),
		filepath.Join("deployments", "utm", "create-utm-vm.sh"),
	} {
		if _, err := os.Stat(filepath.Join(projectDir, relPath)); err != nil {
			t.Fatalf("expected extracted asset %q to exist: %v", relPath, err)
		}
	}
}

func TestEnsureEmbeddedProjectDir_SkipsSyncWhenManifestMatches(t *testing.T) {
	appDataDir := t.TempDir()

	projectDir, err := ensureEmbeddedProjectDir(appDataDir)
	if err != nil {
		t.Fatalf("ensureEmbeddedProjectDir returned error: %v", err)
	}

	managedFile := filepath.Join(projectDir, "ansible.cfg")
	if err := os.WriteFile(managedFile, []byte("locally modified"), 0o644); err != nil {
		t.Fatalf("failed to modify extracted asset: %v", err)
	}

	if _, err := ensureEmbeddedProjectDir(appDataDir); err != nil {
		t.Fatalf("second ensureEmbeddedProjectDir returned error: %v", err)
	}

	content, err := os.ReadFile(managedFile)
	if err != nil {
		t.Fatalf("failed to read managed file after second ensure: %v", err)
	}
	if string(content) != "locally modified" {
		t.Fatalf("expected matching manifest to skip sync, got %q", string(content))
	}
}

func TestEnsureEmbeddedProjectDir_MarksShellScriptsExecutableOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX execute bits are not portable on Windows")
	}

	projectDir, err := ensureEmbeddedProjectDir(t.TempDir())
	if err != nil {
		t.Fatalf("ensureEmbeddedProjectDir returned error: %v", err)
	}

	info, err := os.Stat(filepath.Join(projectDir, "scripts", "macos", "dev-alchemy-install-dependencies.sh"))
	if err != nil {
		t.Fatalf("failed to stat extracted script: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("expected extracted shell script to be executable, got mode %o", info.Mode().Perm())
	}
}

func TestEnsureEmbeddedProjectDir_UsesManagedPermissionsForDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX directory permissions are not portable on Windows")
	}

	projectDir, err := ensureEmbeddedProjectDir(t.TempDir())
	if err != nil {
		t.Fatalf("ensureEmbeddedProjectDir returned error: %v", err)
	}

	info, err := os.Stat(filepath.Join(projectDir, "scripts", "macos"))
	if err != nil {
		t.Fatalf("failed to stat extracted directory: %v", err)
	}
	if got := info.Mode().Perm(); got != managedDirPermission {
		t.Fatalf("expected extracted directory permissions %o, got %o", managedDirPermission, got)
	}
}

func TestEnsureEmbeddedProjectDir_PrunesStaleEntriesWhenManifestChanges(t *testing.T) {
	appDataDir := t.TempDir()

	projectDir, err := ensureEmbeddedProjectDir(appDataDir)
	if err != nil {
		t.Fatalf("ensureEmbeddedProjectDir returned error: %v", err)
	}

	staleFile := filepath.Join(projectDir, "scripts", "macos", ".venv", "keep.txt")
	if err := os.MkdirAll(filepath.Dir(staleFile), 0o755); err != nil {
		t.Fatalf("failed to create stale directory: %v", err)
	}
	if err := os.WriteFile(staleFile, []byte("stale"), 0o644); err != nil {
		t.Fatalf("failed to create stale file: %v", err)
	}

	conflictingPath := filepath.Join(projectDir, "ansible.cfg")
	if err := os.Remove(conflictingPath); err != nil {
		t.Fatalf("failed to remove embedded file for conflict setup: %v", err)
	}
	if err := os.MkdirAll(conflictingPath, 0o755); err != nil {
		t.Fatalf("failed to create conflicting directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(conflictingPath, "nested.txt"), []byte("conflict"), 0o644); err != nil {
		t.Fatalf("failed to create nested conflict file: %v", err)
	}

	manifestPath := filepath.Join(projectDir, embeddedProjectManifestFile)
	if err := os.WriteFile(manifestPath, []byte("stale-manifest\n"), 0o600); err != nil {
		t.Fatalf("failed to write stale manifest: %v", err)
	}

	if _, err := ensureEmbeddedProjectDir(appDataDir); err != nil {
		t.Fatalf("second ensureEmbeddedProjectDir returned error: %v", err)
	}

	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Fatalf("expected stale file to be pruned, got err=%v", err)
	}

	info, err := os.Stat(conflictingPath)
	if err != nil {
		t.Fatalf("expected conflicting path to be restored as file: %v", err)
	}
	if info.IsDir() {
		t.Fatalf("expected conflicting path to be restored as file")
	}
}
