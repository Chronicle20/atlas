# NpcShopOperationSimple (← `CShopDlg::OnPacket#Simple`)

- **IDA:** 0x7cb04e
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (transaction result discriminator @0x7cb0e5)` | ✅ |  |


Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
