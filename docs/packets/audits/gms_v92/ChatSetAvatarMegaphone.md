# ChatSetAvatarMegaphone (← `CWvsContext::OnSetAvatarMegaphone`)

- **IDA:** 0x9d6170
- **Atlas file:** `libs/atlas-packet/chat/clientbound/avatar_megaphone.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | string | string `` | ✅ |  |
| 2 | string | string `` | ✅ |  |
| 3 | int32 | string `` | ❌ | width mismatch |
| 4 | byte | string `` | ❌ | width mismatch |
| 5 | byte | string `` | ❌ | width mismatch |
| 6 | byte | int32 `` | ❌ | width mismatch |
| 7 | int32 | byte `` | ❌ | width mismatch |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

