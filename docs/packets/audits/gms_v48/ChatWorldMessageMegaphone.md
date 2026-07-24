# ChatWorldMessageMegaphone (← `CWvsContext::OnBroadcastMsg#Megaphone`)

- **IDA:** 0x71c356
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=2, MEGAPHONE)` | ✅ |  |
| 1 | string | string `message - mode 2 has no case in the body switch, falls straight to the display switch with no further wire reads (task-123 legacy phase 1, IDA-verified)` | ✅ |  |

