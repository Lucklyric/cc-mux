// Package doctor validates a cc-mux config and its profiles, returning a list
// of findings without touching the filesystem or environment directly (all
// external state is injected, so it is fully testable).
package doctor

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/Lucklyric/cc-mux/internal/config"
)

// Level classifies a finding's severity.
type Level string

const (
	Error Level = "error"
	Warn  Level = "warn"
)

// Finding is a single validation result.
type Finding struct {
	Level   Level
	Profile string // empty for config-wide findings
	Message string
}

// LookPath mirrors exec.LookPath; injected so doctor stays testable.
type LookPath func(command string) (string, error)

// Reserved holds subcommand names a profile should not shadow when using the
// `cc-mux <profile>` shorthand.
var Reserved = map[string]bool{
	"run": true, "list": true, "ls": true, "show": true, "doctor": true,
	"edit": true, "set": true, "init": true, "path": true, "update": true,
	"version": true, "help": true, "completion": true,
}

// Check validates cfg. If profile is non-empty, only that profile is checked
// (plus config-wide checks). lookup resolves ${VAR} refs; lookPath verifies the
// launch command exists.
func Check(cfg *config.Config, profile string, lookup config.Lookup, lookPath LookPath) []Finding {
	var out []Finding

	if cfg.Version != config.SchemaVersion {
		out = append(out, Finding{Error, "", fmt.Sprintf(
			"config version is %d, this build supports %d", cfg.Version, config.SchemaVersion)})
	}
	if len(cfg.Profiles) == 0 {
		out = append(out, Finding{Warn, "", "no profiles defined (add one with `cc-mux set <name> KEY=VALUE --create`)"})
	}

	names := cfg.ProfileNames()
	if profile != "" {
		if _, ok := cfg.Profiles[profile]; !ok {
			return append(out, Finding{Error, profile, "profile not found"})
		}
		names = []string{profile}
	}

	for _, name := range names {
		out = append(out, checkProfile(cfg, name, cfg.Profiles[name], lookup, lookPath)...)
	}
	return out
}

func checkProfile(cfg *config.Config, name string, p *config.Profile, lookup config.Lookup, lookPath LookPath) []Finding {
	var out []Finding

	if Reserved[name] {
		out = append(out, Finding{Warn, name, "profile name shadows a subcommand; launch it with `cc-mux run " + name + "`"})
	}
	if len(p.Env) == 0 {
		out = append(out, Finding{Warn, name, "profile has no env entries"})
	}

	_, missing := p.ResolveEnv(lookup)
	if len(missing) > 0 {
		sort.Strings(missing)
		for _, m := range missing {
			out = append(out, Finding{Error, name, fmt.Sprintf("env references ${%s} but it is unset in the host environment", m)})
		}
	}

	if raw, ok := p.Env["ANTHROPIC_BASE_URL"]; ok && raw != "" {
		if resolved, miss := config.ResolveValue(raw, lookup); len(miss) == 0 {
			if u, err := url.Parse(resolved); err != nil || u.Scheme == "" || u.Host == "" {
				out = append(out, Finding{Warn, name, fmt.Sprintf("ANTHROPIC_BASE_URL %q is not a valid absolute URL", resolved)})
			}
		}
	}

	command := cfg.CommandFor(p)
	if _, err := lookPath(command); err != nil {
		out = append(out, Finding{Error, name, fmt.Sprintf("command %q not found in PATH", command)})
	}
	return out
}

// HasErrors reports whether any finding is error-level.
func HasErrors(findings []Finding) bool {
	for _, f := range findings {
		if f.Level == Error {
			return true
		}
	}
	return false
}
