// Command resourcecheck validates BareMark build metadata and mandatory assets.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ssotnikov/baremark/internal/buildmeta"
)

func main() {
	root := flag.String("root", ".", "project root")
	flag.Parse()

	absoluteRoot, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve project root: %v\n", err)
		os.Exit(1)
	}
	project, err := buildmeta.ValidateProject(absoluteRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resource validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(project.Version.String())
}
