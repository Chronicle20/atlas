# Disable (← `CUserLocal::OnSetStandAloneMode`)

- **IDA:** 0xa2cdcb
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/disable.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bStandAlone (stored to CWvsContext szReserved+1060)` | ✅ |  |

