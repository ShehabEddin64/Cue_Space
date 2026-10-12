package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

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
	launcher := platform.NewLauncher()
	cueService := &services.CueService{
		Store: cues,
		Runner: &runner.Runner{
			Launcher: launcher,
			Arranger: &platform.Arranger{Launcher: launcher, LogDir: filepath.Join(dir, "logs")},
			Home:     home,
		},
	}
	settingsFile := settings.Open(filepath.Join(dir, "settings.json"))
	aiService := &services.AIService{
		Settings: settingsFile,
		Secrets:  secrets.Keychain{},
		System:   system,
	}
	prefsService := &services.PrefsService{Settings: settingsFile}

	app := application.New(application.Options{
		Name:        "Cue Space",
		Description: "Reduce the distance between intention and action.",
		Services: []application.Service{
			application.NewService(cueService),
			application.NewService(system),
			application.NewService(aiService),
			application.NewService(prefsService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	menu, refreshCues := menuBar(app, cues, settingsFile)
	app.Menu.Set(menu)

	// The desktop widget reads a snapshot of the Cues from the data folder.
	exe, _ := os.Executable()
	widgets := &services.WidgetWriter{Dir: dir, Reload: filepath.Join(filepath.Dir(exe), "cuespace-widget-reload")}
	updateWidgets := func() {
		list, _, _ := cues.List()
		st, _ := settingsFile.Load()
		widgets.Update(list, st.Prefs.Pinned)
	}
	go updateWidgets()

	// Keep the Cues menu (and its ⌘1–⌘9 shortcuts) and the widget in step
	// with the library.
	cueService.OnChange = func() {
		refreshCues()
		updateWidgets()
	}
	// Only pins change the menu and widget; greeting edits and the like don't.
	var lastPinned string
	prefsService.OnChange = func(p settings.Prefs) {
		if pinned := strings.Join(p.Pinned, ","); pinned != lastPinned {
			lastPinned = pinned
			refreshCues()
			updateWidgets()
		}
	}

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
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

	// cuespace:// links, used by the widget (and by Shortcuts, Raycast…):
	//   cuespace://start/<id>  starts a Cue
	//   cuespace://open/<id>   shows a Cue
	//   cuespace://open        shows the app
	app.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl, func(e *application.ApplicationEvent) {
		handleURL(app, window, cueService, e.Context().URL())
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// menuBar puts every command in the menu bar, as Mac apps should. Commands
// that act on the UI are forwarded to the frontend as menu events. The
// returned function rebuilds the Cues menu when Cues or pins change.
func menuBar(app *application.App, cues *store.Store, prefs *settings.File) (*application.Menu, func()) {
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
	file.Add("New Cue Manually").SetAccelerator("CmdOrCtrl+Alt+N").OnClick(send("new-blank-cue"))
	file.AddSeparator()
	file.Add("Show Cues Folder in Finder").OnClick(send("reveal-folder"))
	file.AddSeparator()
	file.AddRole(application.CloseWindow)

	menu.AddRole(application.EditMenu)

	view := menu.AddSubmenu("View")
	view.Add("Home").SetAccelerator("CmdOrCtrl+Shift+H").OnClick(send("home"))
	view.Add("All Cues").SetAccelerator("CmdOrCtrl+Shift+C").OnClick(send("library"))
	view.Add("Recently Used").SetAccelerator("CmdOrCtrl+Shift+R").OnClick(send("recents"))
	view.Add("Find Cue").SetAccelerator("CmdOrCtrl+F").OnClick(send("find"))
	view.AddSeparator()
	view.Add("Customize Home…").OnClick(send("customize-home"))
	view.Add("Toggle Sidebar").SetAccelerator("CmdOrCtrl+Ctrl+S").OnClick(send("toggle-sidebar"))
	view.AddRole(application.ToggleFullscreen)

	cuesMenu := menu.AddSubmenu("Cues")

	menu.AddRole(application.WindowMenu)

	help := menu.AddSubmenu("Help")
	help.Add("Welcome Tour").OnClick(send("tour"))
	help.Add("Keyboard Shortcuts").SetAccelerator("CmdOrCtrl+/").OnClick(send("shortcuts"))

	var mu sync.Mutex
	refresh := func() {
		mu.Lock()
		defer mu.Unlock()
		list, _, _ := cues.List()
		st, _ := prefs.Load()
		ordered := services.ShortcutOrder(list, st.Prefs.Pinned)

		cuesMenu.Clear()
		for i, c := range ordered {
			if i == 9 {
				break
			}
			label := "Start " + c.Name
			if c.Icon != "" {
				label = c.Icon + "  " + label
			}
			cuesMenu.Add(label).SetAccelerator(fmt.Sprintf("CmdOrCtrl+%d", i+1)).OnClick(send("start:" + c.ID))
		}
		if len(ordered) == 0 {
			cuesMenu.Add("No Cues Yet").SetEnabled(false)
		}
		cuesMenu.AddSeparator()
		cuesMenu.Add("Start This Cue").SetAccelerator("CmdOrCtrl+Return").OnClick(send("start-cue"))
		cuesMenu.Add("Edit This Cue").SetAccelerator("CmdOrCtrl+E").OnClick(send("edit-cue"))
		cuesMenu.Add("Pin or Unpin This Cue").SetAccelerator("CmdOrCtrl+Shift+P").OnClick(send("pin-cue"))
		cuesMenu.Add("Move This Cue to Trash…").OnClick(send("delete-cue"))
		menu.Update()
	}
	refresh()
	return menu, refresh
}

// RunEvent tells the frontend about a Cue started from outside the window.
type RunEvent struct {
	CueID   string        `json:"cueId"`
	CueName string        `json:"cueName"`
	Result  runner.Result `json:"result"`
	Error   string        `json:"error,omitempty"`
}

func handleURL(app *application.App, window *application.WebviewWindow, cs *services.CueService, raw string) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "cuespace" {
		return
	}
	id := strings.Trim(u.Path, "/")
	switch u.Host {
	case "start":
		if id == "" {
			return
		}
		go func() {
			ev := RunEvent{CueID: id}
			if c, err := cs.Get(id); err == nil {
				ev.CueName = c.Name
			}
			res, err := cs.Start(context.Background(), id)
			ev.Result = res
			if err != nil {
				ev.Error = err.Error()
			}
			app.Event.Emit(services.EventCueRan, ev)
		}()
	case "open":
		window.Show()
		window.Focus()
		if id != "" {
			app.Event.Emit(services.EventMenu, "open:"+id)
		}
	}
}
