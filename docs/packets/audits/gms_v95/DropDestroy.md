# DropDestroy (← `CDropPool::OnDropLeaveField`)

- **IDA:** 0x511e20
- **Atlas file:** `../../libs/atlas-packet/drop/clientbound/destroy.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `destroyType (v3 — 0=expire, 2=pickup-char, 3=pickup-mob, 4=explode, 5=ftxs)` | ✅ |  |
| 1 | int32 | int32 `dwDropID` | ✅ |  |
| 2 | int32 | int32 `pickupCharId — gated destroyType in {2,3,5}` | ✅ |  |
| 3 | int16 | int16 `tLeaveDelay — gated destroyType == 4 (explode)` | ✅ |  |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

