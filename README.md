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
opencode
# Tab → squad agent → "Set up the team for …" → yes
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
| `squad-oc init --preset default` | Scaffold `.squad/` + `.opencode/` |
| `squad-oc doctor` | PATH, scaffold, optional OpenCode server probe (Go SDK) |
| `squad-oc status` | Print team from `.squad/team.md` |

## Layout

```
cmd/squad-oc/           # CLI
internal/squad/         # domain + embedded templates
internal/doctor/        # doctor checks
internal/opencodeclient/# thin opencode-sdk-go wrapper
docs/
```

## Architecture

```
squad-oc (Go)
  ├── embed templates → .squad / .opencode
  └── github.com/sst/opencode-sdk-go → OpenCode server (watch/spawn later)
```

## Non-goals (current MVP)

Ralph/watch execute, upgrade pipeline, export/import, marketplace, Copilot bridge, TS plugins.

## Develop

```bash
go test ./...
go build -o squad-oc ./cmd/squad-oc
```

Requires **Go 1.22+**.

## License

MIT — see [LICENSE](LICENSE).
