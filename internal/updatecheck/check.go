package updatecheck

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xeaser/squad-opencode/internal/version"
)

const cacheTTL = 6 * time.Hour

// Result is local vs latest release.
type Result struct {
	Local   string `json:"local"`
	Latest  string `json:"latest"`
	Status  string `json:"status"` // "up to date" | "update available" | "unknown"
	Message string `json:"message"`
}

type cacheFile struct {
	CheckedAt string `json:"checkedAt"`
	Result    Result `json:"result"`
}

// APILatest is the GitHub releases URL (overridable in tests).
var APILatest = "https://api.github.com/repos/" + version.Repo + "/releases/latest"

// CachePath returns the file-backed update-check cache location.
func CachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "squad-oc", "update-check.json")
}

// Check hits GitHub latest release unless a fresh cache entry exists.
// 404 means no releases yet. refresh forces a network fetch and rewrites the cache.
func Check(client *http.Client, refresh bool) (Result, error) {
	if !refresh {
		if res, ok := readFreshCache(); ok {
			return res, nil
		}
	}
	res, err := fetchLatest(client)
	if err != nil {
		return Result{}, err
	}
	_ = writeCache(res)
	return res, nil
}

func readFreshCache() (Result, bool) {
	b, err := os.ReadFile(CachePath())
	if err != nil {
		return Result{}, false
	}
	var cf cacheFile
	if err := json.Unmarshal(b, &cf); err != nil {
		return Result{}, false
	}
	if cf.CheckedAt == "" {
		return Result{}, false
	}
	checked, err := time.Parse(time.RFC3339, cf.CheckedAt)
	if err != nil {
		return Result{}, false
	}
	if time.Since(checked) >= cacheTTL {
		return Result{}, false
	}
	return cf.Result, true
}

func writeCache(res Result) error {
	path := CachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	cf := cacheFile{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Result:    res,
	}
	b, err := json.Marshal(cf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func fetchLatest(client *http.Client) (Result, error) {
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	url := APILatest
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "squad-oc/"+version.Version)
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	res := Result{Local: version.Version, Status: "unknown"}
	if resp.StatusCode == http.StatusNotFound {
		res.Message = fmt.Sprintf("local %s — no GitHub releases yet", version.Version)
		return res, nil
	}
	if resp.StatusCode != http.StatusOK {
		res.Message = fmt.Sprintf("local %s — update check HTTP %d", version.Version, resp.StatusCode)
		return res, nil
	}
	var payload struct {
		Tag string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return res, err
	}
	res.Latest = payload.Tag
	res.Status = compareStatus(version.Version, payload.Tag)
	res.Message = formatCompare(version.Version, payload.Tag)
	return res, nil
}

func formatCompare(local, latest string) string {
	return fmt.Sprintf("%s — local %s, latest %s", compareStatus(local, latest), local, latest)
}

func compareStatus(local, latest string) string {
	if sameVersion(local, latest) {
		return "up to date"
	}
	return "update available"
}

func sameVersion(local, tag string) bool {
	return strings.TrimPrefix(local, "v") == strings.TrimPrefix(tag, "v")
}
