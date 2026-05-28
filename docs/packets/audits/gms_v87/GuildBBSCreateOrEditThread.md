# GuildBBSCreateOrEditThread (← `CUIGuildBBS::OnRegister`)

- **IDA:** 0x87a5df
- **Atlas file:** `libs/atlas-packet/guild/serverbound/bbs_create_or_edit_thread.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (create/edit)` | ✅ |  |
| 1 | int32 | int32 `threadId (0=create)` | ✅ |  |
| 2 | byte | string `title` | ❌ | width mismatch |
| 3 | string | string `body` | ✅ |  |
| 4 | string | byte `icon` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

