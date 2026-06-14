# PartyDisband (← `CWvsContext::OnPartyResult#Disband`)

- **IDA:** 0xb297e7
- **Atlas file:** `libs/atlas-packet/party/clientbound/disband.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | int32 | int32 `targetId` | ✅ |  |
| 3 | byte | byte `forced/disconnect flag` | ✅ |  |
| 4 | int32 | int32 `partyId (repeated by atlas disband)` | ✅ |  |

