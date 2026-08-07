# SetAccountResult (← `CLogin::OnSetAccountResult`)

- **IDA:** 0x56874d
- **Atlas file:** `libs/atlas-packet/login/clientbound/set_account_result.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `gender @0x568766 (v2)` | ✅ |  |
| 1 | byte | byte `success @0x568768 (if -> resend AFTER_LOGIN)` | ✅ |  |

