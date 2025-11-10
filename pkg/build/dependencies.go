package build

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	getter "github.com/hashicorp/go-getter"
)

type WebFileDependency struct {
	LocalPath        string
	Checksum         string
	Source           string
	RelatedVmConfigs []VirtualMachineConfig
}

func DependencyReconciliation(vmconfig VirtualMachineConfig) {
	for _, dep := range getWebFileDependencies() {
		needsDownload := false
		for _, relatedConfig := range dep.RelatedVmConfigs {
			if relatedConfig.OS == vmconfig.OS && relatedConfig.Arch == vmconfig.Arch && relatedConfig.UbuntuType == vmconfig.UbuntuType {
				if !checkIfWebFileDependencyExists(dep) {
					needsDownload = true
				}
			}
		}
		if needsDownload {
			err := downloadWebFileDependency(dep)
			if err != nil {
				log.Fatalf("Failed to download web file dependency: %v", err)
			}
		}
	}
}

func getWebFileDependencies() []WebFileDependency {
	return []WebFileDependency{
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./vendor/utm/utm-guest-tools-latest.iso"),
			Checksum:  "",
			Source:    "https://getutm.app/downloads/utm-guest-tools-latest.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:   "windows11",
					Arch: "amd64",
				},
				{
					OS:   "windows11",
					Arch: "arm64",
				},
			},
		},
	}
}

func downloadWebFileDependency(dep WebFileDependency) error {
	src := dep.Source
	if dep.Checksum != "" {
		src = src + "?checksum=" + dep.Checksum
	}

	err := getter.GetFile(dep.LocalPath, src)
	if err != nil {
		log.Printf("Failed to download web file dependency from %s to %s: %v", dep.Source, dep.LocalPath, err)
		return err
	}
	log.Printf("Successfully downloaded web file dependency from %s to %s", dep.Source, dep.LocalPath)
	return nil
}

func checkIfWebFileDependencyExists(dep WebFileDependency) bool {
	_, err := filepath.Abs(dep.LocalPath)
	if err != nil {
		return false
	}
	_, err = os.Stat(dep.LocalPath)
	if os.IsNotExist(err) {
		return false
	}

	// check sha256 checksums if dep.Checksum begins with "sha256:"
	if dep.Checksum != "" && len(dep.Checksum) > 7 && dep.Checksum[:7] == "sha256:" {
		file, err := os.Open(dep.LocalPath)
		if err != nil {
			return false
		}
		defer file.Close()

		hasher := sha256.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return false
		}
		computedChecksum := fmt.Sprintf("sha256:%x", hasher.Sum(nil))

		if computedChecksum != dep.Checksum {
			return false
		}
	}

	log.Printf("File exists and checksum matches for %s", dep.LocalPath)
	return err == nil
}
