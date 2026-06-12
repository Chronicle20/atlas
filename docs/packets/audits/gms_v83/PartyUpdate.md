# PartyUpdate (← `CWvsContext::OnPartyResult#Update`)

- **IDA:** 0xa3e31c
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/update.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (7 or 34)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | int32 | bytes `PARTYDATA::Decode — fixed 298-byte block (qmemcpy 0x12A); read as one opaque buffer, NOT field-by-field` | ✅ |  |
| 3 | bytes | byte `` | ✅ | absorbed by trailing opaque buffer |
| 4 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 5 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 6 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 7 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 8 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 9 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 10 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 11 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 12 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 13 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 14 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

