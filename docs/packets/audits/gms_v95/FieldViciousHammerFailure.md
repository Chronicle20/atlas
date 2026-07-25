# FieldViciousHammerFailure (← `CField::OnItemUpgrade#Failure`)

- **IDA:** 0x52a430
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 66; failure) — v5 stored to this->m_nReturnResult, then `if (v5==66)` in OnItemUpgradeResult` | ✅ |  |
| 1 | int32 | int32 `errorCode (v13; switch 1=not-upgradable, 2=cap-reached, 3=horntail-necklace, default=unknown-error using this->m_nResult)` | ✅ |  |

