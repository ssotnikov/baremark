//go:build windows

package main

import "github.com/ssotnikov/baremark"

var version = baremark.Version

func main() {
	_ = version
}
