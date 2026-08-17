---
description: Run independent review of the last specialist change
agent: squad
---

Run independent review of the last specialist change.

1. Identify the last specialist change (recent files, last handoff under `.squad/comms/`, or $ARGUMENTS).
2. Spawn Tester or Lead as a **separate** reviewer — not the Author of the change.
3. The reviewer writes a handoff with a **Review** block: Verdict, Author, Fix owner (must differ from Author), Reasons.
4. If Verdict is reject, assign Fix owner (a different specialist or the human). Do not patch rejected specialist output yourself, and do not send the fix back to the Author.
5. If the outcome is a durable rule, append it to `.squad/decisions.md`.

This is protocol + files, not kernel enforcement.
