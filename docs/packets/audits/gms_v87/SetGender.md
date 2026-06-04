# SetGender (← `CLogin::SendSetGenderPacket`)

- **IDA:** 0x63409f
- **Atlas file:** `../../libs/atlas-packet/account/serverbound/set_gender.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `set flag (literal 1u when setting gender)` | ✅ |  |
| 1 | byte | byte `nGender byte (a2 param)` | ✅ |  |

