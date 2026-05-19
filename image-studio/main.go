package main

import (
	"embed"

	"image-studio/backend"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	svc := backend.NewService()

	err := wails.Run(&options.App{
		Title:     "Image Studio",
		Width:     1440,
		Height:    900,
		MinWidth:  1100,
		MinHeight: 720,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 20, B: 26, A: 1},
		OnStartup:        svc.Startup,
		Bind: []interface{}{
			svc,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
