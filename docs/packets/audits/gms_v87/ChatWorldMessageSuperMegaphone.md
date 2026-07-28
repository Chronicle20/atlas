# ChatWorldMessageSuperMegaphone (← `CWvsContext::OnBroadcastMsg#SuperMegaphone`)

- **IDA:** 0xab9fd5
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=3, SUPER_MEGAPHONE)` | ✅ |  |
| 1 | string | string `message` | ✅ |  |
| 2 | byte | byte `channelId — case 3 shared tail with case 10` | ✅ |  |
| 3 | byte | byte `whispersOn — shared tail second read` | ✅ |  |

