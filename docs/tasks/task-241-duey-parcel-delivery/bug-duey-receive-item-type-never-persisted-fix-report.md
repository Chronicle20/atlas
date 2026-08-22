# Fix report — DUEY_ACTION RECEIVE rejects every item parcel: `itemType` never persisted

## What I implemented

Implemented the `## Fix` table exactly as specified: threaded `ItemType byte`
(the source inventory type at send time) from `TransferToParcelPayload.SourceInventoryType`
through every hop to the persisted `parcels.item_type` column.

Files changed (all match the brief's `### Files`/fix table row-for-row, plus one
transitive wire-contract mirror the brief's file list implied but did not name
explicitly):

- `libs/atlas-saga/payloads.go` — added `ItemType byte \`json:"itemType"\`` to
  `AcceptToParcelPayload`'s item-snapshot block, next to `TemplateId`.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` —
  `expandTransferToParcel`'s `HasItem: true` branch now sets
  `ItemType: byte(payload.SourceInventoryType)`. The meso-only branch is
  untouched (stays zero-valued).
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` —
  `handleAcceptToParcel` maps `payload.ItemType` into `parcel.AcceptToParcelParams`.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/processor.go` —
  added `ItemType byte` to `AcceptToParcelParams`.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/producer.go` —
  `AcceptToParcelProvider` copies `params.ItemType` into the command body.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/parcel/custody/kafka.go` —
  added `ItemType byte \`json:"itemType"\`` to the orchestrator's own mirror of
  `AcceptToParcelCommandBody`. This file is not named in the brief's table but
  its doc comment says it "mirrors atlas-parcel's kafka/message/custody/kafka.go
  byte-for-byte" and is the type `producer.go` actually serializes — omitting it
  would silently drop the field on the wire while still compiling (the two
  `AcceptToParcelCommandBody` types are structurally distinct, package-local
  types). I treated this as part of "the command body" the brief's producer.go
  and consumer.go rows both reference.
- `services/atlas-parcel/atlas.com/parcel/kafka/message/custody/kafka.go` —
  added `ItemType byte \`json:"itemType"\`` to `AcceptToParcelCommandBody`
  (atlas-parcel's own copy, the one the consumer decodes).
- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer.go` —
  `handleAcceptToParcel` maps `b.ItemType` into `parcel.AcceptParams`.
- `services/atlas-parcel/atlas.com/parcel/parcel/processor_custody.go` — added
  `ItemType byte` to `AcceptParams`; `AcceptCustody`'s `params.HasItem` branch
  now calls `b.SetItemType(params.ItemType)` (pre-existing `Builder.SetItemType`,
  no new setter needed).

No DB migration — `parcels.item_type` already exists and is auto-migrated, as
the brief states.

No derive-from-template-id fallback was added anywhere, and no backfill was
written, per the ruling.

## Tests added

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/parcel_expansion_test.go` —
  extended `TestExpandTransferToParcel`:
  - `"item parcel"` subtest: `require.Equal(t, byte(1), acc.ItemType)` (payload's
    `SourceInventoryType: 1`).
  - `"meso only"` subtest: `require.Zero(t, acc.ItemType)`.
- `services/atlas-parcel/atlas.com/parcel/parcel/processor_custody_test.go` (new
  file — no `processor_custody_test.go`/custody test existed yet, so I created
  it alongside `processor_test.go`, reusing its Builder-pattern helpers
  `newProcessorTestDB`/`newTestProcessor`/`fixedClock`; added
  `newTestProcessorImpl` to get at `*ProcessorImpl.AcceptCustody`, which is
  additive to the `Processor` interface per the existing `processor()` helper
  comment in `kafka/consumer/custody/consumer.go`):
  - `TestProcessorAcceptCustody/item parcel persists ItemType` — `AcceptCustody`
    with `HasItem: true, ItemType: 2` persists `ItemType() == 2` both on the
    returned model and on a fresh `GetById` re-read.
  - `TestProcessorAcceptCustody/meso only leaves ItemType zero` — a
    `HasItem: false` accept leaves `ItemType()` zero.
- The third brief-listed test ("saga-orchestrator handler-level assertion that
  `AcceptToParcelParams.ItemType` reaches the producer body, if an existing
  test covers that mapping") — no existing test covers `handleAcceptToParcel`
  or `AcceptToParcelProvider`'s field mapping (`grep` for both across
  `saga/*_test.go` and `parcel/*_test.go` returned nothing), so per the
  brief's conditional wording this was not added.

## Test results

```
cd libs/atlas-saga && go build ./... && go test ./...
ok  	github.com/Chronicle20/atlas/libs/atlas-saga	0.006s

cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./...
ok  	atlas-saga-orchestrator/saga	0.483s
ok  	atlas-saga-orchestrator/saga/mock	0.007s
(...all other packages ok or [no test files], no failures)

cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...
ok  	atlas-parcel/kafka/consumer/custody	0.025s
ok  	atlas-parcel/parcel	(cached)
(...all other packages ok or [no test files], no failures)
```

Output pristine, no failures, no skips.

## Files changed

- `libs/atlas-saga/payloads.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/parcel_expansion_test.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/processor.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/producer.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/parcel/custody/kafka.go`
- `services/atlas-parcel/atlas.com/parcel/kafka/message/custody/kafka.go`
- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer.go`
- `services/atlas-parcel/atlas.com/parcel/parcel/processor_custody.go`
- `services/atlas-parcel/atlas.com/parcel/parcel/processor_custody_test.go` (new)
- `docs/tasks/task-241-duey-parcel-delivery/bug-duey-receive-item-type-never-persisted.md` (added to git — was untracked from a prior phase)

Deliberately left untouched (pre-existing, unrelated to this task, present
before I started): `.idea/inspectionProfiles/Project_Default.xml`,
`go.work.sum`, `.idea/go.imports.xml`, `.idea/golinter.xml`,
`.idea/task-241-duey-parcel-delivery.iml`.

## Self-review

- Every row of the brief's fix table is implemented, field name (`ItemType byte`,
  `json:"itemType"`) matches exactly what was specified.
- The meso-only branch in `expandTransferToParcel` is untouched, so it keeps
  emitting `ItemType` zero-valued as required.
- No derive-from-template-id fallback and no backfill were added anywhere, per
  the ruling.
- Followed the existing Builder pattern for the new test file; no
  `*_testhelpers.go` file created.
- No `// TODO`, stub, or placeholder introduced.
- The one file not literally named in the brief's table
  (`saga-orchestrator/kafka/message/parcel/custody/kafka.go`) was necessary for
  correctness — it's the type actually marshaled onto the wire by `producer.go`
  and unmarshaled by the orchestrator side; skipping it would have made the
  field compile but silently never reach Kafka. Flagged above for visibility.

## Issues or concerns

None. Module-local build and tests are clean across all three affected
modules (`libs/atlas-saga`, `atlas-saga-orchestrator`, `atlas-parcel`).
