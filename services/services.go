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
	"github.com/ShehabEddin64/Cue_Space/internal/runner"
	"github.com/ShehabEddin64/Cue_Space/internal/secrets"
	"github.com/ShehabEddin64/Cue_Space/internal/settings"
	"github.com/ShehabEddin64/Cue_Space/internal/store"
)

// EventCuesChanged fires when any .cue.json file changes, including edits made
// outside the app.
const EventCuesChanged = "cues:changed"

func init() {
	application.RegisterEvent[string](EventCuesChanged)
}

// ---------------------------------------------------------------- Cues

type CueService struct {
	Store  *store.Store
	Runner *runner.Runner
	cancel context.CancelFunc
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
	Cues     *store.Store
	System   *SystemService
}

// AISettings is what the Settings screen shows. The key itself never leaves Go.
type AISettings struct {
	Config ai.Config `json:"config"`
	HasKey bool      `json:"hasKey"`
}

func (s *AIService) GetSettings() (AISettings, error) {
	st, err := s.Settings.Load()
	if err != nil {
		return AISettings{}, err
	}
	key, _ := s.Secrets.Get(secrets.APIKeyName(st.AI.Provider))
	return AISettings{Config: st.AI, HasKey: key != ""}, nil
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
	st, err := s.Settings.Load()
	if err != nil {
		return err
	}
	st.AI = cfg
	return s.Settings.Save(st)
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

// Propose asks the AI for a Cue. With editID set, it edits that Cue. The
// result is only a proposal: nothing is saved until the user saves it.
func (s *AIService) Propose(ctx context.Context, description, editID string) (ai.Proposal, error) {
	if strings.TrimSpace(description) == "" {
		return ai.Proposal{}, errors.New("describe what the Cue is for")
	}
	p, err := s.provider()
	if err != nil {
		return ai.Proposal{}, err
	}
	var existing *cue.Cue
	if editID != "" {
		c, err := s.Cues.Get(editID)
		if err != nil {
			return ai.Proposal{}, err
		}
		existing = &c
	}
	return ai.Propose(ctx, p, description, existing, s.System.InstalledApps())
}
