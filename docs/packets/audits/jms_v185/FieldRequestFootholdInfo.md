# FieldRequestFootholdInfo (← `CField::OnRequestFootHoldInfo`)

- **IDA:** 0x576cd2
- **Atlas file:** `libs/atlas-packet/field/serverbound/request_foothold_info.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nCurState` | ✅ |  |
| 1 | int32 | int32 `nCurX (0 when no moving info)` | ✅ |  |
| 2 | int32 | int32 `nCurY (0 when no moving info)` | ✅ |  |
| 3 | byte | byte `bReverseVertical (0 when no moving info)` | ✅ |  |
| 4 | byte | byte `bReverseHorizontal (0 when no moving info)` | ✅ |  |

