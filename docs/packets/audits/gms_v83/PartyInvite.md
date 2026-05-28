# PartyInvite (← `CWvsContext::OnPartyResult#Invite`)

- **IDA:** 0xa3e31c
- **Atlas file:** `libs/atlas-packet/party/clientbound/invite.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (4)` | ✅ |  |
| 1 | int32 | int32 `partyId (v147)` | ✅ |  |
| 2 | string | string `inviterName` | ✅ |  |
| 3 | int32 | byte `autoJoinFlag (*a2)` | ❌ | width mismatch |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

