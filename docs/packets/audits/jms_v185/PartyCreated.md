# PartyCreated (← `CWvsContext::OnPartyResult#Created`)

- **IDA:** 0xb297e7
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/created.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 1 (Created)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | int32 | int32 `memberId (zeros ok — atlas writes EmptyMapId as placeholder)` | ✅ |  |
| 3 | int32 | int32 `mapId (zeros ok — atlas writes EmptyMapId as placeholder)` | ✅ |  |
| 4 | int16 | int16 `jobId (zeros ok — atlas writes 0)` | ✅ |  |
| 5 | int16 | int16 `level (zeros ok — atlas writes 0)` | ✅ |  |

