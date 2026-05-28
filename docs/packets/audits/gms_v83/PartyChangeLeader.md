# PartyChangeLeader (← `CWvsContext::OnPartyResult#ChangeLeader`)

- **IDA:** 0xa3e31c
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/change_leader.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (27)` | ✅ |  |
| 1 | int32 | int32 `newLeaderId (v64)` | ✅ |  |
| 2 | byte | byte `disconnected flag (*a2)` | ✅ |  |

