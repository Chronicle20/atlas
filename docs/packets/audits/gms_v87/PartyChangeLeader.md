# PartyChangeLeader (← `CWvsContext::OnPartyResult#ChangeLeader`)

- **IDA:** 0xad697a
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/change_leader.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (31)` | ✅ |  |
| 1 | int32 | int32 `newLeaderId` | ✅ |  |
| 2 | byte | byte `isExpedition flag` | ✅ |  |

