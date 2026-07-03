# RemoveTownDoor (← `CWvsContext::OnTownPortal`)

- **IDA:** 0x71c285
- **Atlas file:** `libs/atlas-packet/door/clientbound/remove_town.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 3 | byte | int16 `` | ❌ | atlas: short — missing trailing field |

