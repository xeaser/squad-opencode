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

# HTTP API: run auto-starts `opencode serve` on :4096 if nothing is there
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
| `init [--preset default] [--description <text>]` | Scaffold `.squad/` + `.opencode/` |
| `upgrade [--dry-run] [--force] [--self]` | Refresh host templates; `--self` replaces this binary from GitHub Releases |
| `doctor` | Health checks |
| `status` / `cast` | Team table |
| `cast --add <name> [--role <role>]` | Add a member and regenerate `.opencode/agents` |
| `cast --remove <name>` | Remove a member and regenerate `.opencode/agents` |
| `recast` | Regenerate `.opencode/agents` from `.squad/team.md` |
| `run -p <prompt>` / `--file <path> [--agent name] [--url]` | Prompt the OpenCode HTTP API as `squad`; auto-starts `opencode serve` on :4096 only |
| `watch [--execute] [--interval minutes] [--once] [--url] [--overnight-start HH:MM] [--overnight-end HH:MM]` | Issue triage (Ralph); execute uses `run` |
| `export [file]` / `import <file> [--with-host]` | JSON snapshot of `.squad/` (optional host files) |
| `externalize [--key name]` / `internalize` | Move *this* project's team out of the worktree |
| `nap [--dry-run] [--deep]` / `scrub-emails [directory]` | Context and PII hygiene |
| `upstream add <name> <path\|git-url>` / `list` / `remove` / `sync` | Remember and pull extra agents/skills |
| `pack <path\|git-url>` | One-shot pull of extra agents/skills |
| `link <team-dir>` / `link --off` | Share one team directory across several repos |
| `update-check [--json] [--refresh]` | Prints `up to date` or `update available` vs GitHub latest tag |
| `help` / `version` | Usage and version string |

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

`run` / `watch --execute` attach to an existing server, or start `opencode serve --hostname 127.0.0.1 --port 4096` in this project. They **never** auto-start if `--url` or `OPENCODE_BASE_URL` is anything other than `http://127.0.0.1:4096` / `http://localhost:4096`. The serve process is left running.

`upgrade` refreshes host templates (`.opencode/`). It never overwrites team memory (`team.md`, decisions, knowledge).

`upgrade --self` downloads the latest GitHub Release for this OS/arch and replaces the running binary. On Windows, if the exe is locked, it writes `squad-oc.exe.new` beside it (`replaced on next start`).

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

`link` does not move files. Config stays in each repo; `team.md` / charters / decisions are read from the shared directory. Cannot combine with `externalize` (that *moves* this project's own team out of the worktree).

### Change the cast (`cast --add` / `recast`)

```bash
squad-oc cast --add Designer --role Design
# appends Designer to team.md, writes charter/knowledge, regenerates .opencode/agents

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

## Later

Not in this release:

- **Agent traces** via OpenTelemetry GenAI (not Aspire). Goal: a self-contained `squad-oc traces` (or similar) plus export that any OTEL backend can scrape
- Watch state backends (`git-notes`)

## Non-goals

Original-Squad / Copilot-host pieces we are not building:

- GitHub Copilot CLI / Copilot SDK
- Interactive Ink/`squad` shell (use the OpenCode TUI)
- Aspire / .NET dashboard (traces, if any, will be OTEL)

## Develop

```bash
go test ./...
go build -o squad-oc ./cmd/squad-oc
```

Requires **Go 1.22+**.

## License

MIT — see [LICENSE](LICENSE).
