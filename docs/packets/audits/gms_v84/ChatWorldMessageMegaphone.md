# ChatWorldMessageMegaphone (← `CWvsContext::OnBroadcastMsg#Megaphone`)

- **IDA:** 0xa6dc97
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=2, MEGAPHONE)` | ✅ |  |
| 1 | string | string `message — mode 2 has no case in the body pre-decode switch, falls straight to the display switch with no further wire reads` | ✅ |  |

