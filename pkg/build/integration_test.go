package build

import (
	"os"
	"testing"
)

func requireIntegrationTests(t *testing.T) {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test; set RUN_INTEGRATION_TESTS=1 to run")
	}
}
