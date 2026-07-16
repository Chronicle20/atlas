# FieldViciousHammerFailure (← `CField::OnItemUpgrade#Failure`)

- **IDA:** 0x55fa12
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 64; failure) — read in sub_88F348 (this[38]=v5), reached via sub_88F332 (gates a2==375) from the CField::OnItemUpgrade vtable forwarder` | ✅ |  |
| 1 | int32 | int32 `errorCode (v11; switch 1/2/3 -> StringPool 5063/5064/5066 specific messages, default -> StringPool 5910 unknown-error using this[39])` | ✅ |  |

