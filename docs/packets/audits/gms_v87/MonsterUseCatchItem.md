# MonsterUseCatchItem (← `CWvsContext::SendBridleItemUseRequest`)

- **IDA:** 0xa9f48b
- **Atlas file:** `libs/atlas-packet/monster/serverbound/use_catch_item.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime (get_update_time)` | ✅ |  |
| 1 | int16 | int16 `nPOS` | ✅ |  |
| 2 | int32 | int32 `nItemID` | ✅ |  |
| 3 | int32 | int32 `hit-mob object id (FindHitMobInRect)` | ✅ |  |

