# MonsterUseCatchItem (← `CWvsContext::SendBridleItemUseRequest`)

- **IDA:** 0xaee887
- **Atlas file:** `libs/atlas-packet/monster/serverbound/use_catch_item.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime (get_update_time)` | ✅ |  |
| 1 | int16 | int16 `nPOS` | ✅ |  |
| 2 | int32 | int32 `nItemID` | ✅ |  |
| 3 | int32 | int32 `hit-mob object id (FindHitMobInRect)` | ✅ |  |

