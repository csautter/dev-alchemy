package build

import (
	"context"
	"testing"
	"time"
)

func TestRunExternalProcessWithRetriesStopsOnInterrupt(t *testing.T) {
	interrupt := make(chan bool, 1)
	done := make(chan context.Context, 1)
	start := time.Now()

	go func() {
		done <- RunExternalProcessWithRetries(RunProcessConfig{
			ExecutablePath:     "bash",
			Args:               []string{"-lc", "exit 1"},
			Timeout:            5 * time.Second,
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
