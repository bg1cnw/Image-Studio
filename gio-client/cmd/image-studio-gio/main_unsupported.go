//go:build !windows && !linux

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Image Studio Gio client is supported on Windows and Linux.")
	os.Exit(1)
}
