# Backend Audit — task-181-v79-template-audit (Go diff `main..HEAD`)

- **Worktree:** `.worktrees/task-181-v79-template-audit` (branch `task-181-v79-template-audit`, confirmed via `git branch --show-current`)
- **Guidelines Source:** backend-dev-guidelines skill (ai-guidance.md, file-responsibilities.md, anti-patterns.md, patterns-functional.md) + `docs/packets/IMPLEMENTING_A_PACKET.md` / `docs/packets/DISPATCHER_FAMILY.md` conventions for the packet-codec layer, since this diff touches `libs/atlas-packet` codec structs, not DDD service packages (no `model.go`/`processor.go`/`resource.go` in scope — the generic DOM-01..DOM-24 REST/DB checklist is N/A to this diff; DOM-21 and DOM-25-adjacent concerns and the general immutability/anti-pattern rules still apply).
- **Date:** 2026-07-20
- **Build:** PASS
  - `libs/atlas-packet`: `go build ./...` clean
  - `services/atlas-channel/atlas.com/channel`: `go build ./...` clean
  - `tools/packet-audit`: `go build ./...` clean
- **Tests:** PASS — `go test ./... -count=1` clean in `libs/atlas-packet` (68 packages, all `ok`) and `tools/packet-audit` (all `ok`). No test files in `services/atlas-channel/atlas.com/channel/socket/writer` (thin pass-through wrappers only).
- **Goroutine guard:** PASS — `grep -rnE '^\s*go (func|[A-Za-z_])'` over the changed non-test files in `libs/atlas-packet/field`, `libs/atlas-packet/monster/carnival`, `tools/packet-audit`, `services/atlas-channel/.../socket/writer` returns zero matches.
- **Overall:** PASS — zero FAIL findings against the applicable checklist items. One Minor/non-blocking observation recorded below.

## Scope

Diff `main..HEAD` (17 commits). Go files touched:
- `libs/atlas-packet/field/clientbound/*.go` (+tests) — codec corrections/re-modeling for `SnowballState`, `AriantArenaUserScore`, `ContiMove`, `Tournament`, `TournamentSetPrize`, `TournamentMatchTable`; v79 byte-pin tests added for ~18 other already-verified writers (no codec logic change).
- `libs/atlas-packet/monster/carnival/clientbound/monster_carnival_message.go`, `monster_carnival_summon.go` — comment-only v79 IDA-address additions (no field/logic change).
- `libs/atlas-packet/monster/carnival/clientbound/*_test.go` — new v79 byte-pin tests for Start/Died/Leave/Message/Summon/ObtainedCp/PartyCp/Result (test-only additions).
- `services/atlas-channel/atlas.com/channel/socket/writer/{ariant_arena_user_score,conti_move,snowball_state,tournament,tournament_match_table,tournament_set_prize}.go` — thin wrapper signatures widened to match the corrected codec constructors.
- `tools/packet-audit/internal/idasrc/export.go` (+test) — resolver now skips the `COutPacket` header op.

## Per-File Findings

### 1. `SnowballState` Decode — `Available() >= 6` heuristic (Minor, non-blocking)

**File:** `libs/atlas-packet/field/clientbound/snowball_state.go:117-124`

```go
// The initial snapshot appends three damage shorts; the client gates
// these on its own prior state, so recover `first` from their presence.
if r.Available() >= 6 {
    m.first = true
    m.damageSnowBall = r.ReadUint16()
    m.damageSnowMan0 = r.ReadUint16()
    m.damageSnowMan1 = r.ReadUint16()
}
```

`first` is not a wire field — the client gates the trailing 3 shorts on its own stored `m_nState == -1` check, so the encoder side is correct (it writes the tail only `if m.first`). The `Decode` side reconstructs `first` from "are there ≥6 bytes left in the reader," which is **only** correct if the `request.Reader` passed to `Decode` is scoped to exactly this packet's body and nothing follows it in the same buffer. Verified via `libs/atlas-socket/request/reader.go:133` (`Available()` returns remaining unread length of the reader's own buffer) and confirmed `clientbound.SnowballState.Decode` has **zero production call sites** — `grep -rn "clientbound.SnowballState"` across the repo returns nothing outside the codec file and its own test; it is only exercised by `test.RoundTrip`/`test.Encode` in `snowball_state_test.go`, each of which constructs a reader scoped to that single struct's encoded bytes. So today the heuristic is safe in every place it actually runs.

This is not a FAIL under any checklist item (SnowballState is a clientbound packet — the real server never decodes its own outbound writes; `Decode` exists only for round-trip test symmetry and packet-audit tooling), and the fallback logic is explicitly commented with the rationale. Flagging as a **latent fragility**: if `SnowballState.Decode` is ever called on a reader shared with subsequent packet data (e.g. wired into a batch/multi-packet decode path), the `Available() >= 6` check would silently misfire. No action required for this PR; worth a comment-only hardening (e.g. explicit "packet-scoped reader only" note) if this codec is ever exposed to a shared-buffer decode path.

### 2. `AriantArenaScoreEntry` slice getter returns internal state directly (Minor, non-blocking)

**File:** `libs/atlas-packet/field/clientbound/ariant_arena_user_score.go:29` — `func (m AriantArenaUserScore) Entries() []AriantArenaScoreEntry { return m.entries }`

`patterns-functional.md:10-13` states domain models have private fields with "public getters expose read-only state." `Entries()` returns the backing slice by reference rather than a defensive copy, so a caller can mutate `m.entries[i]` (or append/reslice, though append won't affect the original backing array beyond capacity) after construction, breaking immutability. Contrast with `TournamentMatchTable.Match()` in the same PR (`tournament_match_table.go:44`), which returns `[TournamentMatchTableBufferSize]byte` — a Go **array** value type — so callers get an implicit copy and cannot mutate the model. No explicit DOM/file-responsibilities checklist item mandates defensive-copy slice getters (this is a general "Immutability" principle, not a numbered rule), so this is **not scored as a FAIL**, but it is inconsistent with the array-copy discipline the same PR applies to `TournamentMatchTable` one file over. Non-blocking.

### 3. `export.go` `COutPacket` skip — verified correct (PASS)

**File:** `tools/packet-audit/internal/idasrc/export.go:249-255`

```go
if c.Op == "COutPacket" {
    continue
}
```

This sits inside a `for i, c := range raw.Calls` loop (`export.go:231`) that appends one `FieldCall` per surviving op, in order, to `out.Calls`. `continue` only skips emitting a `FieldCall` for the `COutPacket` entry itself; it does not `return`, break, or otherwise short-circuit the loop, so every call that follows a skipped `COutPacket` entry in `raw.Calls` is still visited and still resolved on subsequent iterations. Confirmed via the new regression test `export_test.go` (`TestExportSourceResolveSkipsCOutPacket`, `export_test.go:37-71`): `Feat::Bodiless` (only a `COutPacket` call) resolves to zero `Calls` (empty body, as intended), and `Feat::HeaderThenField` (`COutPacket` followed by `Encode1`) resolves to exactly one `Calls` entry with `Op == Decode1` — `parsePrim` at `export.go:330-331` maps both `"Decode1"` and `"Encode1"` to the same `Decode1` primitive (the tool's abstraction is direction-agnostic field width, not read-vs-write), so this assertion is correct, not a test bug. **PASS** — the field op immediately after a skipped `COutPacket` is not dropped.

### 4. Version-gating idiom — no raw literal comparisons introduced (PASS)

`grep` across the full Go diff for `MajorVersion`/raw `> N` / `>= N` comparisons (excluding `test.CreateContext(...)` call sites, which are test scaffolding, not gating logic) finds only pre-existing gated code being exercised by new v79 tests — e.g. `witch_tower_score_update_test.go:27` documents that the pre-existing `WitchTowerScoreUpdate` codec's `MajorAtLeast(95)` gate correctly stays OFF at v79 (`witch_tower_score_update.go` itself is unchanged in this diff). None of the new/modified codecs (`SnowballState`, `AriantArenaUserScore`, `ContiMove`, `Tournament`, `TournamentSetPrize`, `TournamentMatchTable`) introduce per-version wire divergence — every doc comment states the layout is "identical in every version checked" (v79/v83/v84/v87/v95/jms), so no `MajorAtLeast` gate was needed and none was added. **PASS**.

### 5. Immutable-model discipline — private fields, getters, no setters (PASS)

Checked every struct touched in `libs/atlas-packet/field/clientbound/{snowball_state,ariant_arena_user_score,conti_move,tournament,tournament_set_prize,tournament_match_table}.go`: all fields are lower-case/private, all mutation happens through `NewXxx(...)` constructors, all exposed accessors are read-only getters (`func (m Model) Field() T`), and `Decode` is the sole in-place mutator, called only on a freshly zero-valued `*Model` immediately after construction — consistent with the rest of the codec library's established pattern (e.g. `AriantArenaUserScore` at `ariant_arena_user_score.go:22-33`, `ContiMove` at `conti_move.go:47-58`). No public field, no `Set*` method, found on any touched struct. **PASS**.

### 6. DOM-21 (atlas-constants type-reuse) — N/A, not a violation

None of the touched structs declare a new domain type, enum, or numeric classification that duplicates something in `libs/atlas-constants` (e.g. no reinvented item-id classification, inventory type, world/channel/map id width, job/skill/monster id type). `TournamentSetPrize.itemId1`/`itemId2` and `AriantArenaScoreEntry.Score` are raw `uint32` wire fields, matching the established convention of this codec library (confirmed: zero files in `libs/atlas-packet/field/clientbound` import `item.Id` or any `atlas-constants` domain type for wire-level struct fields — this package models bytes-on-the-wire, not domain identifiers). No `libs/atlas-constants` equivalent exists for "raw wire item id read via `Decode4`." **N/A**.

### 7. DOM-25 (client-interpreted wire value config-resolution) — N/A to this diff

The corrected fields in this diff (`ContiMove.subState`, `Tournament.value`, `TournamentSetPrize.itemId1/itemId2`, `TournamentMatchTable.match`/`state`, `SnowballState.*`) are payload data (counts, ids, coordinates, HP values, a 768-byte opaque match-table blob) read/written verbatim from IDA-verified fixed offsets — none of them are a client-side lookup-switch code (dispatcher mode byte, notice/fail-reason code) requiring a tenant writer-options table per DOM-25 / `anti-patterns.md:135-165`. The one true dispatcher-mode byte in scope, `ContiMove.state` (selecting one of 6 arms via `state-7`), is unchanged data-shape logic from before this diff (was already a plain field, not newly hardcoded) — this diff only adds the previously-missing conditional `subState` read, it does not introduce a new hardcoded mode-resolution literal. **N/A**.

### 8. No dead code / no TODO/FIXME/stub left behind (PASS)

`grep -n "TODO\|FIXME\|XXX"` over the full diff returns only a documentation reference to a placeholder decompiler symbol name (`sub_XXXXXX`, `tournament_set_prize.go` comment, `tournament.go` comment) — a descriptive comment about IDA's own auto-generated naming convention for an unnamed function, not an unresolved-work marker. No stubs, no `501`, no bare `go` statements. **PASS**.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix / consider)
- `snowball_state.go:117` — `Decode`'s `first`-recovery relies on `r.Available() >= 6`, correct only because `Decode` currently has zero production call sites and is exercised solely by packet-scoped test readers; add a doc comment (or an assertion) pinning that invariant if this codec is ever wired into a shared/multi-packet decode path.
- `ariant_arena_user_score.go:29` — `Entries()` returns the backing slice by reference rather than a defensive copy, inconsistent with the array-value-copy immutability achieved by `TournamentMatchTable.Match()` in the same commit set.
