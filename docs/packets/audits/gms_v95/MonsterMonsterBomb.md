# MonsterMonsterBomb (← `CMob::TryFirstSelfDestruction`)

- **IDA:** 0x640ee0
- **Atlas file:** `libs/atlas-packet/monster/serverbound/monster_bomb.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

