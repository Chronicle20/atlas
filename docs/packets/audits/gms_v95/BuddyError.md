# BuddyError (← `CWvsContext::OnFriendResult#Error`)

- **IDA:** 0xa12630
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/error.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode — error sub-op byte (0x0B=11..0x0F=15, 0x10=16..0x13=19, 0x16=22, 0x17=23); each arm shows a StringPool notice dialog with no further packet reads except cases 0x10/0x11/0x13/0x16 which do Decode1 + optional DecodeStr` | ✅ |  |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

