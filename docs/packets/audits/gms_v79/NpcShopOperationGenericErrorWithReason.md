# NpcShopOperationGenericErrorWithReason (← `CShopDlg::OnPacket#GenericErrorWithReason`)

- **IDA:** 0x6d6eb9
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (generic-error-with-reason sub-op: v79 mode 14)` | ✅ |  |
| 1 | byte | byte `hasReason flag (1 -> reason string follows)` | ✅ |  |
| 2 | string | string `reason (DecodeStr; shown as the Notice text)` | ✅ |  |

