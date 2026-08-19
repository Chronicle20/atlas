# FieldWhisperReceive (← `CField::OnWhisper#Receive`)

- **IDA:** 0x53e2a0
- **Atlas file:** `libs/atlas-packet/field/clientbound/whisper.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | string | string `from` | ✅ |  |
| 2 | byte | byte `channel` | ✅ |  |
| 3 | byte | byte `gm` | ✅ |  |
| 4 | string | string `msg` | ✅ |  |

