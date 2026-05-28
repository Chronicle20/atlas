# ItemUpgrade (← `CUser::ShowItemUpgradeEffect`)

- **IDA:** 0x93354d
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/item_upgrade.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — consumed by CUserPool::OnUserCommonPacket dispatcher (case 0xBA) before calling this function` | ✅ |  |
| 1 | byte | byte `bSuccess (scroll succeeded flag)` | ✅ |  |
| 2 | byte | byte `v4 / bCursed (cursed/failed outcome flag)` | ✅ |  |
| 3 | byte | byte `bEnchantSkill (true = Vega/enchant scroll category; false = normal scroll)` | ✅ |  |
| 4 | byte | byte `v5 (lucky/white-scroll display flag) — NOTE: v83 does NOT read nEnchantCategory(4 bytes) or nEnchantResultFlag(1 byte) that v95 added` | ✅ |  |

