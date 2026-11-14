package main

import (
	"embed"
	"log"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"tuxu/internal/workspace"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	err := wails.Run(&options.App{Title: "EZ软图", Width: 1480, Height: 940, MinWidth: 1120, MinHeight: 740, BackgroundColour: &options.RGBA{R: 247, G: 248, B: 251, A: 255}, AssetServer: &assetserver.Options{Assets: assets}, OnStartup: app.startup, OnBeforeClose: app.beforeClose, Windows: &windows.Options{WebviewUserDataPath: filepath.Join(app.dataDir, workspace.Internal, "webview")}, Bind: []interface{}{app}})
	if err != nil {
		log.Fatal(err)
	}
}
