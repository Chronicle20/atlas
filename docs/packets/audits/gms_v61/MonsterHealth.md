# MonsterHealth (← `CMob::OnHPIndicator`)

- **IDA:** 0x5cc480
- **Atlas file:** `libs/atlas-packet/monster/clientbound/health.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

