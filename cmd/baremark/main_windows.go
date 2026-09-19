//go:build windows

package main

import (
	"os"

	"github.com/ssotnikov/baremark"
	"github.com/ssotnikov/baremark/internal/platform/win32"
)

var version = baremark.Version

func main() {
	_ = version
	if err := win32.Run(); err != nil {
		win32.ShowStartupError()
		os.Exit(1)
	}
}
