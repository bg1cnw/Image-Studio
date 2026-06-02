//go:build windows || linux

package main

import (
	"log"
	"os"

	"image-studio/gio-client/internal/ui"

	"gioui.org/app"
	"gioui.org/unit"
)

func main() {
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
