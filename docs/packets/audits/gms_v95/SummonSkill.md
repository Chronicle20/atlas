# SummonSkill (← `CSummonedPool::OnSkill`)

- **IDA:** 0x759890
- **Atlas file:** `../../libs/atlas-packet/summon/clientbound/skill.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `charId — read by CSummonedPool::OnPacket@0x75ac70 before dispatch` | ✅ |  |
| 1 | int32 | int32 `oid (dwSummonedID) — CUser::OnSummonedSkill@0x8e39b2` | ✅ |  |
| 2 | byte | byte `action/newStance (v4 & 0x7F -> SetAttackAction) — CSummoned::OnSkill@0x74a9ae` | ✅ |  |

