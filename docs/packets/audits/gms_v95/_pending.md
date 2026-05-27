# Pending / Deferred Audit Items — GMS v95

Items deferred from the per-packet audit loop. Each row captures what is unknown,
why it was deferred, and what evidence is needed to resolve it.

---

## OP-FAMILY-messenger-serverbound

| Field | Value |
|---|---|
| Packet | `messenger/serverbound/operation.go` — `Operation` |
| Atlas file | `libs/atlas-packet/messenger/serverbound/operation.go` |
| Reason | Op-byte dispatcher — the single `mode` byte routes to sub-ops 0 (AnswerInvite), 2 (Leave/Destroy), 3 (Invite), 5 (DeclineInvite), 6 (Chat). The full op-family enum (all valid mode values and their semantics) has not been exhaustively verified in IDA. |
| Evidence needed | Confirm that no other mode values exist beyond 0, 2, 3, 5, 6; verify server-side routing in atlas-messengers matches. |
| Verdict | ⚠️ |

---

## OP-FAMILY-messenger-decline

| Field | Value |
|---|---|
| Packet | `messenger/clientbound/invite_declined.go` — `InviteDeclined` |
| Atlas file | `libs/atlas-packet/messenger/clientbound/invite_declined.go` |
| Reason | The `declineMode` byte sub-enum in `OnBlocked` (mode=5) distinguishes between "declined" (0) and "blocked" (non-zero). IDA shows: `if v3` branching to two different StringPool strings (0x31Au vs 0x31Bu). The exact numeric meaning of non-zero values is not confirmed. |
| Evidence needed | Verify declineMode values in server-side atlas-messengers event emissions; confirm whether only 0/1 are used or additional values exist. |
| Verdict | ⚠️ |

---

## AUDIT-TOOL-avatarlook

| Field | Value |
|---|---|
| Affected packets | `messenger/clientbound/add.go` — `Add`; `messenger/clientbound/update.go` — `Update` |
| Reason | The packet audit tool cannot align atlas `WriteByteArray` (AvatarLook encoded as []byte) with IDA `DecodeBuf`. Both use the same AvatarLook encoding; the mismatch is a tool limitation, not a wire bug. Reports show ❌ for Add and Update but the actual encoding is correct. |
| Evidence needed | Tool enhancement to recognize DecodeBuf as opaque byte-blob and compare structurally rather than field-by-field. |
| Verdict | ⚠️ tool limitation — atlas wire is correct |
