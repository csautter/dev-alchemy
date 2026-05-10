package oci

import (
	"strings"
	"testing"
)

func TestParsePushReferenceRejectsDigest(t *testing.T) {
	_, err := parsePushReference("localhost:5000/dev-alchemy/artifact@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err == nil {
		t.Fatal("expected digest push reference to fail")
	}
	if !strings.Contains(err.Error(), "must use a tag") {
		t.Fatalf("expected tag-only error, got %q", err.Error())
	}
}
