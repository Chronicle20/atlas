# RegisterPin (← `CLogin::OnCheckPinCodeResult#RegisterPin`)

- **IDA:** 0x5fc89d
- **Atlas file:** `../../libs/atlas-packet/account/serverbound/register_pin.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `pinInput flag (1=pin provided, 0=cancelled @0x5fc947)` | ✅ |  |
| 1 | string | string `pin string (only when pinInput=1 @0x5fca86)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CLogin::OnCheckPinCodeResult` @ 0x5fc89d, RegisterPin-sending path — Encode1(pinInput=1), EncodeStr(pin). Atlas decoder reads Decode1(flag) then optional DecodeStr. Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
