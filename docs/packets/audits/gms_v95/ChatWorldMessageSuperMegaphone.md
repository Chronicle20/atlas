# ChatWorldMessageSuperMegaphone (← `CWvsContext::OnBroadcastMsg#SuperMegaphone`)

- **IDA:** 0xa04160
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | string | string `message` | ✅ |  |
| 2 | byte | byte `channelId - case 3: goto LABEL_31 directly (no pre-reads)` | ✅ |  |
| 3 | byte | byte `whispersOn - LABEL_31` | ✅ |  |

