package config

import (
	"reflect"
	"testing"
)

func lookupFrom(m map[string]string) Lookup {
	return func(name string) (string, bool) {
		v, ok := m[name]
		return v, ok
	}
}

func TestResolveValue(t *testing.T) {
	look := lookupFrom(map[string]string{"TOKEN": "secret", "HOST": "example.com"})

	got, missing := ResolveValue("Bearer ${TOKEN}", look)
	if got != "Bearer secret" || len(missing) != 0 {
		t.Fatalf("simple ref: got %q missing %v", got, missing)
	}

	got, missing = ResolveValue("https://${HOST}/v1 ${TOKEN}", look)
	if got != "https://example.com/v1 secret" || len(missing) != 0 {
		t.Fatalf("multi ref: got %q missing %v", got, missing)
	}

	got, missing = ResolveValue("${NOPE}", look)
	if got != "${NOPE}" {
		t.Fatalf("unresolved ref should stay verbatim, got %q", got)
	}
	if !reflect.DeepEqual(missing, []string{"NOPE"}) {
		t.Fatalf("expected missing [NOPE], got %v", missing)
	}

	if got, _ := ResolveValue("plain", look); got != "plain" {
		t.Fatalf("plain value changed: %q", got)
	}
}

func TestResolveEnvCollectsUniqueMissing(t *testing.T) {
	p := &Profile{Env: map[string]string{
		"A":     "${MISSING}",
		"B":     "${MISSING}-${ALSO}",
		"C":     "literal",
		"EMPTY": "",
	}}
	resolved, missing := p.ResolveEnv(lookupFrom(nil))

	if resolved["C"] != "literal" || resolved["EMPTY"] != "" {
		t.Fatalf("literal/empty resolution wrong: %+v", resolved)
	}
	if !reflect.DeepEqual(missing, []string{"ALSO", "MISSING"}) {
		t.Fatalf("expected sorted unique missing [ALSO MISSING], got %v", missing)
	}
}
