package deploy

import (
	"errors"
	"regexp"
	"strings"
)

var (
	linuxIPv4Regex     = regexp.MustCompile(`(?m)\b((?:\d{1,3}\.){3}\d{1,3})\b`)
	loopbackAddressSet = map[string]struct{}{
		"127.0.0.1": {},
		"0.0.0.0":   {},
	}
)

func extractLinuxIPv4FromHostOutput(output string) (string, error) {
	matches := linuxIPv4Regex.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return "", errors.New("no IPv4 address found in command output")
	}

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		ip := strings.TrimSpace(match[1])
		if _, isLoopback := loopbackAddressSet[ip]; isLoopback {
			continue
		}
		return ip, nil
	}

	return "", errors.New("only loopback or invalid IPv4 candidates found in command output")
}
