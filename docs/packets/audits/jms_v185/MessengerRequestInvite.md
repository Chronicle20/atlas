# MessengerRequestInvite (← `CUIMessenger::OnPacket#RequestInvite`)

- **IDA:** 0x8e46f2
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/request_invite.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 3 (RequestInvite) — consumed by OnPacket before calling OnInvite` | ✅ |  |
| 1 | string | string `fromName (inviter)` | ✅ |  |
| 2 | byte | byte `pad byte` | ✅ |  |
| 3 | int32 | int32 `messengerId` | ✅ |  |
| 4 | byte | byte `pad byte` | ✅ |  |

