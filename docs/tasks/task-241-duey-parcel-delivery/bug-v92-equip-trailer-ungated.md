# bug: GMS v92+ equip trailer is unimplemented and ungated

**Found by:** controller, closing batch 6's central IDB claim by hand (RULING 23).
**Severity:** blocking. `PARCEL`/`DUEY_ACTION` × `gms_v92` are marked `verified`
in `9d790633d` on fixtures that assert the wrong wire bytes.

## The claim that was wrong

Batch 6 (`9d790633d`) reported that v92's `GW_ItemSlotEquip::RawDecode`@`0x4f35d0`
has "the same three-`Decode4` shape as v84/v87", concluded v92's item encoding is
"byte-identical to v84/v87", and wrote its RULING 22 item-bearing fixtures
asserting v84's bytes.

The three-`Decode4` part is true. The conclusion drawn from it is not: the check
looked at the `Decode4` run and stopped, and the divergence is *after* that run.

## What v92 actually reads

Decompiled `0x4f35d0` on session `019cd393`. After the three `Decode4` reads
(`nEXP_CS` +233, `nDurability_CS` +245, `nIUC_CS` +257), v92 reads **seven more
fields that v84/v87 do not read at all**:

| order | read | field offset |
|---|---|---|
| 1 | `Decode1` | +263 |
| 2 | `Decode1` | +269 |
| 3 | `Decode2` | +277 |
| 4 | `Decode2` | +285 |
| 5 | `Decode2` | +293 |
| 6 | `Decode2` | +301 |
| 7 | `Decode2` | +309 |

…and only then the conditional `DecodeBuffer(+40, 8)`, `DecodeBuffer(+61, 8)`,
`Decode4(+69)` that v84/v87 reach immediately.

**That is 2 bytes + 5 shorts = 12 extra bytes on the wire.**

By shape (2 bytes then 5 shorts) these are the standard potential/socket block —
`nGrade`, `nCHUC`, `nOption1..3`, `nSocket1..2`.

For contrast, v84 @`0x4eaf34` (session `46c2a2eb`) and v87 @`0x502eac`
(session `c0829805`) both go straight from `Decode4 ×3` to the conditional
buffer. Those two ARE byte-identical to each other — batch 5's equivalent claim
was checked the same way and held (RULING 23).

## Why nothing caught it

`libs/atlas-packet/model/asset.go` has no gate above `MajorAtLeast(84)`
(highest gates: lines 260, 605). So the encoder emits v84's shape for v92, the
fixtures assert what the encoder emits, and the tests pass — a closed loop that
agrees with itself and disagrees with the client. `tools/verify.sh` cannot see
this; only the IDB read can.

Note the JMS branch (encode lines 270-277, decode lines 613-621) writes a
*similar but distinct* block — 1 byte + 5 shorts + 1 int (15 bytes). **Do not
reuse it for GMS v92**; v92 is 2 bytes + 5 shorts (12 bytes), no trailing int.

## The fix

1. Add a GMS `MajorAtLeast(92)` gate in `encodeEquipableInfo` (after the
   `hammersApplied` write, before the JMS block) writing byte, byte, short ×5.
2. Mirror it in the decode path (after the `hammersApplied` read).
3. Correct the v92 item-bearing fixtures in
   `libs/atlas-packet/parcel/clientbound/v92_test.go` to the real 12-bytes-longer
   layout, derived from the IDB — not from the encoder's output.
4. Re-run the five `packet-audit` gates and re-confirm the v92 cells.

## Blast radius beyond v92

- **v95 (batch 7) almost certainly needs the same gate** — check `RawDecode` on
  the v95 IDB before writing its fixtures. Batch 7 must not be dispatched
  assuming the v84 shape.
- **jms_v185 (batch 8)** has its own JMS branch; verify it against the IDB rather
  than inheriting either shape.
- Every other codec that embeds an equip asset on v92+ is affected by the same
  encoder gap, since the fix is in the shared `Asset` encoder.

## Process lesson

"Same `Decode4` count" is not "same read order". A shape check must run to the
end of the function, and a fixture whose expectation is copied from another
version is only as good as the divergence check under it. This is the second
batch to lean on a copied expectation; batch 5's held, batch 6's did not.
