package build

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	runtimeassets "github.com/csautter/dev-alchemy"
)

const (
	embeddedProjectDirName      = "project"
	embeddedProjectManifestFile = ".embedded-assets.sha256"
)

func ensureProjectDir(workingDir string, appDataDir string) (string, error) {
	if projectDir := determineTopLevelDirWithGit(workingDir); projectDir != "" {
		return projectDir, nil
	}

	return ensureEmbeddedProjectDir(appDataDir)
}

func determineTopLevelDirWithGit(workingDir string) string {
	dir := workingDir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return ""
		}
	}

	for {
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

func ensureEmbeddedProjectDir(appDataDir string) (string, error) {
	projectDir := filepath.Join(appDataDir, embeddedProjectDirName)
	if err := os.MkdirAll(projectDir, managedDirPermission); err != nil {
		return "", fmt.Errorf("create embedded project directory: %w", err)
	}

	manifestHash, err := runtimeassets.ManifestHash()
	if err != nil {
		return "", fmt.Errorf("build embedded asset manifest: %w", err)
	}

	manifestPath := filepath.Join(projectDir, embeddedProjectManifestFile)
	if err := syncEmbeddedProject(runtimeassets.FS(), projectDir); err != nil {
		return "", fmt.Errorf("extract embedded project assets: %w", err)
	}

	if !manifestMatches(manifestPath, manifestHash) {
		if err := os.WriteFile(manifestPath, []byte(manifestHash+"\n"), 0o600); err != nil {
			return "", fmt.Errorf("write embedded asset manifest: %w", err)
		}
	}

	return projectDir, nil
}

func manifestMatches(path string, want string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(content)) == want
}

func syncEmbeddedProject(source fs.FS, destination string) error {
	return fs.WalkDir(source, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		targetPath := filepath.Join(destination, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		content, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		mode := embeddedFileMode(path, info.Mode())

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		if existing, err := os.ReadFile(targetPath); err == nil && bytes.Equal(existing, content) {
			return os.Chmod(targetPath, mode)
		}

		return os.WriteFile(targetPath, content, mode)
	})
}

func embeddedFileMode(path string, mode fs.FileMode) fs.FileMode {
	perm := mode.Perm()
	if strings.HasSuffix(path, ".sh") {
		return 0o755
	}
	if perm == 0 {
		return 0o644
	}
	if perm&0o111 != 0 {
		return perm
	}
	return 0o644
}
