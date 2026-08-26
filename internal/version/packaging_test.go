package version

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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
		if strings.Contains(s, "#{version}") {
			t.Errorf("%s: pin a literal version, not #{version}", rel)
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
		"url_template:",
		"{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}",
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
		"copy-packaging-from-dist.sh",
		"createCommitOnBranch",
		"graphql --input",
		"fileChanges",
		"chore/packaging-",
		"workflow_dispatch:",
		"require-previous-release.sh",
		"origin/main:.goreleaser.yaml",
		"origin/main:scripts/require-previous-release.sh",
		"gh workflow run",
		"ci.yml",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("release.yml missing %q", want)
		}
	}
	if !strings.Contains(s, "contents: write") || !strings.Contains(s, "pull-requests: write") || !strings.Contains(s, "actions: write") {
		t.Error("packaging-bump needs contents + pull-requests + actions (workflow_dispatch)")
	}
	if !strings.Contains(s, "sort=-v:refname") {
		t.Error("packaging-bump must compare semver tags, not GitHub latest (newest-created)")
	}
	if strings.Contains(s, "GH_TOKEN") {
		t.Error("use GITHUB_TOKEN only; gh already reads GITHUB_TOKEN")
	}
	if strings.Contains(s, "gh auth setup-git") {
		t.Error("gh auth setup-git is unused; commits go through createCommitOnBranch")
	}
	if strings.Contains(s, "PACKAGING_BUMP_TOKEN") {
		t.Error("PACKAGING_BUMP_TOKEN is unused")
	}
	if strings.Contains(s, "git commit") {
		t.Error("do not git commit; createCommitOnBranch is the verified bot signature")
	}
	if strings.Contains(s, "committer") || strings.Contains(s, "author[") {
		t.Error("do not pass committer/author; GitHub will not sign a custom committer")
	}
	if strings.Contains(s, "/contents/") || strings.Contains(s, "/git/blobs") {
		t.Error("use createCommitOnBranch, not Contents or the Git Data API")
	}
	if strings.Contains(s, "gh pr merge") {
		t.Error("do not merge the packaging PR; a human squash-merges after ci")
	}
	if strings.Contains(s, "-F additions") {
		t.Error("gh -F cannot pass a GraphQL [FileAddition!] list; use graphql --input")
	}
}

func TestContributingRetrySameUnpublishedTag(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "CONTRIBUTING.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{
		"workflow_dispatch",
		"same tag",
		"no GitHub Release",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("CONTRIBUTING.md missing %q", want)
		}
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
		"#{version}",
		`${TAG#v}`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("copy script missing %q", want)
		}
	}
}

func TestCopyPackagingRewritesGoreleaserURLs(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq required to run copy-packaging-from-dist.sh")
	}
	bash := lookupBash()
	if bash == "" {
		t.Skip("bash required to run copy-packaging-from-dist.sh")
	}

	root := repoRoot(t)
	dir := t.TempDir()
	dist := filepath.Join(dir, "dist")
	mustWrite(t, filepath.Join(dist, "Casks", "squad-oc.rb"), `# This file was generated by GoReleaser. DO NOT EDIT.
cask "squad-oc" do
  version "0.6.0"
  on_macos do
    on_intel do
      url "https://github.com/xeaser/squad-opencode/releases/download/v#{version}/squad-oc_#{version}_darwin_amd64.tar.gz"
    end
  end
end
`)
	mustWrite(t, filepath.Join(dist, "bucket", "squad-oc.json"), `{
    "version": "0.6.0",
    "architecture": {
        "64bit": {
            "url": "https://github.com/xeaser/squad-opencode/releases/download/v0.6.0/squad-oc_windows_amd64.zip"
        }
    }
}
`)
	mustWrite(t, filepath.Join(dist, "packaging", "winget", "xeaser.squad-oc.yaml"), "ManifestType: version\n")
	mustWrite(t, filepath.Join(dist, "artifacts.json"), `[
  {"type":"Homebrew Cask","path":"`+filepath.ToSlash(filepath.Join(dist, "Casks", "squad-oc.rb"))+`"},
  {"type":"Scoop Manifest","path":"`+filepath.ToSlash(filepath.Join(dist, "bucket", "squad-oc.json"))+`"},
  {"type":"Winget Manifest","path":"`+filepath.ToSlash(filepath.Join(dist, "packaging", "winget", "xeaser.squad-oc.yaml"))+`"}
]
`)

	script := filepath.ToSlash(filepath.Join(root, "scripts", "copy-packaging-from-dist.sh"))
	cmd := exec.Command(bash, script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"DIST="+filepath.ToSlash(dist),
		"REPO=xeaser/squad-opencode",
		"TAG=v0.6.0",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("copy script: %v\n%s", err, out)
	}

	cask, err := os.ReadFile(filepath.Join(dir, "Casks", "squad-oc.rb"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cask), "#{version}") {
		t.Fatal("cask still has #{version}")
	}
	if !strings.Contains(string(cask), "squad-oc_0.6.0_darwin_amd64.tar.gz") {
		t.Fatalf("cask missing pinned asset:\n%s", cask)
	}

	scoopRaw, err := os.ReadFile(filepath.Join(dir, "bucket", "squad-oc.json"))
	if err != nil {
		t.Fatal(err)
	}
	var scoop struct {
		Architecture map[string]struct {
			URL string `json:"url"`
		} `json:"architecture"`
		Autoupdate struct {
			Architecture map[string]struct {
				URL string `json:"url"`
			} `json:"architecture"`
			Hash struct {
				URL string `json:"url"`
			} `json:"hash"`
		} `json:"autoupdate"`
	}
	if err := json.Unmarshal(scoopRaw, &scoop); err != nil {
		t.Fatalf("scoop: %v\n%s", err, scoopRaw)
	}
	got := scoop.Architecture["64bit"].URL
	if !strings.Contains(got, "squad-oc_0.6.0_windows_amd64.zip") {
		t.Fatalf("scoop install url = %q", got)
	}
	auto := scoop.Autoupdate.Architecture["64bit"].URL
	if !strings.Contains(auto, "squad-oc_$version_windows_amd64.zip") {
		t.Fatalf("scoop autoupdate url = %q", auto)
	}
	if !strings.HasSuffix(scoop.Autoupdate.Hash.URL, "/v$version/checksums.txt") {
		t.Fatalf("scoop hash url = %q", scoop.Autoupdate.Hash.URL)
	}
}

func lookupBash() string {
	if runtime.GOOS == "windows" {
		if pf := os.Getenv("PROGRAMFILES"); pf != "" {
			p := filepath.Join(pf, "Git", "bin", "bash.exe")
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	p, err := exec.LookPath("bash")
	if err != nil {
		return ""
	}
	return p
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
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
