# Fix report: clicking Duey opens nothing — "Parcel" writer never registered

## Summary

Registered `parcelcb.ParcelWriter` in `produceWriters()` in
`services/atlas-channel/atlas.com/channel/main.go`, and added a regression
test in `writers_test.go` guarding against the same silent-registration-gap
class documented for the MTS "Charge" button.

Scope was exactly the two files named in the brief's `## Fix` section. No
other files were touched (seed templates, saga handler, parcel consumer left
untouched per the ruling).

## Changes

### `services/atlas-channel/atlas.com/channel/main.go`

- Added import `parcelcb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound"`,
  placed alphabetically immediately before the existing `parcelsb` (serverbound)
  import.
- Added `parcelcb.ParcelWriter` to the `produceWriters()` slice, appended after
  the last entry (`reportcb.ClaimSvrStatusChangedWriter`), consistent with how
  other single-writer feature groups are appended in this list.

### `services/atlas-channel/atlas.com/channel/writers_test.go`

- Added import `parcelcb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound"`.
- Added `TestProduceWriters_RegistersParcelWriter`, in the shape of the existing
  `TestProduceWriters_RegistersMtsWriters`, asserting `parcelcb.ParcelWriter` is
  present in `produceWriters()`.

## TDD evidence — RED then GREEN

**RED** (before the `main.go` fix — temporarily removed the `parcelcb` import
and the `parcelcb.ParcelWriter,` line, restoring immediately after):

```
$ go test . -run TestProduceWriters_RegistersParcelWriter -v
=== RUN   TestProduceWriters_RegistersParcelWriter
    writers_test.go:48: produceWriters() must register writer [Parcel] or Announce fails with 'writer not found'
--- FAIL: TestProduceWriters_RegistersParcelWriter (0.00s)
FAIL
FAIL	atlas-channel	0.011s
FAIL
```

`go build ./...` still succeeded in this state (the unused-import failure
mode was avoided by removing both the import and its use together), confirming
the test itself is what catches the regression, not a compile error.

**GREEN** (after restoring the fix):

```
$ go test . -run TestProduceWriters -v
=== RUN   TestProduceWriters_RegistersMtsWriters
--- PASS: TestProduceWriters_RegistersMtsWriters (0.00s)
=== RUN   TestProduceWriters_RegistersParcelWriter
--- PASS: TestProduceWriters_RegistersParcelWriter (0.00s)
PASS
ok  	atlas-channel	0.096s
```

## Full module-local verification

```
$ cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Build succeeded with no errors. `go test ./...` produced no failing packages
(all `ok` or `no test files`); output is pristine.

## Files changed

- `services/atlas-channel/atlas.com/channel/main.go`
- `services/atlas-channel/atlas.com/channel/writers_test.go`

## Self-review

- Diff is minimal (2 files, +21/-0 lines total for the two files) and matches
  exactly the brief's `## Fix` inventory.
- Import placement follows existing alphabetical-by-path convention (right
  before the existing `parcelsb` serverbound import for the same package).
- `produceWriters()` entry placement follows the existing pattern of appending
  single-writer feature groups at points in the list; placed at the end,
  consistent with how the most recently added groups (e.g. `reportcb.*`) were
  appended.
- Regression test follows the exact shape/doc-comment convention of
  `TestProduceWriters_RegistersMtsWriters`, adapted for the single Parcel
  writer.
- No test-helper files created; no seed templates, saga handler, or parcel
  consumer touched, per the ruling.
- Confirmed RED before the fix and GREEN after, per the brief's requirement.

## Issues or concerns

None. The two "Not yet answered" items in the brief (per-version opcode
coverage sweep across all task-241 clientbound packets, and live confirmation
with a non-empty mailbox) are explicitly out of scope for this fix per the
brief and the ruling — they are follow-up verification items, not blockers on
this registration fix.
