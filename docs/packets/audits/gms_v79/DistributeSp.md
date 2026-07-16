# DistributeSp (← `CWvsContext::SendSkillUpRequest`)

- **IDA:** 0x96debd
- **Atlas file:** `libs/atlas-packet/character/serverbound/distribute_sp.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time()) @0x96def4` | ✅ |  |
| 1 | int32 | int32 `nSkillID (skill to level up) @0x96deff` | ✅ |  |

