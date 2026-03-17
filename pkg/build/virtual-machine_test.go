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
	if GetCurrentHostOs() == HostOsLinux {
		if len(grouped) != 0 {
			t.Fatalf("expected no grouped configs on linux host, got %d", len(grouped))
		}
		return
	}

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
	if GetCurrentHostOs() == HostOsLinux {
		if len(engines) != 0 {
			t.Fatalf("expected no virtualization engines on linux host, got %v", engines)
		}
		return
	}

	if len(engines) == 0 {
		t.Fatal("expected at least one virtualization engine")
	}

	for i := 1; i < len(engines); i++ {
		if engines[i-1] > engines[i] {
			t.Fatalf("expected sorted engines, got %v", engines)
		}
	}
}
