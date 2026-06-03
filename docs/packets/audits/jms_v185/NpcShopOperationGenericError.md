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


## Manual verdict (JMS v185, `CShopDlg::OnPacket#GenericError` @0x7cb04e)

Rows 1-2 (❌ "atlas: extra") reflect a JMS FEATURE ABSENCE, NOT an atlas wire bug. The atlas
`ShopOperationGenericError` writes `Byte(mode)+Bool(hasReason)+[AsciiString(reason) if hasReason]`
— the GMS v95 shape, where case 0x13 reads `Decode1(hasReason)` + optional `DecodeStr(reason)`.
JMS185 has NO reason-string mode: case 0x13 in `CShopDlg::OnPacket@0x7cb04e` is a no-op return
(grouped with cases 4 and 8). The discriminator/opcode mapping is config-driven; the GMS shape
is correct and unchanged. OUT OF SCOPE for an atlas wire fix — a JMS template that maps a code
to mode 0x13 would simply not have the trailing bytes consumed (template/config concern).

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
