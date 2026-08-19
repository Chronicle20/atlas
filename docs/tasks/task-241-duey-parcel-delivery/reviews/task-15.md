# Review: Task 15 — atlas-parcel custody consumer

Commit range: `964196ee0..f1fd89fb9` (single commit `f1fd89fb9`)
Brief: `.superpowers/sdd/plan/task-15-brief.md`
Report: `.superpowers/sdd/plan/task-15-report.md`

## Scope

`git diff --stat 964196ee0..f1fd89fb9` — 7 files, all under
`services/atlas-parcel/atlas.com/parcel/`:

- `kafka/consumer/consumer.go` (new, `NewConfig` helper)
- `kafka/consumer/custody/consumer.go` (new)
- `kafka/consumer/custody/consumer_test.go` (new)
- `kafka/message/custody/kafka.go` (new)
- `kafka/producer/custody/producer.go` (new)
- `parcel/processor_custody.go` (new)
- `main.go` (modified — consumer registration)

Matches the brief's file list exactly. Reviewed against: the orchestrator's
Task 13 command envelope (`kafka/message/parcel/custody/kafka.go`) and status
consumer (`kafka/consumer/parcel/custody/consumer.go`), Task 14's compensator
(`saga/compensator.go`), and the MTS custody twin
(`services/atlas-mts/.../holding/processor_custody.go`,
`services/atlas-mts/.../listing/processor_custody.go`) as the pattern this
task is required to mirror.

## 1. The seam, field for field

`services/atlas-parcel/atlas.com/parcel/kafka/message/custody/kafka.go` was
diffed byte-for-byte against
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/parcel/custody/kafka.go`.
Every struct (`Command[E]`, `AcceptToParcelCommandBody`,
`ReleaseFromParcelCommandBody`, `RestoreParcelCommandBody`,
`RemoveParcelCommandBody`, `StatusEvent[E]`, `StatusEventAcceptedBody`,
`StatusEventReleasedBody`, `StatusEventErrorBody`) and every JSON tag matches
exactly. Constants (`ACCEPT_TO_PARCEL`/`RELEASE_FROM_PARCEL`/
`RESTORE_PARCEL`/`REMOVE_PARCEL`/`ACCEPTED`/`RELEASED`/`ERROR`, both env var
names) match exactly.

Status consumer coverage confirmed: the orchestrator's
`kafka/consumer/parcel/custody/consumer.go` only registers handlers for
ACCEPTED/RELEASED/ERROR; RESTORE_PARCEL/REMOVE_PARCEL are dispatched
fire-and-forget by the compensator
(`saga/compensator.go`'s `RestoreParcelAndEmit`/`RemoveParcelAndEmit`) with no
success ack expected. atlas-parcel's `handleRestoreParcel`/`handleRemoveParcel`
(`kafka/consumer/custody/consumer.go:164-202`) correctly emit ERROR-only,
matching this contract — no dangling success producer that the orchestrator
would never consume.

**PASS.**

## 2. world-0 defaulting

`AcceptToParcelCommandBody.WorldId world.Id `json:"worldId"`` — same on both
sides (`kafka/message/custody/kafka.go:55` here,
orchestrator's `kafka/message/parcel/custody/kafka.go:50`), no `omitempty`,
plain value type. `parcel.AcceptParams.WorldId` (Go-only, not
wire-marshalled) and `parcel.Model`/`parcel.Entity`'s pre-existing
`world.Id`/`byte` cast in `builder.go:205,237` are unchanged by this task and
round-trip world 0 correctly (no pointer, no `omitempty`, casts are
unconditional).

**PASS** — no instance of the world-0 tag-drop class found in this task's
surface.

## 3. `AssetData` field gaps — `ItemLevel`/`RingId`/`ViciousCount`

The implementer's report describes these as having "no home" in
`parcel.AssetData` and disposes of them as out of scope. This is **not
correct** for `ItemLevel`, and the disposition is wrong for all three.

- `parcel.AssetData` (`services/atlas-parcel/atlas.com/parcel/parcel/asset_data.go:14-47`)
  **does** carry a `LevelType byte `json:"levelType"`` field alongside
  `Level byte`. It is a JSONB-backed struct (`entity.go:47`,
  `gorm:"type:jsonb"`) — adding/using fields here costs no migration.
- The orchestrator's own compensator, in the exact code path this task's
  implementer read while checking the wire contract, establishes the
  precedent: `assetDataFromParcelSnapshot` (`saga/compensator.go:2913-2938`)
  maps `LevelType: p.ItemLevel` when reconstructing this identical
  `asset.AssetData` shape from an `AcceptToParcelPayload`. MTS's twin
  function does the same (`compensator.go:2882-2907`,
  `LevelType: p.ItemLevel`).
- `processor_custody.go:26-66` (`AcceptParams`) has **no `ItemLevel` field at
  all** — it was dropped before it ever reached the `AssetData{}` literal at
  `processor_custody.go:102-125`, which never sets `LevelType`. The wire
  field is decoded (`kafka.go:84`, `ItemLevel byte `json:"itemLevel"``) and
  then discarded in `consumer.go:83-120`, which never reads `b.ItemLevel`.

  **Blocking**: `services/atlas-parcel/atlas.com/parcel/parcel/processor_custody.go:26-66,102-125`
  — `ItemLevel` has an existing destination field (`AssetData.LevelType`)
  and an established mapping precedent in this exact codebase; leaving it
  unmapped is a one-line, same-package fix, not a genuine scope boundary.

- `RingId`/`ViciousCount` genuinely have no destination field in
  `parcel.AssetData` today. This mirrors a systemic gap: even the
  orchestrator's own `asset.AssetData` (`kafka/message/asset/kafka.go:29-67`,
  the struct used to re-grant items to a character's inventory) lacks these
  fields, so a ring item or a vicious-hammered item loses this data at the
  inventory/asset-snapshot boundary generally, not only in atlas-parcel.
  However: MTS's *own* custody store does not use this shared `AssetData`
  contract for equip identity — `listing.Entity`/`listing.Model` persist
  `ItemLevel`/`RingId`/`ViciousCount` as first-class columns
  (`services/atlas-mts/atlas.com/mts/listing/entity.go:72-75`) and round-trip
  them through `processor_custody.go:159-162,330-333`. That is the
  established, working pattern this task was told to mirror, and
  atlas-parcel's `Entity`/`AssetData` has no equivalent. Since
  `AssetData` is JSONB (no migration required to add fields), extending it
  with `RingId`/`ViciousCount` and mapping them through `AcceptCustody` was
  producible within this task's own files.

  **Blocking**: `services/atlas-parcel/atlas.com/parcel/parcel/asset_data.go:14-47`,
  `processor_custody.go:26-66,102-125` — ring identity and vicious-hammer
  count are silently dropped for every equipped item mailed through Duey.
  This is real item-property loss in a player-facing custody flow, and the
  established in-repo alternative (MTS's persisted columns) shows the fix is
  producible without a schema migration.

None of `ItemLevel`/`RingId`/`ViciousCount` are asserted by
`TestCustodyCommands` (only `Strength` is, per the brief's table), so the
test suite does not surface this gap — it is a genuine correctness hole, not
a covered-and-accepted risk.

## 4. `ErrAlreadyReleased` sentinel design deviation

`processor_custody.go:19,152-188` — `ReleaseCustody` returns
`(currentModel, ErrAlreadyReleased)` when the row is no longer pending at
write time (replay or lost race). `consumer.go:144-148` checks this with
`errors.Is(err, parcel.ErrAlreadyReleased)`, not `==` or string matching —
correct.

Checked against the brief's table row: "release replay ... second delivery
affects 0 rows and still reports success — no second event." The sentinel is
exactly what is needed to satisfy that literal requirement, and
`consumer_test.go:203-230` ("release replay") asserts it directly
(`require.Len(t, released, 1, "replay must not re-emit RELEASED")` +
`assert.Empty(t, errored, ...)`), and would fail without the sentinel logic
(a naive replay call, without the pre-transaction `m.Status() != StatusPending`
check, would call `UpdateStatusIfPending` a second time, get 0 rows, and — if
mishandled — either erroneously emit RELEASED again or spuriously ERROR).

Checked against the orchestrator's status consumer
(`kafka/consumer/parcel/custody/consumer.go:65-80`): `handleReleasedEvent`
calls `saga.NewProcessor(...).AcceptEvent(...)`, which (`saga/processor.go:422-449`)
already guards on saga lifecycle/current-step state, so a genuine duplicate
RELEASED delivered to an already-completed step would itself be absorbed as a
no-op there too. **The sentinel is therefore not strictly load-bearing for
orchestrator-side correctness** (the consumer is independently idempotent),
but it is still the correct implementation of the brief's explicit table row
and adds a second, cheaper layer of duplicate suppression (skips a wasted
Kafka publish). Not a defect — sound design choice, matches the brief.

**PASS.**

## 5. Idempotency comment wording

`RestoreCustody` (`processor_custody.go:190-206`) and `RemoveCustody`
(`processor_custody.go:208-221`) both state "0 rows affected is success, not
an error" — the actual semantics, not a bare "idempotent". Matches the prior
review's blocking note on this branch.

**PASS.**

## 6. Test quality

`TestCustodyCommands` (`kafka/consumer/custody/consumer_test.go`) has exactly
the 10 subtests named in the brief's table, each exercising the handler
functions directly (not the processor methods in isolation), so they cover
the full seam (decode → processor → producer emit):

- `accept with item` / `accept meso only` — asserts `ItemId`
  nil-vs-non-nil and `ItemSnapshot.Strength`, and exactly one `ACCEPTED`
  event.
- `accept replay` — asserts row count stays 1 after two identical
  deliveries (constrains `AcceptCustody`'s idempotent-create branch;
  without it a naive `Create` would either duplicate-key error or insert
  twice).
- `release` / `release replay` / `release wrong recipient` — asserts status
  transition, `ResolvedAt`, exact RELEASED-event count on replay (see §4),
  and that a recipient mismatch leaves the row `StatusPending` and emits
  exactly one ERROR. These genuinely constrain the code: removing the
  `alreadyReleased`/`ErrNotRecipient` branches would flip these assertions.
- `restore` / `restore on a pending row` — asserts `StatusPending` +
  `ResolvedAt` nil after restore, and 0-error on a no-op restore.
- `remove` / `remove on a received row` — asserts hard-delete
  (`ErrNotFound`) vs. guarded no-op (`StatusReceived` untouched).

These are not happy-path-only; the replay/wrong-recipient/wrong-status
subtests specifically pin the new contract (idempotency + guard behavior),
and none of them would pass against a naive first-draft implementation that
skipped the guard branches. TDD-vs-single-pass is honestly disclosed in the
report; the resulting suite still constrains the code adequately based on
inspection of what each assertion would catch.

**PASS**, modulo the `AssetData` gap in §3 not being covered by any subtest
(expected, since the code doesn't attempt the mapping).

## Other checks

- `errors.Is` used correctly (§4).
- `main.go` wiring (`custodyConsumer.InitConsumers`/`InitHandlers`) mirrors
  MTS's registration exactly; the stale `//nolint:unused` on
  `consumerGroupId` was correctly removed now that it's consumed.
- No new `libs/atlas-constants` duplication found; `world.Id` reused
  throughout.
- No `*_testhelpers.go` file added; test file builds commands via literal
  struct values per the MTS test's own style.

## Findings summary

### Blocking

1. `services/atlas-parcel/atlas.com/parcel/parcel/processor_custody.go:26-66,102-125`
   — `AcceptToParcelCommandBody.ItemLevel` is decoded but never mapped to
   `AssetData.LevelType`, even though that destination field already exists
   and the orchestrator's own compensator code (which the implementer read)
   establishes the exact mapping (`LevelType: p.ItemLevel`). Report's
   rationale ("no home in AssetData") is factually wrong for this field.
2. `services/atlas-parcel/atlas.com/parcel/parcel/asset_data.go:14-47` and
   `processor_custody.go:26-66,102-125` — `RingId`/`ViciousCount` have no
   destination field in `parcel.AssetData` and are silently dropped on
   every equipped-item parcel accept, causing item-property loss (ring
   identity, vicious-hammer count) on any equip mailed through Duey and
   later released to the recipient. `AssetData` is JSONB-backed (no
   migration needed), and MTS's own custody store (`listing.Entity`)
   demonstrates the producible, in-repo pattern for persisting these
   fields. Per this repo's "finish producible work" rule, this should not
   have been left "out of scope."

### Non-blocking

- None beyond what's noted above; the `ErrAlreadyReleased` design deviation
  (§4) is sound and matches the brief.

### Not evaluable

- Whether the released item's `LevelType`/`RingId`/`ViciousCount` gap
  actually manifests as a player-visible bug on the live release path
  (i.e., whether a downstream `accept_to_character`/inventory-grant step
  reads atlas-parcel's stored `ItemSnapshot` at all, vs. relying solely on
  the orchestrator's own cached step payload) was not fully traced — that
  downstream consumer is outside this task's file set and Task 15's diff.
  The compensation path (`RestoreParcel`) reconstructs from the
  orchestrator's own cached `AcceptToParcelPayload`, not from atlas-parcel's
  stored snapshot, so the compensation path is unaffected by this gap; the
  forward "release → grant to recipient character" path was not traced
  because it is out of this task's diff surface. This does not change the
  finding (the data is lost from atlas-parcel's own stored snapshot
  regardless of who reads it), but the precise blast radius on the live
  path is unconfirmed.
