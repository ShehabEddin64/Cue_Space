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

	"github.com/ShehabEddin64/Cue_Space/internal/i18n"
	"github.com/ShehabEddin64/Cue_Space/internal/platform"
	"github.com/ShehabEddin64/Cue_Space/internal/runner"
	"github.com/ShehabEddin64/Cue_Space/internal/secrets"
	"github.com/ShehabEddin64/Cue_Space/internal/settings"
	"github.com/ShehabEddin64/Cue_Space/internal/store"
	"github.com/ShehabEddin64/Cue_Space/internal/tools"
	"github.com/ShehabEddin64/Cue_Space/services"
)

//go:embed all:frontend/dist
var assets embed.FS

// dataDir is where Cue Space keeps the user's Cues and settings. Set
// CUESPACE_DIR to use another folder (handy for development).
func dataDir() string {
	dir, err := tools.DataDir()
	if err != nil {
		log.Fatal(err)
	}
	return dir
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
			application.NewService(&services.ConnectService{Home: home}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Messages and the menu bar follow the Language setting.
	st0, _ := settingsFile.Load()
	lastLang := i18n.Resolve(st0.Prefs.Language, platform.PreferredLanguage())
	i18n.SetLanguage(lastLang)

	// The menu bar is rebuilt when the language changes; refreshCues
	// always points at the current one.
	var menuMu sync.Mutex
	var refreshMenu func()
	setMenu := func() {
		menu, refresh := menuBar(app, cues, settingsFile)
		app.Menu.Set(menu)
		menuMu.Lock()
		refreshMenu = refresh
		menuMu.Unlock()
	}
	refreshCues := func() {
		menuMu.Lock()
		r := refreshMenu
		menuMu.Unlock()
		if r != nil {
			r()
		}
	}
	setMenu()

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
	// Only pins and the language change the menu and widget; greeting edits
	// and the like don't.
	var lastPinned string
	prefsService.OnChange = func(p settings.Prefs) {
		if lang := i18n.Resolve(p.Language, platform.PreferredLanguage()); lang != lastLang {
			lastLang = lang
			i18n.SetLanguage(lang)
			setMenu()
			updateWidgets()
			return
		}
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
	appMenu.Add(i18n.T("Settings…")).SetAccelerator("CmdOrCtrl+,").OnClick(send("settings"))
	appMenu.AddSeparator()
	appMenu.AddRole(application.ServicesMenu)
	appMenu.AddSeparator()
	appMenu.AddRole(application.Hide)
	appMenu.AddRole(application.HideOthers)
	appMenu.AddRole(application.UnHide)
	appMenu.AddSeparator()
	appMenu.AddRole(application.Quit)

	file := menu.AddSubmenu(i18n.T("File"))
	file.Add(i18n.T("New Cue…")).SetAccelerator("CmdOrCtrl+N").OnClick(send("new-cue"))
	file.Add(i18n.T("New Cue Manually")).SetAccelerator("CmdOrCtrl+Alt+N").OnClick(send("new-blank-cue"))
	file.AddSeparator()
	file.Add(i18n.T("Connect AI Apps…")).OnClick(send("connect"))
	file.Add(i18n.T("Show Cues Folder in Finder")).OnClick(send("reveal-folder"))
	file.AddSeparator()
	file.AddRole(application.CloseWindow)

	menu.AddRole(application.EditMenu)

	view := menu.AddSubmenu(i18n.T("View"))
	view.Add(i18n.T("Home")).SetAccelerator("CmdOrCtrl+Shift+H").OnClick(send("home"))
	view.Add(i18n.T("All Cues")).SetAccelerator("CmdOrCtrl+Shift+C").OnClick(send("library"))
	view.Add(i18n.T("Recently Used")).SetAccelerator("CmdOrCtrl+Shift+R").OnClick(send("recents"))
	view.Add(i18n.T("Find Cue")).SetAccelerator("CmdOrCtrl+F").OnClick(send("find"))
	view.AddSeparator()
	view.Add(i18n.T("Customize Home…")).OnClick(send("customize-home"))
	view.Add(i18n.T("Toggle Sidebar")).SetAccelerator("CmdOrCtrl+Ctrl+S").OnClick(send("toggle-sidebar"))
	view.AddRole(application.ToggleFullscreen)

	cuesMenu := menu.AddSubmenu(i18n.T("Cues"))

	menu.AddRole(application.WindowMenu)

	help := menu.AddSubmenu(i18n.T("Help"))
	help.Add(i18n.T("Tutorial")).SetAccelerator("CmdOrCtrl+Shift+T").OnClick(send("tutorial"))
	help.Add(i18n.T("Welcome Tour")).OnClick(send("tour"))
	help.Add(i18n.T("Keyboard Shortcuts")).SetAccelerator("CmdOrCtrl+/").OnClick(send("shortcuts"))

	localizeSystemItems(menu)

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
			label := i18n.T("Start %s", c.Name)
			if c.Icon != "" {
				label = c.Icon + "  " + label
			}
			cuesMenu.Add(label).SetAccelerator(fmt.Sprintf("CmdOrCtrl+%d", i+1)).OnClick(send("start:" + c.ID))
		}
		if len(ordered) == 0 {
			cuesMenu.Add(i18n.T("No Cues Yet")).SetEnabled(false)
		}
		cuesMenu.AddSeparator()
		cuesMenu.Add(i18n.T("Start This Cue")).SetAccelerator("CmdOrCtrl+Return").OnClick(send("start-cue"))
		cuesMenu.Add(i18n.T("Edit This Cue")).SetAccelerator("CmdOrCtrl+E").OnClick(send("edit-cue"))
		cuesMenu.Add(i18n.T("Pin or Unpin This Cue")).SetAccelerator("CmdOrCtrl+Shift+P").OnClick(send("pin-cue"))
		cuesMenu.Add(i18n.T("Move This Cue to Trash…")).OnClick(send("delete-cue"))
		menu.Update()
	}
	refresh()
	return menu, refresh
}

// localizeSystemItems translates the standard items Wails adds (Edit,
// Window, Hide, Quit…), which it labels in English.
func localizeSystemItems(menu *application.Menu) {
	roles := []struct {
		role  application.Role
		label string
	}{
		{application.About, "About Cue Space"},
		{application.Hide, "Hide Cue Space"},
		{application.HideOthers, "Hide Others"},
		{application.UnHide, "Show All"},
		{application.Quit, "Quit Cue Space"},
		{application.CloseWindow, "Close Window"},
		{application.Undo, "Undo"},
		{application.Redo, "Redo"},
		{application.Cut, "Cut"},
		{application.Copy, "Copy"},
		{application.Paste, "Paste"},
		{application.PasteAndMatchStyle, "Paste and Match Style"},
		{application.Delete, "Delete"},
		{application.SelectAll, "Select All"},
		{application.StartSpeaking, "Start Speaking"},
		{application.StopSpeaking, "Stop Speaking"},
		{application.ToggleFullscreen, "Enter Full Screen"},
		{application.Minimise, "Minimize"},
		{application.Zoom, "Zoom"},
		{application.Front, "Bring All to Front"},
	}
	for _, r := range roles {
		if item := menu.FindByRole(r.role); item != nil {
			item.SetLabel(i18n.T(r.label))
		}
	}
	for _, label := range []string{"Edit", "Window", "Speech", "Services"} {
		if item := menu.FindByLabel(label); item != nil {
			item.SetLabel(i18n.T(label))
		}
	}
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
