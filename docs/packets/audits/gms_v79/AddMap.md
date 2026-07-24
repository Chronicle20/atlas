# AddMap (← `CWvsContext::SendMapTransferRequest`)

- **IDA:** 0x970286
- **Atlas file:** `libs/atlas-packet/teleportrock/serverbound/add_map.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nType (1=register current map, 0=delete selected map)` | ✅ |  |
| 1 | byte | byte `flag (VIP saved-map slot set)` | ✅ |  |
| 2 | int32 | int32 `mapId (delete only)` | ✅ |  |

