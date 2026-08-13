package updatecheck

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/squad-opencode/squad-opencode/internal/version"
)

// Result is local vs latest release.
type Result struct {
	Local   string
	Latest  string
	Message string
}

// APILatest is the GitHub releases URL (overridable in tests).
var APILatest = "https://api.github.com/repos/" + version.Repo + "/releases/latest"

// Check hits GitHub latest release. 404 means no releases yet.
func Check(client *http.Client) (Result, error) {
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
	res := Result{Local: version.Version}
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
	res.Message = formatCompare(version.Version, payload.Tag)
	return res, nil
}

func formatCompare(local, latest string) string {
	status := "update available"
	if sameVersion(local, latest) {
		status = "up to date"
	}
	return fmt.Sprintf("%s — local %s, latest %s", status, local, latest)
}

func sameVersion(local, tag string) bool {
	return strings.TrimPrefix(local, "v") == strings.TrimPrefix(tag, "v")
}
