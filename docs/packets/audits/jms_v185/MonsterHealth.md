# MonsterHealth (← `CMob::OnHPIndicator`)

- **IDA:** 0x6eaddf
- **Atlas file:** `libs/atlas-packet/monster/clientbound/health.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by dispatcher` | ✅ |  |
| 1 | byte | byte `nHPpercentage` | ✅ |  |

