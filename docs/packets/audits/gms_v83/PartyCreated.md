# PartyCreated (← `CWvsContext::OnPartyResult#Created`)

- **IDA:** 0xa3e31c
- **Atlas file:** `libs/atlas-packet/party/clientbound/created.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (8)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | int32 | int32 `memberId` | ✅ |  |
| 3 | int32 | int32 `mapId` | ✅ |  |
| 4 | int16 | int16 `jobId` | ✅ |  |
| 5 | int16 | int16 `level` | ✅ |  |

