# BuffCancel (← `CWvsContext::OnTemporaryStatReset`)

- **IDA:** 0xa2071f
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/buff_cancel.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | bytes `uFlagTemp: 16-byte UINT128 stat mask (DecodeBuffer 0x10)` | ✅ |  |
| 1 | byte | byte `nChangedStatPoint (only present when IsMovementAffectingStat(mask) is true)` | ❌ | atlas: short — missing trailing field |

