# AfterLogin (← `CLogin::OnSetAccountResult#AfterLogin`)

- **IDA:** 0x5d0800
- **Atlas file:** `libs/atlas-packet/login/serverbound/after_login.go`
- **Variant:** GMS/v79
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `pinMode @0x5d080e (OnSetAccountResult literal 1u; OnCheckPinCodeResult Encode1(pin-result) @0x5d0abc/@0x5d09cb)` | ✅ |  |
| 1 | byte | byte `opt2 @0x5d0818 (literal; 0 in OnCheckPinCodeResult @0x5d0ad8/@0x5d09e7)` | ✅ |  |
| 2 | int32 | int32 `accountId @0x5d082b (*(g_pWvsContext+8232); OnCheckPinCodeResult @0x5d0aeb/@0x5d09fa) — LEGACY-ONLY int, absent in v83+` | ✅ |  |
| 3 | string | string `pin @0x5d0848 (EncodeStr; empty ZXString byte_B0C24C in OnSetAccountResult)` | ✅ |  |

