# FieldEffectBossHp (← `CField::OnFieldEffect#BossHp`)

- **IDA:** 0x5174bb
- **Atlas file:** `libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `monsterId (case 5) @0x51769d` | ❌ | width mismatch |
| 1 | int32 | int32 `currentHp @0x5176a7` | ✅ |  |
| 2 | int32 | int32 `maxHp @0x5176b1` | ✅ |  |
| 3 | int32 | byte `tagColor @0x5176bd` | ❌ | width mismatch |
| 4 | byte | byte `tagBackgroundColor @0x5176bf` | ✅ |  |
| 5 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

