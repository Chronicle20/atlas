# NpcShopOperationGenericError (← `CShopDlg::OnPacket#GenericError`)

- **IDA:** 0x756da7
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (generic-error sub-op: v83 case 17)` | ✅ |  |
| 1 | byte | byte `hasReason flag` | ✅ |  |
| 2 | string | string `reason (only when hasReason != 0)` | ✅ |  |


Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
