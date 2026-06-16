# MonsterMobEscortCollision (← `CMob::SendCollisionEscort`)

- **IDA:** 0x6efeb7
- **Atlas file:** `libs/atlas-packet/monster/serverbound/mob_escort_collision.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

