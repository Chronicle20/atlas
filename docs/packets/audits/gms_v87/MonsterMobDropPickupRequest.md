# MonsterMobDropPickupRequest (← `CMob::SendDropPickUpRequest`)

- **IDA:** 0x6a98ae
- **Atlas file:** `libs/atlas-packet/monster/serverbound/mob_drop_pickup_request.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

