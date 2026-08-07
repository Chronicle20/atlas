# NpcShopOperationGenericErrorWithReason (← `CShopDlg::OnPacket#GenericErrorWithReason`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |

