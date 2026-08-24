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
	if _, err := os.Stat(filepath.Join(root, "Formula", "squad-oc.rb")); err == nil {
		t.Fatal("Formula/squad-oc.rb must be removed; tap is a cask")
	}
	files := []string{
		filepath.Join("bucket", "squad-oc.json"),
		filepath.Join("Casks", "squad-oc.rb"),
		filepath.Join("packaging", "winget", "xeaser.squad-oc.yaml"),
		filepath.Join("packaging", "winget", "xeaser.squad-oc.installer.yaml"),
		filepath.Join("packaging", "winget", "xeaser.squad-oc.locale.en-US.yaml"),
	}
	repoURL := "https://github.com/" + Repo
	asset := regexp.MustCompile(Name + `_\d+\.\d+\.\d+_(windows_amd64\.zip|darwin_(amd64|arm64)\.tar\.gz|linux_(amd64|arm64)\.tar\.gz)`)
	for _, rel := range files {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		s := string(body)
		if !strings.Contains(s, Repo) && !strings.Contains(s, "xeaser") {
			t.Errorf("%s: missing publisher/repo", rel)
		}
		if rel == filepath.Join("packaging", "winget", "xeaser.squad-oc.yaml") {
			if !strings.Contains(s, "ManifestType: version") {
				t.Errorf("%s: want ManifestType version", rel)
			}
			continue
		}
		if !strings.Contains(s, repoURL) && !strings.Contains(s, Repo) {
			t.Errorf("%s: missing %s", rel, repoURL)
		}
		if !asset.MatchString(s) && !strings.Contains(rel, "locale") && !strings.Contains(rel, filepath.Join("winget", "xeaser.squad-oc.yaml")) {
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

	mig, err := os.ReadFile(filepath.Join(root, "tap_migrations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tm map[string]string
	if err := json.Unmarshal(mig, &tm); err != nil {
		t.Fatalf("tap_migrations: %v", err)
	}
	if tm["squad-oc"] != "squad-oc" {
		t.Fatalf("tap_migrations squad-oc = %q", tm["squad-oc"])
	}
}

func TestGoreleaserGeneratesPackaging(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{
		"homebrew_casks:",
		"scoops:",
		"winget:",
		"skip_upload: true",
		"package_identifier: xeaser.squad-oc",
		"GITHUB_REPOSITORY",
	} {
		if !strings.Contains(s, want) {
			t.Errorf(".goreleaser.yaml missing %q", want)
		}
	}
	if strings.Contains(s, "directory: Formula") {
		t.Error("cask must not use Formula/")
	}
	if strings.Contains(s, "skip_upload: auto") {
		t.Error("skip_upload: auto still pushes official releases to main; use true")
	}
	if strings.Contains(s, "trimPrefix") {
		t.Error("GoReleaser templates use trimprefix, not Sprig trimPrefix")
	}
	if !strings.Contains(s, "trimprefix") {
		t.Error(".goreleaser.yaml should use trimprefix for the repo name")
	}
}

func TestReleaseWorkflowOpensOnePackagingPR(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{
		"packaging-bump:",
		"needs: [goreleaser]",
		"PACKAGING_BUMP_TOKEN",
		"copy-packaging-from-dist.sh",
		"gh pr merge --squash --auto",
		"chore/packaging-",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("release.yml missing %q", want)
		}
	}
	if !strings.Contains(s, "contents: write") || !strings.Contains(s, "pull-requests: write") {
		t.Error("packaging-bump needs contents + pull-requests")
	}
}

func TestCopyPackagingScriptUsesArtifactsJSON(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "scripts", "copy-packaging-from-dist.sh"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{
		"artifacts.json",
		"Homebrew Cask",
		"Scoop Manifest",
		"Winget Manifest",
		`checkver`,
		"checksums.txt",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("copy script missing %q", want)
		}
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
