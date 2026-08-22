# Review — Task 13: rewire Cash/0543 to the submit, implement the submit flow

**Range:** `6673126..50c79fadf`
**Verdict:** APPROVED_WITH_FINDINGS

## Scope

`git diff --stat 6673126..50c79fadf` — exactly 8 files, all under
`services/atlas-channel/atlas.com/channel`:

```
maplelife/registry.go                                    |  51 +--
maplelife/registry_test.go                                | 130 ++----
socket/handler/character_cash_item_use.go                 |   8 +-
socket/handler/character_cash_item_use_maple_life_test.go |  59 ++-
socket/handler/maple_life_create.go                       | 220 ++++++++++
socket/handler/maple_life_create_test.go                  | 472 +++++++++++++
socket/handler/maple_life_open.go                          |  52 --- (deleted)
socket/handler/maple_life_open_test.go                     | 125 --- (deleted)
```

No `libs/`, `deploy/`, `docs/packets/`, seed templates, `atlas-login`,
`atlas-character-factory`, or `atlas-saga-orchestrator` touched. No
`*_testhelpers.go`. No `TODO` markers in the changed files.
`git diff --stat` matches the described unit; no scope mismatch.

`go build ./...` and `go test ./socket/handler/... ./maplelife/...` both pass
(re-run independently, not taken from the report).

## 1. Classification-first regression suite (PRD FR-2.2) — PASS, verified independently

- **(a) The seam genuinely distinguishes arm selection.** Independently
  grepped (not the report's grep): `seedCharacterFunc` is referenced in
  exactly one non-test file, `maple_life_create.go` — its declaration
  (`:68`) and its one call site (`:186`). No other arm in
  `character_cash_item_use.go` calls it. Confirmed at
  `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:800-812`,
  where the classification-543 branch is the only place `handleMapleLifeCreate`
  is invoked.
- **(b) v83 and v95.** `TestCashItemNeighbourArmsUnaffectedByMapleLife`
  (`character_cash_item_use_maple_life_test.go:126-134`) iterates a `v83`/`v95`
  table for all four item rows (pet multi-consumable, sealing lock, vicious
  hammer, maple life) — 8 subtests total. Confirmed by running
  `go test ./socket/handler/... -run TestCashItemNeighbourArmsUnaffectedByMapleLife -v`.
- **(c) Non-vacuous under mis-routing.** `installMapleLifeCreateObservationSeams`
  (`character_cash_item_use_maple_life_test.go:80-114`) forces gates 3-4
  (`accountSlotsFunc`, `charactersInWorldFunc`, `mapleLifeNameValidityFunc`) to
  always pass and records only whether `seedCharacterFunc` fired. The three
  non-Maple-Life rows never reach `handleMapleLifeCreate` at all — their
  `false` observation is real "arm not reached," not a gate rejecting them,
  since the seams that would reject them are already forced to pass. If the
  dispatcher mis-routed maple life to e.g. the pet-consumable arm, `seedCalled`
  would stay `false` against `wantMapleLife: true` and the test would fail; if
  it mis-routed a neighbour into the Maple Life arm, `seedCalled` would flip to
  `true` against `wantMapleLife: false` and fail. Verified this reasoning
  against the actual code rather than accepting the report's assertion.

This property holds. No finding here.

## 2. Submit payload reaching the factory — enumerated, not judged

`maple_life_create.go:68-77` (`seedCharacterFunc`) sends to
`factory.Processor.SeedCharacter(accountId, worldId, name, jobIndex,
subJobIndex, face, hair, color, skinColor, gender, top, bottom, shoes,
weapon, strength, dexterity, intelligence, luck)`
(`character/factory/processor.go:36-39`) as:

| SeedCharacter param | source |
|---|---|
| `accountId`, `worldId` | session (gate 5) |
| `name` | `sub.Name()` |
| `jobIndex` | `sub.CurrentClass()` |
| `subJobIndex` | `0` (hard-coded) |
| `face`, `hair`, `color` (hairColor), `skinColor` | `sub.AL0()`..`sub.AL3()` positionally |
| `gender` | `sub.Gender()` |
| `top`, `bottom`, `shoes`, `weapon` | `0` (hard-coded) |
| `strength`, `dexterity`, `intelligence`, `luck` | `0` (hard-coded) |

`sub.SP()` is decoded by the codec but **never referenced** anywhere in
`maple_life_create.go` (grepped for `sub.SP` / `.SP()` — zero hits) — it is
silently dropped. The in-code doc comment (`maple_life_create.go:45-64`)
enumerates al0-al3→face/hair/hairColor/skinColor and currentClass→jobIndex,
and says "everything else... as 0," but never names `sp` by name as a
dropped field. This is the one enumeration gap: a future reader checking the
comment against the wire fields would not learn that `sp` specifically is
discarded rather than folded into one of the zeroed params. Non-blocking —
the field is genuinely unused in the call, just under-documented.

The invented positional mapping (al0-al3→face/hair/hairColor/skinColor,
currentClass→jobIndex) is **confined to the single `seedCharacterFunc` var**
(`maple_life_create.go:45-77`) — grepped `sub.AL0\|sub.AL1\|sub.AL2\|sub.AL3\|sub.CurrentClass` across the whole diff and every hit is inside that one function. It has not leaked into gates 2-5, tests, or the registry. Per the task's instruction, I have not evaluated whether the mapping itself is correct — that is out of my scope and is tracked separately in `docs/tasks/task-246-maple-life-character-creation/open-selected-al-mapping.md`.

## 3. `beginMapleLife` / open path fully removed — PASS

- `maple_life_open.go` and `maple_life_open_test.go` are deleted (absent from
  `ls socket/handler/`, present only as deletions in the diff stat).
- `grep -rn beginMapleLife` across the whole module returns only doc-comment
  mentions inside the new `maple_life_create.go` (explaining what it replaced)
  — no executable reference remains, and `go build ./...` confirms no dangling
  symbol.
- `mapleLifeSupported` is unchanged and still gates ahead of the sub-body
  decode (`character_cash_item_use.go:800`: `if !mapleLifeSupported(t) { ...;
  return }` precedes `cashsb.NewItemUseMapleLife(...).Decode(...)`).
  `TestMapleLifeSupported` still asserts the gms_v84 exclusion
  (`character_cash_item_use_maple_life_test.go` case `"GMS 84": false // task-246
  Task 2 ruling`).
- `TestMapleLifeUnsupportedVersionWritesNothing` survives with its
  `request.Reader.Position()` assertion intact
  (`character_cash_item_use_maple_life_test.go`, asserts `reader.Position() ==
  commonPrefixLen` for both v79 and v84) — the FR-2.4 proof is present.

One pre-existing, out-of-scope staleness noted for completeness, not a
finding against this unit: `socket/handler/maple_life_check_name.go:48`
(a Task 12 file, not touched by this diff) still reads "...as the player
types a candidate name into the naming dialog **beginMapleLife opened**,"
which is now factually stale since `beginMapleLife` no longer exists. This
file is outside Task 13's diff surface, so it is reported here rather than
treated as a Task 13 defect.

## 4. Registry edit — PASS

`maplelife/registry.go`:
- `PhaseOpen`, `OpenTTL`, `Submit` are gone (grepped, zero hits).
- `Sweep` (`:167-181`) collapsed to `SubmittedTTL` unconditionally, per-tenant
  scoping (`if !k.Tenant.Is(t) { continue }`) and its doc comment
  (`:154-166`, explaining why an unscoped Sweep would steal entries) both
  intact.
- Survive as required: `Phase`/`PhaseSubmitted` (`:27-35`), `Put` (`:88-93`),
  `Get` (`:96-102`), `Take` (`:105-115`), `TakeByTransactionId` (`:118-136`),
  `ClearAccount` (`:139-143`), `SubmittedTTL` (`:41`), `Key` (`:45-48`),
  `Entry` (`:51-60`), `Expired` (`:63-67`) — Task 14's dependency surface is
  undisturbed.
- `TakeByTransactionId`'s doc comment no longer references `PhaseOpen`.

`maplelife/registry_test.go`:
- Tenant-scoping coverage preserved for both mechanisms, not just phase:
  `TestTakeByTransactionIdIsTenantScoped` and `TestSweepIsTenantScoped` both
  present and assert the cross-tenant isolation directly (one tenant's
  `Take`/`Sweep` must not consume another tenant's entry).
- `TestSubmittedTTLOutlivesSagaTimeout` still asserts `SubmittedTTL >
  10*time.Second` against the real constant (not a copy).

No finding here.

## 5. Pre-check gates — PASS, both Disagreement resolutions judged correct

Gate order in `handleMapleLifeCreate` (`maple_life_create.go:97-186`): gate 2
(ownership+classification) → gate 3 (slot limit) → gate 4 (name re-check) →
gate 5 (session sourcing, no packet comparison) → `seedCharacterFunc`. Gate 1
correctly absent. Gate 2 compares `cashItemInSlotFunc(l, ctx, s.CharacterId(),
int16(source))` against `itemId`/`source`, which are parameters threaded from
the 543 packet's own common ItemUse prefix at the call site
(`character_cash_item_use.go:812`: `handleMapleLifeCreate(...)(s, itemId,
source, *sp)`) — not a registry entry, as required.

Both Disagreement-clause resolutions checked against their stated cause:

1. **Gate 5 mismatch logging.** Independently confirmed against
   `libs/atlas-packet/cash/serverbound/item_use_maple_life.go`'s field list
   (`sName, al[0..3], nGender, nCurrentClass, nSP, update_time`) — there is
   genuinely no accountId/worldId on the wire. Narrowing the test to assert
   session-sourced values only, without inventing a mismatch to log, is
   correct; nothing was silently dropped, it was reported.
2. **No pre-existing record to `Take` on the error path.** Confirmed:
   `Put` is called exactly once, on the success branch after
   `seedCharacterFunc` returns `nil` (`maple_life_create.go:196-204`); no
   `Put` occurs on any other path, so there is genuinely nothing to `Take` on
   an error. Correct.

## 6. FR-5.1 — nothing consumes the item — PASS

`assertNoDestroySaga` (`maple_life_create_test.go:456-469`) source-scans
`maple_life_create.go` for `saga.NewProcessor`, `saga.Saga{`, `DestroyAsset` —
none present (confirmed by reading the file). This is a structural
whole-file check rather than a per-row seam assertion, which is a stronger
guarantee than the brief's original per-row design: it holds regardless of
which gate a given test row exercises, since the file never references the
saga package at all. `TestMapleLifeCreateNeverConsumesTheItem` and the source
scan inside `TestMapleLifeCreatePreCheckOrder`'s subtests both invoke it.

## 7. Error arms — PASS, one non-blocking coverage gap

`libs/atlas-packet/maplelife/clientbound/error.go:51,55,60` declares exactly
three arms (`MapleLifeErrorSuccess`, `MapleLifeErrorNameTakenAtSubmit`,
`MapleLifeErrorUnknownError`). `maple_life_create.go` uses only
`MapleLifeErrorUnknownError` (the `fail` closure, `:104-109`) and
`MapleLifeErrorNameTakenAtSubmit` (`:166`) — never `MapleLifeErrorSuccess`,
matching "write nothing on success." No fourth arm invented; no unrecognised
key reaches `MapleLifeErrorBody` (grepped all `mlcb.MapleLifeError*`
references in the diff — exactly these two constants appear).

Success path writes nothing to the client and records a `PhaseSubmitted`
entry carrying the returned `transactionId`
(`maple_life_create.go:196-204`, `TestMapleLifeCreateMapsFactoryOutcomes/success`
asserts both `lastArm()` returns `ok=false` and the registry entry).

**Non-blocking:** the addendum said the 400-vs-500 rows should be
"differentiated by what they log... and worth asserting," since both write
the same wire arm. `logSeedFailure` (`maple_life_create.go:207-217`) does
classify by `errors.Is(err, requests.ErrBadRequest)` into two different log
messages, but `TestMapleLifeCreateMapsFactoryOutcomes`'s `"400 invalid look or
name"` and `"500 server error"` subtests assert only the wire arm and the
registry state — neither subtest asserts which log message fired. The
classification logic exists and is exercised, but is not itself pinned by a
test; a regression that collapsed `logSeedFailure` to one message would not
be caught here.

## 8. Scope — PASS

Already covered above (§ Scope). Matches the addendum's stated boundary
exactly.

## Not evaluable

- Whether the al0-al3→face/hair/hairColor/skinColor and currentClass→jobIndex
  positional mapping is factually correct. Out of this review's scope per the
  task brief — the controller is handling it separately
  (`docs/tasks/task-246-maple-life-character-creation/open-selected-al-mapping.md`
  already exists for this).

## Findings

**Blocking:** none.

**Non-blocking:**
1. `maple_life_create.go:45-64` — the seam-func doc comment enumerates the
   al0-al3/currentClass mapping and says "everything else... as 0," but never
   names `sp` specifically as a dropped field, even though `ItemUseMapleLife.SP()`
   is decoded and is the one field the comment's enumeration misses by name.
2. `maple_life_create_test.go`'s `"400 invalid look or name"` and `"500
   server error"` subtests (`TestMapleLifeCreateMapsFactoryOutcomes`) don't
   assert which `logSeedFailure` message fired, so the HTTP-status
   classification the addendum asked to be "worth asserting" is exercised
   but not pinned by a test.
3. `socket/handler/maple_life_check_name.go:48` (outside this diff, a Task 12
   file) has a doc comment referencing "the naming dialog beginMapleLife
   opened," which is now stale since `beginMapleLife` no longer exists.
   Reported for the record; not a Task 13 defect since the file is untouched
   by this unit.
