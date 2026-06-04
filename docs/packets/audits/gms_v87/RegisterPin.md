# RegisterPin (← `CLogin::OnCheckPinCodeResult#RegisterPin`)

- **IDA:** 0x6342b0
- **Atlas file:** `../../libs/atlas-packet/account/serverbound/register_pin.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `pinInput flag (1=pin provided, 0=cancelled)` | ✅ |  |
| 1 | string | string `pin string (only when pinInput=1)` | ✅ |  |

