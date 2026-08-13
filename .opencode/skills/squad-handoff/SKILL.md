---
name: squad-handoff
description: Write inspectable handoffs between Squad roles
---

# Squad handoffs

When work moves from one role to another (or back to the human), write a short handoff.

## Where

Prefer one of:

1. `.squad/comms/YYYY-MM-DD-<from>-to-<to>.md`
2. The receiving agent's `.squad/agents/<id>/knowledge.md` (append a Handoff section)

## Format

```markdown
## Handoff: <from> → <to>

- **Goal:**
- **Done:**
- **Files:**
- **Open questions:**
- **Suggested next step:**
```

Keep it short. The next agent should not need the full chat history.
