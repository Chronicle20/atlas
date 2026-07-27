# FieldViciousHammerFailure (← `CField::OnItemUpgrade#Failure`)

- **IDA:** 0x537f8c
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 62; failure) — read in sub_82B2C3 (this[34]=v5), reached via sub_82B2AD (gates nType==354) from the CField::OnItemUpgrade vtable forwarder` | ✅ |  |
| 1 | int32 | int32 `errorCode (v13; switch 1=not-upgradable, 2=cap-reached, 3=horntail-necklace, default=unknown-error using this[35])` | ✅ |  |

