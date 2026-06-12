# PartyLeft (← `CWvsContext::OnPartyResult#Left`)

- **IDA:** 0xad697a
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/left.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (12)` | ✅ |  |
| 1 | int32 | int32 `partyLeaderId` | ✅ |  |
| 2 | int32 | int32 `expelledCharId` | ✅ |  |
| 3 | byte | byte `discharge flag` | ✅ |  |
| 4 | byte | bytes `PARTYDATA (298 bytes in v87)` | ✅ |  |
| 5 | string | byte `` | ✅ | absorbed by trailing opaque buffer |
| 6 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
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

