# ChatWorldMessageSuperMegaphone (← `CWvsContext::OnBroadcastMsg#SuperMegaphone`)

- **IDA:** 0x844d49
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=3, SUPER_MEGAPHONE)` | ✅ |  |
| 1 | string | string `message` | ✅ |  |
| 2 | byte | byte `channelId (task-123 legacy phase 1, IDA-verified)` | ✅ |  |
| 3 | byte | byte `whispersOn` | ✅ |  |

