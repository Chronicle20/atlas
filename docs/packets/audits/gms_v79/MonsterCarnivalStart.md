# MonsterCarnivalStart (← `CField_MonsterCarnival::OnEnter`)

- **IDA:** 0x548324
- **Atlas file:** `libs/atlas-packet/monster/carnival/clientbound/monster_carnival_start.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `team` | ✅ |  |
| 1 | int16 | int16 `personalCp` | ✅ |  |
| 2 | int16 | int16 `personalTotal` | ✅ |  |
| 3 | int16 | int16 `myTeamCp` | ✅ |  |
| 4 | int16 | int16 `myTeamTotal` | ✅ |  |
| 5 | int16 | int16 `enemyTeamCp` | ✅ |  |
| 6 | int16 | int16 `enemyTeamTotal` | ✅ |  |
| 7 | byte | byte `spelled (looped once per m_aSummonedMob element; loop bound is the client-local array's own stored count, not wire-read)` | ✅ |  |

