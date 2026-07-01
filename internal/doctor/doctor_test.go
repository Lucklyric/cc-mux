package doctor

import (
	"errors"
	"strings"
	"testing"

	"github.com/Lucklyric/cc-mux/internal/config"
)

func alwaysFound(string) (string, error) { return "/usr/bin/claude", nil }
func neverFound(cmd string) (string, error) {
	return "", errors.New("not found: " + cmd)
}

func lookupFrom(m map[string]string) config.Lookup {
	return func(name string) (string, bool) { v, ok := m[name]; return v, ok }
}

func findMsg(fs []Finding, sub string) *Finding {
	for i := range fs {
		if strings.Contains(fs[i].Message, sub) {
			return &fs[i]
		}
	}
	return nil
}

func TestCheckCleanProfile(t *testing.T) {
	cfg := &config.Config{
		Version: config.SchemaVersion,
		Profiles: map[string]*config.Profile{
			"openrouter": {Env: map[string]string{
				"ANTHROPIC_BASE_URL":   "http://host:8788/",
				"ANTHROPIC_AUTH_TOKEN": "${TOKEN}",
			}},
		},
	}
	fs := Check(cfg, "", lookupFrom(map[string]string{"TOKEN": "x"}), alwaysFound)
	if HasErrors(fs) {
		t.Fatalf("clean profile should have no errors, got %+v", fs)
	}
}

func TestCheckUnresolvedVarIsError(t *testing.T) {
	cfg := &config.Config{
		Version:  config.SchemaVersion,
		Profiles: map[string]*config.Profile{"openrouter": {Env: map[string]string{"K": "${TOKEN}"}}},
	}
	fs := Check(cfg, "", lookupFrom(nil), alwaysFound)
	if !HasErrors(fs) {
		t.Fatal("unresolved ${TOKEN} should be an error")
	}
	if findMsg(fs, "${TOKEN}") == nil {
		t.Fatalf("expected a finding mentioning ${TOKEN}, got %+v", fs)
	}
}

func TestCheckMissingCommandIsError(t *testing.T) {
	cfg := &config.Config{
		Version:  config.SchemaVersion,
		Profiles: map[string]*config.Profile{"openrouter": {Env: map[string]string{"K": "v"}}},
	}
	fs := Check(cfg, "", lookupFrom(nil), neverFound)
	if findMsg(fs, "not found in PATH") == nil {
		t.Fatalf("expected command-not-found error, got %+v", fs)
	}
}

func TestCheckReservedNameWarns(t *testing.T) {
	cfg := &config.Config{
		Version:  config.SchemaVersion,
		Profiles: map[string]*config.Profile{"doctor": {Env: map[string]string{"K": "v"}}},
	}
	fs := Check(cfg, "", lookupFrom(nil), alwaysFound)
	if f := findMsg(fs, "shadows a subcommand"); f == nil || f.Level != Warn {
		t.Fatalf("expected a warn about shadowing, got %+v", fs)
	}
}

func TestCheckBadVersionIsError(t *testing.T) {
	cfg := &config.Config{Version: 99, Profiles: map[string]*config.Profile{}}
	fs := Check(cfg, "", lookupFrom(nil), alwaysFound)
	if findMsg(fs, "config version") == nil {
		t.Fatalf("expected version mismatch error, got %+v", fs)
	}
}
