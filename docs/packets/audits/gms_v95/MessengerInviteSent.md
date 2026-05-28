# MessengerInviteSent (← `CUIMessenger::OnPacket#InviteSent`)

- **IDA:** 0x7f5030
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/invite_sent.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=4, INVITE_SENT) — dispatcher switch byte consumed by CUIMessenger::OnPacket` | ✅ |  |
| 1 | string | string `sMsg — invitee character name` | ✅ |  |
| 2 | byte | byte `success — 0=failure (blocked/offline), non-zero=success` | ✅ |  |

