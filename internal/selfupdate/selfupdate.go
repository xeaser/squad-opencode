// Package selfupdate downloads the latest GitHub release binary and replaces this executable.
package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xeaser/squad-opencode/internal/version"
)

// ErrReplacedOnNextStart means the new binary was written beside the locked executable.
var ErrReplacedOnNextStart = errors.New("replaced on next start")

var (
	apiBase       = "https://api.github.com"
	lookupExe     = os.Executable
	renameFile    = os.Rename
	currentGOOS   = runtime.GOOS
	currentGOARCH = runtime.GOARCH
)

// AssetName is the goreleaser archive for goos/goarch at version.Version.
// Format: squad-oc_<version>_<os>_<arch>.zip (windows) or .tar.gz (else).
func AssetName(goos, goarch string) string {
	return assetName(version.Version, goos, goarch)
}

func assetName(ver, goos, goarch string) string {
	ver = strings.TrimPrefix(ver, "v")
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("squad-oc_%s_%s_%s%s", ver, goos, goarch, ext)
}

// ReplaceExecutable copies downloaded over current.
// On Windows (or any rename failure) it leaves current+".new" and returns ErrReplacedOnNextStart.
func ReplaceExecutable(current, downloaded string) error {
	if current == "" || downloaded == "" {
		return fmt.Errorf("current and downloaded paths are required")
	}
	newPath := current + ".new"
	if err := copyFile(downloaded, newPath); err != nil {
		return err
	}
	if err := renameFile(newPath, current); err == nil {
		return nil
	}
	// Destination exists or is locked: move current aside, then rename.
	bak := current + ".old"
	_ = os.Remove(bak)
	if err := renameFile(current, bak); err == nil {
		if err := renameFile(newPath, current); err != nil {
			_ = renameFile(bak, current)
			return fmt.Errorf("%w: %v", ErrReplacedOnNextStart, err)
		}
		_ = os.Remove(bak)
		return nil
	}
	return ErrReplacedOnNextStart
}

// UpgradeSelf fetches the latest GitHub release for repo and replaces this binary.
// Returns "updated 0.2.1 → 0.3.0" or "already 0.3.0".
func UpgradeSelf(client *http.Client, repo, currentVersion string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	rel, err := fetchLatest(client, repo)
	if err != nil {
		return "", err
	}
	cur := strings.TrimPrefix(currentVersion, "v")
	latest := strings.TrimPrefix(rel.Tag, "v")
	if cur == latest {
		return "already " + cur, nil
	}

	name := assetName(latest, currentGOOS, currentGOARCH)
	url := ""
	for _, a := range rel.Assets {
		if a.Name == name {
			url = a.URL
			break
		}
	}
	if url == "" {
		return "", fmt.Errorf("no release asset for %s/%s (want %s)", currentGOOS, currentGOARCH, name)
	}

	tmp, err := os.MkdirTemp("", "squad-oc-upgrade-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	archivePath := filepath.Join(tmp, name)
	if err := downloadFile(client, url, archivePath); err != nil {
		return "", err
	}
	extracted, err := extractBinary(archivePath, tmp)
	if err != nil {
		return "", err
	}
	exe, err := lookupExe()
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("updated %s → %s", cur, latest)
	if err := ReplaceExecutable(exe, extracted); err != nil {
		if errors.Is(err, ErrReplacedOnNextStart) {
			return msg, err
		}
		return "", err
	}
	return msg, nil
}

type ghRelease struct {
	Tag    string    `json:"tag_name"`
	Assets []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

func fetchLatest(client *http.Client, repo string) (ghRelease, error) {
	var zero ghRelease
	if repo == "" {
		return zero, fmt.Errorf("repo is required")
	}
	url := strings.TrimRight(apiBase, "/") + "/repos/" + repo + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "squad-oc/"+version.Version)
	resp, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return zero, fmt.Errorf("no GitHub releases yet")
	}
	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("GitHub latest release HTTP %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return zero, err
	}
	if rel.Tag == "" {
		return zero, fmt.Errorf("release missing tag_name")
	}
	return rel, nil
}

func downloadFile(client *http.Client, url, dest string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "squad-oc/"+version.Version)
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func extractBinary(archivePath, destDir string) (string, error) {
	name := filepath.Base(archivePath)
	switch {
	case strings.HasSuffix(name, ".zip"):
		return extractZip(archivePath, destDir)
	case strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz"):
		return extractTarGz(archivePath, destDir)
	default:
		return "", fmt.Errorf("unsupported archive: %s", name)
	}
}

func extractZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !isReleaseBinary(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		dest := filepath.Join(destDir, filepath.Base(f.Name))
		err = writeNewFile(dest, rc)
		rc.Close()
		return dest, err
	}
	return "", fmt.Errorf("archive does not contain squad-oc")
}

func extractTarGz(archivePath, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.FileInfo().IsDir() {
			continue
		}
		if !isReleaseBinary(hdr.Name) {
			continue
		}
		dest := filepath.Join(destDir, filepath.Base(hdr.Name))
		if err := writeNewFile(dest, tr); err != nil {
			return "", err
		}
		return dest, nil
	}
	return "", fmt.Errorf("archive does not contain squad-oc")
}

func isReleaseBinary(name string) bool {
	base := filepath.Base(filepath.ToSlash(name))
	return base == "squad-oc" || base == "squad-oc.exe"
}

func writeNewFile(dest string, r io.Reader) error {
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	return writeNewFile(dest, in)
}
