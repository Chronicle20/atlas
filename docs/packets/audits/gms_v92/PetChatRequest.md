# PetChatRequest (← `CPet::DoAction`)

- **IDA:** 0x697910
- **Atlas file:** `libs/atlas-packet/pet/serverbound/chat.go`
- **Variant:** GMS/v92
- **Branch depth:** 2
- **Verdict:** 🚫

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 1 | int32 | bytes `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | byte `` | ✅ |  |
| 4 | string | string `` | ✅ |  |

