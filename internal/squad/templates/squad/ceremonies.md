# Ceremonies

Structured team meetings. The human or coordinator triggers them in chat — there is no scheduler.

## Design Review

Run **before** a multi-agent feature (2+ specialists, shared interfaces).

1. Ask short questions (below). Do not implement during the review.
2. Write `.squad/comms/YYYY-MM-DD-<slug>-design-review.md`.
3. Then spawn specialists.

**Questions:**

- What are we building, and what is out of scope?
- Which agents own which layer?
- What contracts (API, types, files) must they agree on first?
- What could break if they assume different shapes?
- Human: proceed, change the plan, or skip this review?

**Write-up:**

```markdown
## Design Review: <title>

- **Goal:**
- **Owners:**
- **Contracts:**
- **Risks:**
- **Decision:** proceed | change | skip
```

---

## Retro

Offer a retro **after** a failed `watch` / `run`, or a rejected PR. Do not start one unless the human wants it.

1. Ask: what happened, why, what to change?
2. Write `.squad/comms/YYYY-MM-DD-<slug>-retro.md`.
3. If the change is a durable rule, append it to `.squad/decisions.md`.

**Questions:**

- What happened (facts only)?
- Root cause?
- What should change next time?
- Action items, and who owns them?

**Write-up:**

```markdown
## Retro: <title>

- **What happened:**
- **Root cause:**
- **Change:**
- **Actions:**
```
