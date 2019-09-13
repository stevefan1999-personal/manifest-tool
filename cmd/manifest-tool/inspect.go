package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deislabs/oras/pkg/content"
	"github.com/estesp/manifest-tool/pkg/registry"
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

		memoryStore := content.NewMemoryStore()
		// a := getAuthInfo(c)
		descriptor, err := registry.Fetch(context.Background(), memoryStore, types.NewRequest(imageRef, "", allMediaTypes()))
		if err != nil {
			logrus.Error(err)
		}
		_, db, _ := memoryStore.Get(descriptor)
		switch descriptor.MediaType {
		case ocispec.MediaTypeImageIndex, types.MediaTypeDockerSchema2ManifestList:
			fmt.Printf("OCI index/Docker v2 manifest list: %s\n", descriptor.Digest)
			var idx ocispec.Index
			if err := json.Unmarshal(db, &idx); err != nil {
				logrus.Fatal(err)
			}
			fmt.Printf("%v\n", idx)

		case ocispec.MediaTypeImageManifest, types.MediaTypeDockerSchema2Manifest:
			fmt.Printf("OCI image manifest: %s\n", descriptor.Digest)
			var man ocispec.Manifest
			if err := json.Unmarshal(db, &man); err != nil {
				logrus.Fatal(err)
			}
			fmt.Printf("%v\n", man)
		}

		if c.Bool("raw") {
			out, err := json.Marshal(descriptor)
			if err != nil {
				logrus.Fatal(err)
			}
			fmt.Println(string(out))
			return
		}
		// output basic informative details about the image
		/*
			if len(imgInspect) == 1 {
				// this is a basic single manifest
				fmt.Printf("%s: manifest type: %s\n", name, imgInspect[0].MediaType)
				fmt.Printf("      Digest: %s\n", imgInspect[0].Digest)
				fmt.Printf("Architecture: %s\n", imgInspect[0].Architecture)
				fmt.Printf("          OS: %s\n", imgInspect[0].Os)
				fmt.Printf("    # Layers: %d\n", len(imgInspect[0].Layers))
				for i, digest := range imgInspect[0].Layers {
					fmt.Printf("      layer %d: digest = %s\n", i+1, digest)
				}
				return
			}
			// more than one response--this is a manifest list
			fmt.Printf("Name:   %s (Type: %s)\n", name, imgInspect[0].MediaType)
			fmt.Printf("Digest: %s\n", imgInspect[0].Digest)
			fmt.Printf(" * Contains %d manifest references:\n", len(imgInspect)-1)
			for i, img := range imgInspect[1:] {
				fmt.Printf("%d    Mfst Type: %s\n", i+1, img.MediaType)
				fmt.Printf("%d       Digest: %s\n", i+1, img.Digest)
				fmt.Printf("%d  Mfst Length: %d\n", i+1, img.Size)
				fmt.Printf("%d     Platform:\n", i+1)
				fmt.Printf("%d           -      OS: %s\n", i+1, img.Platform.OS)
				fmt.Printf("%d           - OS Vers: %s\n", i+1, img.Platform.OSVersion)
				fmt.Printf("%d           - OS Feat: %s\n", i+1, img.Platform.OSFeatures)
				fmt.Printf("%d           -    Arch: %s\n", i+1, img.Platform.Architecture)
				fmt.Printf("%d           - Variant: %s\n", i+1, img.Platform.Variant)
				fmt.Printf("%d           - Feature: %s\n", i+1, strings.Join(img.Platform.Features, ","))
				fmt.Printf("%d     # Layers: %d\n", i+1, len(img.Layers))
				for j, digest := range img.Layers {
					fmt.Printf("         layer %d: digest = %s\n", j+1, digest)
				}
				fmt.Println()
			}
		*/
	},
}

func allMediaTypes() []string {
	return []string{
		types.MediaTypeDockerSchema2Manifest,
		types.MediaTypeDockerSchema2ManifestList,
		ocispec.MediaTypeImageManifest,
		ocispec.MediaTypeImageIndex,
	}
}
