# FieldWhisperError (← `CField::OnWhisper#Error`)

- **IDA:** 0x5448a0
- **Atlas file:** `libs/atlas-packet/field/clientbound/whisper.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | string | string `target` | ✅ |  |
| 2 | byte | byte `whispersEnabled` | ✅ |  |

