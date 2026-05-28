# SetAccountResult (← `CLogin::OnSetAccountResult`)

- **IDA:** 0x634144
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/set_account_result.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `gender` | ✅ |  |
| 1 | byte | byte `success` | ✅ |  |

