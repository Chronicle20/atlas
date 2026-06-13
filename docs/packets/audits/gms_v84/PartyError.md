# PartyError (← `CWvsContext::OnPartyResult#Error`)

- **IDA:** 0xa89cf3
- **Atlas file:** `libs/atlas-packet/party/clientbound/error.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | string | int32 `` | ❌ | width mismatch |
| 2 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 3 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 4 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
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

