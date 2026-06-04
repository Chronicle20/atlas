# Open (← `CUserLocal::OnOpenUI`)

- **IDA:** 0x9e32c0
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/open.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nUIID (window mode byte; dispatched to CWvsContext::UI_Open)` | ✅ |  |

