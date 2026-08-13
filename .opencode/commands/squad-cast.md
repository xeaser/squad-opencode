---
description: Propose or reconfirm the Squad team cast
agent: squad
---

Set up or reconfirm the Squad team for this project.

1. Read `.squad/team.md` and `.squad/charter.md` if they exist.
2. If the human provided a project description in $ARGUMENTS, use it; otherwise ask what they are building.
3. Propose the default cast (Lead, Frontend, Backend, Tester) with one-line roles, or propose adjustments for the project.
4. Ask the human to reply **yes** to confirm, or specify changes.
5. After confirmation, update `.squad/team.md` if needed and remind them they can use `@frontend`, `@backend`, `@tester`, `@lead`.
