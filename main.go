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

	app.Menu.Set(menuBar(app))

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Cue Space",
		Width:     1180,
		Height:    760,
		MinWidth:  760,
		MinHeight: 520,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 52,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		URL: "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// menuBar puts every command in the menu bar, as Mac apps should. Commands
// that act on the UI are forwarded to the frontend as menu events.
func menuBar(app *application.App) *application.Menu {
	send := func(action string) func(*application.Context) {
		return func(*application.Context) { app.Event.Emit(services.EventMenu, action) }
	}
	menu := application.NewMenu()

	appMenu := menu.AddSubmenu("Cue Space")
	appMenu.AddRole(application.About)
	appMenu.AddSeparator()
	appMenu.Add("Settings…").SetAccelerator("CmdOrCtrl+,").OnClick(send("settings"))
	appMenu.AddSeparator()
	appMenu.AddRole(application.ServicesMenu)
	appMenu.AddSeparator()
	appMenu.AddRole(application.Hide)
	appMenu.AddRole(application.HideOthers)
	appMenu.AddRole(application.UnHide)
	appMenu.AddSeparator()
	appMenu.AddRole(application.Quit)

	file := menu.AddSubmenu("File")
	file.Add("New Cue…").SetAccelerator("CmdOrCtrl+N").OnClick(send("new-cue"))
	file.Add("New Blank Cue").SetAccelerator("CmdOrCtrl+Alt+N").OnClick(send("new-blank-cue"))
	file.AddSeparator()
	file.Add("Start Cue").SetAccelerator("CmdOrCtrl+Return").OnClick(send("start-cue"))
	file.Add("Edit Cue").SetAccelerator("CmdOrCtrl+E").OnClick(send("edit-cue"))
	file.Add("Move Cue to Trash…").OnClick(send("delete-cue"))
	file.AddSeparator()
	file.Add("Show Cues Folder in Finder").OnClick(send("reveal-folder"))
	file.AddSeparator()
	file.AddRole(application.CloseWindow)

	menu.AddRole(application.EditMenu)

	view := menu.AddSubmenu("View")
	view.Add("Home").SetAccelerator("CmdOrCtrl+1").OnClick(send("home"))
	view.Add("Find Cue").SetAccelerator("CmdOrCtrl+F").OnClick(send("find"))
	view.AddSeparator()
	view.Add("Toggle Sidebar").SetAccelerator("CmdOrCtrl+Ctrl+S").OnClick(send("toggle-sidebar"))
	view.AddRole(application.ToggleFullscreen)

	menu.AddRole(application.WindowMenu)
	return menu
}
