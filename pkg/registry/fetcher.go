package registry

import (
	"context"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/estesp/manifest-tool/pkg/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Fetch uses a registry (distribution spec) API to retrieve a specific image manifest from a registry
func Fetch(ctx context.Context, cs *content.Memorystore, image *types.Request) (ocispec.Descriptor, error) {

	resolver := docker.NewResolver(docker.ResolverOptions{})

	// Retrieve manifest from registry
	name, desc, err := resolver.Resolve(ctx, image.Reference().String())
	if err != nil {
		panic(err)
	}
	fetcher, err := resolver.Fetcher(ctx, name)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	r, err := fetcher.Fetch(ctx, desc)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer r.Close()

	// Handler which reads a descriptor and fetches the referenced data (e.g. image layers) from the remote
	h := remotes.FetchHandler(cs, fetcher)
	// This traverses the OCI descriptor to fetch the image and store it into the local store initialized above.
	// All content hashes are verified in this step
	if err := images.Dispatch(ctx, h, nil, desc); err != nil {
		return ocispec.Descriptor{}, err
	}
	return desc, nil
}
