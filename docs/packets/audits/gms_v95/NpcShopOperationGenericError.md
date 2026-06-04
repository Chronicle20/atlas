# NpcShopOperationGenericError (← `CShopDlg::OnPacket#GenericError`)

- **IDA:** 0x6eb7d0
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (generic-error sub-op)` | ✅ |  |
| 1 | byte | byte `hasReason flag` | ✅ |  |
| 2 | string | string `reason (only when hasReason != 0)` | ✅ |  |

