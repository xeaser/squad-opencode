# squad-opencode

**Human-led AI agent teams for [OpenCode](https://opencode.ai)** — implemented in **Go**.

One binary scaffolds a persistent team in your repo (`.squad/`) and OpenCode agents (`.opencode/`). You coordinate work; specialists (Lead, Frontend, Backend, Tester) run as OpenCode agents.

> Inspired by [bradygaster/squad](https://github.com/bradygaster/squad) (MIT). OpenCode-native port — not a Copilot fork.  
> Host API client: official [opencode-sdk-go](https://github.com/anomalyco/opencode-sdk-go) (`github.com/sst/opencode-sdk-go`).

## Quick start

```bash
# 1. Install OpenCode + /connect a provider — https://opencode.ai/docs/

# 2. Build squad-oc
git clone <this-repo> && cd squad-opencode
go build -o squad-oc ./cmd/squad-oc
# optional: move squad-oc onto your PATH

# 3. In your project
mkdir my-app && cd my-app && git init
/path/to/squad-oc init --preset default
/path/to/squad-oc doctor

# Interactive TUI (does not listen on :4096)
opencode
# Tab → squad agent → "Set up the team for …" → yes

# HTTP API for `run` / `watch --execute` (separate terminal)
opencode serve
squad-oc run -p "Summarize .squad/team.md"
```

Full walkthrough: **[docs/get-started.md](docs/get-started.md)**

## What to commit

This repo dogfoods Squad on itself. After `init` + opening OpenCode:

| Commit | Do not commit |
|--------|----------------|
| `.squad/` team, charters, decisions, `.gitignore` | `.squad/comms/*` (folder kept via `.gitkeep`) |
| `.opencode/agents`, `skills`, `commands`, `.gitignore` | `.opencode/node_modules/` |
| `opencode.json` | `.opencode/package.json` + lockfiles OpenCode generates |

OpenCode creates `.opencode/package.json` (`@opencode-ai/plugin`) and runs an install on first launch. That is host runtime, not Squad source.

## Commands

| Command | Role |
|---------|------|
| `init` | Scaffold `.squad/` + `.opencode/` |
| `upgrade` | Refresh host templates; never wipe team state |
| `doctor` / `status` / `cast` | Health and team table |
| `run -p` | Prompt `opencode serve` (HTTP API on :4096) as `squad` |
| `watch [--execute] [--once]` | Issue triage (Ralph); execute uses `run` |
| `export` / `import` | JSON snapshot of `.squad/` |
| `externalize` / `internalize` | Move team state out of the worktree |
| `nap` / `scrub-emails` | Context and PII hygiene |
| `upstream` / `pack` / `link` | Template sources, packs, shared team |
| `update-check` | Prints `up to date` or `update available` vs GitHub latest tag |

## Layout

```
cmd/squad-oc/            # main → internal/cli
internal/cli/            # commands
internal/squad/          # init, upgrade, export, externalize, nap, scrub, templates
internal/opencodeclient/ # SDK + run (needs `opencode serve`)
internal/watch/          # issue triage (Ralph)
internal/githubissues/   # gh issue list
internal/share/          # upstream / pack / link
internal/updatecheck/
internal/version/
docs/
```

## Architecture

```
squad-oc (Go)
  ├── embed templates → .squad / .opencode
  └── github.com/sst/opencode-sdk-go → opencode serve (:4096)
```

`opencode` (TUI) and `opencode serve` (HTTP API) are different. Only **serve** works with `run` and `watch --execute`.

`upgrade` refreshes host templates (`.opencode/`). It never overwrites team memory (`team.md`, decisions, knowledge).

`upgrade --self` is not wired yet — rebuild with `go install ./cmd/squad-oc` (or `go build`).

## Non-goals

- Copilot CLI / Copilot SDK
- Interactive Ink/`squad` shell (use the OpenCode TUI)
- Aspire / .NET dashboard
- npm as the primary distribution
- Auto-spawning `opencode serve` from `run` (start it yourself)
- Recast / generate agents from `.squad/` (not yet)
- Overnight watch windows and git-notes backends

## Develop

```bash
go test ./...
go build -o squad-oc ./cmd/squad-oc
```

Requires **Go 1.22+**.

## License

MIT — see [LICENSE](LICENSE).
