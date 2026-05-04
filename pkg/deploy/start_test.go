package deploy

import (
	"strings"
	"testing"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

func TestSupportsStart(t *testing.T) {
	tests := []struct {
		name string
		vm   alchemy_build.VirtualMachineConfig
		want bool
	}{
		{
			name: "utm supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "windows11",
				Arch:                 "arm64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineUtm,
			},
			want: true,
		},
		{
			name: "tart supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "macos",
				Arch:                 "arm64",
				HostOs:               alchemy_build.HostOsDarwin,
				VirtualizationEngine: alchemy_build.VirtualizationEngineTart,
			},
			want: true,
		},
		{
			name: "hyperv supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "ubuntu",
				UbuntuType:           "server",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsWindows,
				VirtualizationEngine: alchemy_build.VirtualizationEngineHyperv,
			},
			want: true,
		},
		{
			name: "linux qemu supported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "ubuntu",
				UbuntuType:           "server",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsLinux,
				VirtualizationEngine: alchemy_build.VirtualizationEngineQemu,
			},
			want: true,
		},
		{
			name: "virtualbox unsupported",
			vm: alchemy_build.VirtualMachineConfig{
				OS:                   "windows11",
				Arch:                 "amd64",
				HostOs:               alchemy_build.HostOsWindows,
				VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		if got := SupportsStart(tt.vm); got != tt.want {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestRunStartReturnsHelpfulUnsupportedError(t *testing.T) {
	vm := alchemy_build.VirtualMachineConfig{
		OS:                   "windows11",
		Arch:                 "amd64",
		HostOs:               alchemy_build.HostOsWindows,
		VirtualizationEngine: alchemy_build.VirtualizationEngineVirtualBox,
	}

	err := RunStart(vm)
	if err == nil {
		t.Fatal("expected unsupported start to return an error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not implemented error, got %v", err)
	}
}

func TestStartCommandArguments(t *testing.T) {
	vm := alchemy_build.VirtualMachineConfig{
		OS:         "ubuntu",
		UbuntuType: "server",
		Arch:       "arm64",
	}

	if got := startCommandArguments(vm); got != "ubuntu --type server --arch arm64" {
		t.Fatalf("unexpected start command arguments %q", got)
	}
}
