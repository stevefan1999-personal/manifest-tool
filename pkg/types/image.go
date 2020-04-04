package types

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Image struct handles Windows support extensions to OCI spec
type Image struct {
	ocispec.Image
	OSVersion  string   `json:"os.version,omitempty"`
	OSFeatures []string `json:"os.features,omitempty"`
}
