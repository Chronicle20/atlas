# Report: ItemType mapping test coverage (saga-orchestrator → atlas-parcel)

## What I implemented

Closed the two untested mapping hops identified in
`docs/tasks/task-241-duey-parcel-delivery/reviews/review-bug-duey-receive-item-type.md`:
a one-line mapping regression at either hop would previously go undetected.

1. `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/processor_test.go`
   — `TestParcelProcessorDispatch/accept`: added `ItemType: 2` to the
   `AcceptToParcelParams` input, and asserted `cmd.Body.ItemType != 2` fails
   the test if `parcel/producer.go:36`'s `ItemType: params.ItemType` mapping
   regresses.

2. `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer_test.go`
   — `newAcceptCommand` (shared builder used by `TestCustodyCommands`'s
   table rows) now sets `ItemType: 2` on the command body. The "accept with
   item" case asserts `m.ItemType() == byte(2)` on the row returned by
   `GetById`, covering `consumer.go:103`'s `ItemType: b.ItemType` mapping
   into `parcel.AcceptParams`.

No non-test file was changed.

## Testing

```
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./...
```
Result: all packages `ok` (or `[no test files]`); `atlas-saga-orchestrator/parcel` passed
(`ok  	atlas-saga-orchestrator/parcel	0.012s`).

```
cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...
```
Result: `ok  	atlas-parcel/kafka/consumer/custody	0.040s`, `ok  	atlas-parcel/parcel	(cached)`,
others `[no test files]`. No failures.

Both new assertions were confirmed to exercise real code (not dead
assertions): before adding `ItemType` to the `AcceptToParcelParams` struct
literal / command builder, the field defaulted to `byte(0)`, so the new
`!= 2` / `== byte(2)` checks are meaningful given the non-zero value chosen.

## Files changed

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/processor_test.go`
- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer_test.go`

## Self-review

- Scope: only the two named test files touched; no production code changed.
- The `ItemType: 2` value added to `newAcceptCommand` is shared by every
  `TestCustodyCommands` sub-test (accept meso only, accept replay, release,
  etc.) — verified none of the other sub-tests assert `ItemType == 0` or
  otherwise depend on it being zero; only "accept with item" now asserts on
  it, and the rest are unaffected (all still pass).
- Assertion style matches existing conventions in each file (`t.Fatalf` in
  the saga-orchestrator table test, `assert.Equal` with a message in the
  parcel consumer test, mirroring the existing `LevelType`/`RingId`/
  `ViciousCount` assertions in the same block).

## Issues or concerns

None. Both module-local `go build ./... && go test ./...` runs are clean.

## Commit

`c3af819b2` — `test(duey): cover ItemType mapping through parcel accept path`
