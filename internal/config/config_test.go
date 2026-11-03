package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RustyRaptor/deckrotate/internal/core"
)

func TestMissingFileGivesDefaults(t *testing.T) {
	s, err := NewFileStore(filepath.Join(t.TempDir(), "nope", "settings.json")).Load()
	if err != nil {
		t.Fatal(err)
	}
	if s != (core.Settings{Mode: core.Auto}) {
		t.Errorf("defaults = %+v", s)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deckrotate", "settings.json")
	st := NewFileStore(path)
	want := core.Settings{Mode: core.Manual, Locked: true}
	if err := st.Save(want); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "{\n  \"mode\": \"manual\",\n  \"locked\": true\n}\n" {
		t.Errorf("file = %q", b)
	}
	got, err := NewFileStore(path).Load()
	if err != nil || got != want {
		t.Errorf("Load = %+v, %v", got, err)
	}
}

func TestCorruptFileIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"mode":"sideways"}`), 0o644)
	if _, err := NewFileStore(path).Load(); err == nil {
		t.Error("expected error for bad mode")
	}
}

func TestDefaultPathUsesXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/cfg")
	if got := DefaultPath(); got != "/cfg/deckrotate/settings.json" {
		t.Errorf("DefaultPath = %q", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/x")
	if got := DefaultPath(); got != "/home/x/.config/deckrotate/settings.json" {
		t.Errorf("DefaultPath = %q", got)
	}
}

func TestTimingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := core.Settings{Mode: core.Auto, SettleMS: 750, PromptTimeoutMS: 9000}
	if err := NewFileStore(path).Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := NewFileStore(path).Load()
	if err != nil || got != want {
		t.Errorf("Load = %+v, %v; want %+v", got, err, want)
	}
}
