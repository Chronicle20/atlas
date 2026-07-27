# ChatWorldMessageMegaphone (← `CWvsContext::OnBroadcastMsg#Megaphone`)

- **IDA:** 0x844d49
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=2, MEGAPHONE)` | ✅ |  |
| 1 | string | string `message - mode 2 has no header-extras arm (only mode 3/8 read extras); case 2 does chatlog display only (task-123 legacy phase 1, IDA-verified)` | ✅ |  |

