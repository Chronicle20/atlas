# MonsterControl (← `CMobPool::OnMobChangeController`)

- **IDA:** 0x679777
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/control.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `controlMode` | ✅ |  |
| 1 | int32 | int32 `dwMobID` | ✅ |  |
| 2 | byte | byte `aggro byte — atlas hardcodes 5` | ✅ |  |
| 3 | int32 | int32 `dwTemplateID via SetLocalMob — atlas monsterId` | ✅ |  |
| 4 | bytes | bytes `MonsterModel body` | ✅ |  |

