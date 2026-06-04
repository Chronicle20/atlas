# NpcShopOperationGenericError (← `CShopDlg::OnPacket#GenericError`)

- **IDA:** 0x7cb04e
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (generic-error discriminator; JMS case 0x13 is a no-op return -- NO hasReason/reason bytes)` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |

