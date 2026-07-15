# NpcNpcConversation (← `CScriptMan::OnScriptMessage`)

- **IDA:** 0x63f9ca
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v61
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | string `` | ❌ | width mismatch |
| 4 | int32 | byte `` | ❌ | width mismatch |
| 5 | bytes | byte `` | ✅ |  |
| 6 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 15 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 17 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 19 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 25 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 27 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 28 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 29 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 30 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 31 | byte | bytes `` | ❌ | atlas: short — missing trailing field |
| 32 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 33 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 34 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 35 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 36 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 37 | byte | int16 `` | ❌ | atlas: short — missing trailing field |

