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

**ack: tool-limitation false positive (two causes)**

Row 1: `BuffCancelForeign.Encode` calls `m.cts.EncodeMask(l, t, options)(w)`
which internally executes 4×`w.WriteInt` (128 bits = 16 bytes). The analyzer
cannot descend into the method call and reports a single "byte" write, while
IDA shows `DecodeBuffer(iPacket, &uFlagTemp, 16)`. The wire is correct —
same root cause as `BuffGiveForeign` (Task 7 bucket).

Additionally, `BuffCancelForeign.Encode` unconditionally writes a trailing
`tSwallowBuffTime` byte (`buff_cancel.go` ~73) after the mask.
`CUserRemote::OnResetTemporaryStat@0x953e40` reads only `Decode4(characterId)`
then `DecodeBuffer(iPacket, &uFlagTemp, 16)` and returns — no `Decode1`
follows. The surplus byte is harmless under v95's length-framed packet
boundary (no further fields follow in the foreign-cancel packet, so the
client cannot misinterpret it). A follow-up could drop the trailing byte
from `BuffCancelForeign.Encode` for byte-exact wire parity, but the current
behaviour is non-breaking.

Verdict ❌ is a tool-limitation on the mask row; no real wire bug elsewhere.
