# GuildBBSCreateOrEditThread (← `CUIGuildBBS::OnRegister`)

- **IDA:** 0x7c4250
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/bbs_create_or_edit_thread.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op byte (create/edit sub-op)` | ✅ |  |
| 1 | byte | byte `modify bool` | ✅ |  |
| 2 | int32 | int32 `threadId (if modify)` | ✅ |  |
| 3 | byte | byte `notice bool` | ✅ |  |
| 4 | string | string `title` | ✅ |  |
| 5 | string | string `message` | ✅ |  |
| 6 | int32 | int32 `emoticonId` | ✅ |  |

