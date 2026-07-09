# Mount Food Packet Verification — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-09
---

## 1. Overview

The mount food feature (task-086-mount-system) is fully implemented server-side: the
serverbound `USE_MOUNT_FOOD` packet is decoded by
`libs/atlas-packet/mount/serverbound/food.go`, handled by
`services/atlas-channel/atlas.com/channel/socket/handler/mount_food.go`
(`MountFoodHandle`, registered in the gms_83/84/87/95 and jms_185 seed templates), and the
resulting `TamingMobFed` event is applied by
`services/atlas-mounts/.../kafka/consumer/food/consumer.go` (tiredness/exp). What has never
been done is **byte-level verification** of the packet against each client version: STATUS.md
row `USE_MOUNT_FOOD` (fname `CWvsContext::SendTamingMobFoodItemUseRequest`) is ❌ in all five
matrix columns — gms_v83 (0x04D), gms_v84 (0x04D), gms_v87 (0x050), gms_v95 (0x053),
jms_v185 (0x045) — with `status.json` recording each cell as `incomplete` / "no audit report".

This task is a verification campaign, not a feature build: promote all five `USE_MOUNT_FOOD`
cells to ✅ by following `docs/packets/audits/VERIFYING_A_PACKET.md` per version — derive the
client's send order from each version's IDB, harvest the send function into the checked-in
IDA export, add `packet-audit:verify` byte-fixture markers to the existing codec test, generate
the per-version audit reports, and regenerate the matrix.

The v92 half of the original backlog item stays **parked**: no v92 IDB exists, so the v92
inbound opcode remains unverifiable, `MountFoodHandle` stays unregistered in
`template_gms_92_1.json`, and no v92 column is added to the matrix. (The v92 clientbound
writer `SetTamingMobInfo` 0x31 is already present and is untouched.) Per project grounding
rules, the v92 opcode must not be inferred from the v87→v95 shift pattern.

## 2. Goals

Primary goals:
- Promote all five `USE_MOUNT_FOOD` serverbound matrix cells (gms_v83, gms_v84, gms_v87,
  gms_v95, jms_v185) from ❌ to ✅ with byte-level evidence.
- Confirm (or correct) the template-registered opcodes for `MountFoodHandle` in each of the
  five versions against the IDB-derived send opcode.
- Confirm (or correct) the decode order in `libs/atlas-packet/mount/serverbound/food.go`
  against each client's actual send order.
- Leave `packet-audit matrix --check` (and related `--check` gates) exit 0.

Non-goals:
- Any v92 work: no template registration, no matrix column, no opcode inference. v92 unblocks
  only when a v92 IDB exists (tracked in project memory since task-086).
- Changes to feature behavior: `mount_food.go` handler logic, the `food` processor,
  `TamingMobFed` Kafka contract, and atlas-mounts consumption are out of scope unless
  verification proves a decode mismatch (see FR-4).
- Live tenant config patches — confirmed not needed for this task.
- Verifying the clientbound `SetTamingMobInfo` writer (separate matrix row).

## 3. User Stories

- As a maintainer of the packet coverage matrix, I want `USE_MOUNT_FOOD` byte-verified on
  every supported version so the ❌ row stops flagging implemented-but-unproven decode logic.
- As a developer touching the mount food path, I want fixture tests pinned to IDA evidence so
  a future refactor that breaks the wire format fails CI instead of silently corrupting reads.
- As the operator of version-specific tenants, I want the registered opcodes
  (0x4D/0x4D/0x50/0x53/0x45) proven against each client binary so a wrong registration is
  caught now rather than as a live "packet silently dropped" incident.

## 4. Functional Requirements

### FR-1: Per-version send-order derivation (5 versions)

For each of gms_v83, gms_v84, gms_v87, gms_v95, jms_v185:

1. Select the correct IDA instance for the version before reading (per CLAUDE.md
   RE rules); serialize IDB work — never two versions against the shared IDA server in
   parallel.
2. Locate `CWvsContext::SendTamingMobFoodItemUseRequest` (or the version's equivalent send
   function). The fname is currently **absent from all five checked-in exports**
   (`docs/packets/ida-exports/*.json` — only clientbound `CWvsContext::OnSetTamingMobInfo`
   is harvested); if the function is unnamed in an IDB, name it (producible work — do not
   defer). If it cannot be located at all in some IDB, that is a stop-and-ask escalation,
   never a substituted fname or faked hash.
3. Record the encode order (opcode + field writes) from the decompiled send function. The
   `COutPacket` opcode constant in the send function is the source of truth for the
   serverbound opcode — distrust IDB symbol names alone.

### FR-2: Export harvest (surgical splice)

Harvest the send function into each version's `docs/packets/ida-exports/<version>.json` via
the packet-audit harvest flow. The export is **non-idempotent**: splice surgically, never
regenerate/overwrite wholesale, and strip any `COutPacket`-delegate harvest artifacts.

### FR-3: Byte-fixture verification markers

Extend `libs/atlas-packet/mount/serverbound/food_test.go` with byte-fixture tests carrying
one `// packet-audit:verify packet=mount/serverbound/<PacketId> version=<key> ida=<addr>`
marker per version (five markers), following the existing convention (see
`libs/atlas-packet/storage/serverbound/operation_meso_test.go`). Fixtures must encode the
full body per the IDB-derived order, not just the opcode byte — mode/opcode-only enumeration
is an established false pass.

### FR-4: Discrepancy handling

- If a version's derived opcode differs from its template registration
  (`services/atlas-configurations/seed-data/templates/template_<version>_1.json`), fix the
  template in this task and call it out in the PR (existing tenants would then need a live
  config patch — surface that explicitly if it happens; confirmed not needed if opcodes match).
- If a version's send order differs from the shared `Food` codec's decode order, fix the codec
  (version-gated via `readerOptions`/major-version branches per existing atlas-packet
  patterns) and its handler wiring in the same task — no deferral, no follow-up-task split.
- `USE_MOUNT_FOOD` is `tier1: false` in `status.json`: pin evidence records only if the flow
  requires them for promotion or a deferral justification; do not create standing evidence
  liabilities the playbook says to avoid (VERIFYING_A_PACKET.md §7).

### FR-5: Matrix promotion

Regenerate STATUS.md / status.json via the packet-audit tooling (never hand-edit the matrix).
All five `USE_MOUNT_FOOD` cells read ✅; `packet-audit matrix --check` exits 0 with no
dangling-evidence or missing-report failures. Audit reports are generated with the root
`-ida-source docs/packets/ida-exports/<export>.json` per the serverbound flow, into the
correct per-version audit dirs (`docs/packets/audits/gms_v83|gms_v84|gms_v87|gms_v95|jms_v185`;
note jms tooling needs `--audit-dir docs/packets/audits/jms_v185` passed explicitly).

### FR-6: Parked-gap bookkeeping

Update the parked-v92 note (project memory / task docs as appropriate) to reflect that after
this task the only remaining mount-food gap is the v92 inbound registration, still blocked
solely on a v92 IDB.

## 5. API Surface

No REST or Kafka API changes. The only externally visible artifacts are documentation
(`docs/packets/audits/**`) and test files. Template opcode corrections (FR-4) would change
seed data consumed at tenant creation only.

## 6. Data Model

None. No entities, migrations, or schema changes.

## 7. Service Impact

- `libs/atlas-packet` — `mount/serverbound/food_test.go` gains five verify-marker fixtures;
  `food.go` changes only if FR-4 uncovers a decode mismatch.
- `docs/packets/` — five audit reports, export splices for the harvested fname, regenerated
  STATUS.md / status.json.
- `services/atlas-configurations` — seed template opcode fix only if FR-4 triggers.
- `services/atlas-channel`, `services/atlas-mounts` — no changes expected.

## 8. Non-Functional Requirements

- **Grounding**: every opcode/byte cited in fixtures and reports traces to a decompiled
  function address in the matching IDB; nothing inferred from version-shift patterns or
  general MapleStory knowledge.
- **Verification gates**: `go test -race ./...` and `go vet ./...` clean in
  `libs/atlas-packet`; `packet-audit matrix --check` (and `dispatcher-lint`/`fname-doc`
  checks if touched) exit 0. `docker buildx bake` only if any service `go.mod` is touched
  (none expected).
- **IDA discipline**: one version at a time against the shared IDA server; confirm
  `select_instance(port)` matches the target version before reading.
- **Commit hygiene**: per the playbook, each cell's test + report + matrix regen commit
  together; work stays on this task branch (no mid-task forks).

## 9. Open Questions

- Does a v84 IDA instance exist for live decompilation, or is the v84 cell derived from the
  checked-in `gms_v84.json` export plus a one-off IDB session? (v84 discovery was done in
  task-085, so an IDB existed then; confirm availability at execution time. If v84 is
  byte-identical to v83 for this packet, the evidence must still be v84-derived, not assumed.)
- Whether promotion of a non-tier-1 serverbound cell requires pinned evidence records or
  audit reports alone — resolve against VERIFYING_A_PACKET.md §7–10 during planning, not by
  memory.

## 10. Acceptance Criteria

- [ ] STATUS.md `USE_MOUNT_FOOD` row shows ✅ for gms_v83, gms_v84, gms_v87, gms_v95,
      jms_v185; `status.json` cells no longer `incomplete`.
- [ ] `libs/atlas-packet/mount/serverbound/food_test.go` contains five
      `packet-audit:verify` markers with real IDB addresses and full-body byte fixtures.
- [ ] `CWvsContext::SendTamingMobFoodItemUseRequest` (per-version equivalent) present in all
      five `docs/packets/ida-exports/*.json` via surgical splice.
- [ ] Five audit reports exist in the per-version audit dirs; `packet-audit matrix --check`
      exits 0.
- [ ] Template opcodes for `MountFoodHandle` confirmed against IDB-derived opcodes (corrected
      if mismatched, with live-tenant impact called out).
- [ ] No changes to v92 template, no v92 matrix column, no inferred v92 opcode; parked-gap
      note updated to reflect the narrowed remaining gap.
- [ ] `go test -race ./...` and `go vet ./...` clean in `libs/atlas-packet`.
