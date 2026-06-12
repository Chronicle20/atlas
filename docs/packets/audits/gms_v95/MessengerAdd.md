# MessengerAdd (← `CUIMessenger::OnPacket#Add`)

- **IDA:** 0x7f5e40
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/add.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=0, ADD) — dispatcher switch byte consumed by CUIMessenger::OnPacket` | ✅ |  |
| 1 | byte | byte `position — slot index in messenger room (0–2)` | ✅ |  |
| 2 | bytes | bytes `AvatarLook::AvatarLook(&v7, iPacket) — avatar appearance` | ✅ |  |
| 3 | string | string `sID — character name` | ✅ |  |
| 4 | byte | byte `channelId — channel the character is on` | ✅ |  |
| 5 | byte | byte `padding (extra flag, always discarded)` | ✅ |  |

