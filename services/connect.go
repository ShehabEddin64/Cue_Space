package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ShehabEddin64/Cue_Space/internal/i18n"
	"github.com/ShehabEddin64/Cue_Space/internal/tools"
)

// ConnectService helps people use Cue Space from AI apps (Claude Desktop,
// Claude Code, ChatGPT) through the bundled cuespace-cli MCP server.
type ConnectService struct {
	// Home is the user's home folder.
	Home string
	// CLIPath overrides where cuespace-cli is (tests). By default it sits
	// next to the app's executable.
	CLIPath string
}

// Connections describes what is set up, for the Connect sheet.
type Connections struct {
	// CLI is the path of cuespace-cli, or "" when this build doesn't include it.
	CLI string `json:"cli"`
	// ClaudeDesktop is true when the Claude app's settings folder exists.
	ClaudeDesktop bool `json:"claudeDesktop"`
	// ClaudeDesktopStatus is "none", "added" or "outdated" (added, but
	// pointing at another copy of Cue Space).
	ClaudeDesktopStatus string `json:"claudeDesktopStatus"`
	// ClaudeCode is the command that adds Cue Space to Claude Code.
	ClaudeCode string `json:"claudeCode"`
	// HTTPCommand runs the server for URL-based clients such as ChatGPT.
	HTTPCommand string `json:"httpCommand"`
	// HTTPURL is the local address that command serves, including its secret.
	HTTPURL string `json:"httpUrl"`
	// Terminal is the command that makes `cuespace-cli` available in a shell.
	Terminal string `json:"terminal"`
}

const mcpName = "cuespace"

func (s *ConnectService) cli() string {
	if s.CLIPath != "" {
		return s.CLIPath
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	p := filepath.Join(filepath.Dir(exe), "cuespace-cli")
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

func (s *ConnectService) claudeConfigPath() string {
	return filepath.Join(s.Home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
}

// Status reports how Cue Space is connected.
func (s *ConnectService) Status() Connections {
	cli := s.cli()
	c := Connections{CLI: cli, ClaudeDesktopStatus: "none"}
	if _, err := os.Stat(filepath.Dir(s.claudeConfigPath())); err == nil {
		c.ClaudeDesktop = true
	}
	if cfg, err := s.readClaudeConfig(); err == nil {
		if servers, ok := cfg["mcpServers"].(map[string]any); ok {
			if entry, ok := servers[mcpName].(map[string]any); ok {
				c.ClaudeDesktopStatus = "outdated"
				if cmd, _ := entry["command"].(string); cmd == cli && cli != "" {
					c.ClaudeDesktopStatus = "added"
				}
			}
		}
	}
	if cli == "" {
		return c
	}
	q := shellQuote(cli)
	c.ClaudeCode = fmt.Sprintf("claude mcp add --scope user %s -- %s mcp", mcpName, q)
	c.HTTPCommand = q + " mcp --http"
	if token, err := tools.Token(); err == nil {
		c.HTTPURL = "http://127.0.0.1:8765/mcp/" + token
	}
	c.Terminal = fmt.Sprintf("sudo ln -sf %s /usr/local/bin/cuespace-cli", q)
	return c
}

// AddToClaudeDesktop adds (or updates) Cue Space in Claude Desktop's MCP
// servers, keeping everything else in that file as it was.
func (s *ConnectService) AddToClaudeDesktop() (Connections, error) {
	cli := s.cli()
	if cli == "" {
		return s.Status(), i18n.Errorf("This build of Cue Space doesn’t include cuespace-cli. Build the app with ./scripts/build.sh.")
	}
	path := s.claudeConfigPath()
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		return s.Status(), i18n.Errorf("Claude Desktop doesn’t seem to be installed. Open it once, then try again.")
	}
	cfg, err := s.readClaudeConfig()
	if errors.Is(err, os.ErrNotExist) {
		cfg = map[string]any{}
	} else if err != nil {
		return s.Status(), i18n.Errorf("Couldn’t read Claude’s settings: %w", err)
	}
	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers[mcpName] = map[string]any{"command": cli, "args": []string{"mcp"}}
	cfg["mcpServers"] = servers
	if err := writeJSONFile(path, cfg); err != nil {
		return s.Status(), i18n.Errorf("Couldn’t update Claude’s settings: %w", err)
	}
	return s.Status(), nil
}

// RemoveFromClaudeDesktop takes Cue Space out of Claude Desktop's MCP servers.
func (s *ConnectService) RemoveFromClaudeDesktop() (Connections, error) {
	cfg, err := s.readClaudeConfig()
	if err != nil {
		return s.Status(), nil // nothing to remove
	}
	if servers, ok := cfg["mcpServers"].(map[string]any); ok {
		delete(servers, mcpName)
	}
	if err := writeJSONFile(s.claudeConfigPath(), cfg); err != nil {
		return s.Status(), i18n.Errorf("Couldn’t update Claude’s settings: %w", err)
	}
	return s.Status(), nil
}

func (s *ConnectService) readClaudeConfig() (map[string]any, error) {
	data, err := os.ReadFile(s.claudeConfigPath())
	if err != nil {
		return nil, err
	}
	cfg := map[string]any{}
	if len(strings.TrimSpace(string(data))) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// writeJSONFile replaces path atomically. The first time, it keeps a copy
// of the file as it was before Cue Space touched it.
func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	backup := path + ".before-cuespace"
	if _, err := os.Stat(backup); errors.Is(err, os.ErrNotExist) {
		if old, err := os.ReadFile(path); err == nil {
			if err := os.WriteFile(backup, old, 0o600); err != nil {
				return err
			}
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// shellQuote quotes a path for a POSIX shell when it needs it.
func shellQuote(s string) string {
	if strings.IndexFunc(s, func(r rune) bool {
		return !(r == '/' || r == '.' || r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
