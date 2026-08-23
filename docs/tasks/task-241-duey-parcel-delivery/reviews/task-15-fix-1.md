# Review: Task 15 fix round 1 — B1/B2 (`AssetData` field gaps)

Commit range given: `f1fd89fb9..281ce4242`
Fix commit under review: `281ce4242` — `fix(parcel): map ItemLevel/RingId/ViciousCount into the parcel item snapshot`
Prior review being closed: `docs/tasks/task-241-duey-parcel-delivery/reviews/task-15.md`
Report: `.superpowers/sdd/plan/task-15-report.md`, "Fix round 1/5" section

## Scope note

`git log --oneline f1fd89fb9..281ce4242` shows **two** commits in the given
range, not one:

- `66125ac19` — `feat(channel): duey fee table and send validation` (4 files
  under `services/atlas-channel/atlas.com/channel/parcel/`: `fee.go`,
  `fee_test.go`, `validation.go`, `validation_test.go`)
- `281ce4242` — the actual B1/B2 fix (4 files, all `services/atlas-parcel/...`)

The task description says "the fix commit only," but the range as given
spans an unrelated commit that landed in between. `git show --stat 281ce4242`
confirms the fix commit itself touches exactly the 4 files the report claims
and nothing from `atlas-channel`. This review evaluates only `281ce4242`
(the B1/B2 fix); `66125ac19` is a different, unrelated unit of work (a
channel-side fee/validation feature) and is out of this review's scope —
flagging the range mismatch per the task brief's `scope_confirmed`
instruction, not treating it as a defect in the fix itself.

## Files changed by `281ce4242`

```
services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer.go      | 3 +++
services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer_test.go | 8 ++++++++
services/atlas-parcel/atlas.com/parcel/parcel/asset_data.go                    | 10 ++++++++++
services/atlas-parcel/atlas.com/parcel/parcel/processor_custody.go             | 6 ++++++
4 files changed, 27 insertions(+)
```

Matches the report's "Files changed" list for the fix round exactly.

## Check 1 — end-to-end mapping, each of the three fields traced individually

**`ItemLevel`** (wire) → `AssetData.LevelType` (destination):
- Wire: `kafka/message/custody/kafka.go:89` — `ItemLevel byte
  \`json:"itemLevel"\`` on `AcceptToParcelCommandBody` (unchanged this
  round — already existed).
- Consumer pass-through: `kafka/consumer/custody/consumer.go:117` —
  `ItemLevel: b.ItemLevel` (new this round) into `parcel.AcceptParams`.
- `AcceptParams.ItemLevel byte` — `parcel/processor_custody.go:63` (new this
  round).
- `AcceptCustody`'s `AssetData{...}` literal — `processor_custody.go:126` —
  `LevelType: params.ItemLevel` (new this round), landing in the pre-existing
  `AssetData.LevelType byte \`json:"levelType"\`` field
  (`parcel/asset_data.go:38`, unchanged).

No gap. Matches the orchestrator compensator's precedent (`LevelType:
p.ItemLevel`, `saga/compensator.go:2932`) cited in both the original review
and the report.

**`RingId`** (wire) → `AssetData.RingId` (new destination field):
- Wire: `kafka.go:91` — `RingId uint32 \`json:"ringId"\`` (unchanged,
  pre-existing).
- Consumer: `consumer.go:119` — `RingId: b.RingId` (new).
- `AcceptParams.RingId uint32` — `processor_custody.go:65` (new).
- `AssetData{...}` literal: `processor_custody.go:129` — `RingId:
  params.RingId` (new), into the new `AssetData.RingId uint32
  \`json:"ringId"\`` field (`asset_data.go:55`, new).

No gap.

**`ViciousCount`** (wire) → `AssetData.ViciousCount` (new destination field):
Same chain — `kafka.go:92` (pre-existing) → `consumer.go:120` (new) →
`AcceptParams.ViciousCount` (`processor_custody.go:66`, new) →
`AssetData{...}` literal `ViciousCount: params.ViciousCount`
(`processor_custody.go:130`, new) → `AssetData.ViciousCount uint32
\`json:"viciousCount"\`` (`asset_data.go:56`, new).

No gap.

All three fields are mapped end to end at every hop: wire command body →
`AcceptParams` → `AssetData{}` literal → persisted row (via
`Builder.SetItemSnapshot`/`Create`, unchanged from round 1). **B1 and B2 are
both closed** — not a two-of-three fix.

## Check 2 — JSONB, no migration, old-row deserialization

- `parcel/entity.go:47` — `ItemSnapshot AssetData \`gorm:"type:jsonb"\`` —
  confirmed independently (not taken from the report): the column is JSONB,
  read/written via `AssetData.Value()`/`Scan()`
  (`asset_data.go:64-77`, unchanged), both of which go through
  `encoding/json`. `git diff f1fd89fb9..281ce4242 --stat -- '*migrat*'`
  produced no output — no migration file was added or needed, consistent
  with the JSONB claim.
- Existing stored rows without `ringId`/`viciousCount` keys: `AssetData.Scan`
  calls `json.Unmarshal(bytes, a)` into a zero-valued `*AssetData`. Standard
  Go `encoding/json` semantics: a JSON object missing a key leaves the
  corresponding non-pointer struct field at its zero value with no error
  returned — `RingId`/`ViciousCount` are `uint32` (not pointers), so absence
  decodes to `0`, not an error. Nothing in `Scan` or any caller
  (`processor.go` read paths, unchanged this round) treats a missing key or
  a zero value as an error condition — `Scan` only errors on a genuine type
  assertion failure (`value.([]byte)`) or a malformed JSON payload, neither
  of which is triggered by a key simply being absent. Old rows deserialize
  cleanly with `RingId`/`ViciousCount` reading `0`.

## Check 3 — RED evidence genuinely per-field

`consumer_test.go` (`accept_with_item` subtest, lines added this round):

```go
assert.Equal(t, byte(3), m.ItemSnapshot().LevelType, "ItemLevel must map to AssetData.LevelType")
assert.Equal(t, uint32(777), m.ItemSnapshot().RingId, "RingId must survive the parcel round-trip")
assert.Equal(t, uint32(12), m.ItemSnapshot().ViciousCount, "ViciousCount must survive the parcel round-trip")
```

Three separate `assert.Equal` calls, three distinct expected values (`3`,
`777` = `0x309`, `12` = `0xc`), each against a different field of the same
returned `ItemSnapshot()` — not a shared fixture asserted once in aggregate.
`assert.Equal` (testify, non-`require`) continues past a failure, so all
three would independently report if any single mapping regressed — matching
the report's claimed RED transcript exactly:

```
Not equal: expected: 0x3   actual: 0x0   Messages: ItemLevel must map to AssetData.LevelType
Not equal: expected: 0x309 actual: 0x0   Messages: RingId must survive the parcel round-trip
Not equal: expected: 0xc   actual: 0x0   Messages: ViciousCount must survive the parcel round-trip
```

Independently re-ran the full custody suite as committed (not reverted) and
confirmed GREEN:

```
cd services/atlas-parcel/atlas.com/parcel && go build ./... && \
  go test ./kafka/consumer/custody/... -run TestCustodyCommands -v
```
→ all 10 subtests, including `accept_with_item`, PASS.

The RED claim is credible: reverting any one of the three `AssetData{...}`
mapping lines (leaving `AcceptParams`/wire pass-through intact) would fail
exactly that field's assertion while leaving the other two green, since each
is asserted against its own field independently.

## Check 4 — naming/tags

- `AssetData.RingId uint32 \`json:"ringId"\`` / `ViciousCount uint32
  \`json:"viciousCount"\`` (`asset_data.go:55-56`) match the wire body's own
  tags exactly (`kafka.go:91-92`: `\`json:"ringId"\``, `\`json:"viciousCount"\``)
  and match MTS's `listing.Entity` field names (`RingId`, `ViciousCount`,
  `entity.go:74-75`, gorm columns `ring_id`/`vicious_count`).
- `AssetData.LevelType` is the pre-existing field (not touched this round);
  the fix only starts writing to it. No new tag introduced for `ItemLevel`.
- `AcceptParams` fields (`ItemLevel`, `RingId`, `ViciousCount`,
  `processor_custody.go:63,65-66`) are plain Go struct fields, no JSON/gorm
  tags — `AcceptParams` is not marshalled across any boundary (confirmed:
  it's constructed in-process from the already-decoded wire body and
  consumed in-process by `AcceptCustody`), so no tag-mismatch risk applies
  here, unlike the branch's four prior world-0-tag incidents which were all
  at an actual marshal boundary.
- No instance of the world-0-style tag-mismatch class found in this fix's
  diff.

## Check 5 — out-of-scope check

- The fix commit (`281ce4242`) touches exactly 4 files, all inside the B1/B2
  blast radius: `asset_data.go` (new fields + comment), `processor_custody.go`
  (`AcceptParams` fields + `AssetData{}` literal), `consumer.go` (3-line
  pass-through), `consumer_test.go` (fixture values + 3 assertions).
- No edits to `kafka/message/custody/kafka.go`, `kafka/producer/custody/producer.go`,
  `main.go`, the `ErrAlreadyReleased` sentinel/comments, the idempotency
  comment wording, or any of the original 9 subtests — confirmed by the
  file list above and by reading each of the 4 changed files' diffs in full;
  none of the previously-approved code was touched or reworked.
- No new constant or type introduced; `RingId`/`ViciousCount` reuse `uint32`,
  matching both the wire type and the MTS column type — no
  `libs/atlas-constants` duplication.
- The comment added to `asset_data.go:48-54` is accurate and appropriately
  scoped (explains the field addition, cites the review finding, cites the
  precedent) — not scope creep.

## Findings summary

### Blocking

None. Both B1 and B2 are closed: all three fields (`ItemLevel`, `RingId`,
`ViciousCount`) are mapped end to end with no gap at any hop, the new
`AssetData` fields are JSONB-safe for old rows, and the new test assertions
are genuinely per-field (would independently catch a regression in any one
of the three mappings).

### Non-blocking

- Scope/range mismatch: the given commit range `f1fd89fb9..281ce4242`
  contains an unrelated commit (`66125ac19`,
  `feat(channel): duey fee table and send validation`) in addition to the
  fix commit. Not a defect in the fix itself, but worth flagging so the
  range used for the *next* review step is the intended one.

### Not evaluable

- None. The fix's diff surface (4 files) and its immediately-relied-upon
  contracts (wire body tags, `AssetData`'s JSONB `Value`/`Scan`, the MTS
  `listing.Entity` precedent) were all read and traced directly.

## Verdict

Both blocking findings from the prior review are closed by `281ce4242`, no
new defect was introduced, and the previously-approved parts of Task 15 were
left untouched as instructed.
