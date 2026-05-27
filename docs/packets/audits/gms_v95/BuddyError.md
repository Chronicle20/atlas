# BuddyError (← `CWvsContext::OnFriendResult#Error`)

- **IDA:** 0xa12630
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/error.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode — error sub-op byte (0x0B=11..0x0F=15, 0x10=16..0x13=19, 0x16=22, 0x17=23); each arm shows a StringPool notice dialog with no further packet reads except cases 0x10/0x11/0x13/0x16 which do Decode1 + optional DecodeStr` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

ack: sub-op enum modeling deferred — `error.go` `hasExtra` flag models whether certain error modes include a trailing byte. IDA shows error mode arms 0x0B–0x0F and 0x17 read no bytes after the mode; arms 0x10/0x11/0x13/0x16 read Decode1 + optional DecodeStr. Atlas `hasExtra` covers the first category; the conditional string path is not represented. The ❌ on row 1 fires when `hasExtra=true` (or conditionally): the audit pipeline sees the conditional WriteByte as extra. Real behaviour is mode-dependent. Deferred to _pending.md "BuddyError sub-op enum" row.

