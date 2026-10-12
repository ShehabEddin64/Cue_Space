// Package services is the boundary between the Go core and the frontend. Each
// exported method on these structs becomes a typed TypeScript function via
// `wails3 generate bindings`. Keep logic in internal/; keep this layer thin.
package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/ShehabEddin64/Cue_Space/internal/ai"
	"github.com/ShehabEddin64/Cue_Space/internal/cue"
	"github.com/ShehabEddin64/Cue_Space/internal/platform"
	"github.com/ShehabEddin64/Cue_Space/internal/resolve"
	"github.com/ShehabEddin64/Cue_Space/internal/runner"
	"github.com/ShehabEddin64/Cue_Space/internal/secrets"
	"github.com/ShehabEddin64/Cue_Space/internal/settings"
	"github.com/ShehabEddin64/Cue_Space/internal/store"
)

// EventCuesChanged fires when any .cue.json file changes, including edits made
// outside the app.
const EventCuesChanged = "cues:changed"

// EventMenu carries the action of a menu bar command to the frontend, e.g.
// "new-cue" or "start-cue".
const EventMenu = "menu"

func init() {
	application.RegisterEvent[string](EventCuesChanged)
	application.RegisterEvent[string](EventMenu)
}

// ---------------------------------------------------------------- Cues

type CueService struct {
	Store  *store.Store
	Runner *runner.Runner
	// OnChange runs after the Cues on disk change (used to refresh menus).
	OnChange func()
	cancel   context.CancelFunc
}

// Library is everything the Library screen needs.
type Library struct {
	Cues     []cue.Cue         `json:"cues"`
	Problems []store.LoadError `json:"problems"`
	Folder   string            `json:"folder"`
}

func (s *CueService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	ctx, s.cancel = context.WithCancel(ctx)
	return s.Store.Watch(ctx, func() {
		application.Get().Event.Emit(EventCuesChanged, "")
		if s.OnChange != nil {
			s.OnChange()
		}
	})
}

func (s *CueService) ServiceShutdown() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *CueService) List() (Library, error) {
	cues, problems, err := s.Store.List()
	if problems == nil {
		problems = []store.LoadError{}
	}
	return Library{Cues: cues, Problems: problems, Folder: s.Store.Dir()}, err
}

func (s *CueService) Get(id string) (cue.Cue, error) {
	return s.Store.Get(id)
}

// Save validates and stores a Cue, returning it with ids and timestamps set.
func (s *CueService) Save(c cue.Cue) (cue.Cue, error) {
	return s.Store.Save(c)
}

// Problems lists what is wrong with a Cue, for live feedback in the editor.
func (s *CueService) Problems(c cue.Cue) []string {
	c.Normalize()
	var ve *cue.ValidationError
	if err := c.Validate(); errors.As(err, &ve) {
		return ve.Problems
	} else if err != nil {
		return []string{err.Error()}
	}
	return []string{}
}

// Delete moves the Cue's file to the Trash.
func (s *CueService) Delete(id string) error {
	return s.Store.Delete(id)
}

// Start opens everything in the Cue, in order. This never calls the AI.
func (s *CueService) Start(ctx context.Context, id string) (runner.Result, error) {
	c, err := s.Store.Get(id)
	if err != nil {
		return runner.Result{}, err
	}
	res := s.Runner.Run(ctx, c)
	// Recording the run is best-effort; it must not turn a good run into an error.
	_ = s.Store.MarkRun(id, time.Now())
	return res, nil
}

// ---------------------------------------------------------------- System

type SystemService struct {
	CuesFolder string

	mu   sync.Mutex
	apps []platform.App
}

// InstalledApps lists applications on this Mac (cached after the first call).
func (s *SystemService) InstalledApps() []platform.App {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.apps == nil {
		s.apps = platform.InstalledApps(platform.AppDirs())
		if s.apps == nil {
			s.apps = []platform.App{}
		}
	}
	return s.apps
}

// RefreshApps rescans the application folders.
func (s *SystemService) RefreshApps() []platform.App {
	s.mu.Lock()
	s.apps = nil
	s.mu.Unlock()
	return s.InstalledApps()
}

// ChooseFile shows an Open panel and returns the path ("" if cancelled).
func (s *SystemService) ChooseFile() (string, error) {
	p, err := application.Get().Dialog.OpenFile().
		CanChooseFiles(true).CanChooseDirectories(false).
		SetTitle("Add a file").SetButtonText("Add").
		PromptForSingleSelection()
	return collapseHome(p), err
}

// ChooseFolder shows an Open panel for folders ("" if cancelled).
func (s *SystemService) ChooseFolder() (string, error) {
	p, err := application.Get().Dialog.OpenFile().
		CanChooseFiles(false).CanChooseDirectories(true).CanCreateDirectories(true).
		SetTitle("Add a folder").SetButtonText("Add").
		PromptForSingleSelection()
	return collapseHome(p), err
}

// ResolveItems turns text like "figma, ~/Notes, github.com/me/atlas" into
// resources locally, without AI. Anything it can't resolve is returned.
func (s *SystemService) ResolveItems(text string) resolve.Result {
	home, _ := os.UserHomeDir()
	r := resolve.Resolver{Apps: s.InstalledApps(), Home: home}
	return r.Resolve(resolve.Split(text))
}

// Arrangements lists the window arrangements a desktop can use.
func (s *SystemService) Arrangements() []cue.ArrangementInfo {
	return cue.Arrangements
}

// CanArrangeWindows reports whether Cue Space has Accessibility permission.
func (s *SystemService) CanArrangeWindows() bool {
	return platform.AccessibilityTrusted(false)
}

// RequestArrangeWindows shows macOS's Accessibility permission prompt (if
// needed) and opens the settings pane where it's granted.
func (s *SystemService) RequestArrangeWindows() bool {
	if platform.AccessibilityTrusted(true) {
		return true
	}
	_ = platform.OpenAccessibilitySettings()
	return false
}

// RevealCuesFolder shows the folder holding the .cue.json files in Finder.
func (s *SystemService) RevealCuesFolder() error {
	return platform.Reveal(s.CuesFolder)
}

// PathExists reports whether a file or folder path (with ~) exists.
func (s *SystemService) PathExists(path string) bool {
	home, _ := os.UserHomeDir()
	_, err := os.Stat(runner.ExpandHome(path, home))
	return err == nil
}

// collapseHome stores paths under the home folder as ~/…, so Cues survive a
// username change and are shareable.
func collapseHome(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || p == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~/" + p[len(home)+1:]
	}
	return p
}

// ---------------------------------------------------------------- AI

type AIService struct {
	Settings *settings.File
	Secrets  secrets.Store
	System   *SystemService
}

// AISettings is what the UI shows about the AI. The key itself never leaves Go.
type AISettings struct {
	Config ai.Config `json:"config"`
	HasKey bool      `json:"hasKey"`
	// Ready is true when Describe can be used.
	Ready bool `json:"ready"`
	// Disclosure names the model and where requests go.
	Disclosure ai.Disclosure `json:"disclosure"`
}

func (s *AIService) GetSettings() (AISettings, error) {
	st, err := s.Settings.Load()
	if err != nil {
		return AISettings{}, err
	}
	key, _ := s.Secrets.Get(secrets.APIKeyName(st.AI.Provider))
	_, err = ai.New(st.AI, key)
	return AISettings{Config: st.AI, HasKey: key != "", Ready: err == nil, Disclosure: ai.Describe(st.AI)}, nil
}

// SaveSettings stores the provider config. A non-empty apiKey replaces the
// stored key; clearKey removes it.
func (s *AIService) SaveSettings(cfg ai.Config, apiKey string, clearKey bool) error {
	cfg.Model = strings.TrimSpace(cfg.Model)
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	if cfg.Provider == ai.ProviderAnthropic && cfg.Model == "" {
		cfg.Model = ai.DefaultAnthropicModel
	}
	name := secrets.APIKeyName(cfg.Provider)
	switch {
	case clearKey:
		if err := s.Secrets.Delete(name); err != nil {
			return err
		}
	case strings.TrimSpace(apiKey) != "":
		if err := s.Secrets.Set(name, strings.TrimSpace(apiKey)); err != nil {
			return err
		}
	}
	_, err := s.Settings.Update(func(st *settings.Settings) { st.AI = cfg })
	return err
}

func (s *AIService) provider() (ai.Provider, error) {
	st, err := s.Settings.Load()
	if err != nil {
		return nil, err
	}
	key, err := s.Secrets.Get(secrets.APIKeyName(st.AI.Provider))
	if err != nil {
		return nil, err
	}
	return ai.New(st.AI, key)
}

// TestConnection checks the saved provider, key and model.
func (s *AIService) TestConnection(ctx context.Context) error {
	p, err := s.provider()
	if err != nil {
		return err
	}
	return p.Check(ctx)
}

// Resolve asks the AI to turn phrases that couldn't be resolved locally into
// resources. It uses a much smaller prompt than Propose.
func (s *AIService) Resolve(ctx context.Context, phrases []string) (ai.Resolution, error) {
	p, err := s.provider()
	if err != nil {
		return ai.Resolution{}, err
	}
	return ai.Resolve(ctx, p, phrases, s.System.InstalledApps())
}

// Propose asks the AI for a Cue. With base set (a saved Cue or an earlier,
// unsaved proposal), the AI changes that Cue instead of starting over. The
// result is only a proposal: nothing is saved until the user saves it.
func (s *AIService) Propose(ctx context.Context, description string, base *cue.Cue) (ai.Proposal, error) {
	if strings.TrimSpace(description) == "" {
		return ai.Proposal{}, errors.New("describe what the Cue is for")
	}
	p, err := s.provider()
	if err != nil {
		return ai.Proposal{}, err
	}
	return ai.Propose(ctx, p, description, base, s.System.InstalledApps())
}

// ---------------------------------------------------------------- Prefs

type PrefsService struct {
	Settings *settings.File
	// OnChange runs after prefs are saved (used to refresh menus).
	OnChange func(settings.Prefs)
}

func (s *PrefsService) ServiceStartup(context.Context, application.ServiceOptions) error {
	st, err := s.Settings.Load()
	if err == nil {
		platform.SetAppearance(st.Prefs.Appearance)
	}
	return nil
}

func (s *PrefsService) GetPrefs() (settings.Prefs, error) {
	st, err := s.Settings.Load()
	return st.Prefs, err
}

// SavePrefs stores interface preferences and applies the appearance.
func (s *PrefsService) SavePrefs(p settings.Prefs) (settings.Prefs, error) {
	st, err := s.Settings.Update(func(st *settings.Settings) { st.Prefs = p })
	if err != nil {
		return p, err
	}
	platform.SetAppearance(st.Prefs.Appearance)
	if s.OnChange != nil {
		s.OnChange(st.Prefs)
	}
	return st.Prefs, nil
}

// ShortcutOrder is the order Cues get ⌘1–⌘9 in: pinned Cues first, in pinned
// order, then the rest as listed (by name). The sidebar uses the same order.
func ShortcutOrder(cues []cue.Cue, pinned []string) []cue.Cue {
	byID := make(map[string]cue.Cue, len(cues))
	for _, c := range cues {
		byID[c.ID] = c
	}
	out := make([]cue.Cue, 0, len(cues))
	used := map[string]bool{}
	for _, id := range pinned {
		if c, ok := byID[id]; ok && !used[id] {
			out = append(out, c)
			used[id] = true
		}
	}
	for _, c := range cues {
		if !used[c.ID] {
			out = append(out, c)
		}
	}
	return out
}
