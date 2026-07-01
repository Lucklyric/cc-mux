// Package config loads, saves, and validates the cc-mux profile config file.
//
// The config lives at ~/.config/cc-mux/config.json (honoring $XDG_CONFIG_HOME,
// overridable with $CC_MUX_CONFIG). It holds named profiles; each profile is a
// set of environment variables that cc-mux inlines when launching claude.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// SchemaVersion is the config schema version this build understands.
const SchemaVersion = 1

// DefaultCommand is used when neither the profile nor the config overrides it.
const DefaultCommand = "claude"

// Profile is a named launch environment for claude.
type Profile struct {
	Description string            `json:"description,omitempty"`
	Env         map[string]string `json:"env"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
}

// Config is the top-level config document.
type Config struct {
	Version        int                 `json:"version"`
	DefaultCommand string              `json:"default_command,omitempty"`
	Profiles       map[string]*Profile `json:"profiles"`
}

// Path returns the resolved config file path.
func Path() string {
	if p := os.Getenv("CC_MUX_CONFIG"); p != "" {
		return p
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "cc-mux", "config.json")
}

// Load reads and validates the config from Path.
func Load() (*Config, error) {
	return LoadFrom(Path())
}

// LoadFrom reads and validates the config from an explicit path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no config at %s (run `cc-mux init` to create one)", path)
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*Profile{}
	}
	return &cfg, nil
}

// Save writes the config to Path, creating parent dirs. File mode is 0600 since
// profiles may embed secrets.
func (c *Config) Save() error {
	return c.SaveTo(Path())
}

// SaveTo writes the config to an explicit path atomically (temp file + rename).
func (c *Config) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

// CommandFor returns the command to exec for a profile, applying the
// profile → config → built-in default precedence.
func (c *Config) CommandFor(p *Profile) string {
	if p != nil && p.Command != "" {
		return p.Command
	}
	if c.DefaultCommand != "" {
		return c.DefaultCommand
	}
	return DefaultCommand
}

// ProfileNames returns profile names sorted alphabetically.
func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Default returns a starter config with a documented example profile.
func Default() *Config {
	return &Config{
		Version:        SchemaVersion,
		DefaultCommand: DefaultCommand,
		Profiles: map[string]*Profile{
			"openrouter": {
				Description: "Example: OpenRouter, an Anthropic-compatible endpoint. Edit or delete me.",
				Env: map[string]string{
					"ANTHROPIC_BASE_URL":                       "https://openrouter.ai/api",
					"ANTHROPIC_AUTH_TOKEN":                     "${OPENROUTER_API_KEY}",
					"ANTHROPIC_API_KEY":                        "",
					"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
				},
			},
		},
	}
}
