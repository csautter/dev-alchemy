package build

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRunExternalProcessWithRetriesStopsOnInterrupt(t *testing.T) {
	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to resolve test executable: %v", err)
	}

	interrupt := make(chan bool, 1)
	done := make(chan context.Context, 1)
	start := time.Now()

	go func() {
		done <- RunExternalProcessWithRetries(RunProcessConfig{
			ExecutablePath:     execPath,
			Args:               []string{"-test.run=^TestHelperProcessExit1$", "--"},
			Env:                []string{"GO_WANT_HELPER_PROCESS=1"},
			Timeout:            10 * time.Second,
			Retries:            3,
			RetryInterval:      5 * time.Second,
			InterruptRetryChan: interrupt,
		})
	}()

	time.Sleep(150 * time.Millisecond)
	interrupt <- true

	select {
	case ctx := <-done:
		if ctx == nil {
			t.Fatal("expected a cancelled context, got nil")
		}
		if ctx.Err() != context.Canceled {
			t.Fatalf("expected context canceled, got %v", ctx.Err())
		}
		if elapsed := time.Since(start); elapsed >= 2*time.Second {
			t.Fatalf("expected retry loop to stop quickly after interrupt, took %s", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for retry loop to stop")
	}
}

func TestHelperProcessExit1(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	time.Sleep(10 * time.Second)
	os.Exit(1)
}
