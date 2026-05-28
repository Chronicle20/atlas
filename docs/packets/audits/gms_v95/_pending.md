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

---

## Still pending — world domain (task-068 Phase 2c, field/clientbound)

### SETFIELD-old-driver-id

| Field | Value |
|---|---|
| Affected packets | `field/clientbound/set_field.go` — `SetField`; `field/clientbound/warp_to_map.go` — `WarpToMap` |
| Atlas files | `libs/atlas-packet/field/clientbound/set_field.go`, `libs/atlas-packet/field/clientbound/warp_to_map.go` |
| IDA | `CStage::OnSetField` @0x71a0a0 (SET_FIELD, GMS v95 opcode 0x8D/141) |
| Reason | v95 reads `m_dwOldDriverID` as a `Decode4` (line 129) immediately after `m_nChannelID` (line 128), unconditionally — before the `bCharacterData` split, so it affects BOTH the SetField (full) and WarpToMap (warp) paths. Atlas writes only the channelId int32, then emits a `WriteByte(0)+WriteInt(0)` pair under the JMS-only guard; it never emits a 4-byte old-driver-id for GMS. A GMS v95 client therefore reads the following envelope bytes shifted by 4. Every other envelope field matches v95 exactly. |
| Why deferred (not fixed) | The version-introduction point of `m_dwOldDriverID` is unknown. Atlas runs production GMS at v83/v87; only the v95 IDB is loaded. Adding a `(GMS && MajorVersion>83)`-gated `WriteInt` could be correct for v95 yet wrong for v87/v92 if the field was introduced later, breaking the very versions atlas serves. A speculative version gate is riskier than the current (also-wrong-for-v95) state. |
| Evidence needed | v83 / v87 / v92 GMS IDA for `CStage::OnSetField` to pin the exact version where `m_dwOldDriverID` (Decode4 after channelId) was added; then add the correctly-gated 4-byte write to both set_field.go and warp_to_map.go. |
| Verdict | ❌ cross-version structural divergence (v95 confirmed; gate unverifiable) |
