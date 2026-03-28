package deploy

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestIsHypervVagrantTarget(t *testing.T) {
	config := alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "server",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsWindows,
		VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
	}

	if !isHypervVagrantTarget(config) {
		t.Fatal("expected windows hyper-v ubuntu config to be supported")
	}
}

func TestVagrantMachineExistsInStatusOutput(t *testing.T) {
	output := "1737600000,default,state,running\n1737600000,default,provider-name,hyperv\n"

	if !vagrantMachineExistsInStatusOutput(output) {
		t.Fatal("expected running machine to be detected")
	}
}

func TestVagrantMachineExistsInStatusOutputTreatsNotCreatedAsAbsent(t *testing.T) {
	output := "1737600000,default,state,not_created\n"

	if vagrantMachineExistsInStatusOutput(output) {
		t.Fatal("expected not_created machine to be absent")
	}
}

func TestVagrantMachineExistsInStatusOutputTreatsAbortedAsPresent(t *testing.T) {
	output := "1737600000,default,state,aborted\n"

	if !vagrantMachineExistsInStatusOutput(output) {
		t.Fatal("expected aborted machine to still require destroy")
	}
}

func TestVagrantMachineStateFromStatusOutput(t *testing.T) {
	output := "1737600000,default,state,poweroff\n1737600000,default,provider-name,hyperv\n"

	if got := vagrantMachineStateFromStatusOutput(output); got != "poweroff" {
		t.Fatalf("expected poweroff state, got %q", got)
	}
}

func TestStartTargetStateFromVagrantStatusOutput(t *testing.T) {
	state := startTargetStateFromVagrantStatusOutput("1737600000,default,state,running\n")
	if !state.Exists || !state.Running || state.State != "running" {
		t.Fatalf("expected running start target state, got %#v", state)
	}

	missing := startTargetStateFromVagrantStatusOutput("1737600000,default,state,not_created\n")
	if missing.Exists || missing.Running || missing.State != "missing" {
		t.Fatalf("expected missing start target state, got %#v", missing)
	}
}

func TestHypervVagrantVMName(t *testing.T) {
	vmName, err := hypervVagrantVMName([]string{
		"VAGRANT_BOX_NAME=linux-ubuntu-server-packer",
		"VAGRANT_VM_NAME=linux-ubuntu-desktop-packer",
	})
	if err != nil {
		t.Fatalf("expected vm name to be resolved, got %v", err)
	}
	if vmName != "linux-ubuntu-desktop-packer" {
		t.Fatalf("expected desktop vm name, got %q", vmName)
	}
}

func TestHypervVagrantVMNameReturnsErrorWhenMissing(t *testing.T) {
	_, err := hypervVagrantVMName([]string{"VAGRANT_BOX_NAME=linux-ubuntu-server-packer"})
	if err == nil {
		t.Fatal("expected missing vm name to return an error")
	}
}

func TestHypervVMStateFromOutput(t *testing.T) {
	if got := hypervVMStateFromOutput("Running\n"); got != "running" {
		t.Fatalf("expected running state, got %q", got)
	}
	if got := hypervVMStateFromOutput("Off\n"); got != "off" {
		t.Fatalf("expected off state, got %q", got)
	}
	if got := hypervVMStateFromOutput("\n"); got != "missing" {
		t.Fatalf("expected blank output to be treated as missing, got %q", got)
	}
}

func TestHypervStartTargetStateFromVMState(t *testing.T) {
	state := hypervStartTargetStateFromVMState("running")
	if !state.Exists || !state.Running || state.State != "running" {
		t.Fatalf("expected running start target state, got %#v", state)
	}

	stopped := hypervStartTargetStateFromVMState("off")
	if !stopped.Exists || stopped.Running || stopped.State != "off" {
		t.Fatalf("expected stopped start target state, got %#v", stopped)
	}

	missing := hypervStartTargetStateFromVMState("missing")
	if missing.Exists || missing.Running || missing.State != "missing" {
		t.Fatalf("expected missing start target state, got %#v", missing)
	}
}

func TestRunHypervVagrantStopOnWindows_TreatsGracefulHaltErrorAsSuccessOnceStopped(t *testing.T) {
	restore := stubHypervStopDependencies(t)
	defer restore()
	vagrantRoot := setHypervTestVagrantRoot(t)

	commands := make([][]string, 0, 1)
	runHypervVagrantCommandWithEnv = func(_ string, _ time.Duration, executable string, args []string, env []string, _ string) error {
		if executable != "vagrant" {
			t.Fatalf("expected vagrant executable, got %q", executable)
		}
		assertEnvContainsEntry(t, env, "VAGRANT_BOX_NAME=linux-ubuntu-server-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_VM_NAME=linux-ubuntu-server-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_DOTFILE_PATH="+filepath.Join(vagrantRoot, "linux-ubuntu-server-packer"))
		commands = append(commands, append([]string(nil), args...))
		return fmt.Errorf("command failed (vagrant [halt]): exit status 1")
	}

	inspectCalls := 0
	inspectHypervVagrantStopTarget = func(alchemy_build.VirtualMachineConfig) (StartTargetState, error) {
		inspectCalls++
		switch inspectCalls {
		case 1:
			return StartTargetState{Exists: true, Running: true, State: "running"}, nil
		case 2:
			return StartTargetState{Exists: true, Running: false, State: "off"}, nil
		default:
			t.Fatalf("unexpected inspect call %d", inspectCalls)
			return StartTargetState{}, nil
		}
	}

	config := alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "server",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsWindows,
		VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
	}

	if err := RunHypervVagrantStopOnWindows(config); err != nil {
		t.Fatalf("expected graceful halt error to be ignored once stopped, got %v", err)
	}
	if len(commands) != 1 {
		t.Fatalf("expected one halt attempt, got %d", len(commands))
	}
	if got := commands[0]; len(got) != 1 || got[0] != "halt" {
		t.Fatalf("expected graceful halt command, got %#v", got)
	}
}

func TestRunHypervVagrantStopOnWindows_FallsBackToForcedHalt(t *testing.T) {
	restore := stubHypervStopDependencies(t)
	defer restore()
	vagrantRoot := setHypervTestVagrantRoot(t)

	commands := make([][]string, 0, 2)
	runHypervVagrantCommandWithEnv = func(_ string, _ time.Duration, executable string, args []string, env []string, _ string) error {
		if executable != "vagrant" {
			t.Fatalf("expected vagrant executable, got %q", executable)
		}
		assertEnvContainsEntry(t, env, "VAGRANT_BOX_NAME=linux-ubuntu-desktop-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_VM_NAME=linux-ubuntu-desktop-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_DOTFILE_PATH="+filepath.Join(vagrantRoot, "linux-ubuntu-desktop-packer"))
		commands = append(commands, append([]string(nil), args...))
		if len(args) == 1 && args[0] == "halt" {
			return fmt.Errorf("command failed (vagrant [halt]): exit status 1")
		}
		return nil
	}

	inspectCalls := 0
	inspectHypervVagrantStopTarget = func(alchemy_build.VirtualMachineConfig) (StartTargetState, error) {
		inspectCalls++
		switch inspectCalls {
		case 1:
			return StartTargetState{Exists: true, Running: true, State: "running"}, nil
		case 2:
			return StartTargetState{Exists: true, Running: true, State: "running"}, nil
		case 3:
			return StartTargetState{Exists: true, Running: false, State: "off"}, nil
		default:
			t.Fatalf("unexpected inspect call %d", inspectCalls)
			return StartTargetState{}, nil
		}
	}
	hypervVagrantStopSettleTimeout = 0

	config := alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "desktop",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsWindows,
		VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
	}

	if err := RunHypervVagrantStopOnWindows(config); err != nil {
		t.Fatalf("expected forced halt fallback to succeed, got %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("expected graceful and forced halt attempts, got %d", len(commands))
	}
	if got := commands[0]; len(got) != 1 || got[0] != "halt" {
		t.Fatalf("expected first command to be graceful halt, got %#v", got)
	}
	if got := commands[1]; len(got) != 2 || got[0] != "halt" || got[1] != "--force" {
		t.Fatalf("expected second command to be forced halt, got %#v", got)
	}
}

func TestRunHypervVagrantStopOnWindows_ReturnsErrorWhenVMStillRunningAfterForcedHalt(t *testing.T) {
	restore := stubHypervStopDependencies(t)
	defer restore()

	runHypervVagrantCommandWithEnv = func(_ string, _ time.Duration, _ string, args []string, _ []string, _ string) error {
		if len(args) == 1 && args[0] == "halt" {
			return fmt.Errorf("command failed (vagrant [halt]): exit status 1")
		}
		return nil
	}
	inspectHypervVagrantStopTarget = func(alchemy_build.VirtualMachineConfig) (StartTargetState, error) {
		return StartTargetState{Exists: true, Running: true, State: "running"}, nil
	}
	hypervVagrantStopSettleTimeout = 0

	config := alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "server",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsWindows,
		VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
	}

	err := RunHypervVagrantStopOnWindows(config)
	if err == nil {
		t.Fatal("expected stop to fail when VM remains running")
	}
	if err.Error() != "failed to stop Vagrant VM for ubuntu:server:amd64: graceful halt failed: command failed (vagrant [halt]): exit status 1; forced halt completed but VM is still running" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunHypervVagrantStartOnWindows_UsesTypeSpecificVagrantEnv(t *testing.T) {
	restore := stubHypervStopDependencies(t)
	defer restore()
	vagrantRoot := setHypervTestVagrantRoot(t)

	inspectHypervVagrantStartCmdTarget = func(alchemy_build.VirtualMachineConfig) (StartTargetState, error) {
		return StartTargetState{Exists: true, State: "off"}, nil
	}

	runCalls := 0
	runHypervVagrantCommandWithEnv = func(workingDir string, _ time.Duration, executable string, args []string, env []string, _ string) error {
		runCalls++
		if executable != "vagrant" {
			t.Fatalf("expected vagrant executable, got %q", executable)
		}
		if workingDir == "" {
			t.Fatal("expected Vagrant working directory to be set")
		}
		if len(args) != 3 || args[0] != "up" || args[1] != "--provider" || args[2] != "hyperv" {
			t.Fatalf("unexpected args: %v", args)
		}
		assertEnvContainsEntry(t, env, "VAGRANT_BOX_NAME=linux-ubuntu-desktop-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_VM_NAME=linux-ubuntu-desktop-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_DOTFILE_PATH="+filepath.Join(vagrantRoot, "linux-ubuntu-desktop-packer"))
		return nil
	}

	err := RunHypervVagrantStartOnWindows(alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "desktop",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsWindows,
		VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
	})
	if err != nil {
		t.Fatalf("expected start to succeed, got %v", err)
	}
	if runCalls != 1 {
		t.Fatalf("expected one vagrant up call, got %d", runCalls)
	}
}

func TestRunHypervVagrantDestroyOnWindows_UsesTypeSpecificVagrantEnv(t *testing.T) {
	restore := stubHypervStopDependencies(t)
	defer restore()
	vagrantRoot := setHypervTestVagrantRoot(t)

	hypervVagrantMachineExistsChecker = func(workingDir string, env []string) (bool, error) {
		if workingDir == "" {
			t.Fatal("expected Vagrant working directory to be set")
		}
		assertEnvContainsEntry(t, env, "VAGRANT_BOX_NAME=linux-ubuntu-desktop-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_VM_NAME=linux-ubuntu-desktop-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_DOTFILE_PATH="+filepath.Join(vagrantRoot, "linux-ubuntu-desktop-packer"))
		return true, nil
	}
	hypervVagrantBoxInstalledChecker = func(projectDir string, boxName string) (bool, error) {
		if projectDir == "" {
			t.Fatal("expected project dir to be set")
		}
		if boxName != "linux-ubuntu-desktop-packer" {
			t.Fatalf("expected desktop box name, got %q", boxName)
		}
		return false, nil
	}

	runCalls := 0
	runHypervVagrantCommandWithEnv = func(workingDir string, _ time.Duration, executable string, args []string, env []string, _ string) error {
		runCalls++
		if executable != "vagrant" {
			t.Fatalf("expected vagrant executable, got %q", executable)
		}
		if workingDir == "" {
			t.Fatal("expected Vagrant working directory to be set")
		}
		if len(args) != 2 || args[0] != "destroy" || args[1] != "-f" {
			t.Fatalf("unexpected args: %v", args)
		}
		assertEnvContainsEntry(t, env, "VAGRANT_BOX_NAME=linux-ubuntu-desktop-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_VM_NAME=linux-ubuntu-desktop-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_DOTFILE_PATH="+filepath.Join(vagrantRoot, "linux-ubuntu-desktop-packer"))
		return nil
	}

	err := RunHypervVagrantDestroyOnWindows(alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "desktop",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsWindows,
		VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
	})
	if err != nil {
		t.Fatalf("expected destroy to succeed, got %v", err)
	}
	if runCalls != 1 {
		t.Fatalf("expected one vagrant destroy call, got %d", runCalls)
	}
}

func TestDestroyTargetExists_UsesTypeSpecificVagrantEnv(t *testing.T) {
	restore := stubHypervStopDependencies(t)
	defer restore()
	vagrantRoot := setHypervTestVagrantRoot(t)

	hypervVagrantMachineExistsChecker = func(workingDir string, env []string) (bool, error) {
		if workingDir == "" {
			t.Fatal("expected Vagrant working directory to be set")
		}
		assertEnvContainsEntry(t, env, "VAGRANT_BOX_NAME=linux-ubuntu-desktop-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_VM_NAME=linux-ubuntu-desktop-packer")
		assertEnvContainsEntry(t, env, "VAGRANT_DOTFILE_PATH="+filepath.Join(vagrantRoot, "linux-ubuntu-desktop-packer"))
		return false, nil
	}
	hypervVagrantBoxInstalledChecker = func(_ string, boxName string) (bool, error) {
		return boxName == "linux-ubuntu-desktop-packer", nil
	}

	exists, err := DestroyTargetExists(alchemy_build.VirtualMachineConfig{
		OS:                   "ubuntu",
		UbuntuType:           "desktop",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsWindows,
		VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
	})
	if err != nil {
		t.Fatalf("expected destroy target inspection to succeed, got %v", err)
	}
	if !exists {
		t.Fatal("expected desktop destroy target to be reported as existing")
	}
}

func stubHypervStopDependencies(t *testing.T) func() {
	t.Helper()

	originalRun := runHypervVagrantCommandWithEnv
	originalInspectStart := inspectHypervVagrantStartCmdTarget
	originalInspect := inspectHypervVagrantStopTarget
	originalMachineExists := hypervVagrantMachineExistsChecker
	originalBoxInstalled := hypervVagrantBoxInstalledChecker
	originalTimeout := hypervVagrantStopSettleTimeout
	originalPoll := hypervVagrantStopPollInterval
	dirs := alchemy_build.GetDirectoriesInstance()
	originalProjectDir := dirs.ProjectDir
	dirs.ProjectDir = t.TempDir()
	hypervVagrantStopSettleTimeout = time.Millisecond
	hypervVagrantStopPollInterval = 0

	return func() {
		runHypervVagrantCommandWithEnv = originalRun
		inspectHypervVagrantStartCmdTarget = originalInspectStart
		inspectHypervVagrantStopTarget = originalInspect
		hypervVagrantMachineExistsChecker = originalMachineExists
		hypervVagrantBoxInstalledChecker = originalBoxInstalled
		hypervVagrantStopSettleTimeout = originalTimeout
		hypervVagrantStopPollInterval = originalPoll
		dirs.ProjectDir = originalProjectDir
	}
}

func setHypervTestVagrantRoot(t *testing.T) string {
	t.Helper()

	dirs := alchemy_build.GetDirectoriesInstance()
	originalAppDataDir := dirs.AppDataDir
	originalVagrantDir := dirs.VagrantDir
	appDataDir := t.TempDir()
	vagrantDir := filepath.Join(appDataDir, ".vagrant")
	dirs.AppDataDir = appDataDir
	dirs.VagrantDir = vagrantDir
	t.Cleanup(func() {
		dirs.AppDataDir = originalAppDataDir
		dirs.VagrantDir = originalVagrantDir
	})

	return vagrantDir
}

func TestVagrantBoxListIncludesMatchesExactNameAndProvider(t *testing.T) {
	output := "win11-packer (hyperv, 0)\nlinux-ubuntu-server-packer (hyperv, 0)\n"

	if !vagrantBoxListIncludes(output, "win11-packer", "hyperv") {
		t.Fatal("expected hyper-v box to be found")
	}
	if vagrantBoxListIncludes(output, "win11", "hyperv") {
		t.Fatal("did not expect substring box name match")
	}
	if vagrantBoxListIncludes(output, "win11-packer", "virtualbox") {
		t.Fatal("did not expect provider mismatch to match")
	}
}

func assertEnvContainsEntry(t *testing.T, env []string, want string) {
	t.Helper()

	for _, entry := range env {
		if entry == want {
			return
		}
	}

	t.Fatalf("expected env to contain %q, got %v", want, env)
}
