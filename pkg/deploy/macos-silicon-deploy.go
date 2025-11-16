package deploy

import (
	"bufio"
	"context"
	"log"
	"os/exec"
	"path"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func RunUtmDeployOnMacOS(config alchemy_build.VirtualMachineConfig) {
	scriptPath := path.Join(alchemy_build.GetDirectoriesInstance().ProjectDir, "deployments/utm/create-utm-vm.sh")

	os := alchemy_build.GetVirtualMachineNameWithType(config)
	args := []string{"--arch", config.Arch, "--os", os}

	// Set a timeout for the script execution (adjust as needed)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", append([]string{scriptPath}, args...)...)
	cmd.Dir = alchemy_build.GetDirectoriesInstance().ProjectDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic("Failed to get stdout: " + err.Error())
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic("Failed to get stderr: " + err.Error())
	}

	if err := cmd.Start(); err != nil {
		panic("Failed to start command: " + err.Error())
	}

	done := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Printf("%s:%s stdout:  %s", os, config.Arch, scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("%s:%s stderr:  %s", os, config.Arch, scanner.Text())
		}
	}()

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			panic("Script failed: " + err.Error())
		}
		log.Printf("Script finished successfully.")
	case <-ctx.Done():
		// Kill the process if context is done (timeout or cancellation)
		_ = cmd.Process.Kill()
		panic("Script terminated due to timeout or interruption: " + ctx.Err().Error())
	}
}
