// Package launch builds the process environment/argv for a profile and execs
// the target command, replacing the current process.
package launch

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"syscall"
)

// BuildEnviron merges overlay onto a base environment ("KEY=VALUE" slice).
// Overlay entries win; keys already present keep their original position, new
// keys are appended in sorted order. Empty overlay values are preserved (they
// export an explicit blank, e.g. ANTHROPIC_API_KEY="").
func BuildEnviron(base []string, overlay map[string]string) []string {
	values := make(map[string]string, len(base)+len(overlay))
	order := make([]string, 0, len(base)+len(overlay))
	seen := map[string]bool{}

	for _, kv := range base {
		key, val, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if !seen[key] {
			order = append(order, key)
			seen[key] = true
		}
		values[key] = val
	}

	newKeys := make([]string, 0, len(overlay))
	for key := range overlay {
		if !seen[key] {
			newKeys = append(newKeys, key)
		}
	}
	sort.Strings(newKeys)
	for _, key := range newKeys {
		order = append(order, key)
		seen[key] = true
	}
	for key, val := range overlay {
		values[key] = val
	}

	out := make([]string, 0, len(order))
	for _, key := range order {
		out = append(out, key+"="+values[key])
	}
	return out
}

// BuildArgv assembles the argument vector: command name, then the profile's
// fixed args, then any pass-through args from the command line.
func BuildArgv(command string, profileArgs, passthrough []string) []string {
	argv := make([]string, 0, 1+len(profileArgs)+len(passthrough))
	argv = append(argv, command)
	argv = append(argv, profileArgs...)
	argv = append(argv, passthrough...)
	return argv
}

// Exec replaces the current process with command, resolved via PATH. It only
// returns on failure (e.g. command not found); on success the process image is
// replaced and control never comes back.
func Exec(command string, argv, environ []string) error {
	path, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("command %q not found in PATH: %w", command, err)
	}
	return syscall.Exec(path, argv, environ)
}
