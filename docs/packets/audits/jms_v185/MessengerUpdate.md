# MessengerUpdate (← `CUIMessenger::OnPacket#Update`)

- **IDA:** 0x8e4bab
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/update.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | byte | byte `position` | ✅ |  |
| 2 | bytes | bytes `AvatarLook::Decode (opaque block)` | ✅ |  |

