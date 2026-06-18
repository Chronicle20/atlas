# NpcShopOperationGenericErrorWithReason (← `CShopDlg::OnPacket#GenericErrorWithReason`)

- **IDA:** 0x77905b
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (generic-error-with-reason sub-op: case 17)` | ✅ |  |
| 1 | byte | byte `hasReason flag (1 -> reason string follows)` | ✅ |  |
| 2 | string | string `reason (DecodeStr; shown as the Notice text)` | ✅ |  |
