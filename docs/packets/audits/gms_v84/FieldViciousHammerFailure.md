# FieldViciousHammerFailure (← `CField::OnItemUpgrade#Failure`)

- **IDA:** 0x5443af
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 62; failure) — sub_85676C a1[34]=v6` | ✅ |  |
| 1 | int32 | int32 `errorCode (v12; switch 1=not-upgradable StringPool 5057, 2=used-already StringPool 5058, 3=horntail-necklace StringPool 5060, default=unknown-error StringPool 5757 using a1[35])` | ✅ |  |

