# PartyInvite (← `CWvsContext::OnPartyResult#Invite`)

- **IDA:** 0xb297e7
- **Atlas file:** `libs/atlas-packet/party/clientbound/invite.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 4 (Invite)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | string | string `originator name` | ✅ |  |
| 3 | int32 | int32 `jobId (present in JMS v185 — v84plus gate correct)` | ✅ |  |
| 4 | int32 | int32 `level (present in JMS v185)` | ✅ |  |
| 5 | byte | byte `autoJoin flag` | ✅ |  |

