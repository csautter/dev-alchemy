package build

import (
	"strings"
	"testing"
)

func TestSanitizeSensitiveTextMasksKnownPasswordPatterns(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "windows logon password argument",
			input: `config.cmd --windowslogonpassword abc12345`,
		},
		{
			name:  "runner token argument",
			input: `config.cmd --token ghs_verysecretvalue`,
		},
		{
			name:  "winrm password assignment",
			input: `winrm_password = "P@ssw0rd!"`,
		},
		{
			name:  "generated password log line",
			input: `Generated password: example-password`,
		},
		{
			name:  "ansible password yaml",
			input: `ansible_password: P@ssw0rd!`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSensitiveText(tt.input)
			if strings.Contains(got, "P@ssw0rd!") ||
				strings.Contains(got, "abc12345") ||
				strings.Contains(got, "ghs_verysecretvalue") ||
				strings.Contains(got, "example-password") {
				t.Fatalf("expected sensitive value to be redacted, got %q", got)
			}
			if !strings.Contains(got, "[REDACTED]") {
				t.Fatalf("expected redaction marker in %q", got)
			}
		})
	}
}

func TestSanitizeCommandArgsLeavesSliceShapeIntact(t *testing.T) {
	args := []string{"config.cmd", "--windowslogonpassword", "abc12345"}

	got := sanitizeCommandArgs(args)

	if len(got) != len(args) {
		t.Fatalf("expected %d args, got %d", len(args), len(got))
	}
	if got[0] != "config.cmd" {
		t.Fatalf("expected executable to remain unchanged, got %q", got[0])
	}
	if got[2] != "[REDACTED]" {
		t.Fatalf("expected password arg to be redacted, got %q", got[2])
	}
}
