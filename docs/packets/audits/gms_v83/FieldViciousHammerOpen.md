# FieldViciousHammerOpen (← `CField::OnItemUpgrade#Open`)

- **IDA:** 0x537f8c
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (this[34]=v5; open/arm token — any value not 61 or 62) — sub_82B2C3 via sub_82B2AD (nType==354)` | ✅ |  |
| 1 | int32 | int32 `token (this[35]=m_nResult; echoed back verbatim by CUIItemUpgrade::Update in ITEM_UPGRADE_UPDATE)` | ✅ |  |
| 2 | int32 | int32 `hammerCount (this[36]=v19; current hammersApplied) — this[33]=m_nResultState=1; sub_82B7F6 fires if this[32]==2` | ✅ |  |

