# MonsterCarnivalDied (← `CField_MonsterCarnival::OnProcessForDeath`)

- **IDA:** 0x548774
- **Atlas file:** `libs/atlas-packet/monster/carnival/clientbound/monster_carnival_died.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `team (team color selector: !=0 => MAPLE_BLUE, 0 => MAPLE_RED)` | ✅ |  |
| 1 | string | string `name (defeated character name)` | ✅ |  |
| 2 | byte | byte `lostCp (CP lost by the team; <=0 => no-cp-lost message variant)` | ✅ |  |

