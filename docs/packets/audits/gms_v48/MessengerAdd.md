# MessengerAdd (← `CUIMessenger::OnPacket#Add`)

- **IDA:** 0x61d8b8
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/add.go`
- **Variant:** GMS/v48
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | byte | byte `position` | ✅ |  |
| 2 | bytes | bytes `AvatarLook::Decode (opaque block)` | ✅ |  |
| 3 | string | string `name` | ✅ |  |
| 4 | byte | byte `channelId` | ✅ |  |
| 5 | byte | byte `pad` | ✅ |  |

