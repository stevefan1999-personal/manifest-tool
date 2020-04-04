package types

import (
	"github.com/deislabs/oras/pkg/content"
	"github.com/docker/distribution/reference"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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
	Manifests []Manifest
}

// Manifest is an ocispec.Descriptor of media type manifest (OCI or Docker)
// along with a boolean to help determine whether a reference to the manifest
// must be pushed to the target (manifest list) repo location before finalizing
// the manifest list push operation.
type Manifest struct {
	ocispec.Descriptor
	PushRef bool
}

// Push handles the registry interactions required to push
// any required manifest references followed by the OCI "index" or
func (m ManifestList) Push(ms *content.Memorystore) (string, int, error) {
	return "", 0, nil
}
