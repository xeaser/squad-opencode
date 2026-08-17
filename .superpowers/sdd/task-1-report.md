# Task 1 Report — Phase 1 org MCP mcp-config.json (squad-6o4)

**Status:** DONE

**Worktree:** `C:\Users\zephyr\.grok\worktrees\xai-squad-opencode\subagent-01a00e5d-f657-7bc3-9fa7-9dcfa25abd7b`  
**Commit:** `3e89945` — feat: translate org mcp-config.json into opencode.json

## Summary

New `internal/mcpconfig` package translates org/workshop `mcp-config.json` (Copilot `mcpServers` or OpenCode `mcp`) into the project `opencode.json` `mcp` key. CLI: `squad-oc mcp apply|list|init`. Doctor soft-check `MCP apply`. Pack/sync copies missing `mcp-config.json` only (not added to `skipSquad`); pack-root file lands in the resolved team dir.

Did not write workshop/use-case/get-started docs.

## TDD evidence

### RED (expected)

```
go test ./internal/mcpconfig
```

Failed to compile before `mcpconfig.go` existed:

```
undefined: Parse
undefined: Merge
undefined: Server
undefined: Apply
```

Why expected: tests were written first against the intended API (`Parse`, `Merge`, `Apply`, `InitExample`, `List`) with no implementation.

### GREEN

```
go test ./internal/mcpconfig -count=1
# ok  github.com/xeaser/squad-opencode/internal/mcpconfig  1.468s
```

After wiring CLI/doctor/share:

```
go test ./...
# all packages ok (cli, doctor, githubissues, mcpconfig, opencodeclient,
# selfupdate, share, squad, traces, updatecheck, watch)

go build -o squad-oc.exe ./cmd/squad-oc
# exit 0
```

## Required cases

| Case | Where |
|------|--------|
| Copilot `mcpServers` → OpenCode `mcp` | `TestCopilotShapeToOpenCode` + `testdata/org-mcp.json` / `expected-opencode.json` |
| Already-OpenCode pass-through | `TestOpenCodePassthrough` |
| Merge keeps `$schema` (and provider/model) | `TestMergePreservesSchema` |
| `${FOO}` → `{env:FOO}` | `TestEnvVarRewrite` |
| Hardcoded `sk-` / `ghp_` fail apply | `TestRejectHardcodedTokens` |
| Same-name mcp server — org wins | `TestOrgWinsSameName`, `TestApplyOrgWinsOverPackRoot` |
| Linked team: apply reads `ResolveDir` | `TestApplyReadsLinkedTeam`, `TestMCPApplyReadsLinkedTeamViaCLI` |
| Doctor soft-fail if config exists but apply not run | `TestMCPApplySoftFailWhenNotApplied` (`Hard: false`) |
| Pack/sync copies missing mcp-config only | `TestPackCopiesMissingMCPConfigOnly`, `TestPackSquadMCPConfigMissingOnly`, `TestPackRootMCPFollowsLink` |

## Changes

### `internal/mcpconfig/` (new)

- Parse Copilot (`command` string + `args` + `env`) or OpenCode (`type` + `command[]` + `environment` / `url` + `headers`)
- Local → `{type, command, enabled, environment}`; remote → `{type, url, headers, enabled}`
- `${VAR}` rewritten to `{env:VAR}`; `{env:VAR}` left alone
- `enabled: false` written through
- Merge into existing `opencode.json`; never drops `$schema` or unrelated keys
- Apply sources: pack-root `mcp-config.json` then resolved `.squad/mcp-config.json` (org wins)
- Token-like `sk-` / `ghp_` values fail apply with a clear error
- `InitExample` writes commented JSONC into the resolved team dir if missing
- Fixture: `internal/mcpconfig/testdata/org-mcp.json` + `expected-opencode.json`

### `internal/cli/cli.go`

- `squad-oc mcp apply|list|init`
- Unknown subcommand / missing args → exit 2
- One help line: `mcp apply | list | init`
- `init` does not force MCP (only `mcp init` writes the example)

### `internal/doctor/doctor.go`

- Soft check named **MCP apply** (`Hard: false`)
- If resolved org file exists, `opencode.json` must contain those server names
- Soft-fail detail points at `squad-oc mcp apply`

### `internal/share/share.go`

- `mcp-config.json` is **not** in `skipSquad` (nested pack `.squad/mcp-config.json` copies missing-only)
- Pack-root `mcp-config.json` copied into `squad.ResolveDir` only when absent

### `README.md`

- One Commands-table row for `mcp apply` / `list` / `init`

## Self-review

- All required test cases covered and passing
- CLI stays thin; translation lives in `mcpconfig`
- `init` does not write MCP; doctor is soft only
- Secrets are never listed; apply refuses `sk-` / `ghp_` literals
- File ownership respected (no workshop / use-cases / get-started / marketplace)

## Concerns

None.

## Suggested next

- Phase 0 sibling can stitch workshop Section 8
- Manual: after `mcp apply`, reload OpenCode and ask “List the MCP tools you have”

## Review follow-up

Addressed `task-1-review.md` (range after `3e89945`).

| Issue | Status | What changed |
|-------|--------|----------------|
| 1 Apply cwd pack-root | fixed | `Apply` / `List` / `LoadSources` read only `OrgPath`. Dropped `PackPath`. `TestApplyReadsOnlyOrgPath` replaces `TestApplyOrgWinsOverPackRoot`. Pack ingest stays in `copyPackRootMCP`. |
| 2 Token detector | fixed | After rewrite, strip `{env:VAR}` / `${VAR}` spans then substring-scan for `sk-` / `ghp_`. Tests: `ApiKey=sk-…`, `sk-…{env:FOO}`, `token=ghp_…`. |
| 3 Nested copy dest under link | wontfix | Pack-root is the documented ingest. Changing `copySquadSnippets` dest for one filename is a broader share-link change than this phase. |
| 4 Rewrite url/command | fixed | `rewriteEnv` on URL and each Command part. `TestEnvVarRewrite` covers both. |
| 5 Parse example + JSONC | fixed | `TestInitExampleParses` + `TestJSONCCommentsStrip` (trailing comma still errors). |

### Gates

```
go test ./internal/mcpconfig ./internal/cli ./internal/doctor ./internal/share
# ok (mcpconfig, cli, doctor, share)

go test ./...
# all packages ok

go build -o squad-oc.exe ./cmd/squad-oc
# exit 0
```
