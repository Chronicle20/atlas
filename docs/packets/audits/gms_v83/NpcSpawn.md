# NpcSpawn (← `CNpcPool::OnNpcEnterField`)

- **IDA:** 0x6d9993
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/spawn.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (dwNpcId)` | ✅ |  |
| 1 | int32 | int32 `templateId (CNpcTemplate id)` | ✅ |  |
| 2 | int16 | int16 `x (this+86)` | ✅ |  |
| 3 | int16 | int16 `cy (y, this+87)` | ✅ |  |
| 4 | byte | byte `moveAction (atlas f flag, this+43)` | ✅ |  |
| 5 | int16 | int16 `fh (foothold, GetFoothold arg)` | ✅ |  |
| 6 | int16 | int16 `rx0 (this+39)` | ✅ |  |
| 7 | int16 | int16 `rx1 (this+40)` | ✅ |  |
| 8 | byte | byte `enabled (this+77; atlas trailing 1)` | ✅ |  |


Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
