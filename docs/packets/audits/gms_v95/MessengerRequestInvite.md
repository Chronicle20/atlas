# MessengerRequestInvite (← `CUIMessenger::OnPacket#RequestInvite`)

- **IDA:** 0x7f2cb0
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/request_invite.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=3, REQUEST_INVITE) — dispatcher switch byte consumed by CUIMessenger::OnPacket` | ✅ |  |
| 1 | string | string `sInviter — name of the character sending the invite` | ✅ |  |
| 2 | byte | byte `padding byte (v2, used for blacklist check)` | ✅ |  |
| 3 | int32 | int32 `messengerId — room id to join` | ✅ |  |
| 4 | byte | byte `padding byte (v4, checked against config flag)` | ✅ |  |

