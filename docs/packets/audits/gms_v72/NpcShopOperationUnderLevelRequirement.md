# NpcShopOperationUnderLevelRequirement (← `CShopDlg::OnPacket#UnderLevelRequirement`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

