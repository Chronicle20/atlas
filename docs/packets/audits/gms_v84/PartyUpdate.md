# PartyUpdate (← `CWvsContext::OnPartyResult#Update`)

- **IDA:** 0xa89cf3
- **Atlas file:** `libs/atlas-packet/party/clientbound/update.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int32 | string `` | ❌ | width mismatch |
| 3 | bytes | int32 `` | ✅ |  |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | int32 | int32 `` | ✅ |  |
| 6 | int32 | int32 `` | ✅ |  |
| 7 | int32 | int32 `` | ✅ |  |
| 8 | int32 | byte `` | ❌ | width mismatch |
| 9 | int32 | string `` | ❌ | width mismatch |
| 10 | int32 | byte `` | ❌ | width mismatch |
| 11 | int32 | int32 `` | ✅ |  |
| 12 | int32 | int32 `` | ✅ |  |
| 13 | int32 | int16 `` | ❌ | width mismatch |
| 14 | int32 | int16 `` | ❌ | width mismatch |
| 15 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 16 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 17 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 18 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 19 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 20 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 21 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 23 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 24 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 25 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 27 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 28 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 29 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 30 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 31 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 32 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 33 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 34 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 35 | byte | byte `` | ❌ | atlas: short — missing trailing field |

