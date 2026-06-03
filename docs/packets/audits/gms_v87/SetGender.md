# SetGender (← `CLogin::SendSetGenderPacket`)

- **IDA:** 0x63409f
- **Atlas file:** `libs/atlas-packet/account/serverbound/set_gender.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `set flag (literal 1u when setting gender)` | ✅ |  |
| 1 | byte | byte `nGender byte (a2 param)` | ✅ |  |


## Manual analysis

v87 vs v83: `CLogin::SendSetGenderPacket` @ 0x63409f **is PRESENT in v87** (absent in v83). Sends opcode 0x8: Encode1(literal 1u set flag) + Encode1(a2 gender byte). Atlas matches. This function was absent in v83 and is present in v87 — the version gate that was noted as "absent in v83" resolves cleanly here.

Ack: misc-audit Phase 3 v87 on 2026-06-03
