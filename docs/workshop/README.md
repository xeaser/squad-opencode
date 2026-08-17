# Workshop — Squad for OpenCode

Zero to org-shaped team. **squad-oc + OpenCode only.**

You do **not** need GitHub Copilot, Node, the npm `@bradygaster/squad-cli`, or the original Squad Ink shell.

**Time:** about 90 minutes. Sections 1–6 ship today. Section 7 is stubs for later phases.

Modeled on [Tamir's original-Squad workshop](https://github.com/tamirdresher/squad-skills/tree/main/workshop), rewritten for this host. Catalog: [use-cases.md](../use-cases.md). Install detail: [get-started.md](../get-started.md).

| # | Section | Time | Status |
|---|---------|------|--------|
| 1 | [Tools](#1-tools) | 10 min | supported today |
| 2 | [Solo repo](#2-solo-repo) | 15 min | supported today |
| 3 | [Talk to the team](#3-talk-to-the-team) | 15 min | supported today |
| 4 | [Memory](#4-memory) | 10 min | supported today |
| 5 | [Org shape](#5-org-shape) | 20 min | supported today |
| 6 | [Ralph](#6-ralph) | 10 min | supported today |
| 7 | [What's next](#7-whats-next) | 10 min | later phase (stubs) |

---

## 1. Tools

**Time:** about 10 minutes.

| Need | Why |
|------|-----|
| Terminal | Run tools |
| **Go 1.26.6+** (to build `squad-oc`) | Or a prebuilt binary when published |
| Git | Project + recommended for real work |
| [OpenCode](https://opencode.ai) | Host TUI and `opencode serve` |
| An LLM provider | OpenCode talks to a model |

**Do not install:** GitHub Copilot CLI, `npm i -g @bradygaster/squad-cli`, or anything named `squad` that is not this repo’s `squad-oc`.

Optional until [section 6](#6-ralph): GitHub CLI (`gh`) and a repo with Issues.

### Install OpenCode

**macOS / Linux:**

```bash
curl -fsSL https://opencode.ai/install | bash
```

**Windows (pick one):**

```powershell
npm install -g opencode-ai
# or: scoop install opencode
# or: choco install opencode
```

```bash
opencode --version
```

### Connect a model

```bash
opencode
```

Inside OpenCode:

1. Run `/connect`
2. Choose a provider
3. Paste your API key
4. Quit when done

### Build `squad-oc` and put it on `PATH`

From this repository:

```bash
cd /path/to/squad-opencode
go build -o squad-oc ./cmd/squad-oc
```

Put the binary on your `PATH`, or call it with a full path. On Windows the file is `squad-oc.exe`.

### Try it

```bash
opencode --version
squad-oc version
squad-oc help
git --version
```

### Success looks like

- [ ] `opencode --version` prints a version
- [ ] `squad-oc version` and `squad-oc help` run (`init`, `doctor`, `status`, `link`, `upstream`, `watch` are listed)
- [ ] `/connect` accepted a provider (a later prompt will answer)
- [ ] No Copilot CLI and no npm Squad CLI on the critical path

---

## 2. Solo repo

**Time:** about 15 minutes.

One project, default cast, first **yes**.

```bash
mkdir my-recipe-app
cd my-recipe-app
git init
echo "# Recipe app" > README.md
git add README.md
git commit -m "chore: initial commit"

squad-oc init --preset default --description "Recipe sharing app with React and Node"
squad-oc doctor
squad-oc status
```

**Creates:**

- `.squad/` — team state (`team.md`, `decisions.md`, charters)
- `.opencode/agents/` — `squad` + role subagents
- `.opencode/skills/` — team + handoff
- `.opencode/commands/` — `/squad-status`, `/squad-cast`
- `opencode.json` — only if missing

Re-running `init` is safe (no overwrite once `.squad/config.json` exists).

Open the host:

```bash
opencode
```

1. Switch to the **squad** agent (**Tab**, or pick `squad`)
2. Describe the project
3. Reply **yes** when the coordinator proposes Lead / Frontend / Backend / Tester

```text
I'm starting a recipe sharing app with React and Node.
Set up the team and confirm roles.
```

Or run `/squad-cast`, then **yes**.

### Try it

Run the `init` → `doctor` → `opencode` → Tab → `squad` → describe → **yes** path above. Peek at `.squad/team.md`.

### Success looks like

- [ ] `.squad/team.md` lists Lead, Frontend, Backend, Tester
- [ ] Required `squad-oc doctor` checks pass (missing OpenCode **server** is soft; missing binary or scaffold is hard)
- [ ] **squad** agent is selected
- [ ] Cast confirmed with **yes**

---

## 3. Talk to the team

**Time:** about 15 minutes.

Stay in OpenCode as **squad**. The coordinator routes; specialists implement.

Ask the whole team:

```text
Team, add a usage endpoint that returns request counts.
Delegate the API to backend, a small page to frontend, and cases to tester.
```

Address one role when you already know the owner:

```text
@backend Sketch GET /api/usage.
@frontend Scaffold a usage panel.
@tester List cases for an auth-required usage endpoint.
```

Optional:

```text
/squad-status
```

```bash
squad-oc status
```

You review diffs. Humans merge.

### Try it

Send the `Team, add a usage endpoint…` prompt, then at least one `@backend` / `@frontend` / `@tester` line.

### Success looks like

- [ ] Coordinator splits the work instead of implementing every layer itself
- [ ] `@backend`, `@frontend`, and `@tester` each respond in their layer
- [ ] `squad-oc status` still shows the same cast

---

## 4. Memory

**Time:** about 10 minutes.

Durable rules live in `.squad/decisions.md`. Every new **squad** session is told to skim it. There is **no** `squad_state` MCP — files are the memory.

In the same OpenCode session:

```text
Always require auth on usage. Record that as an accepted team decision.
```

Open `.squad/decisions.md`. You should see a newest-first entry (title, status, decision).

Quit OpenCode. Start it again in the same repo, Tab → **squad**:

```text
What is our rule for the usage endpoint?
```

The coordinator should still require auth, from the file, not from chat history.

### Try it

Record “Always require auth on usage”, confirm the file, restart the session, ask again.

### Success looks like

- [ ] `.squad/decisions.md` contains the auth rule (accepted, newest first)
- [ ] A new session’s coordinator repeats the rule without you pasting the old chat
- [ ] You did not install a memory MCP server

---

## 5. Org shape

**Time:** about 20 minutes.

This is the multi-repo DX: one shared team (`link`) + one skills pack (`upstream`). Service repos do not invent a team.

Three roles:

| Who | Repo | What they do |
|-----|------|----------------|
| Platform owner | `acme/squad-platform` | `squad-oc init`, edit `.squad/decisions.md` |
| Pack owner | `acme/squad-skills` | Add `.opencode/skills/…` (and optional extra agents) |
| Service | e.g. `billing` | `init` → `link` → `upstream add` → `sync` → `doctor` |

`link` does not move files. Each service keeps its own `config.json`; `squad-oc status` / `doctor` read `team.md`, charters, and decisions from the platform directory. `upstream sync` copies pack files into **this** repo’s `.opencode/` (new `.squad/agents/…` only if missing). It never overwrites `team.md` or `decisions.md`.

### Phase 0 org example

Use this when the pack is a real git remote. Paths can be `~/teams/…` or sibling folders.

```bash
# platform
cd ~/teams/platform && squad-oc init --preset default --description "Acme platform"
# billing
cd ~/code/billing
squad-oc init --preset default
squad-oc link ~/teams/platform
squad-oc upstream add org git@github.com:acme/squad-skills.git
squad-oc upstream sync org
squad-oc doctor
opencode   # Tab → squad → "Set up the team" → yes
```

### Try it

Runnable with local folders (no `acme` GitHub org required):

```bash
mkdir acme
mkdir acme/squad-platform
mkdir acme/squad-skills
mkdir acme/billing

cd acme/squad-platform
git init
squad-oc init --preset default --description "Acme platform"
```

Edit `.squad/decisions.md` in that platform repo — append, newest first:

```markdown
### 2026-08-17 — Usage endpoints require auth

- **Status:** accepted
- **Context:** Org-wide API rule
- **Decision:** Every usage endpoint requires auth
- **Consequences:** Services do not ship anonymous usage
```

Pack owner — a tiny skill the service will pull:

```bash
cd ../squad-skills
mkdir .opencode
mkdir .opencode/skills
mkdir .opencode/skills/acme-api
```

Write `.opencode/skills/acme-api/SKILL.md`:

```markdown
---
name: acme-api
description: Acme API conventions for service repos
---

# Acme API

Require auth on usage. Prefer boring JSON over new protocols.
```

Service:

```bash
cd ../billing
git init
squad-oc init --preset default
squad-oc link ../squad-platform
squad-oc upstream add org ../squad-skills
squad-oc upstream sync org
squad-oc doctor
squad-oc status
opencode
```

In OpenCode: Tab → **squad** → “Set up the team” → **yes**.

Same commands with a git pack URL: `squad-oc upstream add org <url>` then `squad-oc upstream sync org`.

### Success looks like

- [ ] Platform `.squad/decisions.md` has the org auth rule
- [ ] `squad-oc link` printed the shared team path
- [ ] `squad-oc status` in `billing` shows the platform roster
- [ ] `.opencode/skills/acme-api/SKILL.md` exists in `billing` after `upstream sync org`
- [ ] `squad-oc doctor` required checks pass
- [ ] OpenCode **squad** agent is available in the service repo

Detach later with `squad-oc link --off` (this repo uses its local `.squad/` again).

---

## 6. Ralph

**Time:** about 10 minutes.

Ralph is `squad-oc watch` (aliases: `triage`, `loop`). It lists GitHub issues via `gh` and, with `--execute`, prompts the **squad** agent over the OpenCode **HTTP API** (`opencode serve`), not the TUI.

Needs:

- A git remote `gh` can see
- `gh auth login` already done
- An open issue with a label you pass to `--label`

`watch --execute` starts `opencode serve` on `http://127.0.0.1:4096` if nothing is listening there. It will **not** auto-start if you pass `--url` or set `OPENCODE_BASE_URL` to anything else. Plain `opencode` does not listen on 4096.

```bash
# in the service (or solo) repo, after gh knows the remote
gh label create squad --description "Squad work" --force
gh issue create --title "Add usage endpoint" --body "GET /api/usage, auth required." --label squad

# dry pass — list only
squad-oc watch --once --label squad

# one execute pass
squad-oc watch --execute --once --label squad
```

Inspect the last snapshot:

```bash
squad-oc watch --health
```

Leave a loop running only if you want it (not required here):

```bash
squad-oc watch --execute --interval 10 --label squad
# stop: touch .squad/ralph-stop
```

Overnight quiet (no serve calls in the window) is `--overnight-start 18:00 --overnight-end 08:00`. Skip it in this 90 minutes.

### Try it

Create a `squad`-labeled issue, run `squad-oc watch --once --label squad`, then `squad-oc watch --execute --once --label squad`.

### Success looks like

- [ ] `gh issue list --label squad` shows the issue
- [ ] `watch --once --label squad` prints `issues=1` (or more) without Copilot
- [ ] `watch --execute --once --label squad` talks to `opencode serve` and returns a squad reply
- [ ] `watch --health` shows a last-poll snapshot

If you have no GitHub remote yet, stop after reading the commands — do not fake `gh` output.

---

## 7. What's next

**Time:** about 10 minutes. **Stubs.** These commands are **not** in `squad-oc help` today. Do not run them.

| Later | Phase | Intent |
|-------|-------|--------|
| Org MCP | P1 | Committed `.squad/mcp-config.json` translated into `opencode.json` `mcp` |
| Marketplace browse + install | P2 | Find a skill without memorizing a git URL (today: `pack` / `upstream`) |
| Office cast | P3 | `cast --theme office` — display names only; `@lead` stays stable |
| Named plugins | P4 | `install reflect@acme` instead of a path |
| Review lockout | P5 | Author must not patch a rejected change; protocol + files, not a kernel lock |
| Ceremonies | P6 | Design review / retro as files, not vibes |

Workshop sections that will grow later (do not do them this week):

- **Section 8 — Org MCP** — drop config, apply, ask OpenCode what tools it has
- **Section 9 — Skills marketplace** — browse and install `reflect`
- **Office theme (optional 3b)** — hire the Office; `@lead` still works
- **Independent review** — Tester rejects; someone other than Backend fixes
- **Ceremonies** — design review before a multi-agent feature

WorkIQ, Outlook COM, and Teams Adaptive Cards are **not** product features. P1 may *mention* Teams/ADO/GitHub as example MCP servers.

### Try it

Read the table. Confirm `squad-oc help` has **no** `mcp`, `marketplace`, or `plugin` command.

### Success looks like

- [ ] You can name what is later vs what you already ran in sections 1–6
- [ ] You did not wait on P1–P6 to adopt `init` / `link` / `upstream` / `watch`

---

## Original-Squad DX

Each original ease row is a **squad-oc command** you already ran, a **later** command, or **won’t port**. Full map: [use-cases.md](../use-cases.md).

| Original ease | squad-oc |
|---------------|----------|
| `npm i -g @bradygaster/squad-cli` | `go build` / put `squad-oc` on `PATH` |
| `squad init` | `squad-oc init --preset default` |
| `copilot --agent squad` | `opencode` → Tab → **squad** |
| `squad status` | `squad-oc status` |
| Shared team / org memory | `squad-oc link` |
| Extra skills pack | `squad-oc pack` / `upstream add` + `sync` |
| `squad watch` / Ralph | `squad-oc watch [--execute] [--once] [--label]` |
| Health of the monitor | `squad-oc watch --health` |
| Drop-in org MCP (`mcp-config.json`) | later (`mcp apply` in P1) |
| Marketplace browse + install | later (P2) |
| Office / themed names | later (`cast --theme office` in P3) |
| `install name@marketplace` | later (P4) |
| Reviewer lockout | later (P5) |
| Ceremonies as files | later (P6) |
| GitHub Copilot CLI / Copilot SDK | **won’t port** |
| Ink / `squad` shell | **won’t port** (use the OpenCode TUI) |
| Aspire / .NET dashboard | **won’t port** (`traces` is local OTLP JSON) |
| `squad_state` memory MCP | **won’t port** (`.squad/` files) |

---

## Troubleshooting

| Problem | Try |
|---------|-----|
| `squad-oc: command not found` | Build from this repo; fix `PATH` |
| `opencode: command not found` | Install OpenCode; fix `PATH` |
| No **squad** agent | `squad-oc init --preset default` in the project root |
| `Not initialized` | `squad-oc init --preset default` |
| `link` / no `team.md` | Init the platform repo first; pass that directory (or its `.squad/`) |
| `unknown upstream` | `upstream add org <path\|git-url>` then `upstream sync org` |
| `gh issue list` fails | `gh auth status`; repo must have a GitHub remote |
| Server probe FAIL only | Soft — start `opencode serve` only if you need `run` / `watch --execute` |
| Invented `mcp apply` | Not shipped — section 7 |

---

## Out of scope

Copilot CLI, an Ink/`squad` shell, Aspire, and a `squad_state` MCP server.

`run` / `watch --execute` use **`github.com/sst/opencode-sdk-go`** against `opencode serve` (not the TUI).
