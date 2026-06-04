# Open (← `CUserLocal::OnOpenUI`)

- **IDA:** 0xa2cf38
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/open.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nUIID (dispatched to CWvsContext::UI_Open)` | ✅ |  |

