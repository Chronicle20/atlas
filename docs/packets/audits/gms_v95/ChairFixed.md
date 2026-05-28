# ChairFixed (← `CUserLocal::HandleXKeyDown`)

- **IDA:** 0x90f6d0
- **Atlas file:** `libs/atlas-packet/character/serverbound/chair_fixed.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `chairId (seat index from CField::FindSeatByPosition; 0xFFFF = get-up-from-chair)` | ✅ |  |

