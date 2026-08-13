# ItemUpgrade (← `CUser::ShowItemUpgradeEffect`)

- **IDA:** 0x8d0570
- **Atlas file:** `libs/atlas-packet/character/clientbound/item_upgrade.go`
- **Variant:** GMS/v92
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | int32 `` | ❌ | width mismatch |
| 4 | int32 | byte `` | ❌ | width mismatch |
| 5 | byte | byte `` | ✅ |  |
| 6 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

