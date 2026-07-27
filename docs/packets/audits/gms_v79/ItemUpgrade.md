# ItemUpgrade (← `CUser::ShowItemUpgradeEffect`)

- **IDA:** 0x88d1c4
- **Atlas file:** `libs/atlas-packet/character/clientbound/item_upgrade.go`
- **Variant:** GMS/v79
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | byte `` | ✅ |  |
| 4 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

