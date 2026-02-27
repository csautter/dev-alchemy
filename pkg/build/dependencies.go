package build

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// resolveDebianPackageURL fetches the current download URL for an architecture-independent
// Debian package by querying the official Packages index for the given suite.
// This avoids hardcoding version strings that change frequently and are purged quickly.
func resolveDebianPackageURL(suite, packageName string) (string, error) {
	packagesURL := fmt.Sprintf("https://deb.debian.org/debian/dists/%s/main/binary-all/Packages.gz", suite)
	log.Printf("Resolving latest Debian package URL for %s from %s", packageName, packagesURL)

	resp, err := http.Get(packagesURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Debian package index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status %d fetching %s", resp.StatusCode, packagesURL)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decompress Debian package index: %w", err)
	}
	defer gz.Close()

	scanner := bufio.NewScanner(gz)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var filename string
	inTarget := false

outer:
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "Package: "+packageName:
			inTarget = true
		case inTarget && strings.HasPrefix(line, "Filename: "):
			filename = strings.TrimPrefix(line, "Filename: ")
		case inTarget && filename != "" && line == "":
			break outer
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading Debian package index: %w", err)
	}
	if filename == "" {
		return "", fmt.Errorf("package %q not found in Debian suite %q (binary-all)", packageName, suite)
	}

	url := "https://deb.debian.org/debian/" + filename
	log.Printf("Resolved %s to %s", packageName, url)
	return url, nil
}

func DependencyReconciliation(vmconfig VirtualMachineConfig) {
	for _, dep := range getWebFileDependencies() {
		needsDownload := false
		for _, relatedConfig := range dep.RelatedVmConfigs {
			if string(relatedConfig.HostOs) == string(vmconfig.HostOs) && relatedConfig.OS == vmconfig.OS && relatedConfig.Arch == vmconfig.Arch && relatedConfig.UbuntuType == vmconfig.UbuntuType && string(relatedConfig.VirtualizationEngine) == string(vmconfig.VirtualizationEngine) {
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

// bootstrapPythonEnv ensures the Python virtual environment at workdir/.venv exists
// and has playwright and playwright-stealth installed, then installs the Chromium browser.
// pythonExe is the system Python executable used only when the venv does not yet exist.
func bootstrapPythonEnv(workdir, pythonExe string) error {
	venvDir := filepath.Join(workdir, ".venv")
	if _, err := os.Stat(venvDir); os.IsNotExist(err) {
		log.Printf("Creating Python virtual environment for Windows 11 download script")
		if _, err = RunCliCommand(workdir, pythonExe, []string{"-m", "venv", ".venv"}); err != nil {
			return fmt.Errorf("failed to create Python venv: %w", err)
		}
	} else if err != nil {
		return err
	} else {
		log.Printf("Python virtual environment for Windows 11 download script already exists")
	}

	pipPath := filepath.Join(workdir, ".venv", "Scripts", "pip.exe")
	if runtime.GOOS != "windows" {
		pipPath = filepath.Join(workdir, ".venv", "bin", "pip")
	}
	venvPython := filepath.Join(workdir, ".venv", "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		venvPython = filepath.Join(workdir, ".venv", "bin", "python3")
	}

	log.Printf("Installing required Python packages for Windows 11 download script")
	if _, err := RunCliCommand(workdir, venvPython, []string{"-c", "import playwright"}); err != nil {
		log.Printf("playwright not found, installing...")
		if _, err = RunCliCommand(workdir, pipPath, []string{"install", "playwright"}); err != nil {
			return fmt.Errorf("failed to install playwright: %w", err)
		}
	}
	if _, err := RunCliCommand(workdir, venvPython, []string{"-c", "import playwright_stealth"}); err != nil {
		log.Printf("playwright-stealth not found, installing...")
		if _, err = RunCliCommand(workdir, pipPath, []string{"install", "playwright-stealth"}); err != nil {
			return fmt.Errorf("failed to install playwright-stealth: %w", err)
		}
	}

	log.Printf("Installing Playwright browsers for Windows 11 download script")
	if _, err := RunCliCommand(workdir, venvPython, []string{"-m", "playwright", "install", "chromium"}); err != nil {
		return fmt.Errorf("failed to install Playwright Chromium browser: %w", err)
	}

	return nil
}

func getWindows11Download(arch string, savePath string, download bool) (string, error) {

	_, err := os.Stat(savePath)
	if err != nil && os.IsNotExist(err) {
		log.Printf("Windows 11 ISO not found at %s, will attempt to get download url", savePath)
	} else {
		log.Printf("File already exists at %s, skipping evaluation of download url", savePath)
		return "", nil
	}

	var url_file string
	if arch == "amd64" {
		url_file = "win11_amd64_iso_url.txt"
	}
	if arch == "arm64" {
		url_file = "win11_arm64_iso_url.txt"
	}

	// if running on windows
	var python_executable string
	if runtime.GOOS == "windows" {
		python_executable = "python"
	} else {
		python_executable = "python3"
	}

	workdir := filepath.Join(GetDirectoriesInstance().ProjectDir, "./scripts/macos")
	if err := bootstrapPythonEnv(workdir, python_executable); err != nil {
		return "", fmt.Errorf("dependency bootstrap failed: %w", err)
	}

	venvPython := filepath.Join(workdir, ".venv", "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		venvPython = filepath.Join(workdir, ".venv", "bin", "python3")
	}

	args := []string{"playwright_win11_iso.py", "--save-path", savePath}
	// if arch is arm64, add --arm flag
	if arch == "arm64" {
		args = append(args, "--arm")
	}
	if download {
		args = append(args, "--download")
	}
	config := RunProcessConfig{
		WorkingDir:     workdir,
		ExecutablePath: venvPython,
		Args:           args,
		Timeout:        10 * time.Minute,
	}
	if _, err := RunExternalProcess(config); err != nil {
		return "", fmt.Errorf("playwright script failed: %w", err)
	}

	if download {
		return "", nil
	}

	content, err := os.ReadFile(filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/windows/"+url_file))
	if err != nil {
		return "", err
	}
	url := string(content)

	return url, nil
}

func getWebFileDependencies() []WebFileDependency {
	return []WebFileDependency{
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/utm/utm-guest-tools-latest.iso"),
			Checksum:  "",
			Source:    "https://getutm.app/downloads/utm-guest-tools-latest.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:                   "windows11",
					Arch:                 "amd64",
					HostOs:               HostOsDarwin,
					VirtualizationEngine: VirtualizationEngineUtm,
				},
				{
					OS:                   "windows11",
					Arch:                 "arm64",
					HostOs:               HostOsDarwin,
					VirtualizationEngine: VirtualizationEngineUtm,
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/windows11/iso/win11_25h2_english_amd64.iso"),
			Checksum:  "",
			BeforeHook: func() (string, error) {
				return getWindows11Download("amd64", filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/windows11/iso/win11_25h2_english_amd64.iso"), false)
			},
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:                   "windows11",
					Arch:                 "amd64",
					HostOs:               HostOsDarwin,
					VirtualizationEngine: VirtualizationEngineUtm,
				},
				{
					OS:                   "windows11",
					Arch:                 "amd64",
					HostOs:               HostOsWindows,
					VirtualizationEngine: VirtualizationEngineUtm,
				},
				{
					OS:                   "windows11",
					Arch:                 "amd64",
					HostOs:               HostOsWindows,
					VirtualizationEngine: VirtualizationEngineHyperv,
				},
				{
					OS:                   "windows11",
					Arch:                 "amd64",
					HostOs:               HostOsWindows,
					VirtualizationEngine: VirtualizationEngineVirtualBox,
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/windows11/iso/win11_25h2_english_arm64.iso"),
			Checksum:  "",
			BeforeHook: func() (string, error) {
				return getWindows11Download("arm64", filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/windows11/iso/win11_25h2_english_arm64.iso"), false)
			},
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:                   "windows11",
					Arch:                 "arm64",
					HostOs:               HostOsDarwin,
					VirtualizationEngine: VirtualizationEngineUtm,
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/qemu-efi-aarch64_all.deb"),
			Checksum:  "",
			BeforeHook: func() (string, error) {
				return resolveDebianPackageURL("trixie", "qemu-efi-aarch64")
			},
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:                   "windows11",
					Arch:                 "arm64",
					HostOs:               HostOsDarwin,
					VirtualizationEngine: VirtualizationEngineUtm,
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/windows/virtio-win.iso"),
			Checksum:  "",
			Source:    "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.266-1/virtio-win-0.1.266.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:                   "windows11",
					Arch:                 "arm64",
					HostOs:               HostOsDarwin,
					VirtualizationEngine: VirtualizationEngineUtm,
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/linux/ubuntu-24.04.3-live-server-arm64.iso"),
			Checksum:  "sha256:2ee2163c9b901ff5926400e80759088ff3b879982a3956c02100495b489fd555",
			Source:    "https://cdimage.ubuntu.com/releases/24.04.3/release/ubuntu-24.04.3-live-server-arm64.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:                   "ubuntu",
					UbuntuType:           "server",
					Arch:                 "arm64",
					HostOs:               HostOsDarwin,
					VirtualizationEngine: VirtualizationEngineUtm,
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/linux/ubuntu-24.04.3-live-server-amd64.iso"),
			Checksum:  "sha256:c3514bf0056180d09376462a7a1b4f213c1d6e8ea67fae5c25099c6fd3d8274b",
			Source:    "https://releases.ubuntu.com/24.04.3/ubuntu-24.04.3-live-server-amd64.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:                   "ubuntu",
					UbuntuType:           "server",
					Arch:                 "amd64",
					HostOs:               HostOsDarwin,
					VirtualizationEngine: VirtualizationEngineUtm,
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/linux/ubuntu-24.04.3-live-server-arm64.iso"),
			Checksum:  "sha256:2ee2163c9b901ff5926400e80759088ff3b879982a3956c02100495b489fd555",
			Source:    "https://cdimage.ubuntu.com/releases/24.04.3/release/ubuntu-24.04.3-live-server-arm64.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:                   "ubuntu",
					UbuntuType:           "desktop",
					Arch:                 "arm64",
					HostOs:               HostOsDarwin,
					VirtualizationEngine: VirtualizationEngineUtm,
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/linux/ubuntu-24.04.3-live-server-amd64.iso"),
			Checksum:  "sha256:c3514bf0056180d09376462a7a1b4f213c1d6e8ea67fae5c25099c6fd3d8274b",
			Source:    "https://releases.ubuntu.com/24.04.3/ubuntu-24.04.3-live-server-amd64.iso",
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:                   "ubuntu",
					UbuntuType:           "desktop",
					Arch:                 "amd64",
					HostOs:               HostOsDarwin,
					VirtualizationEngine: VirtualizationEngineUtm,
				},
			},
		},
		{
			LocalPath: filepath.Join(GetDirectoriesInstance().ProjectDir, "./cache/qemu-efi-aarch64_all.deb"),
			Checksum:  "",
			BeforeHook: func() (string, error) {
				return resolveDebianPackageURL("trixie", "qemu-efi-aarch64")
			},
			RelatedVmConfigs: []VirtualMachineConfig{
				{
					OS:                   "ubuntu",
					UbuntuType:           "server",
					Arch:                 "arm64",
					HostOs:               HostOsDarwin,
					VirtualizationEngine: VirtualizationEngineUtm,
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
		// some before hooks may also download the file themselves, so if the new source is empty, we can assume the file has been downloaded and skip the download step
		if newSource == "" {
			log.Printf("BeforeHook for %s returned empty source, assuming file has been downloaded", dep.LocalPath)
			return nil
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
		// delete the file if it was partially downloaded
		_ = os.Remove(dep.LocalPath)
		log.Printf("Failed to download web file dependency from %s to %s: %v", dep.Source, dep.LocalPath, err)
		return err
	}
	if listener.bar != nil {
		listener.bar.Finish()
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
