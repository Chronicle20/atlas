# ChatWorldMessageMegaphone (← `CWvsContext::OnBroadcastMsg#Megaphone`)

- **IDA:** 0x91aaac
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=2, MEGAPHONE)` | ✅ |  |
| 1 | string | string `message - mode 2 has no header-extras arm (cmp edi,3/8/9/11 chain skips mode 2) (task-123 legacy phase 1, IDA-verified)` | ✅ |  |

