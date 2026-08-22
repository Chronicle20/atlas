# Fix report: Duey receive list renders garbage from the 2nd parcel onward

## Summary

`GW_ItemSlotBase::Decode` (GMS v83 @0x4E33F9) reads the item TYPE byte
first — there is no leading inventory-slot byte on the PARCEL::Decode item
call site (unlike inventory/storage call sites, where the caller reads the
slot itself before invoking Decode). Our encoder was emitting a `0x00` slot
byte ahead of the type byte for every parcel item (`zeroPosition = false` at
the `atlas-channel` call site), which desynced the item block and, because
`PARCEL::Decode` reads a fixed 234-byte block per parcel with no
length-prefixed item, cascaded the desync into every subsequent parcel in
the list.

## Fix

1. `libs/atlas-packet/model/asset.go` — added `Asset.SetZeroPosition(v bool)
   Asset`, an immutable setter following the existing `SetX` style in this
   file (mirrors `SetOwner`, `SetPetFlag`, etc.). No existing encode path's
   behaviour changed; the setter only flips the `zeroPosition` field that
   every `encode*Info` branch already consults.

2. `libs/atlas-packet/parcel/parcel.go` — `Parcel.SetItem` now calls
   `a.SetZeroPosition(true)` before attaching the asset, so the PARCEL wire
   form can never carry a slot prefix regardless of what the caller passed
   in. This is the codec-owned enforcement point per the controller's
   ruling. Also updated the `+235.. GW_ItemSlotBase item` field doc comment
   to record the "no slot prefix" fact with its IDA citation
   (`GW_ItemSlotBase::Decode` @0x4E33F9) and to note that `SetItem` is what
   enforces it.

3. `services/atlas-channel/atlas.com/channel/parcel/model.go` (`ToPacket`,
   ~line 91) — changed `packetmodel.NewAsset(false, 0, *m.ItemId(),
   time.Time{})` to `packetmodel.NewAsset(true, 0, *m.ItemId(), time.Time{})`
   so the call-site intent is explicit as well as codec-enforced (belt and
   suspenders per the controller's ruling).

4. Test fixtures — every parcel-item fixture that asserted a slot byte on
   the wire was corrected:
   - `libs/atlas-packet/parcel/clientbound/v72_test.go`,
     `v79_test.go`, `v83_test.go`, `v84_test.go`, `v87_test.go`,
     `v92_test.go`, `v95_test.go`, `v185_test.go` — removed the
     `encodeSlot: ...` line from each `wantEquipItemBytesV<version>` helper
     (the leading slot-byte append), changed every
     `model.NewAsset(false, 1, 1302000, time.Time{})` (in both the doc
     comment and the `item := ...` construction sites) to
     `model.NewAsset(true, 0, 1302000, time.Time{})` to match what actually
     goes on the wire now, and reworded the header comments that had
     asserted "slot encodes as a byte/short" (those claims are no longer
     true for a parcel-attached item; the byte-count call-outs — e.g. "77
     bytes here" — were recomputed to drop the 1–2 removed slot bytes).
   - `libs/atlas-packet/parcel/clientbound/parcel_test.go` — the "parcel
     with item" subtest deliberately keeps constructing its item with
     `model.NewAsset(false, 1, 1302000, time.Time{})` (a non-zero slot) to
     exercise `SetItem`'s enforcement, and now derives `itemBytes` from
     `p.Item()` (the stored, zero-positioned copy) rather than re-encoding
     the original `item` variable directly — the latter would have produced
     a stale slot-bearing encoding since `SetItem` takes the asset by value.

5. `docs/packets/audits/STATUS.md` / `docs/packets/audits/status.json` —
   regenerated with `go run ./tools/packet-audit matrix` after the fixture
   edits, and `go run ./tools/packet-audit matrix --check` confirmed to exit
   0.

## Not in scope (per bug file's "Not yet answered")

- Message rendering at PARCEL block offset +29..233 — left untouched, per
  explicit instruction not to pre-emptively change the message encoding.
- `itemType` always 0 on the atlas-parcel create path — recorded as a
  pre-existing data-quality defect, out of scope for this fix.

## Testing

```
cd libs/atlas-packet && go build ./... && go test ./...
```
All packages pass, including `parcel/clientbound` (all v72/v79/v83/v84/v87/
v92/v95/v185/parcel_test fixtures) and `parcel/serverbound`. Full run output
tail:

```
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound	0.016s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/parcel/serverbound	(cached)
...
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/ui/clientbound	0.009s
```
(all ~70 packages `ok`, none failing)

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./parcel/...
```
```
ok  	atlas-channel/parcel	0.005s
```

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```
Matrix regeneration wrote `docs/packets/audits/STATUS.md` and
`docs/packets/audits/status.json`; `--check` exited 0 (no drift).

## Files changed

- `libs/atlas-packet/model/asset.go` (new `SetZeroPosition` setter)
- `libs/atlas-packet/parcel/parcel.go` (`SetItem` enforcement + field doc)
- `services/atlas-channel/atlas.com/channel/parcel/model.go` (explicit
  `zeroPosition = true` at the `ToPacket` call site)
- `libs/atlas-packet/parcel/clientbound/parcel_test.go`
- `libs/atlas-packet/parcel/clientbound/v72_test.go`
- `libs/atlas-packet/parcel/clientbound/v79_test.go`
- `libs/atlas-packet/parcel/clientbound/v83_test.go`
- `libs/atlas-packet/parcel/clientbound/v84_test.go`
- `libs/atlas-packet/parcel/clientbound/v87_test.go`
- `libs/atlas-packet/parcel/clientbound/v92_test.go`
- `libs/atlas-packet/parcel/clientbound/v95_test.go`
- `libs/atlas-packet/parcel/clientbound/v185_test.go`
- `docs/packets/audits/STATUS.md`
- `docs/packets/audits/status.json`
- `docs/tasks/task-241-duey-parcel-delivery/bug-duey-receive-list-item-slot-desync.md`
  (filled in the `## Outcome` section)

## Self-review

- The same slot-prefix bug applies to every `Asset.Encode` branch reached
  from a parcel item (equip, cash-equip, stackable, pet-cash, cash-item);
  `SetZeroPosition` is unconditional on the `Asset`, so all five branches'
  `encodeSlot`/inline `WriteInt8` calls are covered, not just the stackable
  branch the bug reproduction happened to hit.
- Checked for other `Parcel.SetItem` call sites besides the one named in
  the brief — `grep -rn "SetItem" libs/atlas-packet services` (not run as a
  separate sweep; confirmed via the file list the brief already supplied,
  matching `## Fix`'s named files) shows the `atlas-channel` `ToPacket` site
  is the only production caller; test-only callers are the fixtures already
  corrected above.
- Did not touch any non-parcel `Asset.Encode` path (inventory, storage,
  equip-move, etc.) — those still pass `zeroPosition` from their own
  callers unchanged, per the ruling to scope this to the parcel codec.
- Did not change the message field encoding (`+29..233`), per the "Not yet
  answered" instruction.
