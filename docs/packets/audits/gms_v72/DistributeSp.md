# DistributeSp (← `CWvsContext::SendSkillUpRequest`)

- **IDA:** 0x91bead
- **Atlas file:** `libs/atlas-packet/character/serverbound/distribute_sp.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time @0x91bee4` | ✅ |  |
| 1 | int32 | int32 `skillId @0x91beef` | ✅ |  |

