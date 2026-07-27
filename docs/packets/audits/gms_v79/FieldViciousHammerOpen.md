# FieldViciousHammerOpen (← `CField::OnItemUpgrade#Open`)

- **IDA:** 0x799d61
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (this[32]=v6; open/arm result - any byte not 60 or 61) - CUIItemUpgrade::OnItemUpgradeResult 0x799d61, reached via CUIItemUpgrade::OnPacket 0x799d4b (a2==330)` | ✅ |  |
| 1 | int32 | int32 `token (this[33]=m_nResult; echoed back verbatim by CUIItemUpgrade::Update in ITEM_UPGRADE_UPDATE)` | ✅ |  |
| 2 | int32 | int32 `hammerCount (this[34]=v16; current hammersApplied) - this[31]=m_nResultState=1; sub_79A282 fires if this[30]==2` | ✅ |  |

