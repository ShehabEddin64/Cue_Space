// Package tools is what Cue Space offers to other programs: the
// cuespace-cli command and its MCP server, which lets AI apps such as
// Claude build Cues. Every change goes through the same validation as the
// app, and starting a Cue is handed to the app, so it behaves exactly like
// clicking Start (including desktop layouts and the run report).
package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
	"github.com/ShehabEddin64/Cue_Space/internal/platform"
	"github.com/ShehabEddin64/Cue_Space/internal/runner"
	"github.com/ShehabEddin64/Cue_Space/internal/store"
)

// DataDir is where Cue Space keeps Cues and settings. CUESPACE_DIR
// overrides it, for development.
func DataDir() (string, error) {
	if d := os.Getenv("CUESPACE_DIR"); d != "" {
		return d, nil
	}
	base, err := os.UserConfigDir() // ~/Library/Application Support on macOS
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "CueSpace"), nil
}

// Service performs operations on the user's Cues.
type Service struct {
	Store *store.Store
	Home  string
	// InstalledApps lists installed apps.
	InstalledApps func() []platform.App
	// Open hands a cuespace:// link to the app.
	Open func(ctx context.Context, link string) error
	// Search finds files by name (Spotlight on macOS).
	Search func(ctx context.Context, root, query string, limit int) ([]string, error)

	appsOnce sync.Once
	apps     []platform.App
}

// New opens the Cues folder in Cue Space's data directory.
func New() (*Service, error) {
	dir, err := DataDir()
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	st, err := store.Open(filepath.Join(dir, "cues"))
	if err != nil {
		return nil, err
	}
	st.Trash = platform.MoveToTrash
	return &Service{
		Store:         st,
		Home:          home,
		InstalledApps: func() []platform.App { return platform.InstalledApps(platform.AppDirs()) },
		Open:          platform.OpenCueSpaceURL,
		Search:        platform.SearchFiles,
	}, nil
}

func (s *Service) installed() []platform.App {
	s.appsOnce.Do(func() {
		if s.InstalledApps != nil {
			s.apps = s.InstalledApps()
		}
	})
	return s.apps
}

// --- What callers send and get back -----------------------------------------

// Resource is one item of a Cue, in a form that is easy to write by hand or
// by a model: apps are named rather than given as structs.
type Resource struct {
	ID         string   `json:"id,omitempty" jsonschema:"Optional short id such as \"moodle\", used to place this item in a desktop slot. Keep existing ids when updating a Cue."`
	Type       string   `json:"type" jsonschema:"One of: url, app, file, folder."`
	Label      string   `json:"label,omitempty" jsonschema:"What the person calls it, e.g. \"Textbook\". Optional."`
	URL        string   `json:"url,omitempty" jsonschema:"For url: a full URL with its scheme (https://, http://localhost:3000, obsidian://...)."`
	App        string   `json:"app,omitempty" jsonschema:"For app: the app's name or bundle id, exactly as list_apps shows it."`
	Path       string   `json:"path,omitempty" jsonschema:"For file and folder: an absolute path or one starting with ~/. Never guess: find real paths with search_files."`
	OpenWith   string   `json:"openWith,omitempty" jsonschema:"For file and folder: the app to open it with (name or bundle id). Empty means the default app."`
	Newest     bool     `json:"newest,omitempty" jsonschema:"For folder: open the most recently added or changed file inside it instead of the folder (e.g. the latest download)."`
	Extensions []string `json:"extensions,omitempty" jsonschema:"With newest: only consider these file types, e.g. [\"pdf\"]."`
	Disabled   bool     `json:"disabled,omitempty" jsonschema:"Keep the item but skip it when the Cue starts."`
}

// Desktop is one macOS desktop (Space) in a Cue's layout.
type Desktop struct {
	Arrangement string   `json:"arrangement" jsonschema:"One of: fullscreen (1 slot, the app in native full screen on its own desktop), fill (1 slot), left-right (2), top-bottom (2), left-quarters (3: left half, top right, bottom right), quarters (4: top left, top right, bottom left, bottom right), thirds (3 columns)."`
	Slots       []string `json:"slots" jsonschema:"Resource ids, one per slot in order. Use \"\" to leave a slot empty."`
	NewDesktop  bool     `json:"newDesktop,omitempty" jsonschema:"Put this arrangement on a new desktop instead of the current one. Full screen always gets its own desktop."`
}

// CueInput describes a Cue to create, or the new state of one to update.
type CueInput struct {
	Name        string     `json:"name,omitempty" jsonschema:"Short name of the activity, e.g. \"Math\". Required when creating."`
	Icon        string     `json:"icon,omitempty" jsonschema:"One emoji."`
	Description string     `json:"description,omitempty" jsonschema:"One short sentence."`
	Resources   []Resource `json:"resources,omitempty" jsonschema:"Everything the Cue opens, in order. When updating, this replaces the whole list; omit it to keep the current items."`
	Desktops    []Desktop  `json:"desktops,omitempty" jsonschema:"Optional window layout across macOS desktops. Items not placed in a slot open first on the current desktop. When updating, omit to keep the current layout, or send [] to remove it."`
}

// CueView is a stored Cue in the same shape as CueInput, so it can be read,
// changed and sent back.
type CueView struct {
	ID string `json:"id"`
	CueInput
	Resources []Resource `json:"resources"`
	LastRunAt string     `json:"lastRunAt,omitempty"`
	File      string     `json:"file,omitempty"`
}

// Summary is one line of the Cue list.
type Summary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon,omitempty"`
	Description string `json:"description,omitempty"`
	Items       int    `json:"items"`
	LastRunAt   string `json:"lastRunAt,omitempty"`
}

// Saved is the result of creating or updating a Cue.
type Saved struct {
	Cue CueView `json:"cue"`
	// Warnings point out things worth checking, such as paths that don't exist yet.
	Warnings []string `json:"warnings,omitempty"`
}

// --- Operations ---------------------------------------------------------------

// List returns every Cue, by name.
func (s *Service) List() ([]Summary, error) {
	cues, _, err := s.Store.List()
	if err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(cues))
	for _, c := range cues {
		n := 0
		for _, r := range c.Resources {
			if !r.Disabled {
				n++
			}
		}
		out = append(out, Summary{ID: c.ID, Name: c.Name, Icon: c.Icon, Description: c.Description, Items: n, LastRunAt: timeString(c.LastRunAt)})
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

// Find looks a Cue up by id, then by name: exact, then a unique prefix, then
// a unique part of the name.
func (s *Service) Find(ref string) (cue.Cue, error) {
	cues, _, err := s.Store.List()
	if err != nil {
		return cue.Cue{}, err
	}
	q := strings.ToLower(strings.TrimSpace(ref))
	if q == "" {
		return cue.Cue{}, errors.New("say which Cue (its name or id)")
	}
	for _, c := range cues {
		if c.ID == ref {
			return c, nil
		}
	}
	for _, match := range []func(string) bool{
		func(n string) bool { return n == q },
		func(n string) bool { return strings.HasPrefix(n, q) },
		func(n string) bool { return strings.Contains(n, q) },
	} {
		var found []cue.Cue
		for _, c := range cues {
			if match(strings.ToLower(c.Name)) {
				found = append(found, c)
			}
		}
		if len(found) == 1 {
			return found[0], nil
		}
		if len(found) > 1 {
			names := make([]string, len(found))
			for i, c := range found {
				names[i] = fmt.Sprintf("%q", c.Name)
			}
			return cue.Cue{}, fmt.Errorf("%q matches several Cues: %s. Use the full name or the id", ref, strings.Join(names, ", "))
		}
	}
	return cue.Cue{}, fmt.Errorf("no Cue matches %q", ref)
}

// Get returns one Cue in full.
func (s *Service) Get(ref string) (CueView, error) {
	c, err := s.Find(ref)
	if err != nil {
		return CueView{}, err
	}
	return s.view(c), nil
}

// Create validates and saves a new Cue.
func (s *Service) Create(in CueInput) (Saved, error) {
	if strings.TrimSpace(in.Name) == "" {
		return Saved{}, errors.New("a new Cue needs a name")
	}
	c, warnings, err := s.build(in, nil)
	if err != nil {
		return Saved{}, err
	}
	return s.save(c, warnings)
}

// Update changes an existing Cue. Empty fields keep their current values;
// resources and desktops, when given, replace the current ones.
func (s *Service) Update(ref string, in CueInput) (Saved, error) {
	old, err := s.Find(ref)
	if err != nil {
		return Saved{}, err
	}
	c, warnings, err := s.build(in, &old)
	if err != nil {
		return Saved{}, err
	}
	return s.save(c, warnings)
}

// Delete moves a Cue's file to the Trash.
func (s *Service) Delete(ref string) (Summary, error) {
	c, err := s.Find(ref)
	if err != nil {
		return Summary{}, err
	}
	if err := s.Store.Delete(c.ID); err != nil {
		return Summary{}, err
	}
	return Summary{ID: c.ID, Name: c.Name, Icon: c.Icon}, nil
}

// Start asks the app to start a Cue, exactly as if Start was clicked.
func (s *Service) Start(ctx context.Context, ref string) (Summary, error) {
	c, err := s.Find(ref)
	if err != nil {
		return Summary{}, err
	}
	if s.Open == nil {
		return Summary{}, errors.New("starting Cues isn't available here")
	}
	if err := s.Open(ctx, "cuespace://start/"+c.ID); err != nil {
		return Summary{}, fmt.Errorf("couldn't reach Cue Space: %w", err)
	}
	return Summary{ID: c.ID, Name: c.Name, Icon: c.Icon}, nil
}

// AppInfo is an installed app.
type AppInfo struct {
	Name     string `json:"name"`
	BundleID string `json:"bundleId"`
}

// Apps lists installed apps whose name or bundle id contains query (all
// apps when query is empty), up to limit.
func (s *Service) Apps(query string, limit int) []AppInfo {
	q := strings.ToLower(strings.TrimSpace(query))
	out := []AppInfo{}
	for _, a := range s.installed() {
		if q == "" || strings.Contains(strings.ToLower(a.Name), q) || strings.Contains(strings.ToLower(a.BundleID), q) {
			out = append(out, AppInfo{Name: a.Name, BundleID: a.BundleID})
			if limit > 0 && len(out) == limit {
				break
			}
		}
	}
	return out
}

// PathInfo describes a file or folder on this Mac.
type PathInfo struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Kind   string `json:"kind,omitempty"` // "file" or "folder"
	// NewestFile is the file a "newest" folder item would open right now.
	NewestFile string `json:"newestFile,omitempty"`
}

// CheckPaths reports whether each path exists and what it is.
func (s *Service) CheckPaths(paths []string) []PathInfo {
	out := make([]PathInfo, 0, len(paths))
	for _, p := range paths {
		info := PathInfo{Path: p}
		full := runner.ExpandHome(strings.TrimSpace(p), s.Home)
		if st, err := os.Stat(full); err == nil {
			info.Exists = true
			info.Kind = "file"
			if st.IsDir() && filepath.Ext(full) == "" {
				info.Kind = "folder"
				if newest, err := runner.NewestFile(full, nil); err == nil {
					info.NewestFile = filepath.Base(newest)
				}
			}
		}
		out = append(out, info)
	}
	return out
}

// SearchFiles finds files and folders in the home folder by name.
func (s *Service) SearchFiles(ctx context.Context, query, kind string, limit int) ([]PathInfo, error) {
	if s.Search == nil {
		return nil, errors.New("file search isn't available here")
	}
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("say what to search for")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	// Spotlight is fast but leaves out protected folders (Desktop,
	// Documents, Downloads) unless the asking app has access to them, so a
	// short scan of the usual places fills in.
	found, _ := s.Search(ctx, s.Home, query, limit*4)
	found = append(found, walkSearch(ctx, s.Home, query, limit*4)...)
	seen := map[string]bool{}
	out := []PathInfo{}
	for _, p := range found {
		if seen[p] {
			continue
		}
		seen[p] = true
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		k := "file"
		if st.IsDir() && filepath.Ext(p) == "" {
			k = "folder"
		}
		if kind != "" && kind != "any" && kind != k {
			continue
		}
		out = append(out, PathInfo{Path: s.short(p), Exists: true, Kind: k})
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

// walkSearch looks for names containing every word of query in the usual
// folders under home, a few levels deep, for at most a couple of seconds.
func walkSearch(ctx context.Context, home, query string, limit int) []string {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	matches := func(name string) bool {
		name = strings.ToLower(name)
		for _, w := range words {
			if !strings.Contains(name, w) {
				return false
			}
		}
		return true
	}
	skip := map[string]bool{"Library": true, "Applications": true, "Pictures": true, "Music": true, "Movies": true, "node_modules": true}
	var out []string
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if depth > 5 || len(out) >= limit || ctx.Err() != nil {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") || skip[name] {
				continue
			}
			path := filepath.Join(dir, name)
			if matches(name) {
				out = append(out, path)
				if len(out) >= limit {
					return
				}
			}
			// Don't descend into packages such as .app or .photoslibrary.
			if e.IsDir() && filepath.Ext(name) == "" {
				walk(path, depth+1)
			}
		}
	}
	walk(home, 0)
	return out
}

// --- Conversions -------------------------------------------------------------

func (s *Service) save(c cue.Cue, warnings []string) (Saved, error) {
	saved, err := s.Store.Save(c)
	if err != nil {
		var ve *cue.ValidationError
		if errors.As(err, &ve) {
			return Saved{}, fmt.Errorf("the Cue isn't valid: %s", strings.Join(ve.Problems, "; "))
		}
		return Saved{}, err
	}
	return Saved{Cue: s.view(saved), Warnings: warnings}, nil
}

// build turns input into a Cue, on top of old when updating.
func (s *Service) build(in CueInput, old *cue.Cue) (cue.Cue, []string, error) {
	var c cue.Cue
	if old != nil {
		c = *old
	}
	c.SchemaVersion = cue.SchemaVersion
	if v := strings.TrimSpace(in.Name); v != "" {
		c.Name = v
	}
	if v := strings.TrimSpace(in.Icon); v != "" {
		c.Icon = v
	}
	if v := strings.TrimSpace(in.Description); v != "" {
		c.Description = v
	}

	var warnings []string
	if in.Resources != nil || old == nil {
		c.Resources = make([]cue.Resource, 0, len(in.Resources))
		for i, r := range in.Resources {
			rsc, warn, err := s.resource(r)
			if err != nil {
				return c, nil, fmt.Errorf("item %d: %w", i+1, err)
			}
			if warn != "" {
				warnings = append(warnings, fmt.Sprintf("item %d: %s", i+1, warn))
			}
			c.Resources = append(c.Resources, rsc)
		}
		// Give items without an id one now, so layout slots can't
		// accidentally match an empty id.
		for i := range c.Resources {
			if c.Resources[i].ID == "" {
				c.Resources[i].ID = cue.NewID()
			}
		}
	}

	if in.Desktops != nil {
		if len(in.Desktops) == 0 {
			c.Layout = nil
		} else {
			layout, err := layoutFrom(in.Desktops, c.Resources)
			if err != nil {
				return c, nil, err
			}
			c.Layout = layout
		}
	} else if c.Layout != nil && in.Resources != nil {
		// The items changed but the layout wasn't restated: keep the slots
		// whose items still exist (Normalize drops the rest) and say so.
		if layoutLostItems(c.Layout, c.Resources) {
			warnings = append(warnings, "some desktop slots pointed at items that were removed, so they are now empty")
		}
	}
	return c, warnings, nil
}

func (s *Service) resource(in Resource) (cue.Resource, string, error) {
	r := cue.Resource{
		ID:       strings.TrimSpace(in.ID),
		Type:     cue.ResourceType(strings.ToLower(strings.TrimSpace(in.Type))),
		Label:    in.Label,
		Disabled: in.Disabled,
	}
	var warn string
	switch r.Type {
	case cue.TypeURL:
		r.URL = strings.TrimSpace(in.URL)
	case cue.TypeApp:
		app, err := s.resolveApp(in.App)
		if err != nil {
			return r, "", err
		}
		r.App = app
	case cue.TypeFile, cue.TypeFolder:
		r.Path = s.short(strings.TrimSpace(in.Path))
		if strings.TrimSpace(in.OpenWith) != "" {
			app, err := s.resolveApp(in.OpenWith)
			if err != nil {
				return r, "", err
			}
			r.OpenWith = app
		}
		if r.Type == cue.TypeFolder && in.Newest {
			r.Newest = true
			r.Extensions = in.Extensions
		}
		if r.Path != "" {
			full := runner.ExpandHome(r.Path, s.Home)
			if st, err := os.Stat(full); err != nil {
				warn = fmt.Sprintf("%s doesn't exist on this Mac yet", r.Path)
			} else if r.Type == cue.TypeFolder && !st.IsDir() {
				return r, "", fmt.Errorf("%s is a file; use type \"file\"", r.Path)
			}
		}
	default:
		return r, "", fmt.Errorf("unknown type %q (use url, app, file or folder)", in.Type)
	}
	if in.Newest && r.Type != cue.TypeFolder {
		return r, "", errors.New("newest only applies to folders")
	}
	return r, warn, nil
}

// resolveApp matches a name or bundle id against the installed apps.
func (s *Service) resolveApp(ref string) (*cue.AppRef, error) {
	ref = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(ref), ".app"))
	if ref == "" {
		return nil, errors.New("say which app")
	}
	apps := s.installed()
	if len(apps) == 0 {
		return &cue.AppRef{Name: ref}, nil // can't check; `open -a` decides at run time
	}
	for _, a := range apps {
		if strings.EqualFold(a.BundleID, ref) || strings.EqualFold(a.Name, ref) {
			return &cue.AppRef{BundleID: a.BundleID, Name: a.Name}, nil
		}
	}
	// Then the start of a name ("calc"), then any part of one ("code" for
	// Visual Studio Code), as long as only one app fits.
	q := strings.ToLower(ref)
	for _, fits := range []func(string) bool{
		func(n string) bool { return strings.HasPrefix(n, q) },
		func(n string) bool { return strings.Contains(n, q) },
	} {
		var found []platform.App
		for _, a := range apps {
			if fits(strings.ToLower(a.Name)) {
				found = append(found, a)
			}
		}
		if len(found) == 1 {
			return &cue.AppRef{BundleID: found[0].BundleID, Name: found[0].Name}, nil
		}
		if len(found) > 1 {
			names := make([]string, len(found))
			for i, a := range found {
				names[i] = a.Name
			}
			return nil, fmt.Errorf("%q could be %s", ref, strings.Join(names, ", "))
		}
	}
	return nil, fmt.Errorf("no app named %q is installed (use list_apps to see what is)", ref)
}

func layoutLostItems(l *cue.Layout, resources []cue.Resource) bool {
	ids := map[string]bool{}
	for _, r := range resources {
		ids[r.ID] = true
	}
	for _, d := range l.Desktops {
		for _, id := range d.Slots {
			if id != "" && !ids[id] {
				return true
			}
		}
	}
	return false
}

func layoutFrom(in []Desktop, resources []cue.Resource) (*cue.Layout, error) {
	ids := map[string]bool{}
	for _, r := range resources {
		ids[r.ID] = true
	}
	used := map[string]int{}
	layout := &cue.Layout{}
	for i, d := range in {
		a := cue.Arrangement(strings.ToLower(strings.TrimSpace(d.Arrangement)))
		info, ok := cue.ArrangementByID(a)
		if !ok {
			names := make([]string, len(cue.Arrangements))
			for j, x := range cue.Arrangements {
				names[j] = string(x.ID)
			}
			return nil, fmt.Errorf("desktop %d: unknown arrangement %q (use one of %s)", i+1, d.Arrangement, strings.Join(names, ", "))
		}
		if len(d.Slots) > len(info.Slots) {
			return nil, fmt.Errorf("desktop %d: %s has %d slot(s), not %d", i+1, a, len(info.Slots), len(d.Slots))
		}
		slots := make([]string, len(info.Slots))
		for j, id := range d.Slots {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if !ids[id] {
				return nil, fmt.Errorf("desktop %d, slot %d: no item has id %q (give items ids and use them here)", i+1, j+1, id)
			}
			if prev, ok := used[id]; ok {
				return nil, fmt.Errorf("desktop %d: item %q is already placed on desktop %d", i+1, id, prev)
			}
			used[id] = i + 1
			slots[j] = id
		}
		layout.Desktops = append(layout.Desktops, cue.Desktop{ID: cue.NewID(), Arrangement: a, Slots: slots, NewDesktop: d.NewDesktop})
	}
	return layout, nil
}

func (s *Service) view(c cue.Cue) CueView {
	v := CueView{ID: c.ID, LastRunAt: timeString(c.LastRunAt)}
	v.Name, v.Icon, v.Description = c.Name, c.Icon, c.Description
	v.Resources = make([]Resource, 0, len(c.Resources))
	for _, r := range c.Resources {
		out := Resource{ID: r.ID, Type: string(r.Type), Label: r.Label, URL: r.URL, Path: r.Path, Newest: r.Newest, Extensions: r.Extensions, Disabled: r.Disabled}
		if r.App != nil {
			out.App = appName(r.App)
		}
		if r.OpenWith != nil {
			out.OpenWith = appName(r.OpenWith)
		}
		v.Resources = append(v.Resources, out)
	}
	if c.Layout != nil {
		for _, d := range c.Layout.Desktops {
			v.Desktops = append(v.Desktops, Desktop{Arrangement: string(d.Arrangement), Slots: d.Slots, NewDesktop: d.NewDesktop})
		}
	}
	if path, err := s.Store.PathOf(c.ID); err == nil {
		v.File = s.short(path)
	}
	return v
}

func appName(a *cue.AppRef) string {
	if a.Name != "" {
		return a.Name
	}
	return a.BundleID
}

// short writes paths under the home folder as ~/…
func (s *Service) short(p string) string {
	if s.Home != "" && (p == s.Home || strings.HasPrefix(p, s.Home+"/")) {
		return "~" + strings.TrimPrefix(p, s.Home)
	}
	return p
}

func timeString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format(time.RFC3339)
}
