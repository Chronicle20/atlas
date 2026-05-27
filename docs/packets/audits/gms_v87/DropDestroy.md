# DropDestroy (← `CDropPool::OnDropLeaveField`)

- **IDA:** 0x5287e3
- **Atlas file:** `libs/atlas-packet/drop/clientbound/destroy.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `destroyType` | ✅ |  |
| 1 | int32 | int32 `dwDropID` | ✅ |  |
| 2 | int32 | int32 `pickupCharId — gated destroyType in {2,3,5}` | ✅ |  |
| 3 | int16 | int16 `tLeaveDelay — gated destroyType == 4` | ✅ |  |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

