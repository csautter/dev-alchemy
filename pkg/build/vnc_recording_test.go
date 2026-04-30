package build

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFfmpegImagePipeArgsStreamsMjpegToFragmentedMp4(t *testing.T) {
	t.Parallel()

	outputFile := filepath.Join(t.TempDir(), "qemu.vnc.mp4")
	args := ffmpegImagePipeArgs(outputFile)

	for _, want := range []string{"image2pipe", "mjpeg", "pipe:0", "libx264", "+faststart", outputFile} {
		if !slices.Contains(args, want) {
			t.Fatalf("expected ffmpeg args to contain %q, got %v", want, args)
		}
	}
}

func TestRunFfmpegVideoGenerationProcessSkipsStreamedVideo(t *testing.T) {
	t.Parallel()

	videoFile := filepath.Join(t.TempDir(), "qemu.vnc.mp4")
	if err := os.WriteFile(videoFile, []byte("not a real mp4, just a sentinel"), 0600); err != nil {
		t.Fatalf("failed to write sentinel video file: %v", err)
	}

	ctx := context.Background()
	got := RunFfmpegVideoGenerationProcess(VirtualMachineConfig{}, ctx, RunProcessConfig{}, &VncRecordingConfig{
		OutputVideoFile: videoFile,
		StreamedToVideo: true,
	})

	if got != ctx {
		t.Fatal("expected streamed video post-processing to return the original context")
	}
}

func TestFeedAvailableVncSnapshotFramesKeepsNewestUntilFinalFlush(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	snapshotFile := filepath.Join(dir, "qemu.vnc.jpg")
	firstFrame := filepath.Join(dir, "qemu.vnc00001.jpg")
	secondFrame := filepath.Join(dir, "qemu.vnc00002.jpg")

	if err := os.WriteFile(firstFrame, []byte("first"), 0600); err != nil {
		t.Fatalf("failed to write first frame: %v", err)
	}
	if err := os.WriteFile(secondFrame, []byte("second"), 0600); err != nil {
		t.Fatalf("failed to write second frame: %v", err)
	}

	var out bytes.Buffer
	written, err := feedAvailableVncSnapshotFrames(snapshotFile, &out, false)
	if err != nil {
		t.Fatalf("failed to feed available frames: %v", err)
	}
	if written != 1 {
		t.Fatalf("expected to feed one stable frame, fed %d", written)
	}
	if out.String() != "first" {
		t.Fatalf("expected first frame content, got %q", out.String())
	}
	if _, err := os.Stat(firstFrame); !os.IsNotExist(err) {
		t.Fatalf("expected first frame to be removed, stat err: %v", err)
	}
	if _, err := os.Stat(secondFrame); err != nil {
		t.Fatalf("expected newest frame to remain, stat err: %v", err)
	}

	written, err = feedAvailableVncSnapshotFrames(snapshotFile, &out, true)
	if err != nil {
		t.Fatalf("failed to flush remaining frame: %v", err)
	}
	if written != 1 {
		t.Fatalf("expected to flush one remaining frame, flushed %d", written)
	}
	if out.String() != "firstsecond" {
		t.Fatalf("expected concatenated frame content, got %q", out.String())
	}
	if _, err := os.Stat(secondFrame); !os.IsNotExist(err) {
		t.Fatalf("expected second frame to be removed, stat err: %v", err)
	}
}
