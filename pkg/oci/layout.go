package oci

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

type artifactLayout struct {
	root  string
	files []ArtifactFile
}

func ArtifactFiles(vm alchemy_build.VirtualMachineConfig) ([]ArtifactFile, error) {
	layout, err := resolveArtifactLayout(vm)
	if err != nil {
		return nil, err
	}
	return slices.Clone(layout.files), nil
}

func MediaTypeForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".qcow2":
		return MediaTypeQCOW2
	case ".box":
		return MediaTypeVagrantBox
	default:
		return MediaTypeArtifact
	}
}

func resolveArtifactLayout(vm alchemy_build.VirtualMachineConfig) (artifactLayout, error) {
	paths, err := expectedArtifactPaths(vm)
	if err != nil {
		return artifactLayout{}, err
	}
	root, err := artifactRoot(paths)
	if err != nil {
		return artifactLayout{}, err
	}

	files := make([]ArtifactFile, 0, len(paths))
	for _, artifactPath := range paths {
		absPath, err := filepath.Abs(filepath.Clean(artifactPath))
		if err != nil {
			return artifactLayout{}, fmt.Errorf("resolve artifact path %s: %w", artifactPath, err)
		}
		name, err := relativeArtifactName(root, absPath)
		if err != nil {
			return artifactLayout{}, err
		}
		files = append(files, ArtifactFile{
			Name:      name,
			Path:      absPath,
			MediaType: MediaTypeForPath(absPath),
		})
	}

	return artifactLayout{root: root, files: files}, nil
}

func expectedArtifactPaths(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
	if len(vm.ExpectedBuildArtifacts) > 0 {
		return slices.Clone(vm.ExpectedBuildArtifacts), nil
	}

	for _, candidate := range alchemy_build.AvailableVirtualMachineConfigs() {
		if candidate.HostOs == vm.HostOs &&
			candidate.OS == vm.OS &&
			candidate.UbuntuType == vm.UbuntuType &&
			candidate.Arch == vm.Arch &&
			candidate.VirtualizationEngine == vm.VirtualizationEngine &&
			len(candidate.ExpectedBuildArtifacts) > 0 {
			return slices.Clone(candidate.ExpectedBuildArtifacts), nil
		}
	}

	return nil, fmt.Errorf(
		"no OCI build artifacts defined for OS=%s type=%s arch=%s host_os=%s engine=%s",
		vm.OS,
		vm.UbuntuType,
		vm.Arch,
		vm.HostOs,
		vm.VirtualizationEngine,
	)
}

func artifactRoot(paths []string) (string, error) {
	cacheDir := alchemy_build.GetDirectoriesInstance().GetDirectories().CacheDir
	if cacheDir != "" && allPathsWithin(cacheDir, paths) {
		return filepath.Abs(filepath.Clean(cacheDir))
	}
	return commonArtifactParent(paths)
}

func allPathsWithin(root string, paths []string) bool {
	for _, path := range paths {
		if !pathWithin(root, path) {
			return false
		}
	}
	return true
}

func commonArtifactParent(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", errors.New("no artifact paths provided")
	}

	root, err := filepath.Abs(filepath.Dir(filepath.Clean(paths[0])))
	if err != nil {
		return "", err
	}
	for _, artifactPath := range paths[1:] {
		absPath, err := filepath.Abs(filepath.Clean(artifactPath))
		if err != nil {
			return "", err
		}
		for !pathWithin(root, absPath) {
			next := filepath.Dir(root)
			if next == root {
				return "", fmt.Errorf("could not determine common artifact root for %v", paths)
			}
			root = next
		}
	}
	return root, nil
}

func relativeArtifactName(root string, artifactPath string) (string, error) {
	rel, err := filepath.Rel(root, artifactPath)
	if err != nil {
		return "", fmt.Errorf("make artifact path relative to %s: %w", root, err)
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") || rel == ".." {
		return "", fmt.Errorf("artifact path %s is outside artifact root %s", artifactPath, root)
	}
	return rel, nil
}

func pathWithin(root string, path string) bool {
	absRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == "." || (!strings.HasPrefix(rel, "../") && rel != "..")
}
