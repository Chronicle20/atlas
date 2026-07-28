# ChatWorldMessageSimple (← `CWvsContext::OnBroadcastMsg`)

- **IDA:** 0x91aaac
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | string | byte `` | ❌ | width mismatch |
| 2 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 3 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 4 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | ❌ | atlas: short — missing trailing field |
| 8 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 15 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 17 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | ❌ | atlas: short — missing trailing field |
| 18 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 19 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 20 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 21 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |

