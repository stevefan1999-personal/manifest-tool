package registry

import (
	"context"
	"fmt"

	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/estesp/manifest-tool/pkg/types"
	"github.com/containerd/containerd/remotes/docker"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func Fetch(types.Image image) (ocispec.Descriptor, error) {

	ctx := context.Background()
	resolver := docker.NewResolver(docker.ResolverOptions{})
	memoryStore := content.NewMemoryStore()
	// Push file(s) w custom mediatype to registry
	allowedMediaTypes := []string{image.MediaType()}
	desc, _, err = oras.Pull(ctx, resolver, image.Reference(), memoryStore, oras.WithAllowedMediaTypes(allowedMediaTypes))
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	return desc
}