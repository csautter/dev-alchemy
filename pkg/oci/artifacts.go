package oci

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	ArtifactType = "application/vnd.dev-alchemy.vm-build.v1"

	MediaTypeArtifact   = "application/vnd.dev-alchemy.vm-build.artifact.v1"
	MediaTypeQCOW2      = "application/vnd.dev-alchemy.vm-build.qcow2.v1"
	MediaTypeVagrantBox = "application/vnd.dev-alchemy.vm-build.vagrant-box.v1"

	AnnotationVMOS                   = "dev.alchemy.vm.os"
	AnnotationVMType                 = "dev.alchemy.vm.type"
	AnnotationVMArch                 = "dev.alchemy.vm.arch"
	AnnotationVMHostOS               = "dev.alchemy.vm.host_os"
	AnnotationVMVirtualizationEngine = "dev.alchemy.vm.virtualization_engine"
	AnnotationVMSlug                 = "dev.alchemy.vm.slug"
)

type RegistryOptions struct {
	PlainHTTP                bool
	InsecureSkipTLSVerify    bool
	CAFile                   string
	Username                 string
	Password                 string
	AccessToken              string
	RefreshToken             string
	DisableDockerCredentials bool
}

type PushOptions struct {
	RegistryOptions
	Progress TransferProgress
}

type PullOptions struct {
	RegistryOptions
	Progress                  TransferProgress
	ConfirmForeignArtifactUse ForeignArtifactConfirmation
}

// ForeignArtifactConfirmation decides whether a compatible artifact built for a
// different host OS may be pulled into the requested local artifact slot.
type ForeignArtifactConfirmation func(context.Context, ForeignArtifact) (bool, error)

type ForeignArtifact struct {
	OS                             string
	UbuntuType                     string
	Arch                           string
	SourceHostOS                   alchemy_build.HostOsType
	SourceVirtualizationEngine     alchemy_build.VirtualizationEngine
	TargetHostOS                   alchemy_build.HostOsType
	TargetVirtualizationEngine     alchemy_build.VirtualizationEngine
	SourceHostOSAnnotation         string
	SourceVirtualizationAnnotation string
}

type ArtifactFile struct {
	Name      string
	Path      string
	MediaType string
	Digest    string
	Size      int64
}

type TransferResult struct {
	Reference string
	Digest    string
	MediaType string
	Size      int64
	Artifacts []ArtifactFile
}

type artifactLayout struct {
	root  string
	files []ArtifactFile
}

type remoteReference struct {
	repository string
	reference  string
	registry   string
}

type artifactReplacement struct {
	finalPath string
	backup    string
	replaced  bool
}

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

func Push(ctx context.Context, vm alchemy_build.VirtualMachineConfig, reference string, opts PushOptions) (TransferResult, error) {
	reportTransferStatus(opts.Progress, "Resolving local artifact paths")
	layout, err := resolveArtifactLayout(vm)
	if err != nil {
		return TransferResult{}, err
	}

	reportTransferStatus(opts.Progress, "Parsing OCI reference %s", reference)
	remoteRef, err := parsePushReference(reference)
	if err != nil {
		return TransferResult{}, err
	}

	reportTransferStatus(opts.Progress, "Preparing local OCI artifact store")
	fs, err := file.New(layout.root)
	if err != nil {
		return TransferResult{}, fmt.Errorf("create OCI file store: %w", err)
	}
	defer fs.Close()

	layers := make([]ocispec.Descriptor, 0, len(layout.files))
	files := make([]ArtifactFile, 0, len(layout.files))
	for _, artifact := range layout.files {
		reportTransferStatus(opts.Progress, "Hashing local artifact %s", artifact.Path)
		desc, err := fs.Add(ctx, artifact.Name, artifact.MediaType, artifact.Path)
		if err != nil {
			return TransferResult{}, fmt.Errorf("add artifact %s to OCI store: %w", artifact.Path, err)
		}
		artifact.Digest = desc.Digest.String()
		artifact.Size = desc.Size
		layers = append(layers, desc)
		files = append(files, artifact)
	}

	reportTransferStatus(opts.Progress, "Packing OCI artifact manifest")
	manifestDesc, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, ArtifactType, oras.PackManifestOptions{
		Layers:              layers,
		ManifestAnnotations: manifestAnnotations(vm),
	})
	if err != nil {
		return TransferResult{}, fmt.Errorf("pack OCI artifact manifest: %w", err)
	}
	if err := fs.Tag(ctx, manifestDesc, remoteRef.reference); err != nil {
		return TransferResult{}, fmt.Errorf("tag local OCI artifact: %w", err)
	}

	reportTransferStatus(opts.Progress, "Preparing OCI registry client")
	repo, err := newRepository(remoteRef, opts.RegistryOptions)
	if err != nil {
		return TransferResult{}, err
	}

	reportTransferStatus(opts.Progress, "Uploading OCI artifact")
	pushedDesc, err := copyArtifact(ctx, fs, remoteRef.reference, repo, remoteRef.reference, descriptorTotal(append(layers, manifestDesc)...), opts.Progress)
	if err != nil {
		return TransferResult{}, fmt.Errorf("push OCI artifact %s: %w", reference, err)
	}

	return transferResult(reference, pushedDesc, files), nil
}

func Pull(ctx context.Context, vm alchemy_build.VirtualMachineConfig, reference string, opts PullOptions) (TransferResult, error) {
	reportTransferStatus(opts.Progress, "Resolving local artifact paths")
	layout, err := resolveArtifactLayout(vm)
	if err != nil {
		return TransferResult{}, err
	}

	reportTransferStatus(opts.Progress, "Parsing OCI reference %s", reference)
	remoteRef, err := parsePullReference(reference)
	if err != nil {
		return TransferResult{}, err
	}

	reportTransferStatus(opts.Progress, "Preparing OCI registry client")
	repo, err := newRepository(remoteRef, opts.RegistryOptions)
	if err != nil {
		return TransferResult{}, err
	}
	reportTransferStatus(opts.Progress, "Resolving and validating OCI artifact manifest")
	remoteManifest, err := validateRemoteManifest(ctx, repo, remoteRef.reference, vm, layout.files, opts)
	if err != nil {
		return TransferResult{}, err
	}
	manifestDesc := remoteManifest.descriptor

	if err := os.MkdirAll(layout.root, 0o700); err != nil {
		return TransferResult{}, fmt.Errorf("create artifact root %s: %w", layout.root, err)
	}
	reportTransferStatus(opts.Progress, "Preparing OCI pull staging directory")
	stagingRoot, err := os.MkdirTemp(layout.root, ".dev-alchemy-oci-pull-*")
	if err != nil {
		return TransferResult{}, fmt.Errorf("create OCI pull staging directory: %w", err)
	}
	defer os.RemoveAll(stagingRoot)

	fs, err := file.New(stagingRoot)
	if err != nil {
		return TransferResult{}, fmt.Errorf("create OCI file store: %w", err)
	}
	defer fs.Close()

	reportTransferStatus(opts.Progress, "Downloading OCI artifact")
	if _, err := copyArtifact(ctx, repo, remoteRef.reference, fs, "pulled", descriptorTotal(append(remoteManifest.layers, manifestDesc)...), opts.Progress); err != nil {
		return TransferResult{}, fmt.Errorf("pull OCI artifact %s: %w", reference, err)
	}

	reportTransferStatus(opts.Progress, "Promoting pulled artifacts into the local cache")
	if err := promotePulledArtifacts(stagingRoot, layout.files); err != nil {
		return TransferResult{}, err
	}

	pulledFiles := slices.Clone(layout.files)
	for i := range pulledFiles {
		info, err := os.Stat(pulledFiles[i].Path)
		if err != nil {
			return TransferResult{}, fmt.Errorf("inspect pulled artifact %s: %w", pulledFiles[i].Path, err)
		}
		pulledFiles[i].Size = info.Size()
	}

	return transferResult(reference, manifestDesc, pulledFiles), nil
}

func ArtifactFiles(vm alchemy_build.VirtualMachineConfig) ([]ArtifactFile, error) {
	layout, err := resolveArtifactLayout(vm)
	if err != nil {
		return nil, err
	}
	return slices.Clone(layout.files), nil
}

func MediaTypeForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".qcow2":
		return MediaTypeQCOW2
	case ".box":
		return MediaTypeVagrantBox
	default:
		return MediaTypeArtifact
	}
}

func resolveArtifactLayout(vm alchemy_build.VirtualMachineConfig) (artifactLayout, error) {
	paths, err := expectedArtifactPaths(vm)
	if err != nil {
		return artifactLayout{}, err
	}
	root, err := artifactRoot(paths)
	if err != nil {
		return artifactLayout{}, err
	}

	files := make([]ArtifactFile, 0, len(paths))
	for _, artifactPath := range paths {
		absPath, err := filepath.Abs(filepath.Clean(artifactPath))
		if err != nil {
			return artifactLayout{}, fmt.Errorf("resolve artifact path %s: %w", artifactPath, err)
		}
		name, err := relativeArtifactName(root, absPath)
		if err != nil {
			return artifactLayout{}, err
		}
		files = append(files, ArtifactFile{
			Name:      name,
			Path:      absPath,
			MediaType: MediaTypeForPath(absPath),
		})
	}

	return artifactLayout{root: root, files: files}, nil
}

func expectedArtifactPaths(vm alchemy_build.VirtualMachineConfig) ([]string, error) {
	if len(vm.ExpectedBuildArtifacts) > 0 {
		return slices.Clone(vm.ExpectedBuildArtifacts), nil
	}

	for _, candidate := range alchemy_build.AvailableVirtualMachineConfigs() {
		if candidate.HostOs == vm.HostOs &&
			candidate.OS == vm.OS &&
			candidate.UbuntuType == vm.UbuntuType &&
			candidate.Arch == vm.Arch &&
			candidate.VirtualizationEngine == vm.VirtualizationEngine &&
			len(candidate.ExpectedBuildArtifacts) > 0 {
			return slices.Clone(candidate.ExpectedBuildArtifacts), nil
		}
	}

	return nil, fmt.Errorf(
		"no OCI build artifacts defined for OS=%s type=%s arch=%s host_os=%s engine=%s",
		vm.OS,
		vm.UbuntuType,
		vm.Arch,
		vm.HostOs,
		vm.VirtualizationEngine,
	)
}

func artifactRoot(paths []string) (string, error) {
	cacheDir := alchemy_build.GetDirectoriesInstance().GetDirectories().CacheDir
	if cacheDir != "" && allPathsWithin(cacheDir, paths) {
		return filepath.Abs(filepath.Clean(cacheDir))
	}
	return commonArtifactParent(paths)
}

func allPathsWithin(root string, paths []string) bool {
	for _, path := range paths {
		if !pathWithin(root, path) {
			return false
		}
	}
	return true
}

func commonArtifactParent(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", errors.New("no artifact paths provided")
	}

	root, err := filepath.Abs(filepath.Dir(filepath.Clean(paths[0])))
	if err != nil {
		return "", err
	}
	for _, artifactPath := range paths[1:] {
		absPath, err := filepath.Abs(filepath.Clean(artifactPath))
		if err != nil {
			return "", err
		}
		for !pathWithin(root, absPath) {
			next := filepath.Dir(root)
			if next == root {
				return "", fmt.Errorf("could not determine common artifact root for %v", paths)
			}
			root = next
		}
	}
	return root, nil
}

func relativeArtifactName(root string, artifactPath string) (string, error) {
	rel, err := filepath.Rel(root, artifactPath)
	if err != nil {
		return "", fmt.Errorf("make artifact path relative to %s: %w", root, err)
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") || rel == ".." {
		return "", fmt.Errorf("artifact path %s is outside artifact root %s", artifactPath, root)
	}
	return rel, nil
}

func pathWithin(root string, path string) bool {
	absRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == "." || (!strings.HasPrefix(rel, "../") && rel != "..")
}

func manifestAnnotations(vm alchemy_build.VirtualMachineConfig) map[string]string {
	slugVM := vm
	slug := alchemy_build.GenerateVirtualMachineSlug(&slugVM)
	return map[string]string{
		ocispec.AnnotationTitle:              "dev-alchemy-" + slug,
		ocispec.AnnotationCreated:            time.Now().UTC().Format(time.RFC3339),
		ocispec.AnnotationVendor:             "dev-alchemy",
		ocispec.AnnotationDescription:        "Dev Alchemy VM build artifacts",
		ocispec.AnnotationDocumentation:      "https://github.com/csautter/dev-alchemy",
		ocispec.AnnotationSource:             "https://github.com/csautter/dev-alchemy",
		ocispec.AnnotationAuthors:            "Dev Alchemy",
		ocispec.AnnotationRefName:            slug,
		AnnotationVMOS:                       vm.OS,
		AnnotationVMType:                     vm.UbuntuType,
		AnnotationVMArch:                     vm.Arch,
		AnnotationVMHostOS:                   string(vm.HostOs),
		AnnotationVMVirtualizationEngine:     string(vm.VirtualizationEngine),
		AnnotationVMSlug:                     slug,
		"org.opencontainers.image.component": "vm-build-artifact",
	}
}

func parsePushReference(reference string) (remoteReference, error) {
	remoteRef, err := parseRemoteReference(reference)
	if err != nil {
		return remoteReference{}, err
	}
	parsed, err := registry.ParseReference(reference)
	if err != nil {
		return remoteReference{}, err
	}
	if parsed.Reference != "" {
		if err := parsed.ValidateReferenceAsTag(); err != nil {
			return remoteReference{}, fmt.Errorf("push reference %q must use a tag, not a digest: %w", reference, err)
		}
	}
	return remoteRef, nil
}

func parsePullReference(reference string) (remoteReference, error) {
	return parseRemoteReference(reference)
}

func parseRemoteReference(reference string) (remoteReference, error) {
	parsed, err := registry.ParseReference(reference)
	if err != nil {
		return remoteReference{}, fmt.Errorf("parse OCI reference %q: %w", reference, err)
	}
	if err := parsed.ValidateRegistry(); err != nil {
		return remoteReference{}, fmt.Errorf("invalid OCI registry in %q: %w", reference, err)
	}
	if err := parsed.ValidateRepository(); err != nil {
		return remoteReference{}, fmt.Errorf("invalid OCI repository in %q: %w", reference, err)
	}
	return remoteReference{
		repository: parsed.Registry + "/" + parsed.Repository,
		reference:  parsed.ReferenceOrDefault(),
		registry:   parsed.Registry,
	}, nil
}

func newRepository(ref remoteReference, opts RegistryOptions) (*remote.Repository, error) {
	repo, err := remote.NewRepository(ref.repository)
	if err != nil {
		return nil, fmt.Errorf("create remote OCI repository %s: %w", ref.repository, err)
	}
	repo.PlainHTTP = opts.PlainHTTP

	httpClient, err := registryHTTPClient(opts)
	if err != nil {
		return nil, err
	}
	credential, err := credentialFunc(ref.registry, opts)
	if err != nil {
		return nil, err
	}
	if credential != nil || httpClient != nil {
		repo.Client = &auth.Client{
			Client:     registryAuthHTTPClient(httpClient),
			Header:     auth.DefaultClient.Header.Clone(),
			Cache:      auth.NewCache(),
			Credential: credential,
		}
	}

	return repo, nil
}

func registryAuthHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return retry.DefaultClient
}

func registryHTTPClient(opts RegistryOptions) (*http.Client, error) {
	if !opts.InsecureSkipTLSVerify && opts.CAFile == "" {
		return nil, nil
	}

	tlsConfig := &tls.Config{ // #nosec G402 -- controlled by the explicit --insecure-skip-tls-verify registry option.
		InsecureSkipVerify: opts.InsecureSkipTLSVerify,
	}
	if opts.CAFile != "" {
		rootCAs, err := x509.SystemCertPool()
		if err != nil || rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}

		caBytes, err := os.ReadFile(opts.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read OCI registry CA file %s: %w", opts.CAFile, err)
		}
		if ok := rootCAs.AppendCertsFromPEM(caBytes); !ok {
			return nil, fmt.Errorf("parse OCI registry CA file %s: no PEM certificates found", opts.CAFile)
		}
		tlsConfig.RootCAs = rootCAs
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	return &http.Client{Transport: retry.NewTransport(transport)}, nil
}

func credentialFunc(registryHost string, opts RegistryOptions) (auth.CredentialFunc, error) {
	if opts.Username != "" || opts.Password != "" || opts.AccessToken != "" || opts.RefreshToken != "" {
		return auth.StaticCredential(registryHost, auth.Credential{
			Username:     opts.Username,
			Password:     opts.Password,
			AccessToken:  opts.AccessToken,
			RefreshToken: opts.RefreshToken,
		}), nil
	}
	if opts.DisableDockerCredentials {
		return nil, nil
	}

	store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err != nil {
		return nil, fmt.Errorf("load Docker OCI credentials: %w", err)
	}
	return credentials.Credential(store), nil
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

type remoteManifest struct {
	descriptor ocispec.Descriptor
	layers     []ocispec.Descriptor
}

func validateRemoteManifest(ctx context.Context, repo *remote.Repository, ref string, vm alchemy_build.VirtualMachineConfig, expected []ArtifactFile, opts PullOptions) (remoteManifest, error) {
	desc, err := repo.Resolve(ctx, ref)
	if err != nil {
		return remoteManifest{}, fmt.Errorf("resolve OCI artifact %s: %w", ref, err)
	}
	manifestBytes, err := content.FetchAll(ctx, repo, desc)
	if err != nil {
		return remoteManifest{}, fmt.Errorf("fetch OCI artifact manifest %s: %w", ref, err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return remoteManifest{}, fmt.Errorf("decode OCI artifact manifest %s: %w", ref, err)
	}
	if manifest.ArtifactType != ArtifactType {
		return remoteManifest{}, fmt.Errorf("OCI artifact %s has artifact type %q, expected %q", ref, manifest.ArtifactType, ArtifactType)
	}
	if err := validateManifestForPull(ctx, manifest, vm, expected, opts); err != nil {
		return remoteManifest{}, err
	}
	return remoteManifest{
		descriptor: desc,
		layers:     slices.Clone(manifest.Layers),
	}, nil
}

func validateManifestForPull(ctx context.Context, manifest ocispec.Manifest, vm alchemy_build.VirtualMachineConfig, expected []ArtifactFile, opts PullOptions) error {
	if err := validateManifestTarget(manifest, vm); err == nil {
		return validateManifestLayers(manifest, expected)
	} else if foreign, ok := compatibleForeignArtifact(manifest, vm); ok {
		if err := validateManifestLayers(manifest, expected); err != nil {
			return err
		}
		if opts.ConfirmForeignArtifactUse == nil {
			return err
		}
		confirmed, confirmErr := opts.ConfirmForeignArtifactUse(ctx, foreign)
		if confirmErr != nil {
			return fmt.Errorf("confirm foreign OCI artifact use: %w", confirmErr)
		}
		if !confirmed {
			return errors.New("foreign OCI artifact use cancelled")
		}
		return nil
	} else {
		return err
	}
}

func validateManifestTarget(manifest ocispec.Manifest, vm alchemy_build.VirtualMachineConfig) error {
	annotations := manifest.Annotations
	if annotations == nil {
		return errors.New("OCI artifact manifest is missing Dev Alchemy target annotations")
	}

	checks := manifestTargetChecks(vm)
	for _, check := range checks {
		if got := annotations[check.key]; got != check.want {
			return fmt.Errorf("OCI artifact target annotation %s=%q, expected %q", check.key, got, check.want)
		}
	}
	return nil
}

func manifestTargetChecks(vm alchemy_build.VirtualMachineConfig) []targetAnnotationCheck {
	return []targetAnnotationCheck{
		{key: AnnotationVMOS, want: vm.OS},
		{key: AnnotationVMType, want: vm.UbuntuType},
		{key: AnnotationVMArch, want: vm.Arch},
		{key: AnnotationVMHostOS, want: string(vm.HostOs)},
		{key: AnnotationVMVirtualizationEngine, want: string(vm.VirtualizationEngine)},
	}
}

type targetAnnotationCheck struct {
	key  string
	want string
}

func compatibleForeignArtifact(manifest ocispec.Manifest, vm alchemy_build.VirtualMachineConfig) (ForeignArtifact, bool) {
	annotations := manifest.Annotations
	if annotations == nil {
		return ForeignArtifact{}, false
	}
	if !isForeignArtifactGuestOS(vm.OS) {
		return ForeignArtifact{}, false
	}

	var foreignTargetMismatch bool
	for _, check := range manifestTargetChecks(vm) {
		got := annotations[check.key]
		if got == check.want {
			continue
		}
		switch check.key {
		case AnnotationVMHostOS, AnnotationVMVirtualizationEngine:
			foreignTargetMismatch = true
		default:
			return ForeignArtifact{}, false
		}
	}
	if !foreignTargetMismatch {
		return ForeignArtifact{}, false
	}

	sourceHostOS, ok := normalizeArtifactHostOS(annotations[AnnotationVMHostOS])
	if !ok || !isDarwinLinuxHostPair(sourceHostOS, vm.HostOs) {
		return ForeignArtifact{}, false
	}
	sourceEngine := annotations[AnnotationVMVirtualizationEngine]
	if sourceEngine == "" {
		return ForeignArtifact{}, false
	}

	return ForeignArtifact{
		OS:                             vm.OS,
		UbuntuType:                     vm.UbuntuType,
		Arch:                           vm.Arch,
		SourceHostOS:                   sourceHostOS,
		SourceVirtualizationEngine:     alchemy_build.VirtualizationEngine(sourceEngine),
		TargetHostOS:                   vm.HostOs,
		TargetVirtualizationEngine:     vm.VirtualizationEngine,
		SourceHostOSAnnotation:         annotations[AnnotationVMHostOS],
		SourceVirtualizationAnnotation: sourceEngine,
	}, true
}

func isForeignArtifactGuestOS(osName string) bool {
	normalized := strings.ToLower(osName)
	return normalized == "ubuntu" || strings.HasPrefix(normalized, "windows")
}

func normalizeArtifactHostOS(value string) (alchemy_build.HostOsType, bool) {
	switch strings.ToLower(value) {
	case "linux", string(alchemy_build.HostOsLinux):
		return alchemy_build.HostOsLinux, true
	case "macos", string(alchemy_build.HostOsDarwin):
		return alchemy_build.HostOsDarwin, true
	case string(alchemy_build.HostOsWindows):
		return alchemy_build.HostOsWindows, true
	default:
		return "", false
	}
}

func isDarwinLinuxHostPair(source alchemy_build.HostOsType, target alchemy_build.HostOsType) bool {
	return (source == alchemy_build.HostOsDarwin && target == alchemy_build.HostOsLinux) ||
		(source == alchemy_build.HostOsLinux && target == alchemy_build.HostOsDarwin)
}

func validateManifestLayers(manifest ocispec.Manifest, expected []ArtifactFile) error {
	expectedByName := make(map[string]ArtifactFile, len(expected))
	for _, file := range expected {
		expectedByName[file.Name] = file
	}

	seen := make(map[string]bool, len(manifest.Layers))
	for _, layer := range manifest.Layers {
		name := layer.Annotations[ocispec.AnnotationTitle]
		if name == "" {
			return errors.New("OCI artifact layer is missing title annotation")
		}
		expectedFile, ok := expectedByName[name]
		if !ok {
			return fmt.Errorf("OCI artifact contains unexpected layer %q", name)
		}
		if layer.MediaType != expectedFile.MediaType {
			return fmt.Errorf("OCI artifact layer %q has media type %q, expected %q", name, layer.MediaType, expectedFile.MediaType)
		}
		seen[name] = true
	}

	for _, expectedFile := range expected {
		if !seen[expectedFile.Name] {
			return fmt.Errorf("OCI artifact is missing expected layer %q", expectedFile.Name)
		}
	}
	return nil
}

func promotePulledArtifacts(stagingRoot string, files []ArtifactFile) error {
	replacements := make([]artifactReplacement, 0, len(files))
	for _, file := range files {
		stagedPath := filepath.Join(stagingRoot, filepath.FromSlash(file.Name))
		if _, err := os.Stat(stagedPath); err != nil {
			if rollbackErr := rollbackPulledArtifacts(replacements); rollbackErr != nil {
				return fmt.Errorf("inspect pulled artifact %s: %w; rollback failed: %v", stagedPath, err, rollbackErr)
			}
			if os.IsNotExist(err) {
				return fmt.Errorf("pulled artifact %s is missing from OCI staging directory", file.Name)
			}
			return fmt.Errorf("inspect pulled artifact %s: %w", stagedPath, err)
		}

		if err := os.MkdirAll(filepath.Dir(file.Path), 0o700); err != nil {
			if rollbackErr := rollbackPulledArtifacts(replacements); rollbackErr != nil {
				return fmt.Errorf("create artifact directory for %s: %w; rollback failed: %v", file.Path, err, rollbackErr)
			}
			return fmt.Errorf("create artifact directory for %s: %w", file.Path, err)
		}

		replacement := artifactReplacement{finalPath: file.Path}
		if _, err := os.Lstat(file.Path); err == nil {
			replacement.backup = fmt.Sprintf("%s.dev-alchemy-oci-backup-%d", file.Path, time.Now().UnixNano())
			if err := os.Rename(file.Path, replacement.backup); err != nil {
				if rollbackErr := rollbackPulledArtifacts(replacements); rollbackErr != nil {
					return fmt.Errorf("back up existing artifact %s: %w; rollback failed: %v", file.Path, err, rollbackErr)
				}
				return fmt.Errorf("back up existing artifact %s: %w", file.Path, err)
			}
			replacement.replaced = true
		} else if err != nil && !os.IsNotExist(err) {
			if rollbackErr := rollbackPulledArtifacts(replacements); rollbackErr != nil {
				return fmt.Errorf("inspect existing artifact %s: %w; rollback failed: %v", file.Path, err, rollbackErr)
			}
			return fmt.Errorf("inspect existing artifact %s: %w", file.Path, err)
		}

		replacements = append(replacements, replacement)
		if err := os.Rename(stagedPath, file.Path); err != nil {
			if rollbackErr := rollbackPulledArtifacts(replacements); rollbackErr != nil {
				return fmt.Errorf("promote pulled artifact %s to %s: %w; rollback failed: %v", stagedPath, file.Path, err, rollbackErr)
			}
			return fmt.Errorf("promote pulled artifact %s to %s: %w", stagedPath, file.Path, err)
		}
	}

	var cleanupErrs []error
	for _, replacement := range replacements {
		if replacement.replaced {
			if err := os.RemoveAll(replacement.backup); err != nil {
				cleanupErrs = append(cleanupErrs, fmt.Errorf("remove OCI pull backup %s: %w", replacement.backup, err))
			}
		}
	}
	return errors.Join(cleanupErrs...)
}

func rollbackPulledArtifacts(replacements []artifactReplacement) error {
	var rollbackErrs []error
	for i := len(replacements) - 1; i >= 0; i-- {
		replacement := replacements[i]
		if err := os.RemoveAll(replacement.finalPath); err != nil && !os.IsNotExist(err) {
			rollbackErrs = append(rollbackErrs, fmt.Errorf("remove partially pulled artifact %s: %w", replacement.finalPath, err))
			continue
		}
		if replacement.replaced {
			if err := os.Rename(replacement.backup, replacement.finalPath); err != nil && !os.IsNotExist(err) {
				rollbackErrs = append(rollbackErrs, fmt.Errorf("restore artifact backup %s: %w", replacement.backup, err))
			}
		}
	}
	return errors.Join(rollbackErrs...)
}

func transferResult(reference string, desc ocispec.Descriptor, files []ArtifactFile) TransferResult {
	return TransferResult{
		Reference: reference,
		Digest:    desc.Digest.String(),
		MediaType: desc.MediaType,
		Size:      desc.Size,
		Artifacts: slices.Clone(files),
	}
}
