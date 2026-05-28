# MessengerOperationDeclineInvite (← `CFadeWnd::SendCloseMessage`)

- **IDA:** 0x524180
- **Atlas file:** `../../libs/atlas-packet/messenger/serverbound/operation_decline_invite.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `m_sInviter (fromName) — name of the character who sent the invite; op byte (=5) stripped by atlas Operation dispatcher` | ✅ |  |
| 1 | string | string `selfCharName (myName) — own character name` | ✅ |  |
| 2 | byte | byte `always 0 (alwaysZero padding byte)` | ✅ |  |

