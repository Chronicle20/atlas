# GuildInfo (← `CWvsContext::OnGuildResult#Info`)

- **IDA:** 0xacf7d3
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/info.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (32)` | ✅ |  |
| 1 | byte | bytes `GUILDDATA::Decode (full guild data)` | ✅ |  |
| 2 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 3 | string | byte `` | ✅ | absorbed by trailing opaque buffer |
| 4 | string | byte `` | ✅ | absorbed by trailing opaque buffer |
| 5 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 6 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 7 | bytes | byte `` | ✅ | absorbed by trailing opaque buffer |
| 8 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 9 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 10 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 11 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 12 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 13 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 14 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 15 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 16 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 17 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 18 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 19 | string | byte `` | ✅ | absorbed by trailing opaque buffer |
| 20 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 21 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

