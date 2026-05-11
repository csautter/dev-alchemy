package oci

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistryHTTPClientSkipsTLSVerification(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := registryHTTPClient(RegistryOptions{InsecureSkipTLSVerify: true})
	if err != nil {
		t.Fatalf("expected registry HTTP client: %v", err)
	}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected insecure registry client to reach self-signed server: %v", err)
	}
	resp.Body.Close()
}

func TestRegistryHTTPClientTrustsCAFile(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	caPath := filepath.Join(t.TempDir(), "ca.pem")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := os.WriteFile(caPath, caPEM, 0o600); err != nil {
		t.Fatalf("failed to write test CA file: %v", err)
	}

	client, err := registryHTTPClient(RegistryOptions{CAFile: caPath})
	if err != nil {
		t.Fatalf("expected registry HTTP client with CA file: %v", err)
	}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected registry client to trust CA file: %v", err)
	}
	resp.Body.Close()
}

func TestRegistryHTTPClientRejectsInvalidCAFile(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caPath, []byte("not a certificate"), 0o600); err != nil {
		t.Fatalf("failed to write invalid test CA file: %v", err)
	}

	_, err := registryHTTPClient(RegistryOptions{CAFile: caPath})
	if err == nil {
		t.Fatal("expected invalid CA file to fail")
	}
	if !strings.Contains(err.Error(), "no PEM certificates") {
		t.Fatalf("expected invalid CA error, got %q", err.Error())
	}
}
