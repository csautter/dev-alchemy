package build

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
)

func TestHostSupportsVncRecording(t *testing.T) {
	testCases := []struct {
		goos string
		want bool
	}{
		{goos: "darwin", want: true},
		{goos: "linux", want: true},
		{goos: "windows", want: false},
	}

	for _, tc := range testCases {
		if got := hostSupportsVncRecording(tc.goos); got != tc.want {
			t.Fatalf("hostSupportsVncRecording(%q) = %v, want %v", tc.goos, got, tc.want)
		}
	}
}

func TestHostSupportsVncViewer(t *testing.T) {
	testCases := []struct {
		goos string
		want bool
	}{
		{goos: "darwin", want: true},
		{goos: "linux", want: false},
		{goos: "windows", want: false},
	}

	for _, tc := range testCases {
		if got := hostSupportsVncViewer(tc.goos); got != tc.want {
			t.Fatalf("hostSupportsVncViewer(%q) = %v, want %v", tc.goos, got, tc.want)
		}
	}
}

func TestDetermineBuildCompletionDecisionSuccess(t *testing.T) {
	decision := determineBuildCompletionDecision(nil, nil, nil)
	if decision.err != nil {
		t.Fatalf("expected nil error, got %v", decision.err)
	}
	if !decision.runFfmpeg {
		t.Fatal("expected ffmpeg post-processing to run on success")
	}
	if !decision.buildSuccess {
		t.Fatal("expected build to be marked successful")
	}
}

func TestDetermineBuildCompletionDecisionFailureKeepsFfmpeg(t *testing.T) {
	waitErr := errors.New("build failed")
	decision := determineBuildCompletionDecision(waitErr, nil, nil)
	if !errors.Is(decision.err, waitErr) {
		t.Fatalf("expected wait error to be returned, got %v", decision.err)
	}
	if !decision.runFfmpeg {
		t.Fatal("expected ffmpeg post-processing to run on ordinary build failure")
	}
	if decision.buildSuccess {
		t.Fatal("expected failed build to stay unsuccessful")
	}
}

func TestDetermineBuildCompletionDecisionSkipsFfmpegOnContextCancellation(t *testing.T) {
	decision := determineBuildCompletionDecision(nil, context.Canceled, nil)
	if !errors.Is(decision.err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", decision.err)
	}
	if decision.runFfmpeg {
		t.Fatal("expected ffmpeg post-processing to be skipped after cancellation")
	}
}

func TestDetermineBuildCompletionDecisionSkipsFfmpegOnSignal(t *testing.T) {
	decision := determineBuildCompletionDecision(nil, nil, syscall.SIGINT)
	if decision.err == nil {
		t.Fatal("expected signal error")
	}
	if !decision.runFfmpeg {
		t.Fatal("expected ffmpeg post-processing to run after signal interruption")
	}
}

func TestDrainInterruptedSignalReturnsSignalWhenAvailable(t *testing.T) {
	interruptedSignal := make(chan os.Signal, 1)
	interruptedSignal <- syscall.SIGTERM

	got := drainInterruptedSignal(interruptedSignal)
	if got != syscall.SIGTERM {
		t.Fatalf("expected SIGTERM, got %v", got)
	}
}

func TestDrainInterruptedSignalReturnsNilWhenEmpty(t *testing.T) {
	if got := drainInterruptedSignal(make(chan os.Signal, 1)); got != nil {
		t.Fatalf("expected nil signal, got %v", got)
	}
}

func TestDeferBuildArtifactCleanupUsesFinalBuildSuccessValue(t *testing.T) {
	got := false

	func() {
		buildSucceeded := false
		defer deferBuildArtifactCleanup(func(success bool) {
			got = success
		}, &buildSucceeded)()

		buildSucceeded = true
	}()

	if !got {
		t.Fatal("expected deferred cleanup to observe final successful build state")
	}
}

func TestLogBuildOutputLineEnablesAuxiliaryProcessSilenceOnMarker(t *testing.T) {
	t.Parallel()

	auxiliaryProcessSilent := &atomic.Bool{}

	logBuildOutputLine(auxiliaryLogSilenceStartMarker, "stdout", VirtualMachineConfig{}, auxiliaryProcessSilent)

	if !auxiliaryProcessSilent.Load() {
		t.Fatal("expected auxiliary process silence to be enabled")
	}
}

func TestLogBuildOutputLineDisablesAuxiliaryProcessSilenceOnMarker(t *testing.T) {
	t.Parallel()

	auxiliaryProcessSilent := &atomic.Bool{}
	auxiliaryProcessSilent.Store(true)

	logBuildOutputLine(auxiliaryLogSilenceEndMarker, "stdout", VirtualMachineConfig{}, auxiliaryProcessSilent)

	if auxiliaryProcessSilent.Load() {
		t.Fatal("expected auxiliary process silence to be disabled")
	}
}
