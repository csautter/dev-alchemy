package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	"github.com/spf13/cobra"
)

type installCommandSpec struct {
	executable string
	args       []string
	scriptPath string
}

var runInstallCommand = func(spec installCommandSpec) error {
	if _, err := os.Stat(spec.scriptPath); err != nil {
		return fmt.Errorf("install script not found at %q: %w", spec.scriptPath, err)
	}

	cmd := exec.Command(spec.executable, spec.args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed running install script %q with %s: %w", spec.scriptPath, spec.executable, err)
	}

	return nil
}

func installCommandForHost(hostOs alchemy_build.HostOsType, projectDir string) (installCommandSpec, error) {
	switch hostOs {
	case alchemy_build.HostOsDarwin:
		scriptPath := filepath.Join(projectDir, "scripts", "macos", "dev-alchemy-install-dependencies.sh")
		return installCommandSpec{
			executable: "bash",
			args:       []string{scriptPath},
			scriptPath: scriptPath,
		}, nil
	case alchemy_build.HostOsWindows:
		scriptPath := filepath.Join(projectDir, "scripts", "windows", "dev-alchemy-self-setup.ps1")
		return installCommandSpec{
			executable: "powershell",
			args:       []string{"-ExecutionPolicy", "Bypass", "-File", scriptPath},
			scriptPath: scriptPath,
		}, nil
	default:
		return installCommandSpec{}, fmt.Errorf("install is not supported for host OS: %s", hostOs)
	}
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install host dependencies for the current operating system",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hostOs := alchemy_build.GetCurrentHostOs()
		projectDir := alchemy_build.GetDirectoriesInstance().ProjectDir

		spec, err := installCommandForHost(hostOs, projectDir)
		if err != nil {
			return err
		}

		fmt.Printf("🔧 Running dependency installer for host OS: %s\n", hostOs)
		return runInstallCommand(spec)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
