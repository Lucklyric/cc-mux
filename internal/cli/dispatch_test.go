package cli

import (
	"reflect"
	"testing"
)

func TestPreprocess(t *testing.T) {
	known := knownNames(newRootCmd())
	cases := []struct {
		name    string
		in, out []string
	}{
		{"empty", nil, nil},
		{"help flag", []string{"-h"}, []string{"-h"}},
		{"long help", []string{"--help"}, []string{"--help"}},
		{"version flag", []string{"--version"}, []string{"--version"}},
		{"known command", []string{"list"}, []string{"list"}},
		{"alias", []string{"ls"}, []string{"ls"}},
		{"command with arg", []string{"doctor", "openrouter"}, []string{"doctor", "openrouter"}},
		{"help builtin", []string{"help", "run"}, []string{"help", "run"}},
		{"profile shorthand", []string{"openrouter"}, []string{"run", "openrouter"}},
		{"profile with args", []string{"openrouter", "--model", "x"}, []string{"run", "openrouter", "--model", "x"}},
	}
	for _, c := range cases {
		if got := preprocess(c.in, known); !reflect.DeepEqual(got, c.out) {
			t.Fatalf("%s: preprocess(%v) = %v, want %v", c.name, c.in, got, c.out)
		}
	}
}
