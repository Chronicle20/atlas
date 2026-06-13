# ItemUpgrade (← `CUser::ShowItemUpgradeEffect`)

- **IDA:** 0x9adb79
- **Atlas file:** `libs/atlas-packet/character/clientbound/item_upgrade.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserCommonPacket (case 0xBA)` | ✅ |  |
| 1 | byte | byte `bSuccess (scroll succeeded flag)` | ✅ |  |
| 2 | byte | byte `v5 / bCursed (cursed/failed outcome flag)` | ✅ |  |
| 3 | byte | byte `v30 / bEnchantSkill (enchant scroll category flag)` | ✅ |  |
| 4 | byte | byte `pExceptionObject / result byte — 4th Decode1; NO Decode4(enchantCategory) and NO 2nd result byte. v87 = same 4×Decode1 as v83. enchantCategory+enchantResultFlag are v95+ only.` | ✅ |  |

