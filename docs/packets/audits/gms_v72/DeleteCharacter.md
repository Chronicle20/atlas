# DeleteCharacter (← `CLogin::SendDeleteCharPacket`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/character/serverbound/delete.go`
- **Variant:** GMS/v72
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

