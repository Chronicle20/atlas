# Report: remove the dead `-ida-port` / `select_instance` path from packet-audit

## Status: DONE

## What I implemented

Followed the brief exactly, working in place on `docs/ida-mcp-legacy-sweep` (main
repo checkout, not a worktree), on top of `da2f0048c`.

### `tools/packet-audit/internal/idasrc/mcphttp.go`
- Deleted `InstancePort` and `instanceSelected` struct fields.
- Deleted `NewMCPHTTPClientWithInstance`; rewrote `NewMCPHTTPClient` to
  construct the client directly (preserving the nil-`*http.Client` defaulting
  behavior).
- Deleted `SelectInstance` and `selectInstanceLocked`.
- Deleted the lazy select block in `ensureInit`.
- Rewrote the stale doc comments on the `Database` field, `NewMCPHTTPClient`,
  `NewMCPHTTPClientWithDatabase`, `callStructured`, and `callStructuredLocked`
  that referenced select_instance/port as a live or successor mechanism.
  `Database` is now described as the sole targeting mechanism.

### `tools/packet-audit/cmd/root.go`
- Deleted all seven `var idaPort int` + `fs.IntVar(&idaPort, "ida-port", ...)`
  pairs (one per subcommand: export, validate, infer, diff-shape,
  resolve-dispatch, decompose, triage).
- `newIDAClient` dropped its `port int` parameter and the `case port != 0`
  arm; it's now a two-branch (database set / not set) function. Rewrote its
  doc comment — the "two live servers" framing is gone.
- Updated all seven call sites to the new 3-arg signature.
- `-ida-database` help strings across all seven flag registrations dropped
  the "the session-based successor to -ida-port" clause, keeping "IDA-MCP
  session id (database) from idb_list to target directly; preferred when
  many IDBs are open on one server".

### `tools/packet-audit/cmd/discover_ops.go`
- Deleted the `IDAPort` opts field, its flag registration, and the argument
  at the `newIDAClient` call site. Same help-string cleanup.

### `tools/packet-audit/cmd/verify_serverbound.go`
- Deleted the `IDAPort` opts field and its flag registration. Per the
  brief's judgment call: since this file's inline client-selection switch
  (`case opts.IDADatabase != "": ... case opts.IDAPort != 0: ... default:
  ...`) became functionally identical to `newIDAClient` once the
  `IDAPort` branch was removed, I replaced the inline switch with a call to
  `newIDAClient(opts.IDAURL, 60*time.Second, opts.IDADatabase)` — same
  behavior, less duplication. This also dropped the now-unused `net/http`
  import (build confirms `idasrc` import is still used for the
  `idasrc.MCPClient` return type in `verifyServerboundRun`'s signature).

### `tools/packet-audit/internal/idasrc/mcphttp_test.go`
- Deleted `TestMCPHTTPSelectInstance` and
  `TestMCPHTTPInstancePortSelectedAfterHandshake` in full (not just the
  `select_instance` element of the `want` slice in the latter).
- Fixed the stale comment on `TestMCPHTTPDatabaseInjectedInArgs` that
  referenced "the successor to select_instance/port".
- Left all three `Database`-injection subtests
  (`database set is injected`, `database empty omits key`,
  `preexisting database key is not overwritten`) untouched — this is the
  surviving mechanism and its coverage was not weakened.

## What I tested

```
cd tools/packet-audit && GOTMPDIR="$HOME/.cache/go-tmp" TMPDIR="$HOME/.cache/go-tmp" go build ./... && \
  GOTMPDIR="$HOME/.cache/go-tmp" TMPDIR="$HOME/.cache/go-tmp" go vet ./... && \
  GOTMPDIR="$HOME/.cache/go-tmp" TMPDIR="$HOME/.cache/go-tmp" go test ./...
```

Result: build and vet clean, all 13 packages pass, output pristine:

```
?   	github.com/Chronicle20/atlas/tools/packet-audit	[no test files]
ok  	github.com/Chronicle20/atlas/tools/packet-audit/cmd	2.478s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/atlaspacket	1.255s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/csv	0.006s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/diff	0.177s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/discover	0.007s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/evidence	0.005s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc	0.029s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/marker	0.004s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix	0.012s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry	0.040s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/report	0.007s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/seedcsv	0.004s
ok  	github.com/Chronicle20/atlas/tools/packet-audit/internal/template	0.006s
```

Also confirmed the flag/mechanism is genuinely gone from the repo root:

```
grep -rn "ida-port\|InstancePort\|SelectInstance\|select_instance" tools/packet-audit | grep '\.go:'
```

returned nothing (exit code 1 / no matches).

## Files changed

- `tools/packet-audit/internal/idasrc/mcphttp.go`
- `tools/packet-audit/internal/idasrc/mcphttp_test.go`
- `tools/packet-audit/cmd/root.go`
- `tools/packet-audit/cmd/discover_ops.go`
- `tools/packet-audit/cmd/verify_serverbound.go`

Left untouched (explicitly out of scope per the brief): the working tree's
unrelated `docs/research/missing-features/` changes, and I did not stage or
commit `docs/tasks/ida-port-removal-brief.md`, `.envrc`, or the `.idea/`
files that were already untracked/modified in the working tree before I
started.

## Self-review findings

- Verified every anchor in the brief before editing; no line-number drift
  changed the intended edit targets (deletions in `mcphttp.go` were applied
  top-to-bottom in file order, which is safe against shift).
- Confirmed `NewMCPHTTPClient`'s nil-`*http.Client` defaulting behavior is
  preserved (previously delegated to `NewMCPHTTPClientWithInstance`, now
  inlined identically).
- Confirmed no other production or test file outside the five listed
  referenced `InstancePort`, `SelectInstance`, `selectInstanceLocked`, or the
  `-ida-port` flag (repo-wide grep restricted to `tools/packet-audit`, as the
  brief specifies).
- `verify_serverbound.go`'s consolidation onto `newIDAClient` is a
  same-package call (both in `package cmd`) and does not change externally
  observable behavior — confirmed via the passing `cmd` package test suite,
  which includes coverage for `verify_serverbound`.

## Issues or concerns

None. No behavior change for `-ida-database` or the default (no-selector)
path; `-ida-port` is now an unknown flag everywhere it used to be
"deprecated but accepted."

## Commit

`8b7874774` — `refactor(packet-audit): remove the dead -ida-port /
select_instance path`
