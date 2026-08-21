package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPackagingPinsReleaseRepo(t *testing.T) {
	root := repoRoot(t)
	files := []string{
		filepath.Join("bucket", "squad-oc.json"),
		filepath.Join("Formula", "squad-oc.rb"),
		filepath.Join("packaging", "winget", "xeaser.squad-oc.yaml"),
	}
	repoURL := "https://github.com/" + Repo
	asset := regexp.MustCompile(Name + `_\d+\.\d+\.\d+_(windows_amd64\.zip|darwin_(amd64|arm64)\.tar\.gz|linux_(amd64|arm64)\.tar\.gz)`)
	for _, rel := range files {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		s := string(body)
		if !strings.Contains(s, Repo) {
			t.Errorf("%s: missing repo %q", rel, Repo)
		}
		if !strings.Contains(s, repoURL) {
			t.Errorf("%s: missing %s", rel, repoURL)
		}
		if !asset.MatchString(s) {
			t.Errorf("%s: missing goreleaser asset name", rel)
		}
	}

	raw, err := os.ReadFile(filepath.Join(root, "bucket", "squad-oc.json"))
	if err != nil {
		t.Fatal(err)
	}
	var scoop struct {
		Checkver   string `json:"checkver"`
		Autoupdate struct {
			Hash struct {
				URL string `json:"url"`
			} `json:"hash"`
		} `json:"autoupdate"`
	}
	if err := json.Unmarshal(raw, &scoop); err != nil {
		t.Fatalf("scoop manifest: %v", err)
	}
	if scoop.Checkver != "github" {
		t.Fatalf("scoop checkver = %q, want github", scoop.Checkver)
	}
	if !strings.Contains(scoop.Autoupdate.Hash.URL, Repo+"/releases/download/") || !strings.HasSuffix(scoop.Autoupdate.Hash.URL, "checksums.txt") {
		t.Fatalf("scoop autoupdate hash url = %q", scoop.Autoupdate.Hash.URL)
	}
}

func TestUserDocsDoNotLeadWithGoBuild(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range []string{
		filepath.Join("docs", "get-started.md"),
		filepath.Join("docs", "workshop", "README.md"),
	} {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), "go build") {
			t.Errorf("%s: go build belongs in Develop/CONTRIBUTING only", rel)
		}
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(readme)
	dev := strings.Index(s, "## Develop")
	if dev < 0 {
		t.Fatal("README missing ## Develop")
	}
	if strings.Contains(s[:dev], "go build") {
		t.Error("README leads with go build")
	}
	if !strings.Contains(s[dev:], "go build") {
		t.Error("README Develop should keep go build")
	}
}
