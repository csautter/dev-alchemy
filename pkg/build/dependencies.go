package build

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

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
	// BeforeHook is a function that is called before downloading the dependency. It can be used to modify the Source URL dynamically.
	BeforeHook func() (string, error)
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

func getWindows11DownloadUrl(arch string, args []string) (string, error) {
	var url_file string
	if arch == "amd64" {
		url_file = "win11_amd64_iso_url.txt"
	}
	if arch == "arm64" {
		url_file = "win11_arm64_iso_url.txt"
	}

	args = append(args, "--arch", arch, "--project-root", GetDirectoriesInstance().ProjectDir)

	RunProcessConfig := RunProcessConfig{
		ExecutablePath: "bash",
		WorkingDir:     filepath.Join(GetDirectoriesInstance().ProjectDir, "./scripts/macos"),
		Args:           append([]string{filepath.Join(GetDirectoriesInstance().ProjectDir, "./scripts/macos/playwright_win11_iso.sh")}, args...),
		Timeout:        20 * time.Minute,
	}

	ctx := RunExternalProcess(RunProcessConfig)
	ctxDone := ctx.Done()
	select {
	case <-ctxDone:
		// Process completed
	case <-time.After(RunProcessConfig.Timeout):
		return "", fmt.Errorf("timeout reached while getting Windows 11 download URL")
	}

	content, err := os.ReadFile(filepath.Join(GetDirectoriesInstance().ProjectDir, "./vendor/windows/"+url_file))
	if err != nil {
		return "", err
	}
	url := string(content)

	return url, nil
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
			LocalPath:  filepath.Join(GetDirectoriesInstance().ProjectDir, "./vendor/windows/win11_25h2_english_amd64.iso"),
			Checksum:   "",
			BeforeHook: func() (string, error) { return getWindows11DownloadUrl("amd64", nil) },
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:   "windows11",
					Arch: "amd64",
				},
			},
		},
		{
			LocalPath:  filepath.Join(GetDirectoriesInstance().ProjectDir, "./vendor/windows/win11_25h2_english_arm64.iso"),
			Checksum:   "",
			BeforeHook: func() (string, error) { return getWindows11DownloadUrl("arm64", nil) },
			RelatedVmConfigs: []VirtualMachineConfig{
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
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./vendor/linux/ubuntu-24.04.3-live-server-arm64.iso"),
			Checksum:  "sha256:2ee2163c9b901ff5926400e80759088ff3b879982a3956c02100495b489fd555",
			Source:    "https://cdimage.ubuntu.com/releases/24.04.3/release/ubuntu-24.04.3-live-server-arm64.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:         "ubuntu",
					UbuntuType: "server",
					Arch:       "arm64",
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./vendor/linux/ubuntu-24.04.3-live-server-amd64.iso"),
			Checksum:  "sha256:c3514bf0056180d09376462a7a1b4f213c1d6e8ea67fae5c25099c6fd3d8274b",
			Source:    "https://releases.ubuntu.com/24.04.3/ubuntu-24.04.3-live-server-amd64.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:         "ubuntu",
					UbuntuType: "server",
					Arch:       "amd64",
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./vendor/linux/ubuntu-24.04.3-live-server-arm64.iso"),
			Checksum:  "sha256:2ee2163c9b901ff5926400e80759088ff3b879982a3956c02100495b489fd555",
			Source:    "https://cdimage.ubuntu.com/releases/24.04.3/release/ubuntu-24.04.3-live-server-arm64.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:         "ubuntu",
					UbuntuType: "desktop",
					Arch:       "arm64",
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./vendor/linux/ubuntu-24.04.3-live-server-amd64.iso"),
			Checksum:  "sha256:c3514bf0056180d09376462a7a1b4f213c1d6e8ea67fae5c25099c6fd3d8274b",
			Source:    "https://releases.ubuntu.com/24.04.3/ubuntu-24.04.3-live-server-amd64.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:         "ubuntu",
					UbuntuType: "desktop",
					Arch:       "amd64",
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, fmt.Sprintf("./vendor/qemu-efi-aarch64_%s_all.deb", qemu_efi_version)),
			Checksum:  "",
			Source:    fmt.Sprintf("http://deb.debian.org/debian/pool/main/e/edk2/qemu-efi-aarch64_%s_all.deb", qemu_efi_version),
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:         "ubuntu",
					UbuntuType: "server",
					Arch:       "arm64",
				},
			},
		},
	}
}

func downloadWebFileDependency(dep WebFileDependency) error {
	if dep.BeforeHook != nil {
		newSource, err := dep.BeforeHook()
		if err != nil {
			return err
		}
		dep.Source = newSource
	}

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
