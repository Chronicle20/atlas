# ChatSetAvatarMegaphone (← `CWvsContext::OnSetAvatarMegaphone`)

- **IDA:** 0x9221a8
- **Atlas file:** `libs/atlas-packet/chat/clientbound/avatar_megaphone.go`
- **Variant:** GMS/v72
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
| 8 | byte | byte `` | ✅ |  |
| 9 | int32 | byte `` | ❌ | width mismatch |
| 10 | byte | int32 `` | ❌ | width mismatch |
| 11 | int32 | byte `` | ❌ | width mismatch |
| 12 | byte | int32 `` | ❌ | width mismatch |
| 13 | byte | byte `` | ✅ |  |
| 14 | int32 | int32 `` | ✅ |  |
| 15 | byte | byte `` | ✅ |  |
| 16 | int32 | int32 `` | ✅ |  |
| 17 | int32 | int32 `` | ✅ |  |
| 18 | int32 | bytes `` | ✅ |  |

