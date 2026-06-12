# MessengerChat (← `CUIMessenger::OnPacket#Chat`)

- **IDA:** 0x8b978f
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/chat.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | string | string `message` | ✅ |  |

