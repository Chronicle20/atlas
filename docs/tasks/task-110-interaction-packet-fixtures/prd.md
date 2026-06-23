# Interaction Packet-Fixture Verification Campaign — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-23
---

## 1. Overview

The `interaction` family (CMiniRoomBaseDlg / PLAYER_INTERACTION — trade, personal
store, entrusted-merchant, memory mini-game) is ~90% verified in the packet coverage
matrix (`docs/packets/audits/STATUS.md`), with **12 `incomplete` cells** concentrated
in a handful of merchant/entrusted-store serverbound operations plus one invite hole.
The clientbound dispatcher arms (InteractionEnter / UpdateMerchant) were already
graduated to ✅ across all versions in the task-096 dispatcher campaign; what remains
are scattered **serverbound** operation holes.

Most holes are `v83/v84/v87/jms` gaps where `gms_v95` is verified — a port-from-a-
verified-sibling shape — with a couple of single-version holes (`TieAnswer` v84,
`Invite` v83/v87/jms).

## 2. Goals

Primary goals:
- Drive every `incomplete` cell in the `interaction` family to `verified` (✅) across
  all applicable versions.
- Land each promotion as the three coupled artifacts: byte-fixture (with
  `packet-audit:verify` marker), pinned evidence, regenerated matrix.

Non-goals:
- No new trade/personal-store/merchant features — verification only.
- No changes to the already-verified clientbound dispatcher arms (task-096).
- No opcode reshifts unless a fixture proves the registry opcode wrong (then surface,
  don't silently patch).

## 3. User Stories

- As an Atlas maintainer, I want entrusted-merchant put/remove and trade-invite
  serverbound ops proven byte-correct on every version so merchant/store interactions
  parse correctly across tenants, not just v95.
- As a packet-audit reviewer, I want the interaction family's remaining ❌ holes closed.

## 4. Functional Requirements

Close the following `interaction`-family `incomplete` serverbound cells (current gaps in
brackets):

- `interaction/serverbound/InteractionOperationInvite` — [v83, v87, jms] (v84, v95 verified)
- `interaction/serverbound/InteractionOperationMerchantPutItem` — [v83, v84, v87, jms] (v95 verified)
- `interaction/serverbound/InteractionOperationMerchantRemoveItem` — [v83, v84, v87, jms] (v95 verified)
- `interaction/serverbound/InteractionOperationMemoryGameTieAnswer` — [v84] (others verified)

(Already verified across all versions: the InteractionEnter / UpdateMerchant
clientbound arms, Chat, blacklist ops, all PersonalStore ops, all Trade ops, the other
MemoryGame ops, MerchantBuy/MerchantAddToBlackList/MerchantRemoveFromBlackList.)

For each cell, follow `docs/packets/audits/VERIFYING_A_PACKET.md` serverbound rules
(§9–10): decompile the client *write* order at the per-version opcode (using the
verified `gms_v95` codec as the structural template), distrust IDB names (the COutPacket
opcode is truth), wrap shared-model ops with thin wrapper codecs rather than
duplicating, write the byte-fixture with a `packet-audit:verify` marker, pin evidence +
REPORT, regenerate the matrix, and commit the artifacts together.

## 5. API Surface

None. Wire-format verification of existing interaction handlers.

## 6. Data Model

None.

## 7. Service Impact

- **atlas-channel** (interaction/serverbound handlers) — test files and fixtures added;
  production code changed only if a fixture proves a handler byte-incorrect.
- **docs/packets/** — evidence records and regenerated `STATUS.md` / `status.json`.

## 8. Non-Functional Requirements

- Byte-level verification — no enumeration shortcuts.
- Serverbound verification rules apply (shared-model ops → thin wrapper codecs;
  `routedElsewhere && !routed` conflicts indicate a template-wiring gap to resolve;
  export is non-idempotent surgical splice).
- IDA lookups via the documented MCP API; confirm instance/version per cell
  (v83=13341, v84=13337, v87=13340, v95=13339, jms=13338).

## 9. Open Questions

- `MerchantPutItem` / `MerchantRemoveItem` are verified only on v95 — do the other
  versions share the v95 codec (port) or does the entrusted-merchant slot encoding
  shift per version? Confirm against IDA.
- `Invite` v83/v87/jms holes — same shared trade/store invite codec as the verified
  v84/v95, or version-specific? 
- Are these holes a missing fixture only, or do any have a `routedElsewhere && !routed`
  template-wiring conflict that must be fixed before verification?

## 10. Acceptance Criteria

- [ ] All four listed interaction serverbound packets show `verified` (✅) for v83, v84,
      v87, v95, jms (or `n-a` where genuinely version-absent).
- [ ] Every promoted cell has a `packet-audit:verify` byte-fixture and a fresh pinned
      evidence record + REPORT committed together.
- [ ] `packet-audit matrix --check` (and fname-doc/operations `--check`) exit 0.
- [ ] Affected Go module(s): `go test -race ./...`, `go vet ./...`, `go build ./...`
      clean; `docker buildx bake` for any service whose `go.mod` was touched.
