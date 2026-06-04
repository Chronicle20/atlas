# SetGender (← `CLogin::SendSetGenderPacket`)

- **IDA:** 0x5d4650
- **Atlas file:** `../../libs/atlas-packet/account/serverbound/set_gender.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `set flag (literal 1u when setting gender)` | ✅ |  |
| 1 | byte | byte `nGender byte (only when set=1)` | ✅ |  |

