# FieldEffectBossHp (← `CField::OnFieldEffect#BossHp`)

- **IDA:** 0x55aac5
- **Atlas file:** `libs/atlas-packet/field/clientbound/effect.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 5; boss HP tag)` | ✅ |  |
| 1 | int32 | int32 `monsterId (v34, @0x55aac5)` | ✅ |  |
| 2 | int32 | int32 `currentHp (v40, @0x55aacf)` | ✅ |  |
| 3 | int32 | int32 `maxHp (a2, @0x55aad9)` | ✅ |  |
| 4 | byte | byte `tagColor (v14, @0x55aae5)` | ✅ |  |
| 5 | byte | byte `tagBackgroundColor (v15, @0x55aae7)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
