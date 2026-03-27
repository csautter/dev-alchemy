package build

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestResolveDefaultAppDataDirForOS_UsesOverride(t *testing.T) {
	got, err := resolveDefaultAppDataDirForOS(
		"linux",
		func(key string) string {
			if key == devAlchemyAppDataEnvVar {
				return "/tmp/dev-alchemy-custom"
			}
			return ""
		},
		func() (string, error) { return "/home/tester", nil },
		func() (string, error) { return "/config", nil },
	)
	if err != nil {
		t.Fatalf("resolveDefaultAppDataDirForOS returned error: %v", err)
	}
	if got != filepath.Clean("/tmp/dev-alchemy-custom") {
		t.Fatalf("expected override path, got %q", got)
	}
}

func TestResolveDefaultAppDataDirForOS_Darwin(t *testing.T) {
	got, err := resolveDefaultAppDataDirForOS(
		"darwin",
		func(string) string { return "" },
		func() (string, error) { return "/Users/tester", nil },
		func() (string, error) { return "", nil },
	)
	if err != nil {
		t.Fatalf("resolveDefaultAppDataDirForOS returned error: %v", err)
	}
	want := filepath.Join("/Users/tester", "Library", "Application Support", devAlchemyAppName)
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveDefaultAppDataDirForOS_WindowsPrefersLocalAppData(t *testing.T) {
	got, err := resolveDefaultAppDataDirForOS(
		"windows",
		func(key string) string {
			switch key {
			case "LOCALAPPDATA":
				return `C:\Users\tester\AppData\Local`
			case "APPDATA":
				return `C:\Users\tester\AppData\Roaming`
			default:
				return ""
			}
		},
		func() (string, error) { return "", nil },
		func() (string, error) { return `C:\Users\tester\AppData\Roaming`, nil },
	)
	if err != nil {
		t.Fatalf("resolveDefaultAppDataDirForOS returned error: %v", err)
	}
	want := filepath.Join(`C:\Users\tester\AppData\Local`, devAlchemyAppName)
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveDefaultAppDataDirForOS_LinuxUsesXDGDataHome(t *testing.T) {
	got, err := resolveDefaultAppDataDirForOS(
		"linux",
		func(key string) string {
			if key == "XDG_DATA_HOME" {
				return "/home/tester/.local/share"
			}
			return ""
		},
		func() (string, error) { return "/home/tester", nil },
		func() (string, error) { return "", nil },
	)
	if err != nil {
		t.Fatalf("resolveDefaultAppDataDirForOS returned error: %v", err)
	}
	want := filepath.Join("/home/tester/.local/share", devAlchemyAppName)
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveDefaultAppDataDirForOS_LinuxFallsBackToHome(t *testing.T) {
	got, err := resolveDefaultAppDataDirForOS(
		"linux",
		func(string) string { return "" },
		func() (string, error) { return "/home/tester", nil },
		func() (string, error) { return "", nil },
	)
	if err != nil {
		t.Fatalf("resolveDefaultAppDataDirForOS returned error: %v", err)
	}
	want := filepath.Join("/home/tester", ".local", "share", devAlchemyAppName)
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveDefaultAppDataDirForOS_ReturnsErrorWhenHomeUnavailable(t *testing.T) {
	_, err := resolveDefaultAppDataDirForOS(
		"darwin",
		func(string) string { return "" },
		func() (string, error) { return "", errors.New("boom") },
		func() (string, error) { return "", nil },
	)
	if err == nil {
		t.Fatal("expected error when home directory lookup fails")
	}
}
