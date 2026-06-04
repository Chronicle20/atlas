# Disable (← `CUserLocal::OnSetStandAloneMode`)

- **IDA:** 0x9e3172
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/disable.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bStandAlone (enable flag)` | ✅ |  |

