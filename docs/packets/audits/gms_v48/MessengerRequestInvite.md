# MessengerRequestInvite (← `CUIMessenger::OnPacket#RequestInvite`)

- **IDA:** 0x61d8b8
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/request_invite.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | string | string `fromName` | ✅ |  |
| 2 | byte | byte `pad` | ✅ |  |
| 3 | int32 | int32 `messengerId` | ✅ |  |
| 4 | byte | byte `pad` | ✅ |  |

