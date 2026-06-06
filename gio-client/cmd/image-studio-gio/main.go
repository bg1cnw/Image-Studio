//go:build windows || linux

package main

import (
	"fmt"
	"log"
	"os"

	"image-studio/gio-client/internal/ui"

	"gioui.org/app"
	"gioui.org/unit"
)

func main() {
	if handled, exitCode, err := runCLICommand(os.Args[1:], os.Stdout, os.Stderr); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitCode)
	}
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Image Studio Gio"),
			app.Size(unit.Dp(1440), unit.Dp(980)),
			app.MinSize(unit.Dp(1040), unit.Dp(720)),
		)
		if err := ui.New().Run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
