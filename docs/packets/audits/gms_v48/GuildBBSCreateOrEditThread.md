# GuildBBSCreateOrEditThread (← `CUIGuildBBS::OnRegister`)

- **IDA:** 0x608d55
- **Atlas file:** `libs/atlas-packet/guild/serverbound/bbs_create_or_edit_thread.go`
- **Variant:** GMS/v48
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `modify flag @0x608e6a (after BBS_OPERATION mode 0=REGISTER @0x608e5f)` | ✅ |  |
| 1 | byte | int32 `threadId (only when modify != 0) @0x608e80` | ❌ | width mismatch |
| 2 | int32 | byte `notice flag @0x608e8b` | ❌ | width mismatch |
| 3 | byte | string `title @0x608ea4` | ❌ | width mismatch |
| 4 | string | string `message @0x608ebd` | ✅ |  |
| 5 | string | int32 `emoticonId @0x608ecb` | ❌ | width mismatch |
| 6 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

