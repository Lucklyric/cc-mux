// Package update implements `cc-mux update`: it queries the latest GitHub
// release, downloads the archive for the running OS/arch, verifies its SHA-256
// against checksums.txt, and atomically replaces the running binary.
package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DefaultRepo is the "owner/name" used when CC_MUX_UPDATE_REPO is unset.
const DefaultRepo = "Lucklyric/cc-mux"

// Repo returns the update source repo, overridable via CC_MUX_UPDATE_REPO.
func Repo() string {
	if r := os.Getenv("CC_MUX_UPDATE_REPO"); r != "" {
		return r
	}
	return DefaultRepo
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type release struct {
	Tag    string  `json:"tag_name"`
	Assets []asset `json:"assets"`
}

var httpClient = &http.Client{Timeout: 60 * time.Second}

// apiBase is the GitHub API root; overridden in tests.
var apiBase = "https://api.github.com"

// Run performs the update. When checkOnly is true it reports the latest version
// without downloading. current is the running binary's version string.
func Run(current string, checkOnly bool, out io.Writer) error {
	repo := Repo()
	rel, err := latestRelease(repo)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "current: %s\nlatest:  %s (%s)\n", current, rel.Tag, repo)

	if sameVersion(current, rel.Tag) {
		fmt.Fprintln(out, "already up to date")
		return nil
	}
	if checkOnly {
		fmt.Fprintf(out, "run `cc-mux update` to upgrade to %s\n", rel.Tag)
		return nil
	}
	target, err := selfPath()
	if err != nil {
		return err
	}
	return install(rel, out, target)
}

// install downloads, verifies, extracts, and atomically installs the release
// binary for this OS/arch to target.
func install(rel *release, out io.Writer, target string) error {
	archiveName := fmt.Sprintf("cc-mux_%s_%s_%s.tar.gz",
		strings.TrimPrefix(rel.Tag, "v"), runtime.GOOS, runtime.GOARCH)
	archiveURL := assetURL(rel, archiveName)
	if archiveURL == "" {
		return fmt.Errorf("release %s has no asset %q", rel.Tag, archiveName)
	}
	archive, err := download(archiveURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", archiveName, err)
	}
	if err := verifyChecksum(rel, archiveName, archive); err != nil {
		return err
	}
	bin, err := extractBinary(archive, "cc-mux")
	if err != nil {
		return err
	}
	if err := replace(target, bin); err != nil {
		return err
	}
	fmt.Fprintf(out, "upgraded to %s\n", rel.Tag)
	return nil
}

func latestRelease(repo string) (*release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", apiBase, repo)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "cc-mux")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query latest release for %s: HTTP %d", repo, resp.StatusCode)
	}
	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	if rel.Tag == "" {
		return nil, fmt.Errorf("no releases found for %s", repo)
	}
	return &rel, nil
}

func assetURL(rel *release, name string) string {
	for _, a := range rel.Assets {
		if a.Name == name {
			return a.URL
		}
	}
	return ""
}

func download(url string) ([]byte, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "cc-mux")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// verifyChecksum downloads checksums.txt from the release and compares the
// SHA-256 of archive against the entry for archiveName. A missing checksums
// asset is treated as an error (fail closed).
func verifyChecksum(rel *release, archiveName string, archive []byte) error {
	sumsURL := assetURL(rel, "checksums.txt")
	if sumsURL == "" {
		return fmt.Errorf("release %s has no checksums.txt to verify against", rel.Tag)
	}
	sums, err := download(sumsURL)
	if err != nil {
		return fmt.Errorf("download checksums.txt: %w", err)
	}
	return verifyBytes(sums, archiveName, archive)
}

// parseChecksum returns the hex checksum listed for name in a goreleaser-style
// checksums.txt ("<sha256>  <filename>"), or "" if absent.
func parseChecksum(sums []byte, name string) string {
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			return fields[0]
		}
	}
	return ""
}

// verifyBytes checks archive's SHA-256 against the entry for name in sums.
// A missing entry is an error (fail closed).
func verifyBytes(sums []byte, name string, archive []byte) error {
	want := parseChecksum(sums, name)
	if want == "" {
		return fmt.Errorf("no checksum listed for %s", name)
	}
	sum := sha256.Sum256(archive)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s want %s", name, got, want)
	}
	return nil
}

func extractBinary(archive []byte, binName string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar: %w", err)
		}
		if filepath.Base(hdr.Name) == binName {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary %q not found in archive", binName)
}

// selfPath resolves the running executable's real path.
func selfPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate self: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve self: %w", err)
	}
	return exe, nil
}

// replace writes newBin next to target and renames it over target (atomic on
// the same filesystem).
func replace(target string, newBin []byte) error {
	tmp := target + ".new"
	if err := os.WriteFile(tmp, newBin, 0o755); err != nil {
		return fmt.Errorf("write new binary (need write access to %s): %w", filepath.Dir(target), err)
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

func sameVersion(current, tag string) bool {
	return strings.TrimPrefix(current, "v") == strings.TrimPrefix(tag, "v")
}
