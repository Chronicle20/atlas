# ItemUpgrade (← `CUser::ShowItemUpgradeEffect`)

- **IDA:** 0x9f1a92
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/item_upgrade.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — consumed by CUserPool::OnUserCommonPacket dispatcher before calling this function` | ✅ |  |
| 1 | byte | byte `bSuccess` | ✅ |  |
| 2 | byte | byte `bCursed` | ✅ |  |
| 3 | byte | byte `bEnchantSkill (JMS: no nEnchantCategory Decode4 follows this)` | ✅ |  |
| 4 | byte | byte `v5 (lucky/white-scroll display flag)` | ✅ |  |
| 5 | byte | byte `v6 / enchantResultFlag (display flag — present in JMS v185 even though not passed to CUIEnchantDlg)` | ✅ |  |

