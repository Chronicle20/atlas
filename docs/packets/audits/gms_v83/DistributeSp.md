# DistributeSp (← `CWvsContext::SendSkillUpRequest`)

- **IDA:** 0xa23cf3
- **Atlas file:** `libs/atlas-packet/character/serverbound/distribute_sp.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time())` | ✅ |  |
| 1 | int32 | int32 `nSkillID (skill to level up)` | ✅ |  |

