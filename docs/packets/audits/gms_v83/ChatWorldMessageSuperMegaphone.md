# ChatWorldMessageSuperMegaphone (← `CWvsContext::OnBroadcastMsg#SuperMegaphone`)

- **IDA:** 0xa22785
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=3, SUPER_MEGAPHONE)` | ✅ |  |
| 1 | string | string `message` | ✅ |  |
| 2 | byte | byte `channelId — case 3 goto LABEL_18 (shared tail with case 10)` | ✅ |  |
| 3 | byte | byte `whispersOn — LABEL_18 second read` | ✅ |  |

