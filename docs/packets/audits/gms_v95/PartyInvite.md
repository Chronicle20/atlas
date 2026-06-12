# PartyInvite (← `CWvsContext::OnPartyResult#Invite`)

- **IDA:** 0xa10b5f
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/invite.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `v7 = partyId` | ❌ | width mismatch |
| 1 | int32 | string `sApplierName = inviter character name` | ❌ | width mismatch |
| 2 | string | int32 `nSkillID = inviter job id` | ❌ | width mismatch |
| 3 | int32 | int32 `sName = inviter level` | ✅ |  |
| 4 | int32 | byte `sMsg = auto-join flag (0=show dialog, 1=auto-accept)` | ❌ | width mismatch |
| 5 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

