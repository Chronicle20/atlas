# Brief: remove the dead `-ida-port` / `select_instance` path from packet-audit

## Diagnosis

The `ida-pro` MCP server moved to session-based IDB selection: `idb_list` returns a
session id, which is passed as a `database` argument on every `tools/call`. The older
per-port server — selected via a `select_instance` MCP call and surfaced in the
packet-audit CLI as `-ida-port` — was removed server-side (dead since task-138), and
the user has confirmed **no per-port server is live anywhere in their setup**.

PR #1425 (branch `docs/ida-mcp-legacy-sweep`) already de-documented the flag and
aligned its help text to "deprecated, use -ida-database". This follow-up deletes the
mechanism outright, so no dead branch remains to mislead a future agent. The
`newIDAClient` doc comment currently claims "there are two live IDA-MCP servers" —
that premise is now false and the comment must go with the code.

## Scope — delete, do not deprecate

Work on the existing branch `docs/ida-mcp-legacy-sweep`, in the main repo checkout
(repo root = your cwd). Do NOT create a worktree or a new branch. Commit on top of
`da2f0048c`.

### `tools/packet-audit/internal/idasrc/mcphttp.go`
- Delete the `InstancePort` and `instanceSelected` struct fields (:30-34).
- Delete `NewMCPHTTPClientWithInstance` (:51-60). Rewrite `NewMCPHTTPClient` (:44-49),
  which currently delegates to it with port 0, to construct the client directly —
  preserve its existing nil-`*http.Client` defaulting behavior.
- Delete `SelectInstance` (:339-347) and `selectInstanceLocked` (:349-355).
- Delete the lazy select block in `ensureInit` (:273-278).
- Update the stale prose in the comments at :39, :64, :301, :312 that reference
  select_instance/port as a live or successor-to mechanism. `Database` is now the
  only targeting mechanism, not "the successor" to anything present in the file.

### `tools/packet-audit/cmd/root.go`
- Delete all seven `var idaPort int` + `fs.IntVar(&idaPort, "ida-port", ...)` pairs
  (:133-134, :225-226, :274-275, :313-314, :355-356, :396-397, :439-440).
- Change `newIDAClient` (:197) to drop the `port int` parameter; delete the
  `case port != 0` arm. Rewrite the doc comment at :189-196 — the "two live servers"
  framing and the `-ida-port` explanation are both obsolete.
- Update all seven call sites to the new signature (:185, :254, :294, :333, :375,
  :419, :462).
- The `-ida-database` help strings still read "the session-based successor to
  -ida-port" — drop that clause now that the flag no longer exists. Keep the rest
  ("IDA-MCP session id (database) from idb_list to target directly; preferred when
  many IDBs are open on one server").

### `tools/packet-audit/cmd/discover_ops.go`
- Delete the `IDAPort` opts field (:27), its flag registration (:45), and the
  argument at the `newIDAClient` call (:77). Same "successor to -ida-port" clause
  cleanup on the `-ida-database` help string.

### `tools/packet-audit/cmd/verify_serverbound.go`
- Delete the `IDAPort` opts field (:26), its flag registration (:42), and the
  `case opts.IDAPort != 0` arm of the client switch (:66-67). Same help-string
  cleanup. Note this file builds its client inline rather than via `newIDAClient`;
  consider whether it should now just call `newIDAClient` — use your judgment, but
  do not change its behavior.

### `tools/packet-audit/internal/idasrc/mcphttp_test.go`
- Delete `TestMCPHTTPSelectInstance` (:493-521) and
  `TestMCPHTTPInstancePortSelectedAfterHandshake` (:523-~555) — they test deleted
  behavior. In the latter's `want` sequence, `select_instance` was an expected
  element; the whole test goes, do not merely edit the slice.
- Fix the stale comment at :419 referencing "the successor to select_instance/port".
- Do NOT weaken or delete any test that covers `Database` injection — that is the
  surviving mechanism and its coverage must not regress.

## Constraints

- No behavior change for `-ida-database` or the default (no-selector) path. The only
  user-visible change is that `-ida-port` is now an unknown flag.
- Do not touch any file outside `tools/packet-audit/`. In particular do NOT
  regenerate `docs/packets/audits/STATUS.md` / `status.json` — the controller
  handles the matrix regen (the `toolSha` will shift again because tool source
  changed).
- Leave the working tree's unrelated `docs/research/missing-features/` changes
  untouched and unstaged.

## Verification (module-local only)

```
cd tools/packet-audit && go build ./... && go vet ./... && go test ./...
```

Then confirm the flag is genuinely gone — from the repo root:
```
grep -rn "ida-port\|InstancePort\|SelectInstance\|select_instance" tools/packet-audit | grep '\.go:'
```
should return nothing. Do NOT run `tools/verify.sh` — the controller runs the
repo-wide gate.

Commit on the branch with a `refactor(packet-audit):` subject explaining that the
per-port server is confirmed gone, so the flag is removed rather than deprecated.
