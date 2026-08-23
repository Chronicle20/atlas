# Fix report — DUEY_ACTION RECEIVE rejects any item the recipient already holds

- **Task**: task-241-duey-parcel-delivery
- **Brief**: `docs/tasks/task-241-duey-parcel-delivery/bug-duey-receive-unique-conflict-on-stackables.md`
- **Branch**: task-241-duey-parcel-delivery
- **Commit**: `5dabf1e31` — `fix(duey): gate receive unique-conflict rejection on WZ info/only`

## What was implemented

Followed the brief's `## Fix` inventory exactly, extending the existing
`atlas-channel/data/tradeability` package rather than adding a sibling.

### atlas-data (WZ `info/only` reader)

- `equipment/rest.go` / `equipment/reader.go` — added `Only bool
  \`json:"only"\`` to `RestModel`, set from `info.GetBool("only", false)`
  alongside `TradeBlock`.
- `setup/rest.go` / `setup/reader.go` — same.
- `etc/rest.go` / `etc/reader.go` — same.
- `cash/rest.go` / `cash/reader.go` — same.
- `consumable/rest.go` / `consumable/reader.go` — already carried `Only`;
  untouched.

### atlas-channel (`data/tradeability`)

- `data/tradeability/rest.go`:
  - Widened the package doc from "the two WZ questions the karma gates
    ask" to the WZ item properties atlas-channel gates on (tradeBlock,
    tradeAvailable, only).
  - Added `Only bool \`json:"only"\`` to all five `*RestModel`s.
  - Added `only bool` to `Model` with an `Only() bool` accessor.
  - Extended `NewModel` to `NewModel(tradeBlock bool, tradeAvailable
    int32, only bool) Model` — updated its one non-test call site's
    signature is unchanged (no other production caller), and the five
    call sites in `character_cash_item_use_karma_test.go` that construct
    fixtures via `tradeability.NewModel(...)` were updated to pass
    `false` for the new `only` argument (none of those karma fixtures
    needed a one-of-a-kind item).
  - Extended every `fields()` method and the shared generic `extract` to
    carry the third value through.
- `socket/handler/duey_action_receive.go`:
  - Added `getItemOnly func(it inventory.Type, templateId item.Id)
    (bool, error)` to `dueyReceiveDeps`, documented with the same
    "error means refuse" contract as `tradeability`'s own doc.
  - Wired the real implementation in `handleDueyActionReceive` to
    `tradeability.NewProcessor(l, ctx).Get(it, templateId).Only()`.
  - In `receiveParcel`, the `FindFirstByItemId` rejection now only fires
    `RECV_UNIQUE_CONFLICT` when `getItemOnly` reports `true`. A
    `getItemOnly` error rejects with `ParcelIncorrectRequestBody()` and
    logs at error level — no fallthrough to "assume not unique" and no
    fallback to the old template-equality behavior.

## Not touched (per brief's explicit scope)

- The free-slot check (`duey_action_receive.go:162`,
  `RECV_NO_FREE_SLOTS`) — left exactly as it was; brief's `## Not yet
  answered` calls this out explicitly as a separate defect.

## Tests

### `duey_action_receive_test.go` (table-driven, existing `TestDueyActionReceive`)

Extended the existing case table:

- `"unique conflict"` — now sets `itemOnly: true` (previously implicit);
  proves `only == true` still announces `RECV_UNIQUE_CONFLICT`.
- `"stackable already held merges, no rejection"` (new) — `compDup:
  true`, `itemOnly: false`; proves the reported regression is fixed: no
  rejection is announced and the parcel-receive saga is built (`sagaLen:
  1`).
- `"getItemOnly error"` (new) — `compDup: true`, `itemOnlyErr` set;
  proves a lookup failure announces `INCORRECT_REQUEST` and starts no
  saga (`sagaLen` defaults to 0).

All other existing cases (happy path, meso-only, no-free-slot, not
receivable, not addressed to me, already received, getParcel error) are
unchanged and still pass.

### `data/tradeability/processor_test.go`

- `TestExtractCarriesBothFields` — extended to also assert `Only()`.
- `TestByIdProvider_AllCompartments` — extended fixture response to
  include `"only":true` and asserts `m.Only()` for all five compartments.
- `TestByIdProvider_OnlyFalse` (new) — asserts `Only() == false` for the
  consumable and equipment compartments when the wire field is `false`
  (brief requires "at least the consumable and one non-consumable
  resource").

### atlas-data reader tests

Extended the existing `TradeBlock` coverage in each of the four readers
with an `Only`-surfaces test and an `Only`-defaults-false test, using the
same WZ fixture XML already in each `reader_test.go` (adding an `only`
node to one existing fixture item per package rather than inventing a new
fixture):

- `equipment/reader_test.go` — `TestEquipmentReaderSurfacesOnly` /
  `TestEquipmentReaderOnlyDefaultsFalse` (the existing `testXML` fixture
  already carried `only=1`; `stringTypedReqLUKXML` has none).
- `setup/reader_test.go` — `TestSetupReaderSurfacesOnly` /
  `TestSetupReaderOnlyDefaultsFalse` (added `<int name="only"
  value="1"/>` to the first fixture item).
- `etc/reader_test.go` — `TestEtcReaderSurfacesOnly` /
  `TestEtcReaderOnlyDefaultsFalse` (added `<int name="only" value="1"/>`
  to the third fixture item).
- `cash/reader_test.go` — `TestCashReaderSurfacesOnly` /
  `TestCashReaderOnlyDefaultsFalse` (added `<int name="only" value="1"/>`
  to the `05240000` fixture item).

## Commands run and output

```
cd services/atlas-data/atlas.com/data && go build ./... && go test ./...
```
All packages `ok`, no failures (`go test ./... | grep -v '^ok\|no test files'` → empty).

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
All packages `ok`, no failures (`go test ./... | grep -v '^ok\|no test files'` → empty).

```
gofmt -l ./services/atlas-data/atlas.com/data ./services/atlas-channel/atlas.com/channel
```
No output — no formatting drift.

## Files changed

- `services/atlas-data/atlas.com/data/equipment/rest.go`
- `services/atlas-data/atlas.com/data/equipment/reader.go`
- `services/atlas-data/atlas.com/data/equipment/reader_test.go`
- `services/atlas-data/atlas.com/data/setup/rest.go`
- `services/atlas-data/atlas.com/data/setup/reader.go`
- `services/atlas-data/atlas.com/data/setup/reader_test.go`
- `services/atlas-data/atlas.com/data/etc/rest.go`
- `services/atlas-data/atlas.com/data/etc/reader.go`
- `services/atlas-data/atlas.com/data/etc/reader_test.go`
- `services/atlas-data/atlas.com/data/cash/rest.go`
- `services/atlas-data/atlas.com/data/cash/reader.go`
- `services/atlas-data/atlas.com/data/cash/reader_test.go`
- `services/atlas-channel/atlas.com/channel/data/tradeability/rest.go`
- `services/atlas-channel/atlas.com/channel/data/tradeability/processor_test.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive_test.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_karma_test.go`
  (mechanical `NewModel(...)` call-site update for the new `only` argument
  — no behavior change, all five sites pass `false`)

## Self-review findings

- Confirmed `consumable` was already the one reader carrying `Only`; did
  not duplicate work there.
- Confirmed `tradeability.NewModel`'s only other caller
  (`character_cash_item_use_karma_test.go`) was mechanically updated —
  `go build ./...` would have caught a miss.
- Confirmed the karma gate (`character_cash_item_use.go`) does not read
  `Only()` and needed no change — the brief's scope is the DUEY receive
  path only.
- Did not touch the free-slot check, per explicit brief instruction; a
  sibling bug doc for it already exists in the task folder
  (`bug-duey-receive-free-slot-ignores-stack-merge.md`, not created by
  this task).
- `gofmt -l` clean on both modules; no stray formatting drift from the
  editor/formatter hook.

## Issues or concerns

None. Both module-local `go build ./... && go test ./...` runs are clean
with no flags and no skipped checks.
