# Contributing to squad-opencode

Public repo. MIT. Changes land through **pull requests** on the default branch (`main`). Maintainers with admin can push or merge without a review when they need to; everyone else needs a PR and **one approval that is not their own**.

GitHub already blocks you from approving your own PR. That is the “not self” rule.

## Before you start

- Read [README.md](README.md) and [docs/get-started.md](docs/get-started.md).
- Open an [issue](https://github.com/xeaser/squad-opencode/issues) for bugs or larger features before a big PR.
- Keep the host **OpenCode**. Do not add Copilot, an Ink/`squad` shell, Aspire, or npm as the primary distribution.

## Dev loop

```bash
git clone https://github.com/xeaser/squad-opencode.git
cd squad-opencode
go test ./...
go build -o squad-oc ./cmd/squad-oc
```

Or, with [Task](https://taskfile.dev) (`go install github.com/go-task/task/v3/cmd/task@latest`):

```bash
task                 # list
task test            # go test ./...
task build           # go build -o squad-oc[.exe]
task ci              # local CI gate (fmt/vet/lint/race/build/vuln/actionlint)
task tools           # install lint/vuln/actionlint/goreleaser
task release:check   # validate .goreleaser.yaml
task release         # GoReleaser snapshot into dist/ (no tag, no publish)
task bump TAG=vX.Y.Z # pin Homebrew/Scoop/winget from dist/ (no PR)
task release:tag TAG=vX.Y.Z   # annotated tag on main only (no push)
task release:push TAG=vX.Y.Z  # push tag; GitHub release workflow publishes
```

Do not edit `internal/version/version.go` to bump. Do not tag a feature branch.

Requires **Go 1.26.6+**. CI is `go test ./...` (no live OpenCode). `task ci` is the local mirror of `.github/workflows/ci.yml`. Optional package test `TestLiveEnsureAPI` stays behind `SQUAD_OC_LIVE=1`.

## Branch and PR

1. Branch from the default branch: `feat/…` or `fix/…`.
2. One logical change per PR. Match existing Go style (small packages, table tests, temp dirs).
3. `gofmt`, `go vet ./...`, and `go test ./...` must pass. CI also runs race tests on Linux/Windows/macOS, golangci-lint, govulncheck, actionlint, and a build on all three OSes.
4. Update README / get-started if you add a user-visible command or flag.
5. Open a PR. Wait for the `ci` check (lint + test + vuln + build) and one review.

Do not force-push to the default branch. Do not rewrite published tags.

## Cutting a release

1. Merge the PR to `main`.
2. Wait for the `ci` check on that merge commit to go green.
3. Tag **that main commit**, not the feature-branch SHA:

   ```bash
   git checkout main && git pull
   git tag -a vX.Y.Z -m "squad-oc vX.Y.Z"
   git push origin vX.Y.Z
   ```

4. The release workflow checks the tag is on `origin/main`, waits for `ci`, then GoReleaser publishes. It uses `.goreleaser.yaml` from `origin/main` so a config fix can retry the same tag.
5. The bump job writes the packaging branch with GraphQL `createCommitOnBranch` (one signed commit for every file; do not set `author`/`committer`). `GITHUB_TOKEN` is a GitHub App token, so GitHub signs it as `github-actions[bot]`. It opens the PR and starts `ci` via `workflow_dispatch`. Approve and squash-merge that PR when `ci` is green. `upgrade --self` already works from the GitHub Release if the bump is delayed.

If the release job fails **before** a GitHub Release exists, do **not** tag the next version. Fix the config on `main`, then retry the **same tag**: Actions → release → Run workflow (`workflow_dispatch`). The job checks out that tag’s code and overlays `origin/main`’s `.goreleaser.yaml`.

A newer stable tag is refused while the previous stable tag has no GitHub Release. Only delete a `v*` tag when no GitHub Release exists for it.

`go build` (and CI's compile job) report `dev`. GoReleaser injects the tag into `version.Version`, so `squad-oc version` on a release binary matches the tag. Do not edit `internal/version/version.go` to bump the version.

Do not tag a feature branch. Do not move or delete a published `v*` tag. After a squash-merge the feature-branch SHA is not on `main`; tag the squash commit.

## Signed commits

Commits on the default branch must be **verified** (GPG or SSH signing).

```bash
# SSH signing (simple)
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519.pub
git config --global commit.gpgsign true
```

Add the same public key as a signing key in GitHub → Settings → SSH and GPG keys.

Unsigned history already on a feature branch cannot merge unless an admin bypasses the rule. New work should be signed from the first commit.

## What we will not merge

- Copilot CLI / Copilot SDK
- Interactive Ink/`squad` shell
- Aspire / .NET dashboard
- npm as the install path
- Secrets, API keys, `.opencode/node_modules`

## License

By contributing you agree your work is MIT, same as [LICENSE](LICENSE).
