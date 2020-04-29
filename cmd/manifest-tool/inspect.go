package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/docker/distribution/reference"
	"github.com/estesp/manifest-tool/pkg/registry"
	"github.com/estesp/manifest-tool/pkg/store"
	"github.com/estesp/manifest-tool/pkg/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var inspectCmd = cli.Command{
	Name:  "inspect",
	Usage: "fetch image manifests in a container registry",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "raw",
			Usage: "raw JSON output",
		},
	},
	Action: func(c *cli.Context) {

		name := c.Args().First()
		imageRef, err := parseName(name)
		if err != nil {
			logrus.Fatal(err)
		}

		memoryStore := store.NewMemoryStore()
		descriptor, err := fetchDescriptor(c, memoryStore, imageRef)
		if err != nil {
			logrus.Error(err)
		}

		if c.Bool("raw") {
			out, err := json.Marshal(descriptor)
			if err != nil {
				logrus.Fatal(err)
			}
			fmt.Println(string(out))
			return
		}
		_, db, _ := memoryStore.Get(descriptor)
		switch descriptor.MediaType {
		case ocispec.MediaTypeImageIndex, types.MediaTypeDockerSchema2ManifestList:
			// this is a multi-platform image descriptor; marshal to Index type
			var idx ocispec.Index
			if err := json.Unmarshal(db, &idx); err != nil {
				logrus.Fatal(err)
			}
			outputList(name, memoryStore, descriptor, idx)
		case ocispec.MediaTypeImageManifest, types.MediaTypeDockerSchema2Manifest:
			var man ocispec.Manifest
			if err := json.Unmarshal(db, &man); err != nil {
				logrus.Fatal(err)
			}
			_, cb, _ := memoryStore.Get(man.Config)
			var conf ocispec.Image
			if err := json.Unmarshal(cb, &conf); err != nil {
				logrus.Fatal(err)
			}
			outputImage(name, descriptor, man, conf)
		default:
			logrus.Errorf("Unknown descriptor type: %s", descriptor.MediaType)
		}
	},
}

func outputList(name string, cs *store.MemoryStore, descriptor ocispec.Descriptor, index ocispec.Index) {
	fmt.Printf("Name:   %s (Type: %s)\n", name, descriptor.MediaType)
	fmt.Printf("Digest: %s\n", descriptor.Digest)
	fmt.Printf(" * Contains %d manifest references:\n", len(index.Manifests))
	for i, img := range index.Manifests {
		fmt.Printf("%d    Mfst Type: %s\n", i+1, img.MediaType)
		fmt.Printf("%d       Digest: %s\n", i+1, img.Digest)
		fmt.Printf("%d  Mfst Length: %d\n", i+1, img.Size)
		_, db, _ := cs.Get(img)
		switch img.MediaType {
		case ocispec.MediaTypeImageManifest, types.MediaTypeDockerSchema2Manifest:
			var man ocispec.Manifest
			if err := json.Unmarshal(db, &man); err != nil {
				logrus.Fatal(err)
			}
			fmt.Printf("%d     Platform:\n", i+1)
			fmt.Printf("%d           -      OS: %s\n", i+1, img.Platform.OS)
			fmt.Printf("%d           - OS Vers: %s\n", i+1, img.Platform.OSVersion)
			fmt.Printf("%d           - OS Feat: %s\n", i+1, img.Platform.OSFeatures)
			fmt.Printf("%d           -    Arch: %s\n", i+1, img.Platform.Architecture)
			fmt.Printf("%d           - Variant: %s\n", i+1, img.Platform.Variant)
			fmt.Printf("%d     # Layers: %d\n", i+1, len(man.Layers))
			for j, layer := range man.Layers {
				fmt.Printf("         layer %d: digest = %s\n", j+1, layer.Digest)
			}
			fmt.Println()
		default:
			fmt.Printf("Unknown media type for further display: %s\n", img.MediaType)
		}

	}
}

func outputImage(name string, descriptor ocispec.Descriptor, manifest ocispec.Manifest, config ocispec.Image) {
	fmt.Printf("Name: %s (Type: %s)\n", name, descriptor.MediaType)
	fmt.Printf("      Digest: %s\n", descriptor.Digest)
	fmt.Printf("          OS: %s\n", config.OS)
	fmt.Printf("        Arch: %s\n", config.Architecture)
	fmt.Printf("    # Layers: %d\n", len(manifest.Layers))
	for i, layer := range manifest.Layers {
		fmt.Printf("      layer %d: digest = %s\n", i+1, layer.Digest)
	}
}

func allMediaTypes() []string {
	return []string{
		types.MediaTypeDockerSchema2Manifest,
		types.MediaTypeDockerSchema2ManifestList,
		ocispec.MediaTypeImageManifest,
		ocispec.MediaTypeImageIndex,
	}
}

func fetchDescriptor(c *cli.Context, memoryStore *store.MemoryStore, imageRef reference.Named) (ocispec.Descriptor, error) {
	resolver := newResolver(c.GlobalString("username"), c.GlobalString("password"), c.GlobalBool("insecure"),
		filepath.Join(c.GlobalString("docker-cfg"), "config.json"))
	return registry.Fetch(context.Background(), memoryStore, types.NewRequest(imageRef, "", allMediaTypes(), resolver))
}
