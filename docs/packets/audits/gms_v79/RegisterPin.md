# RegisterPin (← `CLogin::OnCheckPinCodeResult#RegisterPin`)

- **IDA:** 0x5d0921
- **Atlas file:** `libs/atlas-packet/account/serverbound/register_pin.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `pinInput flag (v79 OnCheckPinCodeResult@0x5d0921 v3==1 branch: COutPacket(10)+Encode1(1\|0))` | ✅ |  |
| 1 | string | string `pin string (only when pinInput=1: EncodeStr(pin))` | ✅ |  |

