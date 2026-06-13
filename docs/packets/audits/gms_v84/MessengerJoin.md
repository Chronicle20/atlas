# MessengerJoin (← `CUIMessenger::OnPacket#Join`)

- **IDA:** 0x87cbd8
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/join.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | string `` | ❌ | width mismatch |
| 2 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 3 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 4 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 7 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 15 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 17 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 19 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 21 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | byte `` | ❌ | atlas: short — missing trailing field |

