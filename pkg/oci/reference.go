package oci

import (
	"fmt"

	"oras.land/oras-go/v2/registry"
)

type remoteReference struct {
	repository string
	reference  string
	registry   string
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
