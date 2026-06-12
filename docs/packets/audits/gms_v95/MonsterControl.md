# MonsterControl (← `CMobPool::OnMobChangeController`)

- **IDA:** 0x658d10
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/control.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `controlMode (v4/v5 — 0 = remote, non-zero = local-controlled)` | ✅ |  |
| 1 | int32 | int32 `dwMobID (uniqueId / v7)` | ✅ |  |
| 2 | byte | byte `aggro byte — atlas hardcodes 5, v95 reads as aggro flag` | ✅ |  |
| 3 | int32 | int32 `dwTemplateID via SetLocalMob — atlas monsterId` | ✅ |  |
| 4 | bytes | bytes `MonsterModel body via SetLocalMob's CMob::Init delegate` | ✅ |  |

