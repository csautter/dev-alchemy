package build

import "testing"

func TestAvailableVirtualMachineConfigsForHostOS(t *testing.T) {
	configs := AvailableVirtualMachineConfigsForHostOS(HostOsWindows)
	if len(configs) == 0 {
		t.Fatal("expected windows configs, got none")
	}

	for _, config := range configs {
		if config.HostOs != HostOsWindows {
			t.Fatalf("expected only windows configs, got host OS %q", config.HostOs)
		}
	}
}

func TestAvailableVirtualMachineConfigsForCurrentHostOSByVirtualizationEngine(t *testing.T) {
	grouped := AvailableVirtualMachineConfigsForCurrentHostOSByVirtualizationEngine()
	if len(grouped) == 0 {
		t.Fatal("expected grouped configs, got none")
	}

	for engine, configs := range grouped {
		if len(configs) == 0 {
			t.Fatalf("expected configs for engine %q, got none", engine)
		}
		for _, config := range configs {
			if config.HostOs != GetCurrentHostOs() {
				t.Fatalf("expected current host OS %q, got %q", GetCurrentHostOs(), config.HostOs)
			}
			if config.VirtualizationEngine != engine {
				t.Fatalf("expected engine %q, got %q", engine, config.VirtualizationEngine)
			}
		}
	}
}

func TestCurrentHostVirtualizationEngines(t *testing.T) {
	engines := CurrentHostVirtualizationEngines()
	if len(engines) == 0 {
		t.Fatal("expected at least one virtualization engine")
	}

	for i := 1; i < len(engines); i++ {
		if engines[i-1] > engines[i] {
			t.Fatalf("expected sorted engines, got %v", engines)
		}
	}
}

func TestLinuxHostUbuntuQemuConfigs(t *testing.T) {
	configs := AvailableVirtualMachineConfigsForHostOS(HostOsLinux)
	if len(configs) != 4 {
		t.Fatalf("expected 4 linux build configs, got %d", len(configs))
	}

	want := map[string]bool{
		"ubuntu/server/amd64/qemu":  false,
		"ubuntu/server/arm64/qemu":  false,
		"ubuntu/desktop/amd64/qemu": false,
		"ubuntu/desktop/arm64/qemu": false,
	}

	for _, config := range configs {
		key := config.OS + "/" + config.UbuntuType + "/" + config.Arch + "/" + string(config.VirtualizationEngine)
		if _, ok := want[key]; !ok {
			t.Fatalf("unexpected linux config %q", key)
		}
		want[key] = true
	}

	for key, seen := range want {
		if !seen {
			t.Fatalf("missing linux config %q", key)
		}
	}
}

func TestGroupVirtualMachineConfigsByVirtualizationEngine(t *testing.T) {
	configs := []VirtualMachineConfig{
		{OS: "windows11", VirtualizationEngine: VirtualizationEngineHyperv},
		{OS: "ubuntu", UbuntuType: "server", VirtualizationEngine: VirtualizationEngineUtm},
		{OS: "ubuntu", UbuntuType: "desktop", VirtualizationEngine: VirtualizationEngineHyperv},
	}

	grouped := GroupVirtualMachineConfigsByVirtualizationEngine(configs)

	if len(grouped) != 2 {
		t.Fatalf("expected 2 virtualization engines, got %d", len(grouped))
	}
	if len(grouped[VirtualizationEngineHyperv]) != 2 {
		t.Fatalf("expected 2 hyperv configs, got %d", len(grouped[VirtualizationEngineHyperv]))
	}
	if len(grouped[VirtualizationEngineUtm]) != 1 {
		t.Fatalf("expected 1 utm config, got %d", len(grouped[VirtualizationEngineUtm]))
	}
}

func TestVirtualizationEnginesForVirtualMachineConfigs(t *testing.T) {
	configs := []VirtualMachineConfig{
		{VirtualizationEngine: VirtualizationEngineVirtualBox},
		{VirtualizationEngine: VirtualizationEngineHyperv},
		{VirtualizationEngine: VirtualizationEngineUtm},
		{VirtualizationEngine: VirtualizationEngineHyperv},
	}

	engines := VirtualizationEnginesForVirtualMachineConfigs(configs)
	expected := []VirtualizationEngine{
		VirtualizationEngineHyperv,
		VirtualizationEngineUtm,
		VirtualizationEngineVirtualBox,
	}

	if len(engines) != len(expected) {
		t.Fatalf("expected %d engines, got %d", len(expected), len(engines))
	}
	for i, engine := range expected {
		if engines[i] != engine {
			t.Fatalf("expected engine %q at index %d, got %q", engine, i, engines[i])
		}
	}
}

func TestDisplayVirtualizationEngineMarksUnstableEngines(t *testing.T) {
	if got := DisplayVirtualizationEngine(VirtualizationEngineHyperv); got != "hyperv" {
		t.Fatalf("expected stable engine display name, got %q", got)
	}
	if got := DisplayVirtualizationEngine(VirtualizationEngineVirtualBox); got != "virtualbox (unstable)" {
		t.Fatalf("expected unstable engine display name, got %q", got)
	}
}

func TestWindowsVirtualBoxBuildConfigUsesFixedMemoryProfile(t *testing.T) {
	configs := AvailableVirtualMachineConfigsForHostOS(HostOsWindows)

	var hypervConfig VirtualMachineConfig
	var virtualboxConfig VirtualMachineConfig
	var foundHyperv bool
	var foundVirtualBox bool

	for _, config := range configs {
		if config.OS != "windows11" || config.Arch != "amd64" {
			continue
		}
		switch config.VirtualizationEngine {
		case VirtualizationEngineHyperv:
			hypervConfig = config
			foundHyperv = true
		case VirtualizationEngineVirtualBox:
			virtualboxConfig = config
			foundVirtualBox = true
		}
	}

	if !foundHyperv {
		t.Fatal("expected windows11 hyperv config")
	}
	if !foundVirtualBox {
		t.Fatal("expected windows11 virtualbox config")
	}
	if virtualboxConfig.MemoryMB != hypervConfig.MemoryMB {
		t.Fatalf("expected virtualbox memory %d to match hyperv memory %d", virtualboxConfig.MemoryMB, hypervConfig.MemoryMB)
	}
}
