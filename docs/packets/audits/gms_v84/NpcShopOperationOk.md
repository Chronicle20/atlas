# NpcShopOperationOk (← `CShopDlg::OnPacket#Ok`)

- **IDA:** 0x77905b
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (sub-op discriminator; mode-only arm)` | ✅ |  |

