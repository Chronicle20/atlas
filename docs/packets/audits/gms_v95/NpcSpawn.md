# NpcSpawn (← `CNpcPool::OnNpcEnterField`)

- **IDA:** 0x679680
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/spawn.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (dwNpcId)` | ✅ |  |
| 1 | int32 | int32 `templateId (CNpcTemplate id)` | ✅ |  |
| 2 | int16 | int16 `x (m_ptPosPrev.x)` | ✅ |  |
| 3 | int16 | int16 `cy (y)` | ✅ |  |
| 4 | byte | byte `moveAction (atlas f flag)` | ✅ |  |
| 5 | int16 | int16 `fh (foothold)` | ✅ |  |
| 6 | int16 | int16 `rx0 (m_rgHorz.low)` | ✅ |  |
| 7 | int16 | int16 `rx1 (m_rgHorz.high)` | ✅ |  |
| 8 | byte | byte `enabled (m_bEnabled; atlas trailing 1)` | ✅ |  |

