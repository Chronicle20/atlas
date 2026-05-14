# AfterLogin (← `CLogin::OnSetAccountResult#AfterLogin`)

- **IDA:** 0x5d5e80
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/after_login.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `pinMode (literal 1u from outbound opcode 9 builder)` | ✅ |  |
| 1 | byte | byte `opt2 (literal 1u)` | ✅ |  |
| 2 | string | string `pin (empty ZXString)` | ✅ |  |

