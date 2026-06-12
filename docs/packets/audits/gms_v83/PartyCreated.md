# PartyCreated (← `CWvsContext::OnPartyResult#Created`)

- **IDA:** 0xa3e31c
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/created.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | int32 | int32 `mapId1` | ✅ |  |
| 3 | int32 | int32 `mapId2` | ✅ |  |
| 4 | int16 | int16 `short1` | ✅ |  |
| 5 | int16 | int16 `short2` | ✅ |  |

