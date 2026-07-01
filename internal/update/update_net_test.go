package update

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// startFakeGitHub stands up an httptest server that mimics the GitHub releases
// API + asset downloads for this OS/arch, points the package at it, and returns
// the mock binary bytes the archive contains. When tamper is true, the served
// checksum is wrong so verification must fail.
func startFakeGitHub(t *testing.T, tag string, tamper bool) []byte {
	t.Helper()
	binBytes := []byte("MOCK-cc-mux-binary-bytes")
	archive := makeTarGz(t, map[string][]byte{"cc-mux": binBytes})
	archiveName := fmt.Sprintf("cc-mux_%s_%s_%s.tar.gz",
		strings.TrimPrefix(tag, "v"), runtime.GOOS, runtime.GOARCH)

	sum := sha256.Sum256(archive)
	checkHex := hex.EncodeToString(sum[:])
	if tamper {
		checkHex = strings.Repeat("0", 64)
	}
	checksums := []byte(checkHex + "  " + archiveName + "\n")

	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"tag_name":%q,"assets":[`+
			`{"name":%q,"browser_download_url":%q},`+
			`{"name":"checksums.txt","browser_download_url":%q}]}`,
			tag, archiveName, srvURL+"/dl/archive", srvURL+"/dl/checksums")
	})
	mux.HandleFunc("/dl/archive", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archive) })
	mux.HandleFunc("/dl/checksums", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(checksums) })

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	srvURL = srv.URL

	old := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = old })
	t.Setenv("CC_MUX_UPDATE_REPO", "owner/repo")
	return binBytes
}

func TestLatestReleaseAndCheck(t *testing.T) {
	startFakeGitHub(t, "v9.9.9", false)

	rel, err := latestRelease(Repo())
	if err != nil {
		t.Fatalf("latestRelease: %v", err)
	}
	if rel.Tag != "v9.9.9" {
		t.Fatalf("tag: got %q want v9.9.9", rel.Tag)
	}

	var buf bytes.Buffer
	if err := Run("0.0.1", true, &buf); err != nil {
		t.Fatalf("Run --check: %v", err)
	}
	if !strings.Contains(buf.String(), "upgrade to v9.9.9") {
		t.Fatalf("check output: %q", buf.String())
	}

	buf.Reset()
	if err := Run("9.9.9", true, &buf); err != nil {
		t.Fatalf("Run --check (current): %v", err)
	}
	if !strings.Contains(buf.String(), "already up to date") {
		t.Fatalf("up-to-date output: %q", buf.String())
	}
}

func TestInstallEndToEnd(t *testing.T) {
	want := startFakeGitHub(t, "v9.9.9", false)
	rel, err := latestRelease(Repo())
	if err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(t.TempDir(), "cc-mux")
	if err := os.WriteFile(target, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := install(rel, &buf, target); err != nil {
		t.Fatalf("install: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("installed bytes mismatch:\n got=%q\nwant=%q", got, want)
	}
	if !strings.Contains(buf.String(), "upgraded to v9.9.9") {
		t.Fatalf("install output: %q", buf.String())
	}
}

func TestInstallRejectsTamperedChecksum(t *testing.T) {
	startFakeGitHub(t, "v9.9.9", true) // served checksum is wrong
	rel, err := latestRelease(Repo())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "cc-mux")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	err = install(rel, new(bytes.Buffer), target)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("install should reject tampered checksum, got %v", err)
	}
	// The original target must be left untouched on failure.
	if got, _ := os.ReadFile(target); string(got) != "old" {
		t.Fatalf("target should be unchanged on failed install, got %q", got)
	}
}
