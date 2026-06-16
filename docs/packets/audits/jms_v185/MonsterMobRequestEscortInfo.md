# MonsterMobRequestEscortInfo (← `CMob::SendRequestEscortPath`)

- **IDA:** 0x6eff57
- **Atlas file:** `libs/atlas-packet/monster/serverbound/mob_request_escort_info.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

