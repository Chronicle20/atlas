# NpcShopOperationGenericError (← `CShopDlg::OnPacket#GenericError`)

- **IDA:** 0x7a290d
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (sub-op discriminator)` | ✅ |  |
| 1 | byte | byte `hasMessage flag (case 0x11)` | ✅ |  |
| 2 | string | string `error message (only when hasMessage set)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
