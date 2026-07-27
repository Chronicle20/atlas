# FieldViciousHammerOpen (← `CField::OnItemUpgrade#Open`)

- **IDA:** 0x55fa12
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (this[38]=v5; open/arm token — any value not 63 or 64) — sub_88F332 (a2==375) -> sub_88F348, reached via the CField::OnItemUpgrade vtable forwarder (this[135], vtable 0xbe109c slot 15 / offset 0x3C)` | ✅ |  |
| 1 | int32 | int32 `token (this[39]=m_nResult; echoed back verbatim by CUIItemUpgrade::Update in ITEM_UPGRADE_UPDATE)` | ✅ |  |
| 2 | int32 | int32 `hammerCount (this[40]=v15; current hammersApplied) — this[37]=m_nResultState=1; sub_88F87B fires if this[36]==2` | ✅ |  |

