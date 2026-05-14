# BuffCancel (← `CWvsContext::OnTemporaryStatReset`)

- **IDA:** 0x9f2ab0
- **Atlas file:** `libs/atlas-packet/character/clientbound/buff_cancel.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | bytes `uFlagTemp: 16-byte UINT128 stat mask (DecodeBuffer 0x10)` | ❌ | width mismatch |
| 1 | byte | byte `nChangedStatPoint (only present when IsMovementAffectingStat(mask) is true)` | ❌ | atlas: short — missing trailing field |

---

**ack: tool-limitation false positive**

Row 0: `BuffCancel.Encode` calls `m.cts.EncodeMask(l, t, options)(w)` which
internally executes 4×`w.WriteInt` (128 bits = 16 bytes). The analyzer
cannot descend into the method call and reports a single "byte" write, while
IDA shows `DecodeBuf(16)`. The wire is correct — same pattern as `BuffGive`
(Task 7 bucket, acked in that bucket's commit).

Row 1: `BuffCancel.Encode` unconditionally writes `WriteByte(0)` for
`tSwallowBuffTime`. The client reads this byte only when
`IsMovementAffectingStat` is true. Atlas always emits it so the stream
remains aligned when the condition IS true; when false the client does not
consume it, leaving one trailing byte that is harmless for the cancel packet
(no further fields follow). Consistent with the `BuffGive` pattern.
