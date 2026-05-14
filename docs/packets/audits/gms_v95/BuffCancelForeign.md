# BuffCancelForeign (← `CUserRemote::OnResetTemporaryStat`)

- **IDA:** 0x953e40
- **Atlas file:** `libs/atlas-packet/character/clientbound/buff_cancel.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | bytes `uFlagTemp: 16-byte UINT128 stat mask (DecodeBuffer 0x10)` | ❌ | width mismatch |

---

**ack: tool-limitation false positive**

Row 1: `BuffCancelForeign.Encode` calls `m.cts.EncodeMask(l, t, options)(w)`
which internally executes 4×`w.WriteInt` (128 bits = 16 bytes). The analyzer
cannot descend into the method call and reports a single "byte" write, while
IDA shows `DecodeBuf(16)`. The wire is correct — same pattern as `BuffGiveForeign`
(Task 7 bucket). `CUserRemote::OnResetTemporaryStat` has no trailing
`IsMovementAffectingStat` conditional (unlike the self-cancel variant), so
no extra byte concern exists for the foreign path.
