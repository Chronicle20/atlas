# MonsterHealth (← `CMob::OnHPIndicator`)

- **IDA:** 0x68393b
- **Atlas file:** `libs/atlas-packet/monster/clientbound/health.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

