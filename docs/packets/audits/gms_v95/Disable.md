# Disable (← `CUserLocal::OnSetStandAloneMode`)

- **IDA:** 0x905550
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/disable.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bStandAlone (enable flag; stored to CWvsContext+0xE15)` | ✅ |  |

