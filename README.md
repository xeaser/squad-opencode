# squad-opencode

[![CI](https://github.com/xeaser/squad-opencode/actions/workflows/ci.yml/badge.svg)](https://github.com/xeaser/squad-opencode/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/xeaser/squad-opencode?display_name=tag)](https://github.com/xeaser/squad-opencode/releases/latest)

**Human-led AI agent teams for [OpenCode](https://opencode.ai)** — implemented in **Go**.

One binary scaffolds a persistent team in your repo (`.squad/`) and OpenCode agents (`.opencode/`). You coordinate work; specialists (Lead, Frontend, Backend, Tester) run as OpenCode agents.

> Inspired by [bradygaster/squad](https://github.com/bradygaster/squad) (MIT). OpenCode-native port — not a Copilot fork.  
> Host API client: official [opencode-sdk-go](https://github.com/anomalyco/opencode-sdk-go) (`github.com/sst/opencode-sdk-go`).

## Install

No Go toolchain. Download the archive for your OS from [GitHub Releases](https://github.com/xeaser/squad-opencode/releases/latest) and put `squad-oc` on your `PATH`. Assets are `squad-oc_<version>_<os>_<arch>.zip` (Windows) or `.tar.gz` (macOS/Linux).

```bash
# Scoop
scoop bucket add squad https://github.com/xeaser/squad-opencode
scoop install squad-oc

# Homebrew
brew tap xeaser/squad-opencode https://github.com/xeaser/squad-opencode
brew install squad-oc
# The tap ships a cask; existing formula installs migrate via tap_migrations.json.

# winget (from a clone of this repo)
winget install --manifest packaging/winget
```

`squad-oc upgrade --self` replaces the binary from the latest release.

## Quick start

```bash
# 1. Install OpenCode + /connect a provider — https://opencode.ai/docs/

# 2. Install squad-oc from GitHub Releases (see Install above)

# 3. In your project
mkdir my-app && cd my-app && git init
squad-oc init --preset default
squad-oc doctor

# Interactive TUI (does not listen on :4096)
opencode
# Tab → squad agent → "Set up the team for …" → yes

# HTTP API: run auto-starts `opencode serve` on :4096 if nothing is there
squad-oc run -p "Summarize .squad/team.md"
```

Full walkthrough: **[docs/get-started.md](docs/get-started.md)**

## Also useful

- **[docs/workshop/README.md](docs/workshop/README.md)** — 90-minute adopt path (squad-oc + OpenCode only)
- **[docs/use-cases.md](docs/use-cases.md)** — workshop mapped to supported / later / won’t port

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
| `init [--preset default] [--description <text>] [--global] [--theme office|none]` | Scaffold `.squad/` + `.opencode/` (`--global` uses the user config dir) |
| `upgrade [--dry-run] [--force] [--global] [--self]` | Refresh host templates; `--self` replaces this binary from GitHub Releases |
| `doctor` / `heartbeat` | Health checks |
| `status` / `cast` | Team table |
| `brief [--json]` | Morning listing: team, PRs, tickets, last done, next, Ralph |
| `cast --add <name> [--role <role>] [--model <provider/id>]` | Add a member and regenerate `.opencode/agents` |
| `cast --model <name> <provider/id>` | Pin a member to an OpenCode model id |
| `cast --model squad <provider/id>` | Pin the coordinator model |
| `cast --model <name|-> -` | Clear a pin (inherit Squad, then session) |
| `cast --remove <name>` | Remove a member and regenerate `.opencode/agents` |
| `cast --theme office` / `none` | `init --theme office` (native `@michael`) or later `cast --theme office` (mention map; `@lead` gone) |
| `recast` | Regenerate `.opencode/agents` from `.squad/team.md` |
| `run -p <prompt>` / `--file <path> [--agent name] [--url]` | Prompt the OpenCode HTTP API as `squad`; auto-starts `opencode serve` on :4096 only |
| `watch` / `triage` / `loop` `[--execute] [--interval minutes] [--once] [--health] [--url] [--overnight-start HH:MM] [--overnight-end HH:MM] [--label name] [--log-file path] [--verbose] [--notify-level all\|important\|none] [--state-backend memory\|git-notes\|orphan-branch]` | Issue triage (Ralph); `--execute` uses `run` |
| `export [file]` / `import <file> [--with-host]` | JSON snapshot of `.squad/` (optional host files) |
| `externalize [--key name]` / `internalize` | Move *this* project's team out of the worktree |
| `nap [--dry-run] [--deep]` / `scrub-emails [directory]` | Context and PII hygiene |
| `upstream add <name> <path\|git-url>` / `list` / `remove` / `sync` | Remember and pull extra agents/skills |
| `pack <path\|git-url>` | One-shot pull of extra agents/skills |
| `link <team-dir|git-url>` / `link --sync` / `link --off` | Share one team directory across several repos (git URL clones into `~/.squad-oc/links/`) |
| `update-check [--json] [--refresh]` | Prints `up to date` or `update available` vs GitHub latest tag |
| `traces [--last N] [--json] [--export file]` | Local `run` / `watch` spans (`.squad/traces/spans.jsonl`); `--export` writes OTLP JSON; optional live push via `OTEL_EXPORTER_OTLP_*` |
| `mcp apply` / `list` / `init` | Merge org `.squad/mcp-config.json` into `opencode.json` |
| `marketplace add` / `list` / `remove` / `browse` / `install` | Register a skills pack and copy a plugin into `.opencode/skills/` |
| `plugin install <name>@<marketplace>` / `list` / `uninstall <name>` | Named skill install; uninstall removes only `.opencode/skills/<name>/` |
| `help` / `version` | Usage and version string |

## Layout

```
cmd/squad-oc/            # main → internal/cli
internal/cli/            # commands
internal/squad/          # init, upgrade, export, externalize, nap, scrub, templates
internal/opencodeclient/ # SDK + run (needs `opencode serve`)
internal/watch/          # issue triage (Ralph): health, overnight, backends
internal/githubissues/   # gh issue list
internal/share/          # upstream / pack / link
internal/traces/         # local JSONL spans + OTLP export
internal/selfupdate/     # upgrade --self
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

`run` / `watch --execute` attach to an existing server, or start `opencode serve --hostname 127.0.0.1 --port 4096` in this project. They **never** auto-start if `--url` or `OPENCODE_BASE_URL` is anything other than `http://127.0.0.1:4096` / `http://localhost:4096`. The serve process is left running.

`upgrade` refreshes host templates (`.opencode/`). It never overwrites team memory (`team.md`, decisions, knowledge).

`upgrade --self` downloads the latest GitHub Release for this OS/arch and replaces the running binary. On Windows, if the exe is locked, it writes `squad-oc.exe.new` beside it (`replaced on next start`).

`traces` lists local spans from `run` and `watch --execute`. Default storage is `.squad/traces/spans.jsonl`. `--export file` writes OTLP JSON any collector can ingest. Set `OTEL_EXPORTER_OTLP_ENDPOINT` (or `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`) to push live during `run` / `watch --execute`. Protocols: `http/protobuf` (default) and `grpc` via `OTEL_EXPORTER_OTLP_PROTOCOL`. Prompt/completion bodies always land in local JSONL; OTel message attributes only when `OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT` is on (default off). Langfuse local example: endpoint `http://127.0.0.1:3000/api/public/otel`, Basic Auth in `OTEL_EXPORTER_OTLP_HEADERS`, plus header `x-langfuse-ingestion-version=4`. Aspire Dashboard remains a valid OTLP **consumer**, not a shipped `squad-oc` command.

### Share extra agents (`upstream` / `pack`)

Your company keeps a **security pack** — extra OpenCode agents and a Designer charter — in a git repo or a folder on disk.

```bash
# One-shot: drop those files into *this* project
squad-oc pack https://github.com/acme/squad-security-pack.git
# or a local folder you already cloned
squad-oc pack ~/packs/security

# Remember it and pull again when the pack updates
squad-oc upstream add security https://github.com/acme/squad-security-pack.git
squad-oc upstream sync security     # next quarter: same command
squad-oc upstream list
```

`pack` / `sync` refresh `.opencode/` (agents, skills, commands). New `.squad/agents/<name>/` files are added only if missing. `team.md`, decisions, and existing knowledge are never overwritten.

A pack is just a directory that contains `.opencode/` (or a bare `agents/` / `skills/` / `commands/` tree). Optional `.squad/agents/…` stubs are fine.

### Share one team across repos (`link`)

Platform Lead, decisions, and knowledge live in **one** place. Billing and Checkout both use that team.

```bash
# once: a dedicated team folder (or any repo that already has .squad/)
mkdir -p ~/teams/platform && cd ~/teams/platform
git init && squad-oc init --preset default --description "Platform team"

cd ~/code/billing
squad-oc init --preset default
squad-oc link ~/teams/platform          # also accepts ~/teams/platform/.squad

cd ~/code/checkout
squad-oc init --preset default
squad-oc link ~/teams/platform

squad-oc status                         # both show the same members
# later
squad-oc link --off                     # this repo uses its local .squad/ again
```

```bash
# or, no local clone — cache lives in ~/.squad-oc/links/
squad-oc link https://github.com/acme/squad-platform.git
squad-oc link --sync                    # fetch updates
squad-oc link --off
```

`link` does not move files. Config stays in each repo; `team.md` / charters / decisions are read from the shared directory. Cannot combine with `externalize` (that *moves* this project's own team out of the worktree).

### Change the cast (`cast --add` / `cast --remove` / `recast`)

```bash
squad-oc cast --add Designer --role Design
# appends Designer to team.md, writes charter/knowledge, regenerates .opencode/agents

squad-oc cast --remove Designer
# drops the member and regenerates .opencode/agents

# or edit .squad/team.md by hand, then:
squad-oc recast
```

OpenCode only sees agents that exist under `.opencode/agents/`. Recast is what makes a new row in `team.md` show up as `@designer`. It never touches `squad.md` (the coordinator).

### Overnight watch

```bash
# poll and execute during the day; sleep 18:00–08:00 local
squad-oc watch --execute --interval 10 --overnight-start 18:00 --overnight-end 08:00
```

During the quiet window it does not call `opencode serve`. Stop anytime with `touch .squad/ralph-stop`.

`watch --health` prints the last `ralph-status.json` snapshot. `--state-backend git-notes` or `orphan-branch` persists that snapshot across restarts (default is the local file).

## Non-goals

Original-Squad / Copilot-host pieces we are not building:

- GitHub Copilot CLI / Copilot SDK
- Interactive Ink/`squad` shell (use the OpenCode TUI)
- Aspire / .NET dashboard (won’t ship; point any OTLP consumer at optional push or `traces --export`)

## Develop

```bash
go test ./...
go build -o squad-oc ./cmd/squad-oc
```

Requires **Go 1.26.6+**. How to send a change: **[CONTRIBUTING.md](CONTRIBUTING.md)**.

## License

MIT — see [LICENSE](LICENSE).
