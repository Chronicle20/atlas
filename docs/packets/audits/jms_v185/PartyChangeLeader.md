# PartyChangeLeader (← `CWvsContext::OnPartyResult#ChangeLeader`)

- **IDA:** 0xb297e7
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/change_leader.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (ChangeLeader)` | ✅ |  |
| 1 | int32 | int32 `newLeaderId` | ✅ |  |
| 2 | byte | byte `hasChanged flag` | ✅ |  |

