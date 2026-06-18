# FieldWhisperFindResultCashShop (← `CField::OnWhisper#FindResultCashShop`)

- **IDA:** 0x53e514
- **Atlas file:** `libs/atlas-packet/field/clientbound/whisper.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | string | string `target` | ✅ |  |
| 2 | byte | byte `findMode (=2)` | ✅ |  |
| 3 | int32 | int32 `-1` | ✅ |  |

