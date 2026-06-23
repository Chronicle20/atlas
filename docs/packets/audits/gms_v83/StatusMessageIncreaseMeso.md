# StatusMessageIncreaseMeso (← `CWvsContext::OnMessage#IncreaseMeso`)

- **IDA:** 0xa221f3
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (INCREASE_MESO)` | ✅ |  |
| 1 | int32 | int32 `amount (signed)` | ✅ |  |

