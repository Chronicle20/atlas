# NpcShopOperationSimple (← `CShopDlg::OnPacket#Simple`)

- **IDA:** 0x7a290d
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (sub-op discriminator; mode-only arm)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
