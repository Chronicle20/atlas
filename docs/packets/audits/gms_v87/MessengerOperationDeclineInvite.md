# MessengerOperationDeclineInvite (← `CFadeWnd::SendCloseMessage`)

- **IDA:** 0x54574f
- **Atlas file:** `libs/atlas-packet/messenger/serverbound/operation_decline_invite.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | string | int32 `` | ❌ | width mismatch |
| 2 | string | byte `` | ❌ | width mismatch |
| 3 | byte | string `` | ❌ | width mismatch |
| 4 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | string `` | ❌ | atlas: short — missing trailing field |

