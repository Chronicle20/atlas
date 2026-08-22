# bug: PARCEL's +29 field is isQuickDelivery, not hasMessage

Correction to `bug-duey-parcel-message-offset.md` (fixed by `b21783437`). That
fix put the message at the right offset (+33) and correctly identified +29 as a
32-bit flag gating it, but named the flag `hasMessage` and drove it from
`p.message != ""`. The flag is actually the parcel's **quick-delivery** bit.

## Reproduced

Same environment and data as the two prior parcel bugs: ephemeral env
`atlas-pr-1434`, GMS v83 client, Duey NPC 9010009 → Receive tab; three pending
parcels from `Atlas` with messages `""`, `"lulnub"`, `"lulul"`.

## Observed (client, after `b21783437`)

Notes now render correctly — rows 2 and 3 show their full messages. But rows 2
and 3 also show mojibake ASCII to the right of the sender name; row 1 (no
message) does not.

## Expected

The mojibake column reflects whether the parcel was sent by Quick Delivery,
independent of whether it carries a note.

## Root cause — established, IDA-confirmed

GMS v83 IDB (session `754107bf`, `MapleStory_dump.exe.i64`),
`CTabReceive::Draw` @0x6EFA1F.

The per-row loop draws three columns:

- x=13 — `_bstr_t::_bstr_t(&v150, parcel + 4)` @0x6EFED5, the sender name.
  Our encoder writes it zero-padded to 13 bytes and it is NOT the defect; the
  real wire bytes are `41 74 6c 61 73 00 00 00 00 00 00 00 00`.
- x=113 — gated on `if ( *(parcel + 29) )` @0x6EFF31, then
  `push offset dword_AFCDB0; call ??0_bstr_t@@QAE@PBD@Z` @0x6EFF78. This is a
  **static string constant**, not parcel data. `dword_AFCDB0` holds
  `C4 FC B9 E8 BC DB 00` = EUC-KR **`퀵배송`** ("Quick Delivery"), which a
  cp1252 client renders as `Äü¹è¼Û` — the observed mojibake.
- x=188 — the +21 expiry countdown.

So the field at +29 (`cmp [eax+1Dh], edi` @0x6F07AB — a 32-bit compare) is the
parcel's quick-delivery bit. The detail pane's note read at +33 (@0x6F0801) is
gated on the SAME flag, and the `else` arm @0x6F0816 clears the note control
outright — i.e. the client only displays a note for a quick-delivery parcel,
which matches the game: the note is a Quick Delivery feature.

`b21783437` set the flag from message-presence, so any parcel carrying a note
falsely renders the Quick Delivery label. The correct source is already modeled
end-to-end and simply never reaches the wire:
`atlas-parcel` `parcel/entity.go:73` `Quick bool` → `parcel/model.go:87`
`Model.Quick()` → `parcel/rest.go:37` `json:"quick"` → `atlas-channel`
`parcel/model.go:31` `quick` / `:53` `Model.Quick()`.
`atlas-channel`'s `Model.ToPacket()` never reads it.

**Accepted client behaviour, not to be worked around:** the marker string is
hardcoded EUC-KR in this binary, so a genuine quick-delivery parcel will always
render mojibake there on a non-Korean client. That is intended (ruling from the
task owner, this session) — do not attempt to suppress the flag to hide it.

## Fix

- `libs/atlas-packet/parcel/parcel.go`
  - Rename the +29 concept from `hasMessage` to quick-delivery throughout:
    add a `quick bool` field to `Parcel`, a `Quick() bool` accessor, and an
    immutable `SetQuick(bool) Parcel` setter in the existing setter style
    (`SetItem` is the model to follow).
  - `Encode`: write `uint32(1)` when `p.quick` and `uint32(0)` otherwise at
    +29 — driven by the flag, NOT by `p.message`.
  - Update the `+29` / `+33` field docs on the `Parcel` struct comment: cite
    `dword_AFCDB0` = EUC-KR `퀵배송` and the gated draw @0x6EFF31/@0x6EFF78 as
    the evidence that the field is quick-delivery, keep the 32-bit compare
    citation @0x6F07AB, and record that the note at +33 is gated on this same
    flag (`else` arm @0x6F0816 clears the note control) so a non-quick parcel
    can never display a note client-side. Correct the "set iff the message is
    non-empty" wording introduced by `b21783437`.
  - Leave the client's empty-note fallback note in place (flag set + empty
    +33 → `strcpy` from StringPool 3886, @0x6F07C1).
- `services/atlas-channel/atlas.com/channel/parcel/model.go` — `ToPacket()`
  must carry `m.Quick()` onto the wire parcel (`.SetQuick(m.Quick())` on the
  `packetparcel.NewParcel(...)` result, before the item branch, so both the
  item and no-item return paths keep it).
- `services/atlas-channel/atlas.com/channel/parcel/model_test.go` — extend the
  existing `TestToPacket*` coverage with a case pinning that `Quick()` is
  projected onto the wire parcel in both the item and no-item paths.
- `libs/atlas-packet/parcel/clientbound/v72_test.go`, `v79_test.go`,
  `v83_test.go`, `v84_test.go`, `v87_test.go`, `v92_test.go`, `v95_test.go`,
  `v185_test.go`, and `libs/atlas-packet/parcel/clientbound/parcel_test.go` —
  every fixture that `b21783437` taught to derive the 4-byte flag from its
  message must derive it from the fixture parcel's `quick` value instead.
  Keep at least one fixture exercising quick=true with a message and one
  exercising quick=false with a message, so the two are not conflated again.
- `docs/packets/audits/STATUS.md` and `docs/packets/audits/status.json` —
  regenerate with `go run ./tools/packet-audit matrix` AFTER the fixture edits,
  and commit.

Verification: module-local `go build ./... && go test ./...` in
`libs/atlas-packet` and `services/atlas-channel`, plus
`go run ./tools/packet-audit matrix --check` exiting 0.

## Not yet answered

- **Does the quick bit survive the send path?** `atlas-parcel`
  `parcel/task.go:194` calls `SetQuick(false)` and
  `parcel/processor_custody.go:98` sets it from `params.Quick`. Whether the
  Duey quick-send flow actually populates `params.Quick` is NOT verified here.
  If, after this fix, a parcel sent with a Quick Delivery ticket still shows no
  marker and no note, that is a separate defect on the send path, not in the
  codec.
- **`itemType` is always 0** — still out of scope, carried from
  `bug-duey-receive-list-item-slot-desync.md`.

## Outcome

_(to be filled in by the fix)_
