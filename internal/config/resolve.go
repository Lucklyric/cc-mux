package config

import (
	"os"
	"regexp"
	"sort"
)

// refRe matches ${VAR} references where VAR is a shell-style identifier.
var refRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// Lookup resolves a variable name to a value; ok is false when unset.
type Lookup func(name string) (string, bool)

// OsLookup resolves ${VAR} references against the host process environment.
func OsLookup(name string) (string, bool) {
	return os.LookupEnv(name)
}

// ResolveValue expands every ${VAR} in v using lookup. Unresolved references are
// left verbatim in the returned string and their names collected in missing.
func ResolveValue(v string, lookup Lookup) (out string, missing []string) {
	out = refRe.ReplaceAllStringFunc(v, func(match string) string {
		name := refRe.FindStringSubmatch(match)[1]
		if val, ok := lookup(name); ok {
			return val
		}
		missing = append(missing, name)
		return match
	})
	return out, missing
}

// ResolveEnv expands ${VAR} references across all of a profile's env values.
// The returned map has fully-resolved values; missing lists every unique
// unresolved variable name (sorted) so callers can fail fast with a clear error.
func (p *Profile) ResolveEnv(lookup Lookup) (resolved map[string]string, missing []string) {
	resolved = make(map[string]string, len(p.Env))
	seen := map[string]bool{}
	for _, key := range sortedKeys(p.Env) {
		val, miss := ResolveValue(p.Env[key], lookup)
		resolved[key] = val
		for _, m := range miss {
			if !seen[m] {
				seen[m] = true
				missing = append(missing, m)
			}
		}
	}
	sort.Strings(missing)
	return resolved, missing
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
