package main

import (
	"embed"
	"path/filepath"

	"gobchat-log-browser/api"
	"gobchat-log-browser/internal/config"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := api.NewApp()

	// Keep WebView2 browser data inside our own app dir instead of the Wails
	// default %APPDATA%\<binaryname>.exe (see docs/adr/0010).
	webviewDataPath := ""
	if dir, err := config.AppDataDir(); err == nil {
		webviewDataPath = filepath.Join(dir, "webview2")
	}

	err := wails.Run(&options.App{
		Title:  "Gobchat Log Browser",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		Windows: &windows.Options{
			WebviewUserDataPath: webviewDataPath,
		},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
