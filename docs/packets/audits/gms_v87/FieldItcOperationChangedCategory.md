# FieldItcOperationChangedCategory (← `CITC::OnChangedCategory`)

- **IDA:** 0x5ceffa
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x5cf04f` | ✅ |  |
| 1 | int32 | int32 `category @0x5cf05a` | ✅ |  |
| 2 | int32 | int32 `categorySub (const 0) @0x5cf063` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x5cf06c` | ✅ |  |
| 4 | byte | byte `sortType (const 1) @0x5cf078` | ✅ |  |
| 5 | byte | byte `sortColumn (const 1) @0x5cf081` | ✅ |  |
| 6 | int32 | int32 `searchOption (const 1) @0x5cf08a` | ✅ |  |
| 7 | string | string `searchCondition (const empty) @0x5cf0a2` | ✅ |  |

