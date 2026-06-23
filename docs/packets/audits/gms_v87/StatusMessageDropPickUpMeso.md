# StatusMessageDropPickUpMeso (← `CWvsContext::OnMessage#DropPickUpMeso`)

- **IDA:** 0xab818c
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (0 = drop pick-up)` | ✅ |  |
| 1 | byte | byte `inner disc int8 = 1 (meso)` | ✅ |  |
| 2 | byte | byte `partial bool` | ✅ |  |
| 3 | int32 | int32 `amount` | ✅ |  |
| 4 | int16 | int16 `internet-cafe bonus` | ✅ |  |

