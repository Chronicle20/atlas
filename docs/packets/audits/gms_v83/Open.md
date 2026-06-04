# Open (← `CUserLocal::OnOpenUI`)

- **IDA:** 0x9600f0
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/open.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nUIID (window mode byte; dispatched to CWvsContext::UI_Open)` | ✅ |  |

