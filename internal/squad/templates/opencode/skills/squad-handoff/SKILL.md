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

## Review

When Tester or Lead reviews specialist work, add this block to the handoff:

```markdown
## Review

- **Verdict:** accept | reject
- **Author:** <agent id who wrote the change>
- **Fix owner:** <next agent id — must differ from Author>
- **Reasons:**
```

**Fix owner** must differ from **Author**. On reject, the coordinator assigns Fix owner (or asks the human). The Author must not apply the fix.

This is protocol + files. OpenCode has no hard file-write lockout.

Record a durable review outcome in `.squad/decisions.md` when it becomes a team rule.
