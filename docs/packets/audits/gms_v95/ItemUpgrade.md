# ItemUpgrade (← `CUser::ShowItemUpgradeEffect`)

- **IDA:** 0x8e7b00
- **Atlas file:** `libs/atlas-packet/character/clientbound/item_upgrade.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — consumed by CUserPool::OnUserCommonPacket dispatcher (case 186 = 0xBA) before calling this function` | ✅ |  |
| 1 | byte | byte `bSuccess (scroll succeeded flag)` | ✅ |  |
| 2 | byte | byte `v4 / bCursed (cursed/failed outcome flag)` | ✅ |  |
| 3 | byte | byte `bEnchantSkill (true = Vega/enchant scroll category; false = normal scroll)` | ✅ |  |
| 4 | int32 | int32 `nEnchantCategory (enchant category type; 0 for normal scrolls; passed to CUIEnchantDlg::SetResult)` | ✅ |  |
| 5 | byte | byte `v5 (lucky/white-scroll display flag; selects 'Lucky Day' vs 'Scroll of Goodness' string)` | ✅ |  |
| 6 | byte | byte `v6 (enchant result flag; passed to CUIEnchantDlg::SetResult; 0 for normal scrolls)` | ✅ |  |

