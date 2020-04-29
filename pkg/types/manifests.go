package types

import (
	"context"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/docker/distribution/reference"
	"github.com/estesp/manifest-tool/pkg/store"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ManifestType specifies whether to use the OCI index media type and
// format or the Docker manifestList media type and format for the
// registry push operation.
type ManifestType int

const (
	// OCI is used to specify the "index" type
	OCI ManifestType = iota
	// Docker is used for the "manifestList" type
	Docker
)

// ManifestList represents the information necessary to assemble and
// push the right data to a registry to form a manifestlist or OCI index
// entry.
type ManifestList struct {
	Name      string
	Type      ManifestType
	Reference reference.Named
	Resolver  remotes.Resolver
	Manifests []Manifest
}

// Manifest is an ocispec.Descriptor of media type manifest (OCI or Docker)
// along with a boolean to help determine whether a reference to the manifest
// must be pushed to the target (manifest list) repo location before finalizing
// the manifest list push operation.
type Manifest struct {
	Descriptor ocispec.Descriptor
	PushRef    bool
}

// Push handles the registry interactions required to push
// any required manifest references followed by the OCI "index"
// or Docker v2 "manifest list" itself
func (m ManifestList) Push(ms *store.MemoryStore) (string, int, error) {
	// push manifest references to target ref (if required)
	for _, man := range m.Manifests {
		if man.PushRef {
			ref, err := reference.Parse(reference.TrimNamed(m.Reference).String() + "@" + man.Descriptor.Digest.String())
			if err != nil {
				return "", 0, errors.Wrapf(err, "Error parsing reference for target manifest component push: %s", m.Reference.String())
			}
			err = push(ref, man.Descriptor, m.Resolver, ms)
			if err != nil {
				return "", 0, errors.Wrapf(err, "Error pushing target manifest component reference: %s", ref.String())
			}
			logrus.Infof("pushed manifest component reference (%s) to target namespace: %s", man.Descriptor.Digest.String(), ref.String())
		}
	}
	return "", 0, nil
}

func push(ref reference.Reference, desc ocispec.Descriptor, resolver remotes.Resolver, ms *store.MemoryStore) error {
	ctx := context.Background()
	pusher, err := resolver.Pusher(ctx, ref.String())
	if err != nil {
		return err
	}
	var wrapper func(images.Handler) images.Handler

	return remotes.PushContent(ctx, pusher, desc, ms, nil, wrapper)
}
