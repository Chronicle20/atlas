# FieldViciousHammerOpen (← `CField::OnItemUpgrade#Open`)

- **IDA:** 0x52a430
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (this->m_nReturnResult; open/arm token — any value not 65 or 66)` | ✅ |  |
| 1 | int32 | int32 `token (this->m_nResult; echoed back verbatim by CUIItemUpgrade::Update in ITEM_UPGRADE_UPDATE)` | ✅ |  |
| 2 | int32 | int32 `hammerCount (v23 -> this->m_nIUC) — this->m_nResultState=1; CUIItemUpgrade::ShowResult 0x7bec20 fires if this->m_nState==2` | ✅ |  |

