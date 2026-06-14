# BuffCancelForeign (← `CUserRemote::OnResetTemporaryStat`)

- **IDA:** 0xa574f5
- **Atlas file:** `libs/atlas-packet/character/clientbound/buff_cancel.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | bytes `` | ✅ |  |
| 1 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

