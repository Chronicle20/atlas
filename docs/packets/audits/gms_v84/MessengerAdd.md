# MessengerAdd (← `CUIMessenger::OnPacket#Add`)

- **IDA:** 0x87cbd8
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/add.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | string `` | ❌ | width mismatch |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | int32 `` | ❌ | width mismatch |
| 4 | int32 | byte `` | ❌ | width mismatch |
| 5 | byte | byte `` | ✅ |  |
| 6 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 7 | byte | string `` | ❌ | width mismatch |
| 8 | int32 | byte `` | ❌ | width mismatch |
| 9 | byte | byte `` | ✅ |  |
| 10 | byte | byte `` | ✅ |  |
| 11 | int32 | byte `` | ❌ | width mismatch |
| 12 | byte | string `` | ❌ | width mismatch |
| 13 | int32 | byte `` | ❌ | width mismatch |
| 14 | int32 | string `` | ❌ | width mismatch |
| 15 | int32 | byte `` | ❌ | width mismatch |
| 16 | string | string `` | ✅ |  |
| 17 | byte | byte `` | ✅ |  |
| 18 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 19 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |
| 21 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | byte `` | ❌ | atlas: short — missing trailing field |

