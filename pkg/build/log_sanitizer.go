package build

import (
	"os"
	"regexp"
	"strings"
)

var sensitiveTextPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(--windowslogonpassword\s+)(\S+)`),                                 // #nosec G101 -- redaction pattern for log sanitization, not a credential.
	regexp.MustCompile(`(?i)(--token\s+)(\S+)`),                                                // #nosec G101 -- redaction pattern for log sanitization, not a credential.
	regexp.MustCompile(`(?i)(winrm_password\s*=\s*)(\"[^\"]*\"|\S+)`),                          // #nosec G101 -- redaction pattern for log sanitization, not a credential.
	regexp.MustCompile(`(?i)(ansible_(?:become_)?password\s*[:=]\s*)(\"[^\"]*\"|'[^']*'|\S+)`), // #nosec G101 -- redaction pattern for log sanitization, not a credential.
	regexp.MustCompile(`(?i)(generated password:\s*)(\S+)`),                                    // #nosec G101 -- redaction pattern for log sanitization, not a credential.
	regexp.MustCompile(`(?i)(password for user:\s*\S+\s*[:=]?\s*)(\S+)`),                       // #nosec G101 -- redaction pattern for log sanitization, not a credential.
}

func sanitizeSensitiveText(input string) string {
	sanitized := input
	for _, pattern := range sensitiveTextPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, `${1}[REDACTED]`)
	}

	for _, secret := range discoverRuntimeSecrets() {
		if secret == "" {
			continue
		}
		sanitized = strings.ReplaceAll(sanitized, secret, "[REDACTED]")
	}

	return sanitized
}

func sanitizeCommandArgs(args []string) []string {
	sanitized := make([]string, len(args))
	redactNext := false

	for i, arg := range args {
		if redactNext {
			sanitized[i] = "[REDACTED]"
			redactNext = false
			continue
		}

		sanitized[i] = sanitizeSensitiveText(arg)
		if expectsSecretValue(arg) {
			redactNext = true
		}
	}
	return sanitized
}

func expectsSecretValue(arg string) bool {
	switch strings.ToLower(arg) {
	case "--windowslogonpassword", "--token":
		return true
	default:
		return false
	}
}

func discoverRuntimeSecrets() []string {
	secrets := []string{
		"P@ssw0rd!", // #nosec G101 -- redact the documented default VM password from logs when it appears.
	}

	for _, envEntry := range os.Environ() {
		parts := strings.SplitN(envEntry, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToUpper(parts[0])
		value := parts[1]
		if value == "" || len(value) < 6 {
			continue
		}

		if strings.Contains(key, "TOKEN") ||
			strings.Contains(key, "PASSWORD") ||
			strings.Contains(key, "SECRET") ||
			strings.Contains(key, "PAT") {
			secrets = append(secrets, value)
		}
	}

	return secrets
}
