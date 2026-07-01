package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitListShowPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	out, err := runCLI(t, path, "init")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if !strings.Contains(out, "wrote starter config") {
		t.Fatalf("init output: %q", out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("init did not write config: %v", err)
	}

	if out, _ = runCLI(t, path, "list"); !strings.Contains(out, "openrouter") {
		t.Fatalf("list output: %q", out)
	}

	// Default show is unmasked-but-verbatim: ${VAR} refs and (empty) are shown.
	out, _ = runCLI(t, path, "show", "openrouter")
	if !strings.Contains(out, "${OPENROUTER_API_KEY}") || !strings.Contains(out, "(empty)") {
		t.Fatalf("show output: %q", out)
	}

	if out, _ = runCLI(t, path, "path"); !strings.Contains(out, path) {
		t.Fatalf("path output: %q", out)
	}
}

func TestInitRefusesOverwriteWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if _, err := runCLI(t, path, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCLI(t, path, "init"); err == nil {
		t.Fatal("second init without --force should fail")
	}
	if _, err := runCLI(t, path, "init", "--force"); err != nil {
		t.Fatalf("init --force should succeed: %v", err)
	}
}

func TestDisplayValue(t *testing.T) {
	cases := []struct {
		v    string
		mask bool
		want string
	}{
		{"", true, "(empty)"},
		{"", false, "(empty)"},
		{"abc", false, "abc"},
		{"${OPENROUTER_API_KEY}", false, "${OPENROUTER_API_KEY}"},
		{"secretvalue", true, "••••alue"},
		{"abcd", true, "••••"},
	}
	for _, c := range cases {
		if got := displayValue(c.v, c.mask); got != c.want {
			t.Fatalf("displayValue(%q, %v) = %q, want %q", c.v, c.mask, got, c.want)
		}
	}
}
