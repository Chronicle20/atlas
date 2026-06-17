# FieldWhisperFindResultMap (← `CField::OnWhisper#FindResultMap`)

- **IDA:** 0x56f4df
- **Atlas file:** `libs/atlas-packet/field/clientbound/whisper.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | string | string `target` | ✅ |  |
| 2 | byte | byte `findMode (=1)` | ✅ |  |
| 3 | int32 | int32 `mapId` | ✅ |  |
| 4 | int32 | int32 `x (mode 0x09 only)` | ✅ |  |
| 5 | int32 | int32 `y (mode 0x09 only)` | ✅ |  |

