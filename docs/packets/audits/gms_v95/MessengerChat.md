# MessengerChat (← `CUIMessenger::OnPacket#Chat`)

- **IDA:** 0x7f52d0
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/chat.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=6, CHAT) — dispatcher switch byte consumed by CUIMessenger::OnPacket` | ✅ |  |
| 1 | string | string `chatLine — formatted as 'name : message' (parsed client-side to extract speaker)` | ✅ |  |

