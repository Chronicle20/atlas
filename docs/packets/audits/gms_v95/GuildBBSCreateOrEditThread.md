# GuildBBSCreateOrEditThread (← `CUIGuildBBS::OnRegister`)

- **IDA:** 0x7c4250
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/bbs_create_or_edit_thread.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op byte (create/edit sub-op)` | ✅ |  |
| 1 | int32 | byte `modify bool` | ❌ | width mismatch |
| 2 | byte | int32 `threadId (if modify)` | ❌ | width mismatch |
| 3 | string | byte `notice bool` | ❌ | width mismatch |
| 4 | string | string `title` | ✅ |  |
| 5 | int32 | string `message` | ❌ | width mismatch |
| 6 | byte | int32 `emoticonId` | ❌ | atlas: short — missing trailing field |

