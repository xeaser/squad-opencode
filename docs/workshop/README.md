# Workshop — Squad for OpenCode

Zero to org-shaped team. **squad-oc + OpenCode only.**

You do **not** need GitHub Copilot, Node, the npm `@bradygaster/squad-cli`, or the original Squad Ink shell.

**Time:** about 100 minutes. Sections 1–6 and 8 ship today. Section 7 is stubs for later phases.

Modeled on [Tamir's original-Squad workshop](https://github.com/tamirdresher/squad-skills/tree/main/workshop), rewritten for this host. Catalog: [use-cases.md](../use-cases.md). Install detail: [get-started.md](../get-started.md).

| # | Section | Time | Status |
|---|---------|------|--------|
| 1 | [Tools](#1-tools) | 10 min | supported today |
| 2 | [Solo repo](#2-solo-repo) | 15 min | supported today |
| 3 | [Talk to the team](#3-talk-to-the-team) | 15 min | supported today |
| 3b | [Hire the Office](#3b-hire-the-office) | 5 min | optional |
| 4 | [Memory](#4-memory) | 10 min | supported today |
| 5 | [Org shape](#5-org-shape) | 20 min | supported today |
| 6 | [Ralph](#6-ralph) | 10 min | supported today |
| 7 | [What's next](#7-whats-next) | 5 min | later phase (stubs) |
| 8 | [Org MCP](#8-org-mcp) | 10 min | supported today |
| 9 | [Skills marketplace](#9-skills-marketplace) | 10 min | supported today |
| 10 | [Independent review](#10-independent-review) | 5 min | supported today (protocol) |
| 11 | [Ceremonies](#11-ceremonies) | 5 min | supported today |

---

## 1. Tools

**Time:** about 10 minutes.

| Need | Why |
|------|-----|
| Terminal | Run tools |
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

### Install `squad-oc` and put it on `PATH`

You do **not** need Go. Download the archive for your OS from [GitHub Releases](https://github.com/xeaser/squad-opencode/releases/latest) and put `squad-oc` (`squad-oc.exe` on Windows) on your `PATH`. Assets are `squad-oc_<version>_<os>_<arch>.zip` (Windows) or `.tar.gz` (macOS/Linux).

```bash
# Scoop
scoop bucket add squad https://github.com/xeaser/squad-opencode
scoop install squad-oc

# Homebrew
brew tap xeaser/squad-opencode https://github.com/xeaser/squad-opencode
brew install squad-oc

# winget (from a clone of this repo)
winget install --manifest packaging/winget
```

Later: `squad-oc upgrade --self`.

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

## 3b. Hire the Office

**Time:** about 5 minutes. **Optional.** Two paths. Never both `lead.md` and `michael.md`.

### Birth — `init --theme office`

A new project can start already in Office. Agent ids **are** the tags:

```bash
squad-oc init --theme office --preset default --description "Recipe sharing app with React and Node"
squad-oc status
# Michael  Lead
# Jim      Frontend
# Dwight   Backend
# Pam      Tester
```

Host files are native: `.opencode/agents/michael.md` exists; `lead.md` does not. There is **no** `.squad/mentions.md`. Chat with `@michael` (not `@lead`).

### Later apply — `cast --theme office`

On an existing default team, names change and mention files are replaced:

```bash
squad-oc cast --theme office
squad-oc status
# Michael  Lead
# Jim      Frontend
# Dwight   Backend
# Pam      Tester
```

Memory ids stay `lead` / `frontend` / `backend` / `tester`. Host mention files do not: `michael.md` exists, `lead.md` is gone. `.squad/mentions.md` maps **Tag now** (`@michael`) to **Was** (`@lead`). The coordinator uses **Tag now**; treat **Was** as the same specialist.

| Role | Office name | Tag now (later apply) | Was |
|------|-------------|----------------------|-----|
| Lead | Michael | `@michael` | `@lead` |
| Frontend | Jim | `@jim` | `@frontend` |
| Backend | Dwight | `@dwight` | `@backend` |
| Tester | Pam | `@pam` | `@tester` |
| Coordinator | Squad (not a character) | `@squad` | `@squad` |

Restore with `--theme none`:

```bash
squad-oc cast --theme none
```

- After **later apply**: display names and `@lead` host files come back; the mention map is dropped. Memory ids were always `lead`/….
- After **birth** (`init --theme office`): this is a real rename. Memory dirs go back to `lead`/`frontend`/…, host files become `lead.md` again, and there is still no mention map.

### Try it

**Birth:** `init --theme office`, then `status`. Confirm `.opencode/agents/michael.md` exists, `lead.md` does not, and there is no `.squad/mentions.md`. Restore with `--theme none` and confirm `agents/lead/` and `lead.md` are back.

**Later apply:** in a default project, `cast --theme office`, then `status`. Confirm `michael.md` yes, `lead.md` no, `.squad/mentions.md` yes. Restore with `--theme none`.

### Success looks like

- [ ] Birth: status shows Michael / Jim / Dwight / Pam; only `@michael` (no mentions.md, no `lead.md`)
- [ ] Later apply: mention map present; `@lead` gone; `@michael` works
- [ ] Never both `lead.md` and `michael.md`
- [ ] `--theme none` after later apply brings back Lead / Frontend / Backend / Tester and `@lead`
- [ ] `--theme none` after birth renames memory ids back to `lead`/… and restores `@lead`

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

`link` does not move files. Each service keeps its own `config.json`. `squad-oc status` reads roster and decisions from the platform; `doctor`'s team-file check follows the link. `upstream sync` copies pack files into **this** repo's `.opencode/` (new `.squad/agents/…` only if missing). It never overwrites `team.md` or `decisions.md`.

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

Runnable with local folders (no `acme` GitHub org required). Leave the solo recipe repo first so this is not nested inside `my-recipe-app`:

```bash
cd ..
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
# relative pack path — run add and later sync from this service root
squad-oc upstream add org ../squad-skills
squad-oc upstream sync org
squad-oc doctor
squad-oc status
opencode
```

In OpenCode: Tab → **squad** → "Set up the team" → **yes**.

Same commands with a git pack URL: `squad-oc upstream add org <url>` then `squad-oc upstream sync org` (still from the service root).

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

Ralph is `squad-oc watch` (aliases: `triage`, `loop`). It lists GitHub issues via `gh` and, with `--execute`, prompts the **squad** agent over the OpenCode **HTTP API** (`opencode serve`), not the TUI. Optional `--project N` keeps only issues on that GitHub Project v2 (`--column` matches Status). Issues that already have an open linked PR are skipped; `--force` or `--retry-label` (default `ralph-retry`) re-enables them.

Needs:

- A git remote `gh` can see
- `gh auth login` already done
- An open issue with a label you pass to `--label`

`watch --execute` starts `opencode serve` on `http://127.0.0.1:4096` if nothing is listening there. It will **not** auto-start if you pass `--url` or set `OPENCODE_BASE_URL` to anything else. Plain `opencode` does not listen on 4096.

```bash
# in the service (or solo) repo — or any repo gh already sees
gh repo create --source=. --public --push
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
# stop (Unix): touch .squad/ralph-stop
# stop (PowerShell): New-Item .squad/ralph-stop
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

**Time:** about 5 minutes.

P1–P6 from the org DX roadmap are in this workshop now (MCP, marketplace, Office names, named plugins, review protocol, ceremonies). What we still **will not** port is in [Out of scope](#out-of-scope) and [use-cases.md](../use-cases.md).

WorkIQ, Outlook COM, and Teams Adaptive Cards are **not** product features. Section 8 may *mention* Teams/ADO/GitHub as example MCP servers.

### Try it

Confirm `squad-oc help` lists `mcp`, `marketplace`, `plugin`, and `cast --theme`.

### Success looks like

- [ ] You can run the shipped commands without reading original Squad docs
- [ ] You can name the won’t-port rows (Copilot CLI, Ink shell, Aspire, `squad_state`)

---

## 8. Org MCP

**Time:** about 10 minutes.

The org authors one file: `.squad/mcp-config.json` (after `link`, that is the **shared** team dir). `squad-oc mcp apply` translates it into `opencode.json` `mcp`. OpenCode reads the host file. `squad-oc` does not start MCP processes.

`init` does not write MCP. `doctor` has a **soft** check: if the org file exists, `opencode.json` must list those server names.

Use `${VAR}` or `{env:VAR}`. Never put `sk-` / `ghp_` literals in the file — apply refuses them.

This workshop uses a **disabled remote stub** so you do not need Teams, ADO, or a GitHub token.

```bash
# in the platform repo (or a service after link — apply reads the linked team dir)
cd acme/squad-platform
squad-oc mcp init
```

If the file already exists, `mcp init` leaves it. Edit `.squad/mcp-config.json` so it looks like this (comments are allowed):

```json
{
  "mcpServers": {
    "example-remote": {
      "url": "https://example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${MCP_TOKEN}"
      },
      "enabled": false
    }
  }
}
```

```bash
squad-oc mcp apply
squad-oc mcp list
squad-oc doctor
```

`mcp list` prints name / source / enabled / applied — not secret values.

Reload OpenCode if it was already open (the host may cache config). Ask:

```text
List the MCP tools you have
```

A disabled stub will not connect. That is success for this section: the name is in `opencode.json` and doctor is green.

### Try it

`mcp init` → edit the disabled stub → `mcp apply` → `mcp list` → `doctor`. Peek at `opencode.json` `mcp`.

### Success looks like

- [ ] `.squad/mcp-config.json` exists in the resolved team dir (platform if linked)
- [ ] `opencode.json` has `"example-remote"` (or your server name) and still has `"$schema"`
- [ ] `mcp list` shows `enabled=false` `applied=true` and does not print tokens
- [ ] `doctor` **MCP apply** is OK (soft check; missing apply is FAIL but exit stays 0 if required checks pass)

---

## 9. Skills marketplace

**Time:** about 10 minutes.

Browse a pack without memorizing a git URL, then install one skill into this repo’s `.opencode/skills/`. This does **not** overwrite `team.md` or `decisions.md`.

A tiny offline fixture lives in this repo: `docs/workshop/fixtures/skills-pack/` (`reflect` and `fact-checking`).

```bash
# from the squad-opencode checkout — or pass an absolute path to the fixture
squad-oc marketplace add community /path/to/squad-opencode/docs/workshop/fixtures/skills-pack
squad-oc marketplace browse
# name                         source      triggers
# reflect                      community   retrospective, lessons learned
# fact-checking                community   verify, validate claims

squad-oc marketplace install reflect --from community
# → .opencode/skills/reflect/SKILL.md
```

If only one marketplace is registered, `--from` is optional.

Prefer the named form when you know the source:

```bash
squad-oc plugin install reflect@community
squad-oc plugin list
# path-based install remains the fallback: marketplace install reflect --from community
```

In OpenCode as **squad**:

```text
Use the reflect skill on the last change.
```

### Try it

Add the fixture pack, `browse`, `install reflect`, confirm `.opencode/skills/reflect/SKILL.md`.

### Success looks like

- [ ] `marketplace browse` lists `reflect` and `fact-checking`
- [ ] `.opencode/skills/reflect/SKILL.md` exists after install
- [ ] `.squad/team.md` and `.squad/decisions.md` are unchanged

---

## 10. Independent review

**Time:** about 5 minutes. **Protocol + files**, not a kernel lock. OpenCode will not block the author from editing; the coordinator must not assign the author to apply a reject.

Demo: Tester rejects Backend; Lead (or Frontend) applies the fix.

```text
Team, add GET /api/usage with tests.
```

Tester writes `.squad/comms/YYYY-MM-DD-tester-to-lead.md` using the handoff **Review** block:

```text
Verdict: reject — no auth check
Author: backend
Fix owner: lead
Reasons: usage must require auth (see decisions.md)
```

Coordinator assigns `@lead`, **not** `@backend`. Optional: `/squad-review`.

### Try it

Have Tester reject a Backend change. Open the handoff file and check **Author ≠ Fix owner**.

### Success looks like

- [ ] Handoff file has Verdict, Author, Fix owner, Reasons
- [ ] Fix owner is not the author
- [ ] Coordinator does not patch the rejected diff itself

---

## 11. Ceremonies

**Time:** about 5 minutes.

Fresh `init` writes `.squad/ceremonies.md` (Design Review + Retro only). `upgrade` will not overwrite your edits.

Before a multi-agent feature, ask **squad** to run a Design Review (short questions, then `.squad/comms/…-design-review.md`). After a failed `watch` / `run`, it should **offer** a Retro — not start one unless you want it.

### Try it

```text
Run a design review for adding billed usage export.
```

### Success looks like

- [ ] `.squad/ceremonies.md` exists after `init`
- [ ] A design-review note appears under `.squad/comms/`
- [ ] You did not get an unsolicited retro

---

## Original-Squad DX

Each original ease row is a **squad-oc command** you already ran, a **later** command, or **won’t port**. Full map: [use-cases.md](../use-cases.md).

| Original ease | squad-oc |
|---------------|----------|
| `npm i -g @bradygaster/squad-cli` | GitHub Releases / Scoop / Homebrew / winget |
| `squad init` | `squad-oc init --preset default` |
| `copilot --agent squad` | `opencode` → Tab → **squad** |
| `squad status` | `squad-oc status` |
| Shared team / org memory | `squad-oc link` |
| Extra skills pack | `squad-oc pack` / `upstream add` + `sync` |
| `squad watch` / Ralph | `squad-oc watch [--execute] [--once] [--label]` |
| Health of the monitor | `squad-oc watch --health` |
| Drop-in org MCP (`mcp-config.json`) | `squad-oc mcp init` / `apply` / `list` |
| Marketplace browse + install | `squad-oc marketplace browse` / `install` |
| Office / themed names | `squad-oc init --theme office` (native `@michael`) or later `cast --theme office` (mention map; `@lead` gone) |
| `install name@marketplace` | `squad-oc plugin install reflect@community` |
| Reviewer lockout | protocol + files (`/squad-review`, handoff Review block) |
| Ceremonies as files | `.squad/ceremonies.md` (design review / retro) |
| GitHub Copilot CLI / Copilot SDK | **won’t port** |
| Ink / `squad` shell | **won’t port** (use the OpenCode TUI) |
| Aspire / .NET dashboard | **won’t port** (`traces` is local OTLP JSON) |
| `squad_state` memory MCP | **won’t port** (`.squad/` files) |

---

## Troubleshooting

| Problem | Try |
|---------|-----|
| `squad-oc: command not found` | Install from GitHub Releases; fix `PATH` |
| `opencode: command not found` | Install OpenCode; fix `PATH` |
| No **squad** agent | `squad-oc init --preset default` in the project root |
| `Not initialized` | `squad-oc init --preset default` |
| `link` / no `team.md` | Init the platform repo first; pass that directory (or its `.squad/`) |
| `unknown upstream` | `upstream add org <path\|git-url>` then `upstream sync org` |
| `gh issue list` fails | `gh auth status`; repo must have a GitHub remote |
| Server probe FAIL only | Soft — start `opencode serve` only if you need `run` / `watch --execute` |
| `mcp apply` missing servers | Soft — run `squad-oc mcp apply`; see section 8 |

---

## Out of scope

Copilot CLI, an Ink/`squad` shell, Aspire, and a `squad_state` MCP server.

`run` / `watch --execute` use **`github.com/sst/opencode-sdk-go`** against `opencode serve` (not the TUI).
