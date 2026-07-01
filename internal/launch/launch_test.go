package launch

import (
	"reflect"
	"testing"
)

func TestBuildEnvironOverlayWins(t *testing.T) {
	base := []string{"PATH=/usr/bin", "ANTHROPIC_API_KEY=old"}
	overlay := map[string]string{
		"ANTHROPIC_API_KEY":  "", // explicit blank
		"ANTHROPIC_BASE_URL": "http://x",
	}
	got := BuildEnviron(base, overlay)

	want := []string{
		"PATH=/usr/bin",
		"ANTHROPIC_API_KEY=", // overwritten in place with blank
		"ANTHROPIC_BASE_URL=http://x",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("environ mismatch:\n got=%v\nwant=%v", got, want)
	}
}

func TestBuildEnvironSkipsMalformedBase(t *testing.T) {
	got := BuildEnviron([]string{"NOEQUALS", "A=1"}, nil)
	if !reflect.DeepEqual(got, []string{"A=1"}) {
		t.Fatalf("malformed base entry not skipped: %v", got)
	}
}

func TestBuildArgvOrder(t *testing.T) {
	got := BuildArgv("claude", []string{"--profile-arg"}, []string{"--model", "x"})
	want := []string{"claude", "--profile-arg", "--model", "x"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv mismatch:\n got=%v\nwant=%v", got, want)
	}
}
