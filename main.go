package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/ShehabEddin64/Cue_Space/internal/platform"
	"github.com/ShehabEddin64/Cue_Space/internal/runner"
	"github.com/ShehabEddin64/Cue_Space/internal/secrets"
	"github.com/ShehabEddin64/Cue_Space/internal/settings"
	"github.com/ShehabEddin64/Cue_Space/internal/store"
	"github.com/ShehabEddin64/Cue_Space/services"
)

//go:embed all:frontend/dist
var assets embed.FS

// dataDir is where Cue Space keeps the user's Cues and settings. Set
// CUESPACE_DIR to use another folder (handy for development).
func dataDir() string {
	if d := os.Getenv("CUESPACE_DIR"); d != "" {
		return d
	}
	base, err := os.UserConfigDir() // ~/Library/Application Support on macOS
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(base, "CueSpace")
}

func main() {
	dir := dataDir()
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	cues, err := store.Open(filepath.Join(dir, "cues"))
	if err != nil {
		log.Fatal(err)
	}
	cues.Trash = platform.MoveToTrash

	system := &services.SystemService{CuesFolder: cues.Dir()}
	cueService := &services.CueService{
		Store:  cues,
		Runner: &runner.Runner{Launcher: platform.NewLauncher(), Home: home},
	}
	aiService := &services.AIService{
		Settings: settings.Open(filepath.Join(dir, "settings.json")),
		Secrets:  secrets.Keychain{},
		Cues:     cues,
		System:   system,
	}

	app := application.New(application.Options{
		Name:        "Cue Space",
		Description: "Reduce the distance between intention and action.",
		Services: []application.Service{
			application.NewService(cueService),
			application.NewService(system),
			application.NewService(aiService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Cue Space",
		Width:     1100,
		Height:    720,
		MinWidth:  720,
		MinHeight: 480,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 52,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(18, 18, 20),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
