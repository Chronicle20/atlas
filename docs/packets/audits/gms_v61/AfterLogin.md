# AfterLogin (← `CLogin::OnSetAccountResult#AfterLogin`)

- **IDA:** 0x56874d
- **Atlas file:** `libs/atlas-packet/login/serverbound/after_login.go`
- **Variant:** GMS/v61
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `pinMode @0x5687bb (OnSetAccountResult literal 1u; OnCheckPinCodeResult Encode1(pin-result) @0x568a69/@0x568979)` | ✅ |  |
| 1 | byte | byte `opt2 @0x5687c5 (literal 1u; 0 in OnCheckPinCodeResult @0x568a85/@0x568995)` | ✅ |  |
| 2 | int32 | int32 `accountId @0x5687d8 (*(g_pWvsContext+8232); OnCheckPinCodeResult @0x568a98/@0x5689a8) - LEGACY-ONLY int, absent in v83+` | ✅ |  |
| 3 | string | string `pin @0x5687f5 (EncodeStr; empty ZXString in OnSetAccountResult)` | ✅ |  |

