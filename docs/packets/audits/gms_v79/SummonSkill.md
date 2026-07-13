# SummonSkill (← `CSummonedPool::OnSkill`)

- **IDA:** 0x71d5df
- **Atlas file:** `libs/atlas-packet/summon/clientbound/skill.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket@0x8c8c84 (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_892500@0x89253f (read before leaf dispatch)` | ✅ |  |
| 2 | byte | byte `single stance byte, masked 0x7F (sub_71D5DF@0x71d62e); oid read in dispatcher` | ✅ |  |

