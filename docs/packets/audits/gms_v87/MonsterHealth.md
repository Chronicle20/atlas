# MonsterHealth (← `CMob::OnHPIndicator`)

- **IDA:** 0x6a8505
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/health.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | byte | byte `nHPpercentage (0..100)` | ✅ |  |

