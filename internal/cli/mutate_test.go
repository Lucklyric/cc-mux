package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/Lucklyric/cc-mux/internal/config"
)

// runCLI executes the root command with args against an isolated config file.
func runCLI(t *testing.T, configPath string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("CC_MUX_CONFIG", configPath)
	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(preprocess(args, knownNames(root)))
	err := root.Execute()
	return buf.String(), err
}

func TestSetCreatesAndUpdatesMultipleEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	// --create makes a new profile and config in one shot, with two entries.
	if _, err := runCLI(t, path, "set", "openrouter",
		"ANTHROPIC_BASE_URL=http://host:8788/",
		"ANTHROPIC_AUTH_TOKEN=${OPENROUTER_API_KEY}",
		"--create"); err != nil {
		t.Fatalf("set --create: %v", err)
	}

	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	p, ok := cfg.Profiles["openrouter"]
	if !ok {
		t.Fatal("profile openrouter not created")
	}
	if p.Env["ANTHROPIC_BASE_URL"] != "http://host:8788/" || p.Env["ANTHROPIC_AUTH_TOKEN"] != "${OPENROUTER_API_KEY}" {
		t.Fatalf("entries not set: %+v", p.Env)
	}

	// A second set updates one key and adds another (no --create needed).
	if _, err := runCLI(t, path, "set", "openrouter",
		"ANTHROPIC_API_KEY=",
		"ANTHROPIC_BASE_URL=http://new:9000/"); err != nil {
		t.Fatalf("set update: %v", err)
	}
	cfg, _ = config.LoadFrom(path)
	p = cfg.Profiles["openrouter"]
	if _, ok := p.Env["ANTHROPIC_API_KEY"]; !ok {
		t.Fatal("empty-value key not stored")
	}
	if p.Env["ANTHROPIC_BASE_URL"] != "http://new:9000/" {
		t.Fatalf("key not updated: %q", p.Env["ANTHROPIC_BASE_URL"])
	}
}

func TestSetUnknownProfileWithoutCreateFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Default().SaveTo(path); err != nil {
		t.Fatal(err)
	}
	if _, err := runCLI(t, path, "set", "ghost", "K=V"); err == nil {
		t.Fatal("expected error setting on unknown profile without --create")
	}
}
