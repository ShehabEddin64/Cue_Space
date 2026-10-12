package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddToClaudeDesktopKeepsOtherSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CUESPACE_DIR", t.TempDir())
	dir := filepath.Join(home, "Library", "Application Support", "Claude")
	os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "claude_desktop_config.json")
	os.WriteFile(path, []byte(`{"mcpServers":{"other":{"command":"x"}},"theme":"dark"}`), 0o644)

	s := &ConnectService{Home: home, CLIPath: "/Applications/Cue Space.app/Contents/MacOS/cuespace-cli"}
	if st := s.Status(); !st.ClaudeDesktop || st.ClaudeDesktopStatus != "none" {
		t.Fatalf("before: %+v", st)
	}
	st, err := s.AddToClaudeDesktop()
	if err != nil {
		t.Fatal(err)
	}
	if st.ClaudeDesktopStatus != "added" {
		t.Errorf("status = %s", st.ClaudeDesktopStatus)
	}
	var cfg map[string]any
	data, _ := os.ReadFile(path)
	json.Unmarshal(data, &cfg)
	servers := cfg["mcpServers"].(map[string]any)
	if cfg["theme"] != "dark" || servers["other"] == nil {
		t.Errorf("other settings lost: %s", data)
	}
	entry := servers["cuespace"].(map[string]any)
	if entry["command"] != s.CLIPath || entry["args"].([]any)[0] != "mcp" {
		t.Errorf("entry = %v", entry)
	}
	if _, err := os.Stat(path + ".before-cuespace"); err != nil {
		t.Error("no backup of the original settings")
	}
	if !strings.Contains(st.ClaudeCode, `'/Applications/Cue Space.app/Contents/MacOS/cuespace-cli' mcp`) {
		t.Errorf("claude code command = %s", st.ClaudeCode)
	}

	// Another copy of the app sees the entry as outdated.
	moved := &ConnectService{Home: home, CLIPath: "/tmp/elsewhere/cuespace-cli"}
	if got := moved.Status().ClaudeDesktopStatus; got != "outdated" {
		t.Errorf("moved app status = %s", got)
	}

	if st, _ := s.RemoveFromClaudeDesktop(); st.ClaudeDesktopStatus != "none" {
		t.Errorf("after remove: %s", st.ClaudeDesktopStatus)
	}
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), `"other"`) {
		t.Errorf("remove dropped other servers: %s", data)
	}
}

func TestAddToClaudeDesktopNotInstalled(t *testing.T) {
	s := &ConnectService{Home: t.TempDir(), CLIPath: "/x/cuespace-cli"}
	if _, err := s.AddToClaudeDesktop(); err == nil || !strings.Contains(err.Error(), "Open it once") {
		t.Errorf("err = %v", err)
	}
}
