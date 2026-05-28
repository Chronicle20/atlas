# PartyInviteReject (← `CWvsContext::OnPartyResult#InviteReject`)

- **IDA:** 0xb297e7
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/invite_reject.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode = 5 (InviteReject/decline)` | ✅ |  |
| 1 | string | int32 `partyId` | ❌ | width mismatch |
| 2 | byte | byte `rejected = 0 (always 0 for reject)` | ❌ | atlas: short — missing trailing field |

