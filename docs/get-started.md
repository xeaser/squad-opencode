# Get started — Squad for OpenCode (Go)

Zero to first team. You do **not** need GitHub Copilot, Node, or the original Squad CLI.

**Time:** about 10–15 minutes.

---

## Prerequisites

| Need | Why |
|------|-----|
| Terminal | Install and run tools |
| Git | Project + recommended for real work |
| An LLM provider | OpenCode talks to a model |

---

## 1. Install OpenCode

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

Check:

```bash
opencode --version
```

---

## 2. Connect a model provider

```bash
opencode
```

Inside OpenCode:

1. Run `/connect`
2. Choose a provider
3. Paste your API key
4. Quit when done

---

## 3. Create a project

```bash
mkdir my-recipe-app
cd my-recipe-app
git init
echo "# Recipe app" > README.md
git add README.md
git commit -m "chore: initial commit"
```

---

## 4. Install `squad-oc`

You do **not** need Go. Download the archive for your OS from [GitHub Releases](https://github.com/xeaser/squad-opencode/releases/latest) and put `squad-oc` (`squad-oc.exe` on Windows) on your `PATH`. Assets are `squad-oc_<version>_<os>_<arch>.zip` (Windows) or `.tar.gz` (macOS/Linux).

**Scoop**

```powershell
scoop bucket add squad https://github.com/xeaser/squad-opencode
scoop install squad-oc
```

**Homebrew**

```bash
brew tap xeaser/squad-opencode https://github.com/xeaser/squad-opencode
brew install squad-oc
```

**winget** (from a clone of this repo)

```powershell
winget install --manifest packaging/winget
```

Update later with `squad-oc upgrade --self`.

Check:

```bash
squad-oc version
squad-oc help
```

---

## 5. Scaffold Squad in the project

```bash
cd my-recipe-app
squad-oc init --preset default
```

With description:

```bash
squad-oc init --preset default --description "Recipe sharing app with React and Node"
```

**Creates:**

- `.squad/` — team state  
- `.opencode/agents/` — `squad` + role subagents  
- `.opencode/skills/` — team + handoff  
- `.opencode/commands/` — `/squad-status`, `/squad-cast`  
- `.opencode/.gitignore` — ignores OpenCode’s local `node_modules` / lockfiles  
- `opencode.json` — only if missing  

**Commit** `.squad/` (except `comms/` scratch) and `.opencode/agents|skills|commands`.  
**Do not commit** `.opencode/node_modules/` or the `package.json` OpenCode generates there.  

Re-running `init` is safe (no overwrite once `.squad/config.json` exists).

To refresh OpenCode agents/skills after a `squad-oc` update (team files stay put):

```bash
squad-oc upgrade --dry-run
squad-oc upgrade
```

Validate:

```bash
squad-oc doctor
squad-oc status
```

`doctor` uses the **OpenCode Go SDK** for an *optional* server probe (`http://127.0.0.1:4096`). Missing server is soft; missing OpenCode binary or scaffold is hard.

`squad-oc run` and `watch --execute` use the **HTTP API**, not the TUI. If nothing is listening on `http://127.0.0.1:4096`, they start `opencode serve` there. They will **not** start a server if you pass `--url` or set `OPENCODE_BASE_URL` to anything else.

```bash
squad-oc run -p "Summarize .squad/team.md"
# or, if you already run serve on another port:
squad-oc run --url http://127.0.0.1:5000 -p "Summarize .squad/team.md"
```

Plain `opencode` opens the interactive UI and does **not** listen on 4096.

### Live check

From this repo root, ground `doctor` / `run` against a dummy project (never this git tree):

```powershell
pwsh -File scripts/live-e2e.ps1
```

The script builds `squad-oc.exe`, inits `D:\xAI\squad-oc-dummy` if needed, starts `opencode serve --hostname 127.0.0.1 --port 4096` there (or attaches if that localhost port is already up), then runs `doctor` and `run -p "Reply with exactly PONG and nothing else."`. It fails if `:4096` is bound on a non-localhost address. The TUI is optional; if no window is open, skip it.

`TestLiveEnsureAPI` stays behind `SQUAD_OC_LIVE=1`. CI stays unit-only (`go test ./...` without that env).

---

## Also useful

- [Workshop](workshop/README.md) — adopt path for squad-oc + OpenCode only (includes org MCP)
- [Use-case catalog](use-cases.md) — workshop mapped to supported / later / won’t port
- `squad-oc init --global` (and `upgrade --global`) for a personal team outside the repo
- `squad-oc cast --add Designer --role Design` then OpenCode sees `@designer` after recast
- `squad-oc cast --remove Tester` drops a member and recasts agents
- `squad-oc import snapshot.json --with-host` restores `.squad/` plus host agents
- `squad-oc link ~/teams/platform` (or a `git@` / `https` URL) to share one `.squad/` (`link --sync` / `link --off`)
- `squad-oc watch --health` prints the last Ralph snapshot
- `squad-oc watch --execute --overnight-start 18:00 --overnight-end 08:00`
- `squad-oc traces` lists local run/watch spans (`--export` writes OTLP JSON)
- `squad-oc run -p "…"` starts `opencode serve` on `127.0.0.1:4096` if nothing is there; a custom `--url` never auto-starts
- `squad-oc pack <path|git-url>` or `squad-oc upstream add <name> <path|git-url>` to pull extra agents/skills (see README)
- `squad-oc mcp init` / `apply` / `list` — org `.squad/mcp-config.json` into `opencode.json` (workshop §8)

`export`/`import`, `nap`, `scrub-emails`, and the rest are in `squad-oc help` and the README.

---

## 6. Open OpenCode and select Squad

```bash
cd my-recipe-app
opencode
```

Optional:

```text
/init
```

Switch to the **squad** agent (Tab).

---

## 7. Confirm the team

```text
I'm starting a recipe sharing app with React and Node.
Set up the team and confirm roles.
```

Or:

```text
/squad-cast
```

Reply **yes** to confirm Lead / Frontend / Backend / Tester.

---

## 8. Do work

```text
Add a recipe list page and a simple API.
Delegate UI to frontend and API to backend.
```

```text
@backend Sketch REST endpoints for recipes.
@frontend Scaffold a recipe list page.
```

```text
/squad-status
```

```bash
squad-oc status
```

---

## 9. Success checklist

- [ ] `opencode` runs and a model answers  
- [ ] `.squad/team.md` lists members  
- [ ] **squad** agent is available  
- [ ] Cast confirmed with **yes**  
- [ ] `@frontend` / `@backend` work  
- [ ] Required `squad-oc doctor` checks pass  

---

## Troubleshooting

| Problem | Try |
|---------|-----|
| `opencode: command not found` | Install OpenCode; fix PATH |
| No **squad** agent | `squad-oc init --preset default` in project root |
| `Not initialized` | `squad-oc init --preset default` |
| Server probe FAIL only | Soft check — run `opencode serve` for the HTTP API on :4096 |
| `squad-oc: command not found` | Download a release binary; fix PATH |

---

## Out of scope

Copilot, an Ink/`squad` shell, and Aspire.

`run` / `watch --execute` use **`github.com/sst/opencode-sdk-go`** against `opencode serve` (not the “OpenCode Go” model subscription, and not the TUI).
