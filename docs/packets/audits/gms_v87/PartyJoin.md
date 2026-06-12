# PartyJoin (← `CWvsContext::OnPartyResult#Join`)

- **IDA:** 0xad697a
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/join.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (15)` | ✅ |  |
| 1 | int32 | int32 `partyLeaderId` | ✅ |  |
| 2 | string | string `inviteeName` | ✅ |  |
| 3 | int32 | bytes `PARTYDATA (298 bytes in v87 — same as v83; v95+ is 378)` | ✅ |  |
| 4 | bytes | byte `` | ✅ | absorbed by trailing opaque buffer |
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
| 15 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

