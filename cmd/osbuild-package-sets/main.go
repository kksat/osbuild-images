// Simple tool to dump a JSON object containing all package sets for a specific
// distro x arch x image type.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/ostree"
)

func main() {
	var distroName string
	var archName string
	var imageName string

	flag.StringVar(&distroName, "distro", "", "Distribution name")
	flag.StringVar(&archName, "arch", "", "Architecture name")
	flag.StringVar(&imageName, "image", "", "Image name")
	flag.Parse()

	if distroName == "" || archName == "" || imageName == "" {
		flag.Usage()
		os.Exit(1)
	}

	df := distrofactory.NewDefault()

	d := df.GetDistro(distroName)
	if d == nil {
		panic(fmt.Errorf("Distro %q does not exist", distroName))
	}

	arch, err := d.GetArch(archName)
	if err != nil {
		panic(err)
	}

	image, err := arch.GetImageType(imageName)
	if err != nil {
		panic(err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	var options distro.ImageOptions
	if image.OSTreeRef() != "" {
		options.OSTree = &ostree.ImageOptions{
			URL: "https://example.com", // required by some image types
		}
	}
	manifest, _, err := image.Manifest(&blueprint.Blueprint{}, options, nil, nil)
	if err != nil {
		panic(err)
	}
	_ = encoder.Encode(manifest.GetPackageSetChains())
}
