# Config-driven version, package, and release identity

> **Beads:** `squad-h7s.2`. Parent inbox `squad-h7s`. Broader CI overhaul stays `squad-h7s.4`.
>
> Approved 2026-08-18. Read this whole file before coding.

## Goal

Release identity (version, module, GitHub repo, binary name) comes from existing sources (`go.mod`, git tag via GoReleaser, goreleaser `project_name`), then is populated into the binary. Local `go build` does not lie about being a released version.

## Locked decisions

- **No new config format.** `go.mod` + git tag + `.goreleaser.yaml` `project_name` are enough.
- **`Version` is a `var` defaulting to `dev`.** GoReleaser injects the tag with `-ldflags -X`. Local and CI `go build` report `dev`.
- **`Module` is a `const` matching `go.mod`.** A test fails if they drift.
- **`Repo` defaults from `Module`:** `github.com/owner/name` → `owner/name`. Optional release `-X` of `GITHUB_REPOSITORY` when that env is set, so a fork’s published binary self-updates from the fork. Do not probe git remotes at runtime.
- **`Name` is a `const` `squad-oc` matching goreleaser `project_name`.** Self-update asset and binary names use `version.Name`.
- **Drop hardcoded `release.github.owner` / `name`.** GitHub Actions `GITHUB_REPOSITORY` is the source. Snapshot/local GoReleaser does not publish.
- **Go toolchain** already comes from `go.mod` (`go-version-file` in h7s.1). Do not retouch workflow Go pins.
- **CLI surface unchanged.** `squad-oc version` still prints one line; the value changes from a stale `0.4.0` to `dev` or the injected tag.
- **Docs:** CONTRIBUTING only (dev vs tagged release). No README/workshop change.

## Non-goals

- `.squad/` project config, MCP, marketplace, themes
- New identity/config file
- Runtime git-remote detection
- Rewriting GoReleaser archives
- actionlint, SHA-pin, OS test matrix (`squad-h7s.4`)
- A live `v*` tag in this change

## Architecture

```text
go.mod                          .goreleaser.yaml
  module github.com/owner/name    project_name: squad-oc
  go 1.26.6                       ldflags -X Version={{.Version}}
        │                         optional -X Repo=$GITHUB_REPOSITORY
        ▼                                    │
internal/version                             │
  Module  = module path                      │
  Repo    = owner/name (or -X)               │
  Name    = squad-oc                         │
  Version = "dev" or -X tag  <───────────────┘

consumers: cli version / help, updatecheck, selfupdate (upgrade --self)
```

`GITHUB_REPOSITORY` is only applied when the env var is non-empty so an empty value cannot wipe the module-derived default.

## Files

| File | Change |
|------|--------|
| `internal/version/version.go` | `var Version`, `var Repo`, `const Module`, `const Name` |
| `internal/version/version_test.go` | Identity vs `go.mod` / goreleaser; default `dev`; ldflags smoke build |
| `.goreleaser.yaml` | ldflags `-X Version`; conditional `-X Repo`; drop hardcoded GitHub owner/name |
| `internal/selfupdate/selfupdate.go` | Asset/binary names use `version.Name` |
| `internal/selfupdate/selfupdate_test.go` | Use `version.Repo` instead of a literal |
| `CONTRIBUTING.md` | Local `dev` vs injected tag |

## Testing / verification

- `go test ./internal/version ./internal/selfupdate ./internal/updatecheck ./internal/cli`
- `go test ./...` and `go build -o squad-oc ./cmd/squad-oc` (`version` prints `dev`)
- Ldflags smoke: build with `-X ...Version=9.9.9` and `squad-oc version` prints `9.9.9`
- No live `v*` tag. Next real tag after merge is the live proof.

## Out of scope (already tracked)

| Topic | Bead |
|-------|------|
| Broader CI overhaul | `squad-h7s.4` |
| OpenCode v2 plugin spike | `squad-h7s.3` |
