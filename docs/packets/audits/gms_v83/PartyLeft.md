# PartyLeft (← `CWvsContext::OnPartyResult#Left`)

- **IDA:** 0xa3e31c
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/left.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | int32 | int32 `targetId` | ✅ |  |
| 3 | byte | byte `hasName flag` | ✅ |  |
| 4 | byte | byte `forced (expel/leave)` | ✅ |  |
| 5 | string | string `targetName` | ✅ |  |
| 6 | int32 | bytes `PARTYDATA::Decode — fixed opaque block (read as one buffer)` | ✅ |  |
| 7 | bytes | byte `` | ✅ | absorbed by trailing opaque buffer |
| 8 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 9 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 10 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 11 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 12 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 13 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 14 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 15 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 16 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 17 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 18 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

