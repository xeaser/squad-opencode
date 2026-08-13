# Get started — Squad for OpenCode (Go)

Zero to first team. You do **not** need GitHub Copilot, Node, or the original Squad CLI.

**Time:** about 10–15 minutes.

---

## Prerequisites

| Need | Why |
|------|-----|
| Terminal | Install and run tools |
| **Go 1.22+** (to build `squad-oc`) | Or a prebuilt binary when published |
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

## 4. Build and install `squad-oc`

From this repository:

```bash
cd /path/to/squad-opencode
go build -o squad-oc ./cmd/squad-oc
```

Put the binary on your `PATH`, or call it with a full path.

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

Validate:

```bash
squad-oc doctor
squad-oc status
```

`doctor` uses the **OpenCode Go SDK** for an *optional* server probe (`http://127.0.0.1:4096`). Missing server is soft; missing OpenCode binary or scaffold is hard.

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
| Server probe FAIL only | Soft check — run `opencode` if you want a live server |
| Build fails | `go version` ≥ 1.22; `go mod tidy` |

---

## Out of scope (MVP)

Watch/Ralph execute, upgrade pipeline, export/import, marketplace, Copilot.

Orchestration will use **`github.com/sst/opencode-sdk-go`** (not the “OpenCode Go” model subscription).
