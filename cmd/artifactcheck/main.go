// Command artifactcheck validates generated Windows build artifacts.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ssotnikov/baremark/internal/buildmeta"
)

func main() {
	path := flag.String("path", "", "path to the generated Windows artifact")
	kind := flag.String("kind", "", "artifact kind: syso or exe")
	architecture := flag.String("arch", "", "target architecture: amd64 or arm64")
	publishPath := flag.String("publish", "", "stable output path to replace after successful validation")
	iconsRoot := flag.String("icons-root", "", "directory of canonical app and file ICO resources")
	manifest := flag.String("manifest", "", "canonical Windows manifest for EXE validation")
	version := flag.String("version", "", "expected application version in EXE resources")
	originalName := flag.String("original-name", "", "expected OriginalFilename in EXE version info")
	flag.Parse()

	if *path == "" || *kind == "" || *architecture == "" {
		fmt.Fprintln(os.Stderr, "artifact validation failed: -path, -kind, and -arch are required")
		os.Exit(2)
	}
	var err error
	if *kind == string(buildmeta.WindowsArtifactEXE) && (*manifest != "" || *version != "" || *originalName != "" || *publishPath != "") {
		if *manifest == "" || *version == "" || *originalName == "" {
			fmt.Fprintln(os.Stderr, "artifact validation failed: EXE publication requires -manifest, -version, and -original-name")
			os.Exit(2)
		}
		if err = buildmeta.ValidatePEResourceMetadata(*path, *manifest, *version, *originalName); err != nil {
			fmt.Fprintf(os.Stderr, "artifact validation failed: %v\n", err)
			os.Exit(1)
		}
	}
	if *publishPath == "" {
		err = buildmeta.ValidateWindowsArtifact(*path, buildmeta.WindowsArtifactKind(*kind), *architecture)
		if err == nil && *kind == string(buildmeta.WindowsArtifactEXE) && *iconsRoot != "" {
			err = buildmeta.ValidatePEIconResources(*path, *iconsRoot)
		}
	} else {
		err = buildmeta.PublishWindowsArtifact(*path, *publishPath, buildmeta.WindowsArtifactKind(*kind), *architecture, *iconsRoot)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "artifact validation failed: %v\n", err)
		os.Exit(1)
	}

	label := strings.ToUpper(*kind)
	if *publishPath == "" {
		fmt.Printf("Validated %s (%s): %s\n", label, *architecture, *path)
		if *kind == string(buildmeta.WindowsArtifactEXE) && *manifest != "" {
			fmt.Printf("Validated manifest and version info (%s): %s\n", *architecture, *path)
		}
		if *kind == string(buildmeta.WindowsArtifactEXE) && *iconsRoot != "" {
			fmt.Printf("Validated fixed icon resources (%s): %s\n", *architecture, *iconsRoot)
		}
		return
	}
	fmt.Printf("Validated %s (%s): %s\n", label, *architecture, *path)
	if *kind == string(buildmeta.WindowsArtifactEXE) {
		fmt.Printf("Validated manifest and version info (%s): %s\n", *architecture, *path)
	}
	if *iconsRoot != "" {
		fmt.Printf("Validated fixed icon resources (%s): %s\n", *architecture, *iconsRoot)
	}
	fmt.Printf("Published %s (%s): %s\n", label, *architecture, *publishPath)
}
