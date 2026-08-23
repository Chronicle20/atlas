# Fix report: bug-duey-parcel-quick-flag-mislabeled

## What was implemented

Renamed the PARCEL wire struct's `+29` concept from `hasMessage` (driven by
`p.message != ""`) to `quick` (the parcel's quick-delivery bit), matching the
IDA-confirmed root cause in the bug file: `CTabReceive::Draw`'s row rendering
(@0x6EFF31/@0x6EFF78) draws a static EUC-KR "퀵배송" (Quick Delivery) label
gated on `*(parcel+29)`, and the detail-pane note read at `+33` is gated on
the SAME flag.

- `libs/atlas-packet/parcel/parcel.go`
  - Added `quick bool` field to `Parcel`, a `Quick() bool` accessor, and an
    immutable `SetQuick(bool) Parcel` setter following the `SetItem` style
    (returns a copy, no mutation of the receiver's storage).
  - `Encode` now writes `uint32(1)`/`uint32(0)` at `+29` from `p.quick`,
    never from `p.message`.
  - Rewrote the `+29` field doc: cites `dword_AFCDB0` = EUC-KR `퀵배송`, the
    gated draw call sites `@0x6EFF31`/`@0x6EFF78`, keeps the `@0x6F07AB`
    32-bit compare citation, and records that the note at `+33` is gated on
    this same flag (`else` arm `@0x6F0816` clears the note control) — i.e. a
    non-quick parcel can never display a note client-side. Also records the
    accepted-mojibake ruling inline so a future reader doesn't reopen it.
  - Left the client's empty-note fallback (StringPool 3886) noted, unchanged
    behaviourally (no code for it lives here — it's client-side).

- `services/atlas-channel/atlas.com/channel/parcel/model.go`
  - `ToPacket()` now calls `p.SetQuick(m.Quick())` on the `packetparcel.
    NewParcel(...)` result before the item branch, so both the item and
    no-item return paths carry the flag.

- `services/atlas-channel/atlas.com/channel/parcel/model_test.go`
  - Added `TestToPacketQuick`, covering all four combinations
    (quick × message-present) across both the no-item and item-attached
    `ToPacket()` paths, pinning that `Quick()` is projected independent of
    `Message()`.

- `libs/atlas-packet/parcel/clientbound/parcel_test.go` and the eight
  version fixture files (`v72_test.go`, `v79_test.go`, `v83_test.go`,
  `v84_test.go`, `v87_test.go`, `v92_test.go`, `v95_test.go`,
  `v185_test.go`) — every fixture that previously derived the `+29` 4-byte
  flag from `message != ""` now derives it from the fixture `Parcel`'s
  `quick` value:
  - `parcel_test.go`'s `TestParcelEncode` t.Run("parcel no item") now builds
    a message-carrying, `quick=false` fixture (flag bytes `00 00 00 00`);
    t.Run("parcel with item") now calls `.SetQuick(true)` on a
    message-carrying fixture (flag bytes `01 00 00 00`) — both fixtures keep
    the message non-empty so the two concepts (quick vs. has-message) can
    never silently re-conflate.
  - Each version file's `TestParcel*V<ver>WithItem` (the "Arrived" variant)
    now calls `.SetQuick(true)` and expects `01 00 00 00`, with a comment
    `quick flag LE (quick=true, message non-empty)`.
  - Each version file's `TestParcelOpen*V<ver>WithItem` variant is left with
    no `SetQuick` call (default `quick=false`) and now expects
    `00 00 00 00`, with a comment `quick flag LE (quick=false, message
    non-empty)`.

- `docs/packets/audits/STATUS.md` / `docs/packets/audits/status.json` — ran
  `go run ./tools/packet-audit matrix` after the fixture edits. No diff was
  produced: the `packet-audit:verify` markers and their addressed packets
  did not change (only fixture byte literals and internal struct fields
  changed), so the matrix was already current. Confirmed with
  `go run ./tools/packet-audit matrix --check`, exit 0.

## What was tested

```
cd libs/atlas-packet && go build ./... && go test ./...
```
All packages pass, including `parcel/clientbound` (was previously red on
the `hasMessage`-derived fixtures before the edits below fixed them).

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
All packages pass (no `FAIL`/`panic` in output), including the new
`TestToPacketQuick` in `parcel`.

```
go run ./tools/packet-audit matrix --check
```
Exit 0, no diff to `STATUS.md`/`status.json`.

## Files changed

- `libs/atlas-packet/parcel/parcel.go`
- `services/atlas-channel/atlas.com/channel/parcel/model.go`
- `services/atlas-channel/atlas.com/channel/parcel/model_test.go`
- `libs/atlas-packet/parcel/clientbound/parcel_test.go`
- `libs/atlas-packet/parcel/clientbound/v72_test.go`
- `libs/atlas-packet/parcel/clientbound/v79_test.go`
- `libs/atlas-packet/parcel/clientbound/v83_test.go`
- `libs/atlas-packet/parcel/clientbound/v84_test.go`
- `libs/atlas-packet/parcel/clientbound/v87_test.go`
- `libs/atlas-packet/parcel/clientbound/v92_test.go`
- `libs/atlas-packet/parcel/clientbound/v95_test.go`
- `libs/atlas-packet/parcel/clientbound/v185_test.go`

## Self-review

- Confirmed no remaining `hasMessage` references, in code or comments, in
  `libs/atlas-packet/parcel/` (`grep -rn hasMessage libs/atlas-packet/parcel`
  returns nothing after the fix).
- Confirmed `ToPacket()` carries `Quick()` on both return paths (the
  no-item early return and the item-attached path), since `SetQuick` is
  called before the `if m.ItemId() == nil` branch.
- Confirmed the fixture edits keep at least one quick=true-with-message and
  one quick=false-with-message case (both in `parcel_test.go` and, per the
  bug's per-file instruction, split across each version file's two
  `WithItem` tests), so the flag can't silently re-derive from message
  presence again without a test catching it.
- Did not touch `atlas-parcel`'s send path (`task.go:194`,
  `processor_custody.go:98`) — out of scope per the bug file's "Not yet
  answered" section; that's a separate, unverified concern about whether
  `params.Quick` is actually populated on the Duey quick-send flow.
- Did not attempt to suppress or special-case the EUC-KR mojibake — per the
  ruling, it is accepted client behaviour for a genuine quick-delivery
  parcel.

## Issues or concerns

None. The fix is a straightforward field-rename plus a one-line wiring
correction in `ToPacket()`.

## Outcome

Fixed. The `+29` wire field on `libs/atlas-packet/parcel.Parcel` is now
driven exclusively by the parcel's quick-delivery flag, and
`atlas-channel`'s `Model.ToPacket()` projects `Model.Quick()` onto it. All
module-local builds and tests pass; the packet-audit matrix required no
regeneration.
