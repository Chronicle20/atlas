# FieldViciousHammerFailure (← `CField::OnItemUpgrade#Failure`)

- **IDA:** 0x799d61
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 61; failure) - this[32]=v6 read by CUIItemUpgrade::OnItemUpgradeResult 0x799d61, reached via CUIItemUpgrade::OnPacket 0x799d4b (a2==330)` | ✅ |  |
| 1 | int32 | int32 `errorCode (v12; switch 1=not-upgradable str5014, 2=cap-reached str5015, 3=horntail-necklace str5017, default=unknown-error str5343 using this[33])` | ✅ |  |

