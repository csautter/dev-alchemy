package build

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-getter"
	"github.com/schollz/progressbar/v3"
)

// ProgressBarListener implements getter.ProgressListener to show a progress bar
type ProgressBarListener struct {
	bar *progressbar.ProgressBar
}

func (p *ProgressBarListener) TrackProgress(src string, current, total int64, r io.ReadCloser) io.ReadCloser {
	if p.bar == nil {
		p.bar = progressbar.DefaultBytes(total, fmt.Sprintf("downloading %s", src))
	}

	// Wrap the reader so the bar updates as data is read
	return &progressReader{
		reader: r,
		bar:    p.bar,
	}
}

type progressReader struct {
	reader io.ReadCloser
	bar    *progressbar.ProgressBar
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		_ = pr.bar.Add(n)
	}
	return n, err
}

func (pr *progressReader) Close() error {
	return pr.reader.Close()
}

type WebFileDependency struct {
	LocalPath        string
	Checksum         string
	Source           string
	RelatedVmConfigs []VirtualMachineConfig
}

const qemu_efi_version = "2025.05-1"

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
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, fmt.Sprintf("./vendor/qemu-efi-aarch64_%s_all.deb", qemu_efi_version)),
			Checksum:  "",
			Source:    fmt.Sprintf("http://deb.debian.org/debian/pool/main/e/edk2/qemu-efi-aarch64_%s_all.deb", qemu_efi_version),
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:   "windows11",
					Arch: "arm64",
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./vendor/windows/virtio-win.iso"),
			Checksum:  "",
			Source:    "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.266-1/virtio-win-0.1.266.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
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

	listener := &ProgressBarListener{}
	client := &getter.Client{
		Src:              src,
		Dst:              dep.LocalPath,
		Mode:             getter.ClientModeFile,
		ProgressListener: listener,
	}
	err := client.Get()
	if err != nil {
		log.Printf("Failed to download web file dependency from %s to %s: %v", dep.Source, dep.LocalPath, err)
		return err
	}
	listener.bar.Finish()
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
