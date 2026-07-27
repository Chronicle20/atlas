# Mount Food Packet Verification — Product Requirements Document

Version: v2
Status: Draft
Created: 2026-07-09
Updated: 2026-07-25 (scope expanded to the legacy version columns v48/v61/v72/v79)
---

## 1. Overview

The mount food feature (task-086-mount-system) is fully implemented server-side: the
serverbound `USE_MOUNT_FOOD` packet is decoded by
`libs/atlas-packet/mount/serverbound/food.go`, handled by
`services/atlas-channel/atlas.com/channel/socket/handler/mount_food.go`
(`MountFoodHandle`), and the resulting `TamingMobFed` event is applied by
`services/atlas-mounts/.../kafka/consumer/food/consumer.go` (tiredness/exp). What has never
been done is **byte-level verification** of the packet against each client version.

This task is a verification campaign, not a feature build: promote every applicable
`USE_MOUNT_FOOD` matrix cell to ✅ by following `docs/packets/audits/VERIFYING_A_PACKET.md`
per version — derive the client's send order from each version's IDB, harvest the send
function into the checked-in IDA export, add `packet-audit:verify` byte-fixture markers to the
codec test, generate the per-version audit reports, and regenerate the matrix.

### 1.1 Scope change (2026-07-25)

The original PRD (v1, 2026-07-09) targeted **five** columns (gms_v83, gms_v84, gms_v87,
gms_v95, jms_v185), because those were the only matrix columns when it was written. Since then
main has brought up the legacy version columns, so the `USE_MOUNT_FOOD` row now spans **nine**
columns. Per the maintainer's direction, this task now covers **all nine**: the original five
plus **gms_v48, gms_v61, gms_v72, gms_v79**.

Grounding done at scope-expansion time (all four legacy senders are symbol-named in the DEVM
IDBs — `has_type: true`, no renaming contingency needed):

| Version | Send fn | Address (DEVM IDB) | Opcode | Source |
|---|---|---|---|---|
| gms_v48 | `CWvsContext::SendTamingMobFoodItemUseRequest` | `0x70e00b` | **0x3D (61)** | **decompile-verified 2026-07-25** |
| gms_v61 | `CWvsContext::SendTamingMobFoodItemUseRequest` | `0x831f44` | 0x4C (76) | registry `ida-discovered` (re-derive at exec) |
| gms_v72 | `CWvsContext::SendTamingMobFoodItemUseRequest` | `0x904419` | 0x4C (76) | registry `ida-discovered` (re-derive at exec) |
| gms_v79 | `CWvsContext::SendTamingMobFoodItemUseRequest` | `0x955781` | 0x4B (75) | registry `ida-discovered` (re-derive at exec) |
| gms_v83 | `CWvsContext::SendTamingMobFoodItemUseRequest` | derive at exec | 0x4D (77) | registry/template |
| gms_v84 | `CWvsContext::SendTamingMobFoodItemUseRequest` | derive at exec | 0x4D (77) | registry/template |
| gms_v87 | `CWvsContext::SendTamingMobFoodItemUseRequest` | derive at exec | 0x50 (80) | registry/template |
| gms_v95 | `CWvsContext::SendTamingMobFoodItemUseRequest` | derive at exec | 0x53 (83) | registry/template |
| jms_v185 | `CWvsContext::SendTamingMobFoodItemUseRequest` | derive at exec | 0x45 (69) | registry/template |

**The v48 cell is a correction, not just a verification.** The matrix currently records
gms_v48 as `n-a` (opcode -1). That is a **false negative** inherited from the legacy bring-up:
the v48 DEVM IDB contains `?SendTamingMobFoodItemUseRequest@CWvsContext@@QAEXJJ@Z` at
`0x70e00b`, sending opcode `0x3D` with the identical `update_time u32 / slot i16 / itemId u32`
body and a `nItemID/10000 == 226` (taming-mob food category) guard. v48 also carries the
clientbound `OnSetTamingMobInfo` (op 40) and the pet-food twin `SendPetFoodItemUseRequest`.
The `n-a` arose only because the checked-in `gms_v48.json` export is a selective harvest that
never captured this sender — absence from the export is not absence from the binary. So the v48
cell requires: adding the op to the `gms_v48.yaml` registry, registering `MountFoodHandle`
(opcode `0x3D`, no collision — 0x3D is unused in `template_gms_48_1.json`), splicing the export,
and promoting the matrix `n-a → verified`.

**The v61/v72/v79 cells are plain verifications.** Their seed templates already register
`MountFoodHandle` (v61/v72 at `0x4C`, v79 at `0x4B`) and their registries already carry
`ida-discovered` `USE_MOUNT_FOOD` opcodes — they need the same byte-fixture + audit-report pass
as the original five.

### 1.2 Codec is version-invariant

The v48 decompile confirms the send body is byte-for-byte the same order as v83+
(`Encode4(update_time), Encode2(slot/nPOS), Encode4(nItemID)`). Only the **opcode** differs
per version, and the opcode is config-resolved (seed template / registry), not encoded in the
codec. Therefore `libs/atlas-packet/mount/serverbound/food.go` needs **no version gating** and
is not expected to change (FR-4 branch (b) is a contingency only).

### 1.3 Parked / out-of-scope versions

- **gms_12** stays parked: **no v12 IDB exists**, so its opcode is unverifiable. Its template
  does not register `MountFoodHandle` and it is not a matrix column. (v12 is a login-minimal
  template whose `MajorVersion < 83` lands in the legacy guard regardless.)
- **gms_92** is **out of scope for this task's matrix work** but for a different reason than
  before: a v92 IDB now exists and the opcode **is verified** (`0x54`, v92
  `SendTamingMobFoodItemUseRequest @0x9ab430`). However, **gms_92 is not a packet-matrix
  column** (`matrix.VersionKeys` tracks 9 columns; gms_92 has no registry/export/audit dir), so
  there is no cell to promote. Closing the v92 gap is a one-line seed-template registration
  (`{opCode:"0x54", validator:"LoggedInValidator", handler:"MountFoodHandle", services:["channel"]}`
  in `template_gms_92_1.json`) — see Open Questions §9. It is **not** part of the verification
  scope unless the maintainer folds it in.

## 2. Goals

Primary goals:
- Promote **all nine** `USE_MOUNT_FOOD` serverbound matrix cells (gms_v48, gms_v61, gms_v72,
  gms_v79, gms_v83, gms_v84, gms_v87, gms_v95, jms_v185) to ✅ with byte-level evidence.
- Correct the false `gms_v48` `n-a`: add the registry op, register the template handler, and
  promote the cell to verified.
- Confirm (or correct) the template-registered opcodes for `MountFoodHandle` in each version
  against the IDB-derived send opcode.
- Confirm the decode order in `libs/atlas-packet/mount/serverbound/food.go` against each
  client's actual send order.
- Leave `packet-audit matrix --check` (and related `--check` gates, including the feature-family
  n-a consistency gate) exit 0 with no new problems.

Non-goals:
- **gms_12**: no template registration, no matrix column, no opcode inference. Parked on the
  absence of a v12 IDB.
- **gms_92 matrix work**: gms_92 is not a matrix column; no verification cell exists. (Optional
  template registration only if the maintainer folds it in — see §1.3 / §9.)
- Changes to feature behavior: `mount_food.go` handler logic, the `food` processor,
  `TamingMobFed` Kafka contract, and atlas-mounts consumption are out of scope unless
  verification proves a decode mismatch (see FR-4).
- Verifying the clientbound `SetTamingMobInfo` writer (separate matrix row).

## 3. User Stories

- As a maintainer of the packet coverage matrix, I want `USE_MOUNT_FOOD` byte-verified on
  every supported version so the row stops flagging implemented-but-unproven decode logic.
- As a developer touching the mount food path, I want fixture tests pinned to IDA evidence so
  a future refactor that breaks the wire format fails CI instead of silently corrupting reads.
- As the operator of version-specific tenants, I want the registered opcodes
  (0x3D/0x4C/0x4C/0x4B/0x4D/0x4D/0x50/0x53/0x45) proven against each client binary so a wrong
  registration is caught now rather than as a live "packet silently dropped" incident.
- As the operator of a **v48** tenant, I want mount feeding actually routed — the missing
  `MountFoodHandle` registration means feeding a mount is currently unhandled on v48.

## 4. Functional Requirements

### FR-1: Per-version send-order derivation (9 versions)

For each of gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v95, jms_v185:

1. Select the correct IDB session before reading (per CLAUDE.md RE rules); the session-based
   IDA API scopes reads to the `database` id, but still serialize the shared mutating steps
   (test file, export splice, matrix regen).
2. Locate `CWvsContext::SendTamingMobFoodItemUseRequest`. For the four legacy versions the
   function is already symbol-named at the addresses in §1.1; for the original five, locate via
   `name_regex` (name it if unnamed — producible work, not a defer). If it cannot be located at
   all in some IDB, that is a stop-and-ask escalation, never a substituted fname or faked hash.
3. Record the encode order (opcode + field writes) from the decompiled send function. The
   `COutPacket::COutPacket(&pkt, OPCODE)` integer is the source of truth for the serverbound
   opcode — distrust the IDB symbol name and the csv-seeded registry alike.

### FR-2: Export harvest (surgical splice)

Harvest the send function into each version's `docs/packets/ida-exports/<version>.json` via
the packet-audit harvest flow. The export is **non-idempotent**: splice surgically, never
regenerate/overwrite wholesale, and strip any `COutPacket`-delegate harvest artifacts.

### FR-3: Byte-fixture verification markers

Extend `libs/atlas-packet/mount/serverbound/food_test.go` with byte-fixture tests carrying one
`// packet-audit:verify packet=mount/serverbound/MountFood version=<key> ida=<addr>` marker per
version (**nine** markers). Because the body is version-invariant (§1.2), every variant asserts
the same 10-byte body — but each variant's marker/comment cites its own version's decompile
address; the evidence is per-version even when the layout is not.

**Prerequisite:** `libs/atlas-packet/test/context.go` `Variants` has no v48/v61/v72/v79 entries
(only v28/v83/v87/v95/jms185/v84/v86). Append the four legacy variants (never insert — positional
`Variants[N]` references must stay valid) before writing their fixture cases.

### FR-4: Discrepancy handling

- **v48 registration (expected, not a discrepancy):** add `USE_MOUNT_FOOD` (opcode `0x3D`,
  `ida-discovered`, address `0x70e00b`) to `docs/packets/registry/gms_v48.yaml`, and register
  `{opCode:"0x3D", validator:"LoggedInValidator", handler:"MountFoodHandle", services:["channel"]}`
  in `services/atlas-configurations/seed-data/templates/template_gms_48_1.json`. This is
  producible unblock work, done in-task.
- If any other version's derived opcode differs from its template registration, fix the
  template in this task and call it out in the PR (existing tenants would then need a live
  config patch — surface that explicitly).
- If a version's send order differs from the shared `Food` codec's decode order, fix the codec
  (version-gated per existing atlas-packet patterns) and its handler wiring in the same task —
  no deferral. **Not expected**: the body is version-invariant per §1.2.
- `USE_MOUNT_FOOD` is `tier1: false`: pin evidence records only if the flow requires them for
  promotion; do not create standing evidence liabilities (VERIFYING_A_PACKET.md §7).

### FR-5: Matrix promotion

Regenerate STATUS.md / status.json via the packet-audit tooling (never hand-edit the matrix).
All nine `USE_MOUNT_FOOD` cells read ✅ (v48 transitions `n-a → verified` once its registry op
exists); `packet-audit matrix --check` exits 0 with no new dangling-evidence, missing-report,
or feature-family n-a-consistency failures. Audit reports are generated with the root
`-ida-source docs/packets/ida-exports/<export>.json` per the serverbound flow, into the correct
per-version audit dirs (jms tooling needs `--audit-dir docs/packets/audits/jms_v185` explicitly).

### FR-6: Parked-gap bookkeeping

Update the task docs and project memory to reflect the post-task state: all nine matrix cells
byte-verified; **gms_12** remains parked solely on the absence of a v12 IDB; **gms_92** mount
food is unblocked (IDB + opcode `0x54`) and reduced to a one-line template registration that is
out of this task's matrix scope.

## 5. API Surface

No REST or Kafka API changes. Externally visible artifacts are documentation
(`docs/packets/audits/**`, `docs/packets/registry/gms_v48.yaml`) and test files. Template
registrations (v48 always; others only under FR-4 correction) change seed data consumed at
tenant creation only.

## 6. Data Model

None. No entities, migrations, or schema changes.

## 7. Service Impact

- `libs/atlas-packet` — `mount/serverbound/food_test.go` gains nine verify-marker fixtures;
  `test/context.go` gains four legacy variants; `food.go` changes only if FR-4 branch (b)
  fires (not expected).
- `docs/packets/` — nine audit reports, export splices, regenerated STATUS.md/status.json, plus
  the `gms_v48.yaml` registry op.
- `services/atlas-configurations` — `template_gms_48_1.json` gains the `MountFoodHandle`
  registration (opcode `0x3D`); other templates only under an FR-4 correction.
- `services/atlas-channel`, `services/atlas-mounts` — no changes expected.

## 8. Non-Functional Requirements

- **Grounding**: every opcode/byte cited in fixtures and reports traces to a decompiled
  function address in the matching IDB; nothing inferred from version-shift patterns or memory.
- **Verification gates**: `go test -race ./...` and `go vet ./...` clean in `libs/atlas-packet`
  and `tools/packet-audit`; `tools/lint.sh --check` clean; `packet-audit matrix --check` exits
  0 (no new problems). `docker buildx bake` only if a service `go.mod` is touched (none expected
  — template JSON and registry YAML do not touch `go.mod`).
- **IDA discipline**: confirm the session id matches the target version before reading;
  serialize the shared mutating steps (test file, matrix regen).
- **Commit hygiene**: each cell's test marker + export splice + audit report + matrix regen
  commit together; work stays on this task branch (no mid-task forks).

## 9. Open Questions

- **gms_92 fold-in.** The v92 mount-food gap is now a one-line template registration (opcode
  `0x54`, verified) rather than a blocked unknown. Should this task also register
  `MountFoodHandle` in `template_gms_92_1.json` (cheap, closes the live gap) even though gms_92
  is not a matrix column and so gets no verification cell? **Default: no** — kept out unless the
  maintainer folds it in. (Recorded so it is not silently dropped.)
- v84 IDB availability at execution (present this turn: session `79511a2a`,
  `GMS_v84.1_U_DEVM`). If v84 is byte-identical to v83 for this packet, the evidence must still
  be v84-derived, not assumed.

## 10. Acceptance Criteria

- [ ] STATUS.md `USE_MOUNT_FOOD` row shows ✅ for **all nine** columns (gms_v48, gms_v61,
      gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v95, jms_v185); `status.json` cells no
      longer `incomplete`, and the gms_v48 cell is no longer `n-a`.
- [ ] `libs/atlas-packet/mount/serverbound/food_test.go` contains **nine** `packet-audit:verify`
      markers with real IDB addresses and full-body byte fixtures; `test/context.go` carries the
      four appended legacy variants.
- [ ] `CWvsContext::SendTamingMobFoodItemUseRequest` present in all nine
      `docs/packets/ida-exports/*.json` via surgical splice.
- [ ] `docs/packets/registry/gms_v48.yaml` has the `USE_MOUNT_FOOD` op (opcode `0x3D`,
      `ida-discovered @0x70e00b`); `template_gms_48_1.json` registers `MountFoodHandle` at
      `0x3D`.
- [ ] Nine audit reports exist in the per-version audit dirs; `packet-audit matrix --check`
      exits 0 (including the feature-family n-a consistency gate).
- [ ] Template opcodes for `MountFoodHandle` confirmed against IDB-derived opcodes (corrected
      if mismatched, with live-tenant impact called out).
- [ ] No gms_12 work and no inferred gms_12 opcode; gms_92 matrix work excluded; bookkeeping
      updated to reflect the narrowed gaps (v12 parked on IDB; v92 reduced to a template line).
- [ ] `go test -race ./...`, `go vet ./...`, and `tools/lint.sh --check` clean.
