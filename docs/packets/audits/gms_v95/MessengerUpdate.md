# MessengerUpdate (← `CUIMessenger::OnPacket#Update`)

- **IDA:** 0x7f2ea0
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/update.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=7, UPDATE) — dispatcher switch byte consumed by CUIMessenger::OnPacket` | ✅ |  |
| 1 | byte | byte `position — slot index whose avatar changed` | ✅ |  |
| 2 | bytes | bytes `AvatarLook::AvatarLook(&v5, iPacket) — updated avatar appearance` | ✅ |  |

