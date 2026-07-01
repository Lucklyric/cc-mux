package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestRepoOverride(t *testing.T) {
	t.Setenv("CC_MUX_UPDATE_REPO", "")
	if got := Repo(); got != DefaultRepo {
		t.Fatalf("default repo: got %q want %q", got, DefaultRepo)
	}
	t.Setenv("CC_MUX_UPDATE_REPO", "you/fork")
	if got := Repo(); got != "you/fork" {
		t.Fatalf("override repo: got %q", got)
	}
}

func TestSameVersion(t *testing.T) {
	cases := []struct {
		current, tag string
		want         bool
	}{
		{"0.1.0", "v0.1.0", true},
		{"v0.1.0", "0.1.0", true},
		{"v0.1.0", "v0.1.0", true},
		{"0.1.0", "v0.2.0", false},
		{"dev", "v0.1.0", false},
	}
	for _, c := range cases {
		if got := sameVersion(c.current, c.tag); got != c.want {
			t.Fatalf("sameVersion(%q,%q)=%v want %v", c.current, c.tag, got, c.want)
		}
	}
}

func TestAssetURL(t *testing.T) {
	rel := &release{Assets: []asset{
		{Name: "cc-mux_0.1.0_darwin_arm64.tar.gz", URL: "http://x/a"},
		{Name: "checksums.txt", URL: "http://x/sums"},
	}}
	if got := assetURL(rel, "checksums.txt"); got != "http://x/sums" {
		t.Fatalf("assetURL match: got %q", got)
	}
	if got := assetURL(rel, "missing"); got != "" {
		t.Fatalf("assetURL miss should be empty, got %q", got)
	}
}

func TestParseChecksumAndVerifyBytes(t *testing.T) {
	archive := []byte("pretend-tarball-bytes")
	sum := sha256.Sum256(archive)
	hexsum := hex.EncodeToString(sum[:])
	name := "cc-mux_0.1.0_darwin_arm64.tar.gz"
	sums := []byte(hexsum + "  " + name + "\n" +
		"deadbeef  cc-mux_0.1.0_linux_amd64.tar.gz\n")

	if got := parseChecksum(sums, name); got != hexsum {
		t.Fatalf("parseChecksum: got %q want %q", got, hexsum)
	}
	if got := parseChecksum(sums, "absent.tar.gz"); got != "" {
		t.Fatalf("parseChecksum absent should be empty, got %q", got)
	}

	if err := verifyBytes(sums, name, archive); err != nil {
		t.Fatalf("verifyBytes valid: unexpected error %v", err)
	}
	if err := verifyBytes(sums, name, []byte("tampered")); err == nil {
		t.Fatal("verifyBytes should fail on checksum mismatch")
	}
	if err := verifyBytes(sums, "absent.tar.gz", archive); err == nil {
		t.Fatal("verifyBytes should fail when no checksum is listed (fail closed)")
	}
}

func TestExtractBinary(t *testing.T) {
	want := []byte("#!/binary\x00\x01mock")
	archive := makeTarGz(t, map[string][]byte{
		"README.md": []byte("ignore me"),
		"cc-mux":    want,
	})

	got, err := extractBinary(archive, "cc-mux")
	if err != nil {
		t.Fatalf("extractBinary: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("extracted bytes mismatch:\n got=%q\nwant=%q", got, want)
	}

	if _, err := extractBinary(archive, "nope"); err == nil {
		t.Fatal("extractBinary should error when the binary is absent")
	}
}

// makeTarGz builds an in-memory .tar.gz with the given files.
func makeTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, data := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: name, Mode: 0o755, Size: int64(len(data)), Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
