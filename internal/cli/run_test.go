package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Lucklyric/cc-mux/internal/config"
)

func TestRunUnknownProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Default().SaveTo(path); err != nil {
		t.Fatal(err)
	}
	_, err := runCLI(t, path, "definitely-not-a-profile")
	if err == nil || !strings.Contains(err.Error(), "unknown profile") {
		t.Fatalf("want unknown-profile error, got %v", err)
	}
}

// TestRunUnresolvedVarAborts covers the fail-fast path only; the success path
// execs and replaces the process, so it is exercised by scripts/e2e.sh instead.
func TestRunUnresolvedVarAborts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := &config.Config{
		Version: config.SchemaVersion,
		Profiles: map[string]*config.Profile{
			"x": {Command: "env", Env: map[string]string{"TOK": "${CC_MUX_TEST_UNSET_VAR_ZZZ}"}},
		},
	}
	if err := cfg.SaveTo(path); err != nil {
		t.Fatal(err)
	}
	_, err := runCLI(t, path, "x")
	if err == nil || !strings.Contains(err.Error(), "unset ${VAR}") {
		t.Fatalf("want unresolved-var error, got %v", err)
	}
}
