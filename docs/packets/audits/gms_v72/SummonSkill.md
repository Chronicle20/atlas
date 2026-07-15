# SummonSkill (← `CSummonedPool::OnSkill`)

- **IDA:** 0x6e97d5
- **Atlas file:** `libs/atlas-packet/summon/clientbound/skill.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_848023@0x848062 (read before leaf dispatch)` | ✅ |  |
| 2 | byte | byte `single stance byte, masked 0x7F (sub_6E97D5@0x6e9824); oid read in dispatcher` | ✅ |  |

