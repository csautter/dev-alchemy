package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelpDoesNotDuplicateGeneratedSections(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	if err := rootCmd.Help(); err != nil {
		t.Fatalf("expected no help error, got %v", err)
	}

	output := buf.String()
	if got := strings.Count(output, "Available Commands:"); got != 1 {
		t.Fatalf("expected one Available Commands section, got %d in help output:\n%s", got, output)
	}
	if got := strings.Count(output, "Usage:"); got != 1 {
		t.Fatalf("expected one Usage section, got %d in help output:\n%s", got, output)
	}
	if strings.Contains(output, "Build the VM Images for multiple target OS") {
		t.Fatalf("expected help output to use generated command descriptions, got stale hard-coded text:\n%s", output)
	}
}
