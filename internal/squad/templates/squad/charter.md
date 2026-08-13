# Team charter

Project: {{PROJECT_DESCRIPTION}}

## Principles

1. **Human-led** — People set priorities, approve merges, and own production risk.
2. **Coordinator routes** — The `squad` agent coordinates; specialists implement within scope.
3. **Persistent state** — Team knowledge lives in `.squad/` so work survives sessions.
4. **Inspectable** — Decisions and handoffs are written to files, not only chat.
5. **Small cast** — Default team: Lead, Frontend, Backend, Tester.

## Handoffs

When work moves between roles, write a short note under `.squad/comms/` or the receiving agent's `knowledge.md` with:

- Goal
- Done so far
- Files touched
- Open questions for the human

## Out of bounds for agents

- Force-pushing or rewriting shared history without explicit human request
- Merging PRs without human approval
- Committing secrets or reading `.env` secrets into chat
