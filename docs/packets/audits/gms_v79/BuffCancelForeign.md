# BuffCancelForeign (← `CUserRemote::OnResetTemporaryStat`)

- **IDA:** 0x8d9ac7
- **Atlas file:** `libs/atlas-packet/character/clientbound/buff_cancel.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | bytes `` | ✅ |  |
| 1 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

