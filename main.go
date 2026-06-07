package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Orbita",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		OnShutdown: app.shutdown,
		Menu: menu.NewMenuFromItems(
			menu.SubMenu("Orbita", menu.NewMenuFromItems(
				&menu.MenuItem{
					Label:       "Settings",
					Accelerator: keys.CmdOrCtrl(","),
					Click: func(cd *menu.CallbackData) {
						runtime.EventsEmit(app.ctx, "open-settings")
					},
				},
			)),
		),
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
