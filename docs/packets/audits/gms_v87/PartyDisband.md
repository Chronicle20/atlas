# PartyDisband (← `CWvsContext::OnPartyResult#Disband`)

- **IDA:** 0xad697a
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/disband.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (14)` | ✅ |  |
| 1 | int32 | int32 `partyLeaderId` | ✅ |  |
| 2 | int32 | bytes `PARTYDATA (298 bytes in v87)` | ✅ |  |
| 3 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 4 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

