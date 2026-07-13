# AfterLogin (← `CLogin::OnSetAccountResult#AfterLogin`)

- **IDA:** 0x5b5598
- **Atlas file:** `libs/atlas-packet/login/serverbound/after_login.go`
- **Variant:** GMS/v72
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `pinMode @0x5b55a6 (literal 1u)` | ✅ |  |
| 1 | byte | byte `opt2 @0x5b55b0 (literal 1u)` | ✅ |  |
| 2 | int32 | int32 `accountId @0x5b55c3 (*(g_pWvsContext+8232)) — LEGACY-ONLY int, absent in v83+` | ✅ |  |
| 3 | string | string `pin @0x5b55e0 (EncodeStr, empty ZXString)` | ✅ |  |

