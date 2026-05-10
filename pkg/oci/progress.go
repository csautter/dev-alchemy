package oci

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
)

// TransferProgress receives aggregate byte progress for OCI push and pull
// operations. Implementations must be safe for concurrent Add calls.
type TransferProgress interface {
	Start(totalBytes int64)
	Add(bytes int64)
	Done(success bool)
}

// TransferStatus receives human-readable phase updates for work that happens
// before or after the byte transfer itself.
type TransferStatus interface {
	Status(message string)
}

func reportTransferStatus(progress TransferProgress, format string, args ...any) {
	reporter, ok := progress.(TransferStatus)
	if !ok {
		return
	}
	reporter.Status(fmt.Sprintf(format, args...))
}

func copyArtifact(
	ctx context.Context,
	src oras.ReadOnlyTarget,
	srcRef string,
	dst oras.Target,
	dstRef string,
	totalBytes int64,
	progress TransferProgress,
) (desc ocispec.Descriptor, err error) {
	copyOptions := oras.DefaultCopyOptions
	if progress == nil {
		return oras.Copy(ctx, src, srcRef, dst, dstRef, copyOptions)
	}

	progress = newCappedTransferProgress(progress, totalBytes)
	progress.Start(totalBytes)
	defer func() {
		progress.Done(err == nil)
	}()

	src = progressReadOnlyTarget{
		ReadOnlyTarget: src,
		progress:       progress,
	}
	copyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		addProgress(progress, desc.Size)
		return nil
	}
	copyOptions.OnMounted = func(ctx context.Context, desc ocispec.Descriptor) error {
		addProgress(progress, desc.Size)
		return nil
	}

	return oras.Copy(ctx, src, srcRef, dst, dstRef, copyOptions)
}

func descriptorTotal(descs ...ocispec.Descriptor) int64 {
	var total int64
	for _, desc := range descs {
		if desc.Size > 0 {
			total += desc.Size
		}
	}
	return total
}

type cappedTransferProgress struct {
	progress TransferProgress
	total    int64
	current  atomic.Int64
}

func newCappedTransferProgress(progress TransferProgress, totalBytes int64) *cappedTransferProgress {
	return &cappedTransferProgress{
		progress: progress,
		total:    totalBytes,
	}
}

func (p *cappedTransferProgress) Start(totalBytes int64) {
	p.progress.Start(totalBytes)
}

func (p *cappedTransferProgress) Add(bytes int64) {
	if bytes <= 0 {
		return
	}
	if p.total <= 0 {
		p.progress.Add(bytes)
		return
	}

	for {
		current := p.current.Load()
		remaining := p.total - current
		if remaining <= 0 {
			return
		}

		delta := min(bytes, remaining)
		if p.current.CompareAndSwap(current, current+delta) {
			p.progress.Add(delta)
			return
		}
	}
}

func (p *cappedTransferProgress) Done(success bool) {
	p.progress.Done(success)
}

type progressReadOnlyTarget struct {
	oras.ReadOnlyTarget
	progress TransferProgress
}

func (t progressReadOnlyTarget) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	rc, err := t.ReadOnlyTarget.Fetch(ctx, target)
	if err != nil {
		return nil, err
	}
	return progressReadCloser{
		ReadCloser: rc,
		progress:   t.progress,
	}, nil
}

type progressReadCloser struct {
	io.ReadCloser
	progress TransferProgress
}

func (r progressReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	addProgress(r.progress, int64(n))
	return n, err
}

func addProgress(progress TransferProgress, bytes int64) {
	if progress != nil && bytes > 0 {
		progress.Add(bytes)
	}
}
