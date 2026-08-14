---
name: squad-team
description: Coordinate the Squad team on OpenCode — casting, routing, and .squad state
---

# Squad team operations

Use this skill whenever you are the **squad** coordinator or doing multi-role work.

## State files

| Path | Purpose |
|------|---------|
| `.squad/team.md` | Who is on the team |
| `.squad/charter.md` | Team norms |
| `.squad/decisions.md` | Durable decisions |
| `.squad/agents/<id>/charter.md` | Role mission |
| `.squad/agents/<id>/knowledge.md` | Role learnings |
| `.squad/comms/` | Handoff notes |

## Casting flow

1. Propose Lead, Frontend, Backend, Tester (or current `.squad/team.md`).
2. Wait for human **yes** or edits.
3. Update `.squad/team.md` if the cast changes.
4. After file edits, run `squad-oc recast` so `.opencode/agents/` matches the team.

## Routing

- Multi-role feature → plan with Lead, implement with Frontend/Backend, verify with Tester.
- Invoke specialists with `@lead`, `@frontend`, `@backend`, `@tester` or the task tool.
- Keep the human as approver for merges and product calls.

## Status

When asked for status, summarize `.squad/team.md` and the newest decisions.
