# RegisterPin (← `CLogin::OnCheckPinCodeResult#RegisterPin`)

- **IDA:** 0x5688ce
- **Atlas file:** `libs/atlas-packet/account/serverbound/register_pin.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `pinInput flag @0x568b59 (1u register / @0x568b42 0u cancel)` | ✅ |  |
| 1 | string | string `pin @0x568b9a (only when pinInput=1)` | ✅ |  |

