package registry

import (
	"context"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/estesp/manifest-tool/pkg/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Fetch uses a registry (distribution spec) API to retrieve a specific image manifest from a registry
func Fetch(ctx context.Context, image *types.Request) (ocispec.Descriptor, error) {

	resolver := docker.NewResolver(docker.ResolverOptions{})
	memoryStore := content.NewMemoryStore()
	// Retrieve manifest from registry
	desc, _, err := oras.Pull(ctx, resolver, image.Reference().String(), memoryStore, oras.WithAllowedMediaTypes(image.MediaTypes()))
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	return desc, nil
}
