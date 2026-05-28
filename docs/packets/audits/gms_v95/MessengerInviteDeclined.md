# MessengerInviteDeclined (← `CUIMessenger::OnPacket#InviteDeclined`)

- **IDA:** 0x7f51a0
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/invite_declined.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=5, INVITE_DECLINED) — dispatcher switch byte consumed by CUIMessenger::OnPacket` | ✅ |  |
| 1 | string | string `sBlockedUser — name of the character who declined` | ✅ |  |
| 2 | byte | byte `declineMode — sub-enum (0=declined, non-zero=blocked); exact values deferred to _pending.md` | ✅ |  |

