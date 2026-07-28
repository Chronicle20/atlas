# FieldViciousHammerOpen (← `CField::OnItemUpgrade#Open`)

- **IDA:** 0x5443af
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (a1[34]=m_nReturnResult; open/arm token — any value not 61 or 62) — sub_85676C via CUIItemUpgrade::OnPacket sub_856756 (header==364)` | ✅ |  |
| 1 | int32 | int32 `token (a1[35]=m_nResult; echoed back verbatim by CUIItemUpgrade::Update in ITEM_UPGRADE_UPDATE)` | ✅ |  |
| 2 | int32 | int32 `hammerCount (a1[36]=v16; current hammersApplied) — a1[33]=m_nResultState=1; ShowResult sub_856C9F fires if a1[32]==2` | ✅ |  |

