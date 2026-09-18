// Command resourcegen creates BareMark Windows resources with stable icon IDs.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ssotnikov/baremark/build/tools/internal/resourcegen"
)

func main() {
	root := flag.String("root", ".", "project root")
	arch := flag.String("arch", "", "target architecture: amd64 or arm64")
	version := flag.String("version", "", "application version")
	originalName := flag.String("original-name", "", "original executable filename")
	output := flag.String("o", "", "output SYSO path")
	flag.Parse()
	if *arch == "" || *version == "" || *originalName == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "resourcegen: -arch, -version, -original-name and -o are required")
		os.Exit(2)
	}
	if err := resourcegen.Generate(*root, *arch, *version, *originalName, *output); err != nil {
		fmt.Fprintf(os.Stderr, "resourcegen: %v\n", err)
		os.Exit(1)
	}
}
