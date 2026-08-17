# Use cases — original Squad → squad-oc

What the [90-minute workshop](workshop/README.md) covers, mapped to **supported today**, a **later phase**, or **won’t port**.

This is not a second issue tracker. Product work stays on GitHub Issues. This catalog is the adopt map.

Host is **OpenCode**. You do **not** need Copilot, Node, or the original Squad CLI.

---

## Workshop sections

| Workshop | What you do | squad-oc today | Status |
|----------|-------------|----------------|--------|
| [1. Tools](workshop/README.md#1-tools) | OpenCode, `squad-oc` on `PATH`, git, `/connect` | `version`, `help` | **supported today** |
| [2. Solo repo](workshop/README.md#2-solo-repo) | Scaffold + first **yes** | `init --preset default`, `doctor`, `status` | **supported today** |
| [3. Talk to the team](workshop/README.md#3-talk-to-the-team) | `Team, add a usage endpoint…` and `@backend` / `@frontend` / `@tester` | OpenCode agents from `init`; `status` | **supported today** |
| [4. Memory](workshop/README.md#4-memory) | “Always require auth on usage” → `.squad/decisions.md` → new session | files under `.squad/` | **supported today** |
| [5. Org shape](workshop/README.md#5-org-shape) | Platform `init` + decisions; pack `.opencode/skills`; service `link` + `upstream` | `init`, `link`, `upstream add`, `upstream sync`, `doctor` | **supported today** |
| [6. Ralph](workshop/README.md#6-ralph) | One labeled issue | `watch --execute --once --label`, `watch --health` | **supported today** |
| [7. What's next](workshop/README.md#7-whats-next) | Read stubs only | — | **later phase** |

Section 7 is documentation only. `squad-oc help` has no `mcp`, `marketplace`, or `plugin` command until those phases land.

---

## Later phases

Do not treat these as shipped. Spec: [2026-08-14-org-dx-roadmap.md](superpowers/specs/2026-08-14-org-dx-roadmap.md).

| Phase | Original ease | Planned squad-oc | Workshop add-on |
|-------|---------------|------------------|-----------------|
| P1 | Drop-in org MCP (`mcp-config.json`) | `mcp init` / `mcp apply` / `mcp list` | Section 8 — Org MCP |
| P2 | Marketplace browse + install | `marketplace add` / `browse` / `install` | Section 9 — Skills marketplace |
| P3 | Themed cast (Office first) | `cast --theme office` (ids stay `@lead`) | Optional 3b — Hire the Office |
| P4 | Named plugins (`name@marketplace`) | `plugin install reflect@acme` | Update section 9 |
| P5 | Coordinator spawn + independent review | templates + handoff **Review** block | Tester rejects; author ≠ fix owner |
| P6 | Ceremonies as files | `.squad/ceremonies.md` at `init` | Design review / retro in chat |

Today’s stand-ins:

| Need | Use now | Later |
|------|---------|-------|
| Extra skills / agents | `squad-oc pack <path\|git-url>` or `upstream add` + `sync` | P2 browse, P4 `name@source` |
| Shared team + decisions | `squad-oc link <team-dir>` | still `link` (MCP file will follow the link in P1) |
| Memory | `.squad/decisions.md` | still files — not `squad_state` |

---

## Won’t port

Matches README **Non-goals** (plus the memory MCP the roadmap refuses):

| Original / Copilot-host piece | Why not |
|------------------------------|---------|
| GitHub Copilot CLI / Copilot SDK | Different host. OpenCode TUI + `opencode serve`. |
| Interactive Ink / `squad` shell | Use the OpenCode TUI. |
| Aspire / .NET dashboard | `squad-oc traces` is local OTLP JSON, not Aspire. |
| `squad_state` memory MCP | OpenCode + `.squad/` files remain memory. |

Also not product features (workshop may mention them as **example** org MCP servers after P1): WorkIQ, Outlook COM, Teams Adaptive Cards.

---

## DX table

Each original-Squad ease row is a **squad-oc command** you already ran, a **later** command, or **won’t port**. Later rows stay later — they are not won’t-port.

| Original ease | Mapping |
|---------------|---------|
| `npm i -g @bradygaster/squad-cli` | `go build -o squad-oc ./cmd/squad-oc` (put it on `PATH`) |
| `squad init` / `npx squad init` | `squad-oc init --preset default` |
| `squad init` with a blurb | `squad-oc init --preset default --description "…"` |
| Personal / global team | `squad-oc init --global` |
| Open Copilot → agent Squad | `opencode` → Tab → **squad** |
| `squad status` | `squad-oc status` |
| Confirm the cast | chat **yes** (or `/squad-cast`) |
| Add / drop a member | `squad-oc cast --add` / `cast --remove` / `recast` |
| Health check | `squad-oc doctor` (alias `heartbeat`) |
| Refresh host templates | `squad-oc upgrade` |
| Shared org team | `squad-oc link` / `link --off` |
| Pull a skills pack once | `squad-oc pack <path\|git-url>` |
| Remember + re-pull a pack | `squad-oc upstream add` / `sync` / `list` / `remove` |
| `squad watch` / Ralph | `squad-oc watch` (aliases `triage`, `loop`) |
| One Ralph pass | `squad-oc watch --once` |
| Ralph executes work | `squad-oc watch --execute --once` |
| Label filter | `squad-oc watch --label <name>` |
| Ralph snapshot | `squad-oc watch --health` |
| Overnight quiet window | `watch --overnight-start` / `--overnight-end` |
| Prompt without the TUI | `squad-oc run -p "…"` |
| Snapshot team files | `export` / `import` |
| Move team out of the worktree | `externalize` / `internalize` |
| Context / PII hygiene | `nap` / `scrub-emails` |
| Local spans | `squad-oc traces` |
| Drop-in `mcp-config.json` | later — `mcp apply` (P1) |
| Marketplace browse + install | later — `marketplace browse` / `install` (P2) |
| Office / themed names | later — `cast --theme office` (P3) |
| `install reflect@acme` | later — `plugin install` (P4) |
| Reviewer lockout (author cannot fix) | later — protocol + files (P5) |
| Design review / retro ceremonies | later — `ceremonies.md` (P6) |
| GitHub Copilot CLI / Copilot SDK | **won’t port** |
| Ink / `squad` interactive shell | **won’t port** |
| Aspire dashboard | **won’t port** |
| `squad_state` MCP | **won’t port** |

---

## Tamir workshop → here

[Tamir's workshop](https://github.com/tamirdresher/squad-skills/tree/main/workshop) is Copilot-hosted. Same intent, different commands.

| Tamir section | Here |
|---------------|------|
| 1. Prerequisites (Copilot, Node, `gh`) | Workshop §1 — OpenCode, `squad-oc`, git, `/connect` |
| 2. `npm i -g @bradygaster/squad-cli` + `squad init` | Workshop §2 — `squad-oc init --preset default` |
| 3. First conversation / hire the team | Workshop §2 — Tab → **squad** → **yes** |
| 4. Playground (talk, parallel, decisions) | Workshop §3–4 |
| 5. GitHub Issues routing | Workshop §6 — `watch --label` (no Copilot label bot) |
| 6. Project board tracking | **later** / not a Phase 0 command |
| 7. Ralph | Workshop §6 — `watch --execute --once` |
| 8. Skills marketplace | **later** P2 / P4 — today `pack` / `upstream` |
| 9. MCP (Teams, Outlook, …) | **later** P1 — not `mcp apply` today |
| 10. `@copilot` coding agent | **won’t port** |
| 11. Ceremonies, humans, cross-machine | ceremonies **later** P6; `link` is today’s cross-repo team |

---

## Verify

- Every command in workshop sections 1–6 exists in `squad-oc help`: `init`, `doctor`, `status`, `link`, `upstream`, `watch`.
- A teammate who never used original Squad can finish 1–6 without Copilot.
- Won’t-port rows match README non-goals plus `squad_state`.
