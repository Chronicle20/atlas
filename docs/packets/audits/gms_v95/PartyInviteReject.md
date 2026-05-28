# PartyInviteReject (← `CWvsContext::OnPartyResult#InviteReject`)

- **IDA:** 0xa10ab0
- **Atlas file:** `libs/atlas-packet/party/serverbound/invite_reject.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `unk = decline response code` | ✅ |  |
| 1 | string | string `from = inviter character name` | ✅ |  |

