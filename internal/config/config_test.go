package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestPathPrecedence(t *testing.T) {
	t.Setenv("CC_MUX_CONFIG", "/explicit/config.json")
	if got := Path(); got != "/explicit/config.json" {
		t.Fatalf("CC_MUX_CONFIG should win, got %q", got)
	}

	t.Setenv("CC_MUX_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	if got, want := Path(), "/xdg/cc-mux/config.json"; got != want {
		t.Fatalf("XDG path: got %q want %q", got, want)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	orig := Default()
	if err := orig.SaveTo(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(orig, got) {
		t.Fatalf("round trip mismatch:\n orig=%+v\n got =%+v", orig, got)
	}
}

func TestLoadMissingHasHint(t *testing.T) {
	_, err := LoadFrom(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil {
		t.Fatal("expected error for missing config")
	}
	if want := "cc-mux init"; !contains(err.Error(), want) {
		t.Fatalf("missing-config error should mention %q, got %q", want, err.Error())
	}
}

func TestCommandForPrecedence(t *testing.T) {
	cfg := &Config{DefaultCommand: "claude-default"}
	if got := cfg.CommandFor(&Profile{Command: "claude-x"}); got != "claude-x" {
		t.Fatalf("profile command should win, got %q", got)
	}
	if got := cfg.CommandFor(&Profile{}); got != "claude-default" {
		t.Fatalf("config default should win, got %q", got)
	}
	empty := &Config{}
	if got := empty.CommandFor(&Profile{}); got != DefaultCommand {
		t.Fatalf("built-in default should win, got %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
