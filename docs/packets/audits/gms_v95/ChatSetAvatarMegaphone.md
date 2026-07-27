# ChatSetAvatarMegaphone (← `CWvsContext::OnSetAvatarMegaphone`)

- **IDA:** 0xa017e0
- **Atlas file:** `libs/atlas-packet/chat/clientbound/avatar_megaphone.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `itemId` | ✅ |  |
| 1 | string | string `name (sender name)` | ✅ |  |
| 2 | string | string `lines[0]` | ✅ |  |
| 3 | int32 | string `lines[1]` | ❌ | width mismatch |
| 4 | byte | string `lines[2]` | ❌ | width mismatch |
| 5 | byte | string `lines[3]` | ❌ | width mismatch |
| 6 | byte | int32 `channelId` | ❌ | width mismatch |
| 7 | int32 | byte `whispersOn` | ❌ | width mismatch |
| 8 | byte | bytes `AvatarLook::Decode(look) - opaque avatar block (model.Avatar recurse)` | ✅ |  |
| 9 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 10 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 11 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 12 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 13 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 14 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 15 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 16 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 17 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 18 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |

