# PinUpdate (← `CLogin::OnUpdatePinCodeResult`)

- **IDA:** 0x6345d4
- **Atlas file:** `libs/atlas-packet/login/clientbound/pin_update.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode (success/failure dispatch)` | ✅ |  |

