package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/estesp/manifest-tool/pkg/types"

	"github.com/deislabs/oras/pkg/content"
	"github.com/docker/distribution/reference"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"
)

var pushCmd = cli.Command{
	Name:  "push",
	Usage: "push a manifest list/OCI index entry to a registry with provided image details",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "type",
			Value: "docker",
			Usage: "image manifest type: docker (manifest list) or oci (index)",
		},
	},
	Subcommands: []cli.Command{
		{
			Name:  "from-spec",
			Usage: "push a manifest list to a registry via a YAML spec",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "ignore-missing",
					Usage: "only warn on missing images defined in YAML spec",
				},
			},
			Action: func(c *cli.Context) {
				filePath := c.Args().First()
				var yamlInput types.YAMLInput

				filename, err := filepath.Abs(filePath)
				if err != nil {
					logrus.Fatalf(fmt.Sprintf("Can't resolve path to %q: %v", filePath, err))
				}
				yamlFile, err := ioutil.ReadFile(filename)
				if err != nil {
					logrus.Fatalf(fmt.Sprintf("Can't read YAML file %q: %v", filePath, err))
				}
				err = yaml.Unmarshal(yamlFile, &yamlInput)
				if err != nil {
					logrus.Fatalf(fmt.Sprintf("Can't unmarshal YAML file %q: %v", filePath, err))
				}

				err = pushManifestList(c, yamlInput, c.Bool("ignore-missing"), c.GlobalBool("insecure"))
				if err != nil {
					logrus.Fatal(err)
				}
			},
		},
		{
			Name:  "from-args",
			Usage: "push a manifest list to a registry via CLI arguments",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "platforms",
					Usage: "comma-separated list of the platforms that images should be pushed for",
				},
				cli.StringFlag{
					Name:  "template",
					Usage: "the pattern the source images have. OS and ARCH in that pattern will be replaced with the actual values from the platforms list",
				},
				cli.StringFlag{
					Name:  "target",
					Usage: "the name of the manifest list image that is going to be produced",
				},
				cli.BoolFlag{
					Name:  "ignore-missing",
					Usage: "only warn on missing images defined in platform list",
				},
			},
			Action: func(c *cli.Context) {
				platforms := c.String("platforms")
				templ := c.String("template")
				target := c.String("target")
				srcImages := []types.ManifestEntry{}

				if len(platforms) == 0 || len(templ) == 0 || len(target) == 0 {
					logrus.Fatalf("You must specify all three arguments --platforms, --template and --target")
				}

				platformList := strings.Split(platforms, ",")

				for _, platform := range platformList {
					osArchArr := strings.Split(platform, "/")
					if len(osArchArr) != 2 && len(osArchArr) != 3 {
						logrus.Fatal("The --platforms argument must be a string slice where one value is of the form 'os/arch'")
					}
					variant := ""
					os, arch := osArchArr[0], osArchArr[1]
					if len(osArchArr) == 3 {
						variant = osArchArr[2]
					}
					srcImages = append(srcImages, types.ManifestEntry{
						Image: strings.Replace(strings.Replace(strings.Replace(templ, "ARCH", arch, 1), "OS", os, 1), "VARIANT", variant, 1),
						Platform: ocispec.Platform{
							OS:           os,
							Architecture: arch,
							Variant:      variant,
						},
					})
				}
				yamlInput := types.YAMLInput{
					Image:     target,
					Manifests: srcImages,
				}
				err := pushManifestList(c, yamlInput, c.Bool("ignore-missing"), c.GlobalBool("insecure"))
				if err != nil {
					logrus.Fatal(err)
				}
			},
		},
	},
}

func pushManifestList(c *cli.Context, input types.YAMLInput, ignoreMissing, insecure bool) error {
	// resolve the target image reference for the combined manifest list/index
	targetRef, err := reference.ParseNormalizedNamed(input.Image)
	if err != nil {
		return fmt.Errorf("Error parsing name for manifest list (%s): %v", input.Image, err)
	}

	manifestList := types.ManifestList{
		Name:      input.Image,
		Reference: targetRef,
	}
	// create an in-memory store for OCI descriptors and content used during the push operation
	memoryStore := content.NewMemoryStore()

	logrus.Info("Retrieving digests of member images")
	for _, img := range input.Manifests {
		ref, err := parseName(img.Image)
		if err != nil {
			return fmt.Errorf("Unable to parse image reference: %s: %v", img.Image, err)
		}
		if reference.Domain(targetRef) != reference.Domain(ref) {
			return fmt.Errorf("Cannot use source images from a different registry than the target image: %s != %s", reference.Domain(ref), reference.Domain(targetRef))
		}
		descriptor, err := fetchDescriptor(c, memoryStore, ref)
		if err != nil {
			if ignoreMissing {
				logrus.Warnf("Couldn't access image '%q'. Skipping due to 'ignore missing' configuration.", img.Image)
				continue
			}
			return fmt.Errorf("Inspect of image %q failed with error: %v", img.Image, err)
		}

		// Check that only member images of type OCI manifest or Docker v2.2 manifest are included
		switch descriptor.MediaType {
		case ocispec.MediaTypeImageIndex, types.MediaTypeDockerSchema2ManifestList:
			return fmt.Errorf("Cannot include an image in a manifest list/index which is already a multi-platform image: %s", img.Image)
		case ocispec.MediaTypeImageManifest, types.MediaTypeDockerSchema2Manifest:
			// valid image type to include
		default:
			return fmt.Errorf("Cannot include unknown media type '%s' in a manifest list/index push", descriptor.MediaType)
		}
		_, db, _ := memoryStore.Get(descriptor)
		var man ocispec.Manifest
		if err := json.Unmarshal(db, &man); err != nil {
			return fmt.Errorf("Could not unmarshal manifest object from descriptor for image '%s': %v", img.Image, err)
		}
		_, cb, _ := memoryStore.Get(man.Config)
		var imgConfig types.Image
		if err := json.Unmarshal(cb, &imgConfig); err != nil {
			return fmt.Errorf("Could not unmarshal config object from descriptor for image '%s': %v", img.Image, err)
		}

		// finalize the platform object that will be used to push with this manifest
		platform, err := resolvePlatform(descriptor, img, imgConfig)
		manifest := types.Manifest{
			PushRef: false,
		}
		manifest.Platform = platform
		manifest.Digest = descriptor.Digest
		manifest.Size = descriptor.Size
		manifest.MediaType = descriptor.MediaType

		if reference.Path(ref) != reference.Path(targetRef) {
			// the target manifest list/index is located in a different repo; need to push
			// the manifest as a digest to the target repo before the list/index is pushed
			manifest.PushRef = true
		}
		manifestList.Manifests = append(manifestList.Manifests, manifest)
	}

	if ignoreMissing && len(manifestList.Manifests) == 0 {
		// we need to verify we at least have one valid entry in the list
		// otherwise our manifest list will be totally empty
		return fmt.Errorf("all entries were skipped due to missing source image references; no manifest list to push")
	}

	digest, len, err := manifestList.Push(memoryStore)
	if err != nil {
		return err
	}
	fmt.Printf("Digest: %s %d\n", digest, len)
	return nil
}

func resolvePlatform(descriptor ocispec.Descriptor, img types.ManifestEntry, imgConfig types.Image) (*ocispec.Platform, error) {
	var platform *ocispec.Platform
	// fill os/arch from inspected image if not specified in input YAML
	if img.Platform.OS == "" && img.Platform.Architecture == "" {
		// prefer a full platform object, if one is already available (and appears to have meaningful content)
		if descriptor.Platform.OS != "" || descriptor.Platform.Architecture != "" {
			platform = descriptor.Platform
		} else if imgConfig.OS != "" || imgConfig.Architecture != "" {
			platform.OS = imgConfig.OS
			platform.Architecture = imgConfig.Architecture
		}
	}
	// Windows: if the origin image has OSFeature and/or OSVersion information, and
	// these values were not specified in the creation YAML, then
	// retain the origin values in the Platform definition for the manifest list:
	if imgConfig.OSVersion != "" && img.Platform.OSVersion == "" {
		platform.OSVersion = imgConfig.OSVersion
	}
	if len(imgConfig.OSFeatures) > 0 && len(img.Platform.OSFeatures) == 0 {
		platform.OSFeatures = imgConfig.OSFeatures
	}

	// validate os/arch input
	if !isValidOSArch(platform.OS, platform.Architecture, platform.Variant) {
		return nil, fmt.Errorf("Manifest entry for image %s has unsupported os/arch or os/arch/variant combination: %s/%s/%s", img.Image, platform.OS, platform.Architecture, platform.Variant)
	}
	return platform, nil
}
