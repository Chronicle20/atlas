# Review: Task 2 — atlas-maps `state` column, model, and `SetState`

Range reviewed: `6839f59e5..c99525e21` (single commit `c99525e21`, module
`services/atlas-maps/atlas.com/maps`).

## Scope confirmation

`git diff --stat` for the range touches exactly the five files the brief
named:

```
administrator.go     | 42 ++++++++++++++++++++--
entity.go            |  3 ++
model.go             | 23 +++++++-----
processor.go         | 19 ++++++++++
processor_test.go    | 137 +++++++++++++++++++++
```

No file outside `character/location/` is touched. The diff matches the work
described — no scope mismatch.

## Requirement-by-requirement

### 1. `PresenceState` values cross the REST boundary unchanged

`libs/atlas-constants/character/presence.go` (Task 1's deliverable, consumed
here, not modified) defines `PresenceStateOffline = "OFFLINE"`,
`PresenceStateInField = "IN_FIELD"`, `PresenceStateInCashShop = "IN_CASH_SHOP"`.
Task 2 only imports these constants (`entity.go:10`, `model.go:9`,
`processor.go:13`, `administrator.go:8`) and never redefines or re-cases them.
PASS.

### 2. `OFFLINE` is the zero value; unrecognised/empty resolves to `OFFLINE`

- `entity.go:24` — `State string \`gorm:"not null;default:'OFFLINE'"\`` gives
  the DB column a real default.
- `entity.go:41` — `Make` maps via `characterconst.ParsePresenceState(e.State)`,
  which (per `libs/atlas-constants/character/presence.go:29-37`, unmodified
  by this task, confirmed by its own test table including `{"", OFFLINE}` and
  `{"IN_ORBIT", OFFLINE}`) resolves both an empty string and any unrecognised
  value to `PresenceStateOffline`.
- Verified empirically: `TestSetState_TransitionsWithoutDisturbingPosition`
  writes a fresh row via `Set` (which never touches `state`) then asserts
  `m.State() == PresenceStateOffline` — passes, confirming the GORM
  `default:'OFFLINE'` tag is actually applied on INSERT under sqlite, not
  just asserted in a comment.

PASS.

### 3. `upsertLocation` no longer clobbers state on a position write (the load-bearing check)

Before: `db.Save(&e)` — a full-row overwrite that would write `e.State`
(built from a fresh `Builder` with zero-value state, i.e. `""`) into every
row on every position write, silently knocking a live character back to
`OFFLINE`-equivalent on `CHANGE_MAP`.

After (`administrator.go:20-42`): `db.Clauses(clause.OnConflict{Columns:
[tenant_id, character_id], DoUpdates: AssignmentColumns([world_id,
channel_id, map_id, instance, updated_at])}).Create(&e)`, followed by a
re-read. `state` is deliberately absent from `DoUpdates`, so an `ON CONFLICT
DO UPDATE` never touches the column regardless of what value was in the
INSERT's value list.

I did not just read this — I mutation-tested it. I temporarily reverted
`upsertLocation` to the pre-change `db.Save(&e); return e, nil` body (keeping
everything else at HEAD) and reran `TestSet_PreservesState` in isolation:

```
=== RUN   TestSet_PreservesState
    processor_test.go:265: Set reset the state to "OFFLINE", want IN_FIELD preserved
--- FAIL: TestSet_PreservesState (0.00s)
```

Then restored `administrator.go` to the committed content (`git diff` on the
file after restoration is empty, confirming a clean revert) and reran the
full package:

```
ok  	atlas-maps/character/location	0.010s
```

all tests pass, including the four new ones. This proves `TestSet_PreservesState`
is not vacuous — it fails against the exact regression the brief called out
and passes against the fix. PASS, with direct evidence.

### 4. `SetStateIfOnline` genuinely refuses to write over an `OFFLINE` row

`administrator.go:53-66` (`setLocationState`): when `conditional=true`, the
query adds `.Where("state <> ?", string(characterconst.PresenceStateOffline))`
before `.Update("state", ...)`. A row whose persisted `state` column is the
literal string `"OFFLINE"` (which — per point 2 above — is what every fresh
or explicitly-offlined row actually holds, not empty string) is excluded from
the `UPDATE`'s row set, so the `Update` call affects 0 rows and the value
stays `OFFLINE`.

`TestSetStateIfOnline_DoesNotResurrectOfflineRow` (processor_test.go:216-243)
exercises exactly this: `SetState(..., Offline)` then `SetStateIfOnline(...,
InField)`, asserts the row is still `OFFLINE`. Ran in isolation, passes (see
full-suite run above).
`TestSetStateIfOnline_AppliesWhenOnline` (processor_test.go:245-267) is the
complementary case on a non-`OFFLINE` row, also passes. PASS.

### 5. No invented staleness/TTL constant

`administrator.go` and `processor.go` contain no timeout, TTL, or sweeper
logic — `SetStateIfOnline`'s only condition is the `state <> 'OFFLINE'`
predicate, matching the brief's "SetStateIfOnline is the only staleness
control" constraint. PASS.

### 6. No placeholder/stub

Both `SetState` and `SetStateIfOnline` are fully implemented (not stubs),
each backed by a real `UPDATE ... WHERE ...` through `setLocationState`.
PASS.

### 7. Builder pattern; no new `*_testhelpers.go`

`git diff --stat` shows no new file. `processor_test.go`'s four new tests
reuse the file's existing `newCtxTenant(t)` / `newTestDB(t)` helpers
(processor_test.go:29, 102, confirmed present and unmodified in this diff)
and construct locations only through `NewProcessor(...).Set(...)` /
`.SetState(...)` — the existing public surface, not a bypass. PASS.

### 8. Exact interface names/signatures Tasks 3-5 will consume

Checked against current file contents (not just the diff hunks), matched
verbatim:

- `model.go:29` — `func (m Model) State() characterconst.PresenceState { return m.state }`
- `model.go:63` — `func (b *Builder) SetState(v characterconst.PresenceState) *Builder { b.m.state = v; return b }`
- `processor.go:19-20` — `Processor` interface:
  `SetState(characterId uint32, state characterconst.PresenceState) error` and
  `SetStateIfOnline(characterId uint32, state characterconst.PresenceState) error`
- `processor.go:96-108` — `ProcessorImpl` implements both; `var _ Processor =
  (*ProcessorImpl)(nil)` (processor.go:38, pre-existing, unmodified) statically
  pins the interface satisfaction, so a signature drift would fail to build.

All exact-spelling requirements satisfied. PASS.

### 9. `libs/atlas-constants` reuse, no re-definition

`character/location` only imports `characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`
(Task 1's package); it does not define a parallel `PresenceState`-like type
locally. PASS.

## Cross-file consistency checks

- No other file in the module constructs `location.entity{}` literally
  (`grep -rn "entity{"` in `character/location/*.go` outside
  administrator.go/entity.go/model.go/their tests returns nothing), so the
  new `State` field addition to the struct cannot have silently broken an
  un-reviewed construction site.
- `Migration(db)` (`entity.go:15-17`) is unchanged and still just calls
  `db.AutoMigrate(&entity{})`, which is sufficient to add the new column —
  no manual migration script needed, matching "additive, no backfill."
- `SetField` (`model.go:65-71`) is untouched, confirmed by direct read — it
  still sets only worldId/channelId/mapId/instance, which is what makes
  `Set`'s call to `NewBuilder(...).SetField(f).Build()` produce a
  zero-value-`state` `Model` on every position write, and is exactly why
  `upsertLocation` must not carry that zero value into `Save`/`Create`
  unconditionally (point 3).

## Not evaluable

- REST-layer exposure of `State()` (a GET returning the new field) is Task 3
  work, not in this diff, and is correctly absent here — nothing to flag.
- Whether `AutoMigrate` runs cleanly against a real Postgres instance with
  existing rows (vs. the sqlite in-memory test DB used here) was not
  exercised; the brief's "existing rows adopt the Go zero value" claim rests
  on Postgres's `ADD COLUMN ... DEFAULT` semantics, which I did not verify
  against a live Postgres migration. This is a reasonable gap for a
  module-local unit test suite and is standard AutoMigrate behavior, but it
  is genuinely outside what this diff's tests exercise.

## Verdict

All step-by-step brief requirements are met; the exact interface surface
Tasks 3-5 depend on is present with the exact names and signatures specified;
the load-bearing regression (`db.Save` clobbering state) is fixed and the
regression test that pins the fix was mutation-tested and confirmed to fail
against the old code and pass against the new. No invented constants, no
stubs, no new test-helper file, no re-definition of shared constants. No
blocking findings.
