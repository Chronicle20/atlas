# Phase 1.5 — Real-decompile hardening (inserted between Phase 1 GATE and Phase 2)

**Why this phase exists.** Phase 1 built the exporter against synthetic Hex-Rays
fixtures. A live smoke-test against the v83 IDB (`MapleStory_dump.exe`, md5
`80ff438ced539b831f0d2ed95099275d`) at `http://192.168.20.3:13337/mcp` proved the
synthetic fixtures were far cleaner than real decompiler output. Running the live
re-export as-is would silently drop real struct-helper reads and abort on the first
missing/undecompilable function — producing a *misleading* export (false
`atlas: extra` blockers on exactly the packets we care about). The user chose to
harden the parser/client before any live run.

Canonical evidence: v83 `CWvsContext::OnFriendResult` @`0xa3f2e8`, case 9 (#Invite):
```c
v2 = Index;
switch ( CInPacket::Decode1(Index) ) { ...
  case 9:
    v23 = CInPacket::Decode4(v2);        // friendId
    v24 = CInPacket::DecodeStr(v2, v40); // name
    sub_A40028(..., v2);                  // GW_Friend reader (unnamed sub)
```
`sub_A40028` @`0xa40028` → `CInPacket::Decode1(a2)` + `sub_4E4427(v4, a2)`; and
`sub_4E4427` @`0x4e4427` **fails to decompile** (`-32000 Decompilation failed`).
Committed real fixtures: `internal/idasrc/testdata/real_onfriendresult_v83.c`,
`real_sub_a40028_v83.c`.

## Confirmed live-server contract (drives Task A)

- `initialize` issues an **`Mcp-Session-Id`** response header; it must be sent on
  every subsequent request. `notifications/initialized` returns `-32601 Method not
  found` (server doesn't implement it); `tools/call` works with just the session id.
- `get_function_by_name` returns a JSON object in `content[].text`:
  `{"address":"0xa3f2e8","name":"?OnFriendResult@...","size":"0x3d5"}` — NOT a bare
  address. Must parse + extract `address`.
- not-found → JSON-RPC `error -32000 "No function found with name X"`.
  decompile-fail → JSON-RPC `error -32000 "Decompilation failed at ..."`.
- `decompile_function` arg is `address`; `get_callees` arg is `function_address`.
- decompile text is line-annotated: `/* line: N, address: 0x.. */ <code>` with
  register `// eax` trailing comments.

## Tasks

### Task A — MCP-HTTP client real-payload fixes (`mcphttp.go`)
TDD against fixture strings mirroring the captured shapes above.
1. Capture `Mcp-Session-Id` from the `initialize` response header; send it on
   `notifications/initialized` (best-effort; ignore its error) and every `tools/call`.
2. `GetFunctionByName`: parse `content[].text` as JSON `{address,...}` → return
   `address`. If the rpc error message matches "No function found" (or content is
   empty/"not found"), return `("", false, nil)` — NOT a fatal error.
3. Distinguish a **not-found / decompilation-failed** rpc error (`-32000` with those
   messages) from a genuine transport/protocol failure: the former is a soft signal
   the caller maps to ok=false / a decompile error value; the latter stays loud.
   Add an exported sentinel/typed error (e.g. `ErrToolSoftFail` or a predicate) so
   Harvest (Task B) can branch on it.
4. `get_callees`: use arg key `function_address`.
5. Tolerate the `/* line: N */` decompile prefixes (no client change needed beyond
   returning the text verbatim; the parser handles prefixes in Task D).

### Task B — Harvest robustness (`harvest.go`)
A `DecompileFunction` (or `ParseDecompile`) failure on a **discovered/roster**
function must produce an `Unresolved` export entry and CONTINUE the BFS, not abort.
- GetFunctionByName not-found already → Unresolved entry (keep).
- DecompileFunction soft-fail (decompilation failed) → `{Unresolved:true,
  Calls:[{Op:"Unresolved", Comment:"decompilation failed; hand-trace"}]}`, continue.
- A genuine transport error (not a per-function soft-fail) still aborts loudly.
- Regression test with a fake client that errors on one helper's decompile.

### Task C — Parser: packet-pointer alias SET (`parse.go`)
Replace the single `pktVar` with a **set** of packet-pointer aliases:
- Seed: the first-arg identifier of every `CInPacket::Decode*` / `COutPacket::Encode*`
  call across the function.
- Closure: any `X = Y;` or `X = &Y;` assignment where `Y` is already an alias adds `X`.
- `Delegate`/`Unresolved` arg-matching uses set membership (so a helper passing the
  alias `v2` is recognized even though reads also use `Index`).
- Regression: the real OnFriendResult fixture's `sub_A40028(..., v2)` is NOT silently
  dropped (it is descended — see Task D).

### Task D — Parser: unnamed-sub descent + switch-expr + line prefixes (`parse.go`)
1. Emit `Delegate{Ref:name}` for a call passing a packet alias whose callee is a
   resolvable NAME: `Class::method`, a bare identifier, OR `sub_[0-9A-Fa-f]+` —
   denylist-filtered. Reserve `Unresolved` for a true indirect `(*..)(..)` call
   passing a packet alias (no name to resolve).
2. `switch ( <expr> )` where the discriminator is an expression (e.g.
   `CInPacket::Decode1(Index)`): still emit the inline read; derive a case guard
   (guard text is cosmetic — the audit does not compare guard text — so a stable
   synthesized label like `case == N` is acceptable). Case labels may be hex
   (`case 0xA:`) or decimal.
3. Tolerate `/* line: N, address: .. */` line prefixes and register `// reg` trailing
   comments (strip/ignore so they don't corrupt op/order; labels stay best-effort).
4. Must-not-regress: all existing synthetic-fixture tests stay green.

### Task E — Real end-to-end regression (`harvest_test.go`)
Harvest over the real fixtures with a fake client:
`real_onfriendresult_v83.c` (@0xa3f2e8) + `real_sub_a40028_v83.c` (@0xa40028) +
`sub_4E4427` decompile = soft-fail. Assert the faithful #Invite-path sequence:
`Decode4` (friendId), `DecodeStr` (name), then DESCEND `sub_A40028` →
`Decode1` + DESCEND `sub_4E4427` → `Unresolved` (decompile failed). I.e. the
GW_Friend read is no longer silently dropped — it is a descent chain ending in an
honest `Unresolved`, never a guess and never a phantom truncation.

### Phase 1.5 GATE
`go test -race ./...`, `go vet ./...`, `go build ./...` clean in `tools/packet-audit`;
the real OnFriendResult descent regression green. THEN proceed to Phase 2 (live
re-export), re-running the smoke-test (a small roster) before the full 1145-fn run.
