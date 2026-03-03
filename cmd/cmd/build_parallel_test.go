package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

// buildDuration is the simulated duration of one real packer/QEMU build.
// Keep at 10 s to realistically exercise timing, parallelism and cancellation.
// Run: go test ./cmd/cmd/... -run TestParallelBuilds -v -timeout 120s
const buildDuration = 10 * time.Second

// testVMs returns fixed VM configs that do not depend on the current host OS.
func testVMs() []alchemy_build.VirtualMachineConfig {
	return []alchemy_build.VirtualMachineConfig{
		{OS: "ubuntu", Arch: "amd64", UbuntuType: "server"},
		{OS: "ubuntu", Arch: "arm64", UbuntuType: "server"},
		{OS: "ubuntu", Arch: "amd64", UbuntuType: "desktop"},
		{OS: "windows11", Arch: "amd64"},
	}
}

// slug returns a human-readable VM identifier used in assertions.
func slug(vm alchemy_build.VirtualMachineConfig) string {
	return fmt.Sprintf("%s/%s/%s", vm.OS, vm.UbuntuType, vm.Arch)
}

// successRunner sleeps for buildDuration then returns nil.
// It honours ctx cancellation so SIGINT tests finish in seconds.
func successRunner(ctx context.Context, vm alchemy_build.VirtualMachineConfig) error {
	select {
	case <-time.After(buildDuration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// =============================================================================
// Test 1 - All builds succeed
// =============================================================================

// TestParallelBuilds_AllSucceed runs 4 dummy 10-second builds with parallelism=2.
// Expected wall time ~20 s (2 batches x 10 s).
func TestParallelBuilds_AllSucceed(t *testing.T) {
	t.Log("Running 4 x 10s dummy builds with parallelism=2; expected ~20s total")

	ctx := context.Background()
	vms := testVMs()

	start := time.Now()
	errs := runParallelBuilds(ctx, vms, 2, successRunner)
	elapsed := time.Since(start)

	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d:", len(errs))
		for _, e := range errs {
			t.Errorf("  %v", e)
		}
	}
	// With parallelism=2 wall time should be >= ~2xbuildDuration.
	minExpected := 2*buildDuration - time.Second
	if elapsed < minExpected {
		t.Errorf("elapsed %v suspiciously short (expected >= %v); parallel governor may be broken", elapsed, minExpected)
	}
	t.Logf("All 4 builds completed in %v", elapsed)
}

// =============================================================================
// Test 2 - Some builds fail; all others still run
// =============================================================================

// TestParallelBuilds_SomeFailOthersStillRun verifies that a failing build does
// NOT prevent remaining builds from completing.
//
// Setup: 4 VMs, parallelism 4 (all launched concurrently).
//   - ubuntu/server/arm64  fails immediately.
//   - windows11//amd64     fails after 2 s.
//   - ubuntu/server/amd64  succeeds after 10 s.
//   - ubuntu/desktop/amd64 succeeds after 10 s.
//
// Expected: 2 errors, all 4 VMs attempted, 2 successful completions.
func TestParallelBuilds_SomeFailOthersStillRun(t *testing.T) {
	t.Log("4 builds, 2 will fail - verifying others still complete")

	vms := testVMs()

	var startedMu sync.Mutex
	started := make(map[string]bool)
	var completedSuccess int32 // atomic counter

	runner := func(ctx context.Context, vm alchemy_build.VirtualMachineConfig) error {
		s := slug(vm)
		startedMu.Lock()
		started[s] = true
		startedMu.Unlock()

		switch s {
		case "ubuntu/server/arm64":
			return errors.New("simulated immediate build failure")
		case "windows11//amd64":
			select {
			case <-time.After(2 * time.Second):
				return fmt.Errorf("simulated late build failure after 2s")
			case <-ctx.Done():
				return ctx.Err()
			}
		default:
			select {
			case <-time.After(buildDuration):
				atomic.AddInt32(&completedSuccess, 1)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	start := time.Now()
	errs := runParallelBuilds(context.Background(), vms, 4, runner)
	elapsed := time.Since(start)

	if len(errs) != 2 {
		t.Errorf("expected exactly 2 errors, got %d:", len(errs))
		for _, e := range errs {
			t.Errorf("  %v", e)
		}
	}

	startedMu.Lock()
	for _, vm := range vms {
		if !started[slug(vm)] {
			t.Errorf("VM %q was never started - a failure must have skipped it", slug(vm))
		}
	}
	startedMu.Unlock()

	if got := atomic.LoadInt32(&completedSuccess); got != 2 {
		t.Errorf("expected 2 successful completions, got %d", got)
	}
	t.Logf("Partial-failure run finished in %v (2 failed, 2 succeeded)", elapsed)
}

// =============================================================================
// Test 3 - Context cancellation (SIGINT simulation) stops all builds
// =============================================================================

// TestParallelBuilds_SIGINTCancelsAll simulates Ctrl-C by cancelling the context
// 3 s into a run where each build would otherwise take 30 s.
//
// The test exercises two behaviours:
//  1. Running builds receive ctx.Done() and abort early.
//  2. Builds queued behind the semaphore are never started after cancellation.
//
// Setup: 6 VMs, parallelism 2, cancel after 3 s.
// Expected: wall time <8 s, all errors wrap context.Canceled, >=2 runners started.
func TestParallelBuilds_SIGINTCancelsAll(t *testing.T) {
	vms := append(testVMs(),
		alchemy_build.VirtualMachineConfig{OS: "ubuntu", Arch: "arm64", UbuntuType: "desktop"},
		alchemy_build.VirtualMachineConfig{OS: "windows11", Arch: "arm64"},
	)

	var startedMu sync.Mutex
	started := make(map[string]int)

	longRunner := func(ctx context.Context, vm alchemy_build.VirtualMachineConfig) error {
		s := slug(vm)
		startedMu.Lock()
		started[s]++
		startedMu.Unlock()
		select {
		case <-time.After(30 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Simulate SIGINT: cancel after 3 s (mirrors the signal goroutine in buildCmd).
	go func() {
		time.Sleep(3 * time.Second)
		t.Log("Simulating SIGINT: cancelling context")
		cancel()
	}()

	start := time.Now()
	errs := runParallelBuilds(ctx, vms, 2, longRunner)
	elapsed := time.Since(start)

	maxExpected := 8 * time.Second
	if elapsed > maxExpected {
		t.Errorf("expected finish within %v after SIGINT, took %v", maxExpected, elapsed)
	}
	t.Logf("runParallelBuilds returned %v after cancellation", elapsed)

	if len(errs) == 0 {
		t.Error("expected at least one cancellation error but got none")
	}
	for _, e := range errs {
		if !errors.Is(e, context.Canceled) {
			t.Errorf("expected error to wrap context.Canceled, got: %v", e)
		}
	}
	t.Logf("Received %d cancellation error(s)", len(errs))

	startedMu.Lock()
	totalStarted := 0
	for _, c := range started {
		totalStarted += c
	}
	startedMu.Unlock()
	if totalStarted < 2 {
		t.Errorf("expected >= 2 VMs to start before cancellation, only %d did", totalStarted)
	}
	t.Logf("%d VM runner(s) started before/during cancellation", totalStarted)
}

// =============================================================================
// Test 4 - Actual OS SIGINT wires correctly to context cancellation
// =============================================================================

// TestParallelBuilds_OSSIGINTSignal sends a real SIGINT to the current process
// and validates the signal-to-context wiring that buildCmd relies on.
// This test does NOT invoke runParallelBuilds directly.
func TestParallelBuilds_OSSIGINTSignal(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	// Register signal.Notify BEFORE sending the signal so the OS routes it to
	// sigCh instead of invoking the default handler (which would kill the process).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	defer signal.Stop(sigCh)

	// Mirror the signal handler goroutine from buildCmd.
	go func() {
		sig := <-sigCh
		t.Logf("Received OS signal: %v", sig)
		stop()
	}()

	// Deliver a real SIGINT to the process after signal.Notify is active.
	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	select {
	case <-ctx.Done():
		t.Log("Context was cancelled as expected after SIGINT")
	case <-time.After(2 * time.Second):
		t.Error("Context was NOT cancelled within 2s after SIGINT - signal wiring is broken")
	}
}

// =============================================================================
// Test 5 - Sequential builds (parallelism=1), all succeed
// =============================================================================

// TestSequentialBuilds_AllSucceed runs 3 dummy 10-second builds one after the
// other (parallelism=1) and verifies:
//   - Each build finishes before the next one starts.
//   - Wall time is ~30 s (3 x 10 s with zero overlap).
//   - Zero errors are returned.
//   - Execution order matches the input slice order.
func TestSequentialBuilds_AllSucceed(t *testing.T) {
	t.Log("Running 3 x 10s sequential builds (parallelism=1); expected ~30s total")

	vms := []alchemy_build.VirtualMachineConfig{
		{OS: "ubuntu", Arch: "amd64", UbuntuType: "server"},
		{OS: "ubuntu", Arch: "arm64", UbuntuType: "server"},
		{OS: "windows11", Arch: "amd64"},
	}

	var orderMu sync.Mutex
	var executionOrder []string
	// concurrentActive tracks how many runners are executing at the same time.
	var concurrentActive int32

	runner := func(ctx context.Context, vm alchemy_build.VirtualMachineConfig) error {
		// Check for concurrent execution — with parallelism=1 this must never exceed 1.
		if n := atomic.AddInt32(&concurrentActive, 1); n > 1 {
			atomic.AddInt32(&concurrentActive, -1)
			return fmt.Errorf("concurrency violation: %d runners active simultaneously (parallelism=1)", n)
		}

		orderMu.Lock()
		executionOrder = append(executionOrder, slug(vm))
		orderMu.Unlock()

		select {
		case <-time.After(buildDuration):
		case <-ctx.Done():
			atomic.AddInt32(&concurrentActive, -1)
			return ctx.Err()
		}

		atomic.AddInt32(&concurrentActive, -1)
		return nil
	}

	start := time.Now()
	errs := runParallelBuilds(context.Background(), vms, 1, runner)
	elapsed := time.Since(start)

	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d:", len(errs))
		for _, e := range errs {
			t.Errorf("  %v", e)
		}
	}

	// Wall time must be at least 3 x buildDuration (no parallelism).
	minExpected := 3*buildDuration - time.Second
	if elapsed < minExpected {
		t.Errorf("elapsed %v too short for sequential execution (expected >= %v)", elapsed, minExpected)
	}

	// Wall time must be less than what parallel execution could achieve.
	// Two concurrent runners would finish in ~20 s; strictly sequential must take longer.
	maxParallelEquivalent := 2 * buildDuration
	if elapsed < maxParallelEquivalent {
		t.Errorf("elapsed %v suggests builds ran in parallel (expected serial, >= %v)", elapsed, maxParallelEquivalent)
	}

	// Execution order must exactly match the input slice order.
	orderMu.Lock()
	for i, vm := range vms {
		if i >= len(executionOrder) {
			t.Errorf("build %d (%s) was never executed", i, slug(vm))
			continue
		}
		if executionOrder[i] != slug(vm) {
			t.Errorf("build %d: expected %q, got %q", i, slug(vm), executionOrder[i])
		}
	}
	orderMu.Unlock()

	t.Logf("Sequential builds finished in %v, order: %v", elapsed, executionOrder)
}

// =============================================================================
// Test 6 - Sequential builds (parallelism=1), one fails; others still run
// =============================================================================

// TestSequentialBuilds_FailureDoesNotSkipRemainder verifies that when the middle
// build in a sequential run fails, the builds that come after it are still
// executed to completion.
//
// Setup: 3 VMs, parallelism=1.
//   - Build 1 (ubuntu/server/amd64)  → succeeds after 10 s.
//   - Build 2 (ubuntu/server/arm64)  → fails immediately.
//   - Build 3 (windows11//amd64)     → succeeds after 10 s.
//
// Expected:
//   - Exactly 1 error (build 2).
//   - All 3 builds were attempted in order.
//   - Build 1 and build 3 complete successfully.
//   - Wall time ~20 s (build 1 + build 3; build 2 contributes almost nothing).
func TestSequentialBuilds_FailureDoesNotSkipRemainder(t *testing.T) {
	t.Log("3 sequential builds, middle one fails — verifying build 3 still runs")

	vms := []alchemy_build.VirtualMachineConfig{
		{OS: "ubuntu", Arch: "amd64", UbuntuType: "server"}, // succeeds
		{OS: "ubuntu", Arch: "arm64", UbuntuType: "server"}, // fails
		{OS: "windows11", Arch: "amd64"},                    // must still run
	}

	var orderMu sync.Mutex
	var executionOrder []string
	var completedSuccess int32

	runner := func(ctx context.Context, vm alchemy_build.VirtualMachineConfig) error {
		s := slug(vm)
		orderMu.Lock()
		executionOrder = append(executionOrder, s)
		orderMu.Unlock()

		switch s {
		case "ubuntu/server/arm64":
			return errors.New("simulated sequential build failure")
		default:
			select {
			case <-time.After(buildDuration):
				atomic.AddInt32(&completedSuccess, 1)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	start := time.Now()
	errs := runParallelBuilds(context.Background(), vms, 1, runner)
	elapsed := time.Since(start)

	// Exactly one error expected.
	if len(errs) != 1 {
		t.Errorf("expected exactly 1 error, got %d:", len(errs))
		for _, e := range errs {
			t.Errorf("  %v", e)
		}
	}

	// All 3 builds must have been attempted, in order.
	orderMu.Lock()
	expectedOrder := []string{slug(vms[0]), slug(vms[1]), slug(vms[2])}
	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("expected %d builds to run, got %d: %v", len(expectedOrder), len(executionOrder), executionOrder)
	} else {
		for i, want := range expectedOrder {
			if executionOrder[i] != want {
				t.Errorf("build %d: expected %q, got %q", i, want, executionOrder[i])
			}
		}
	}
	orderMu.Unlock()

	// Builds 1 and 3 must have completed successfully.
	if got := atomic.LoadInt32(&completedSuccess); got != 2 {
		t.Errorf("expected 2 successful completions, got %d", got)
	}

	// Wall time should be ~20 s (two 10 s builds; failed build adds ~0 s).
	minExpected := 2*buildDuration - time.Second
	if elapsed < minExpected {
		t.Errorf("elapsed %v too short; build 3 may have been skipped (expected >= %v)", elapsed, minExpected)
	}

	t.Logf("Sequential failure run finished in %v, order: %v (1 failed, 2 succeeded)", elapsed, executionOrder)
}
