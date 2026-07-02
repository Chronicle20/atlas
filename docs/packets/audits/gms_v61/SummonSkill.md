# SummonSkill (← `CSummonedPool::OnSkill`)

- **IDA:** 0x67c8d2
- **Atlas file:** `../../libs/atlas-packet/summon/clientbound/skill.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_7922E8 Decode4@0x792327 (read before leaf dispatch)` | ✅ |  |
| 2 | byte | byte `newStance — skill leaf sub_67C8D2 Decode1@0x67c921 (masked 0x7F)` | ✅ |  |

