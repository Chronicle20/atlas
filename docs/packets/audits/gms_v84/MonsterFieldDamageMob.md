# MonsterFieldDamageMob (← `CMob::Update`)

- **IDA:** 0x67d4ea
- **Atlas file:** `libs/atlas-packet/monster/serverbound/field_damage_mob.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

