# MobEscortFullPath Wire Model Correction — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-04
Issue: https://github.com/Chronicle20/atlas/issues/1626
---

## 1. Overview

`libs/atlas-packet/monster/clientbound/mob_escort_full_path.go` models the clientbound
`MOB_ESCORT_FULL_PATH` packet (`CMob::OnEscortFullPath`) with **two** leading `int32`
fields — `mode`, then `count` — before the waypoint loop. The client reads **three**
unconditional `Decode4` ints before the loop. The first is the loop bound (`count`);
the second and third are stored to escort struct offsets `+1924` and `+1928` and are
absent from the codec entirely. So the model is both missing a field and ordering the
two it has incorrectly relative to the client read.

This is not a version-specific divergence. The three-leading-`Decode4` shape is present
identically in the checked-in IDA exports for all three columns where the op exists:
`gms_v92` (`CMob::OnEscortFullPath` @ `0x6374c0`), `gms_v95` (@ `0x643d90`), and
`jms_v185` (@ `0x6efa01`). Two of those columns — `gms_v95` and `jms_v185` — are marked
✅ verified in the coverage matrix against a single golden test that pins the wrong
2-leading-int shape. Their byte fixtures and pinned evidence therefore encode bytes the
client cannot correctly parse.

The defect was found during task-270 (`gms_v92` packet coverage bring-up), batch D-013b.
That work carried a standing constraint against making a wire change to an
already-verified version, so the batch agent reported the defect and withheld the
`gms_v92` cell rather than fixing the shared codec — the correct outcome, since the fix
is not a `gms_v92` version gate but a correction to a shared model three columns depend
on. This task is that fix, with its own review.

## 2. Goals

Primary goals:

- Correct `MobEscortFullPath` to the client's actual read order: `count` first, then the
  two previously-unmodelled `int32` fields, then the existing waypoint loop and tail.
- Re-derive the byte golden and re-pin evidence for the two already-verified columns
  (`gms_v95`, `jms_v185`) so their ✅ status rests on a correct model.
- Route `MOB_ESCORT_FULL_PATH` at `gms_v92` (opcode `0x128`) and promote that cell from
  ❌ to ✅ against the corrected codec.
- Name the two new fields from live IDA analysis of how offsets `+1924` / `+1928` are
  consumed, not from a placeholder.

Non-goals:

- Implementing an escort-mob emitter. `MobEscortFullPathBody` in `atlas-channel` is an
  intentional unwired seam and stays one.
- Any other escort op (`MobEscortStop`, `MobEscortStopSay`, `MobEscortReturnBefore`) or
  any other `CMob` coverage cell withheld by task-270.
- Adding the op to `gms_v83` / `gms_v84` / `gms_v87` — the escort family is absent there
  and those matrix cells remain ⬜.
- Any behavioral change in `atlas-channel` beyond the writer's parameter list.

## 3. User Stories

- As a packet engineer, I want `MobEscortFullPath` to encode the bytes the client
  actually reads, so that when an escort emitter is eventually written the client does
  not desync mid-packet.
- As a coverage-matrix consumer, I want the ✅ on `gms_v95` and `jms_v185` to mean the
  codec matches the client, so that a verified cell is trustworthy evidence rather than
  a pin on a disproved model.
- As the task-270 bring-up, I want `MOB_ESCORT_FULL_PATH` at `gms_v92` promoted, so the
  D-013 queue row closes without a permanently withheld cell.

## 4. Functional Requirements

### 4.1 Field derivation (prerequisite)

- FR-1. Derive the read order live from the IDBs, not from the checked-in export alone.
  The task-270 review (`review-task-21-D013b.md`, "Not evaluable") establishes that the
  exports record op sequence and guards but **not** field names or struct offsets; the
  `+1924` / `+1928` labels rest on a live IDB read that must be reproduced here.
  IDBs: `GMS_v92_1_DEVM.exe.i64` (@ `0x6374c0`), `GMS_v95.0_U_DEVM.exe.i64`
  (@ `0x643d90`), `MapleStory_dump_SCY.exe.i64` (jms_v185, @ `0x6efa01`).
- FR-2. Name the two new fields from observed behavior — how the escort struct fields at
  `+1924` and `+1928` (jms byte offsets `+0x764` / `+0x768`) are subsequently read or
  passed. Placeholder names of the form `field481` / `escortFieldA` are not acceptable
  on an exported accessor. If a field's semantics genuinely cannot be resolved from the
  disassembly, record that in §9 and raise it rather than shipping a guess.
- FR-3. Confirm whether the codec's current `mode` field corresponds to any real wire
  field or is purely an artifact of the mis-derivation. Per the task-270 finding the
  first wire int is the loop bound; if `mode` has no wire counterpart it is removed, not
  renamed.
- FR-4. Confirm the post-loop tail is unchanged: `Decode4` tail, `Decode1` hasArrive
  [+ `Decode4` arriveDelay], `Decode1` hasReset. Both the exports and the task-270 report
  agree this section matches today's codec; the derivation must re-confirm rather than
  assume.
- FR-5. Confirm the waypoint loop body is unchanged: `x`, `y`, `kind` int32s with an
  extra `int32` only when `kind == 2`.

### 4.2 Codec correction

- FR-6. `MobEscortFullPath` gains one `int32` field and drops or repurposes `mode` per
  FR-3, with the leading section ordered exactly as the client reads it.
- FR-7. Both `Encode` and `Decode` are updated symmetrically; the struct stays immutable
  with a `New…` constructor and value-receiver accessors, matching the existing file.
- FR-8. The waypoint count is still derived from `len(waypoints)` on encode and drives
  the loop on decode — the count field is not stored redundantly on the struct.
- FR-9. The doc comment above the struct is rewritten to the corrected layout, citing the
  three IDA addresses. The stale "The harvest summary 8×Decode4 + Decode1 + Decode4 +
  Decode1 corresponds to…" paragraph, which arithmetically encodes the wrong 2-field
  shape, is corrected to the 3-field arithmetic.
- FR-10. No `MajorAtLeast` version gate is introduced unless the live derivation shows an
  actual per-version divergence among v92/v95/jms. The current evidence says the three
  columns share one shape; a gate added without a derived divergence is a defect.

### 4.3 Fixtures and evidence

- FR-11. `TestMobEscortFullPath` in `mob_escort_full_path_test.go` is re-derived against
  the corrected layout. The golden byte array and its per-line comments must both reflect
  the new field order.
- FR-12. The two existing `packet-audit:verify` markers (`gms_v95` @ `0x643d90`,
  `jms_v185` @ `0x6efa01`) are retained, and a third is added for `gms_v92` @ `0x6374c0`.
  Whether that is one test with three markers or separate per-version tests is a design
  decision; either is acceptable if `packet-audit matrix --check` exits 0.
- FR-13. Evidence records at `docs/packets/evidence/gms_v95/...yaml` and
  `docs/packets/evidence/jms_v185/...yaml` are re-pinned — `decompile_sha256` must be
  regenerated from the current decompile, not carried forward.
- FR-14. A new evidence record is created at
  `docs/packets/evidence/gms_v92/monster.clientbound.MonsterMobEscortFullPath.yaml`,
  category `TIER1-FIXTURE`, `--ida "CMob::OnEscortFullPath"`, address `0x6374c0`.
- FR-15. The round-trip assertion over `pt.Variants` is preserved.

### 4.4 gms_v92 routing and promotion

- FR-16. Add the `MOB_ESCORT_FULL_PATH` row to `docs/packets/registry/gms_v92.yaml` at
  opcode `0x128` with writer `MobEscortFullPath` and fname `CMob::OnEscortFullPath`, if
  not already present.
- FR-17. Add the corresponding entry to the `gms_v92` seed template under
  `services/atlas-configurations/seed-data/templates/`, matching the shape of the
  existing `template_gms_95_1.json` (`0x130`) and `template_jms_185_1.json` (`0x110`)
  rows: `opCode`, `writer`, `fname`, `services: ["channel"]`.
- FR-18. Regenerate `docs/packets/audits/STATUS.md` and `status.json` via
  `packet-audit matrix`. The `MOB_ESCORT_FULL_PATH` row's `gms_v92` cell moves ❌ → ✅.
- FR-19. No other matrix cell may move. A diff of `STATUS.md` over the branch must show
  only this row's `gms_v92` cell plus the `gms_v92` summary-row totals.

### 4.5 Consumer update

- FR-20. `MobEscortFullPathBody` in
  `services/atlas-channel/atlas.com/channel/socket/writer/mob_escort_full_path.go` is
  updated to the corrected parameter list. No compatibility shim and no deprecated
  overload — nothing calls it (verified: no caller outside the writer file itself).
- FR-21. Its doc comment is updated to state the corrected field list and to record that
  the op is also routed at `gms_v92` after this change.

## 5. API Surface

No HTTP or JSON:API surface changes. The affected surfaces are:

**Go (breaking, internal):**

```go
// libs/atlas-packet/monster/clientbound
func NewMobEscortFullPath(<corrected leading fields>, waypoints []MobEscortWaypoint,
    tail int32, hasArrive bool, arriveDelay int32, hasReset bool) MobEscortFullPath

// services/atlas-channel/.../socket/writer
func MobEscortFullPathBody(<corrected leading fields>, waypoints []monsterpkt.MobEscortWaypoint,
    tail int32, hasArrive bool, arriveDelay int32, hasReset bool) packet.Encode
```

The exact leading-field parameter list is fixed by the FR-1/FR-2 derivation; the design
doc names them. `MobEscortWaypoint` and its accessors are unchanged.

**Wire (clientbound, breaking):** the on-wire byte layout of `MOB_ESCORT_FULL_PATH`
changes for `gms_v95` (`0x130`) and `jms_v185` (`0x110`) and is newly defined for
`gms_v92` (`0x128`). No emitter sends this packet today, so no live traffic changes.

**Seed data:** one new opcode row in the `gms_v92` configuration template.

## 6. Data Model

No database entities, no migrations, no `tenant_id` scoping — this is a packet codec and
its verification artifacts. The only persisted-data change is the `gms_v92` seed template
row (FR-17), which is configuration seed data, not tenant state.

## 7. Service Impact

| Area | Change |
|---|---|
| `libs/atlas-packet` | `monster/clientbound/mob_escort_full_path.go` — struct, constructor, accessors, `Encode`, `Decode`, doc comment. `mob_escort_full_path_test.go` — golden + markers. |
| `services/atlas-channel` | `socket/writer/mob_escort_full_path.go` — signature + doc comment. No handler, emitter, or processor change. `main.go` already references the writer name. |
| `services/atlas-configurations` | One new opcode row in the `gms_v92` seed template. |
| `docs/packets` | New `gms_v92` evidence record; re-pinned `gms_v95` and `jms_v185` records; `registry/gms_v92.yaml` row; regenerated `audits/STATUS.md` + `audits/status.json`. |
| All other services | None. |

## 8. Non-Functional Requirements

- **Verification.** `go run ./tools/packet-audit matrix --check` must exit 0 at the tip.
  `tools/verify.sh` (flagless) must exit 0 before the branch is called done.
- **Evidence integrity.** A re-pinned evidence record must carry a `decompile_sha256`
  regenerated from the decompile actually read in this task. Carrying the old hash forward
  while changing the model is a silent evidence lie and fails review.
- **No collateral movement.** Per FR-19, exactly one matrix cell changes state. Any other
  cell moving indicates the matrix regeneration picked up unrelated drift and must be
  investigated before commit.
- **Observability / security / multi-tenancy.** No new logging, no auth surface, no
  tenant-scoped data. The seed-template row is tenant-template configuration and follows
  the existing per-version template convention.
- **Backward compatibility.** None required — the corrected bytes are the only correct
  bytes; the previous shape was never emitted by any running service.

## 9. Open Questions

- **Semantics of `+1924` and `+1928`.** The task-270 review explicitly marks the field
  labels as not evaluable from checked-in artifacts. The design phase must resolve these
  from a live IDB read (FR-1/FR-2). If either resists resolution, surface it rather than
  naming it speculatively.
- **Does `mode` survive?** The task-270 report states the real wire is `count` first, then
  two fields "the codec never writes at all" — implying `mode` was a mislabeling of
  `count`. To be confirmed by FR-3, not assumed.
- **Marker layout.** One test with three version markers vs. per-version tests — a design
  decision, constrained only by `matrix --check` exiting 0.
- **`gms_v92` registry row.** Whether `docs/packets/registry/gms_v92.yaml` already carries
  a `0x128` row (possibly as an unrouted or stale entry) must be checked before adding one;
  task-270 recorded the opcode as `0x128` but withheld the cell.

## 10. Acceptance Criteria

- [ ] The read order at all three addresses (`0x6374c0` / `0x643d90` / `0x6efa01`) is
      re-derived live and recorded in the design doc, including what `+1924` and `+1928`
      are used for.
- [ ] `MobEscortFullPath` encodes and decodes three leading `int32`s in the client's order,
      with the two new fields named from derived semantics.
- [ ] The struct doc comment describes the corrected layout; the stale `8×Decode4`
      arithmetic paragraph is corrected.
- [ ] `TestMobEscortFullPath`'s golden bytes match the corrected layout, and its inline
      comments annotate the new field order.
- [ ] `packet-audit:verify` markers exist for `gms_v95` @ `0x643d90`, `jms_v185` @
      `0x6efa01`, and `gms_v92` @ `0x6374c0`.
- [ ] `docs/packets/evidence/gms_v95/...` and `.../jms_v185/...` carry regenerated
      `decompile_sha256` values.
- [ ] `docs/packets/evidence/gms_v92/monster.clientbound.MonsterMobEscortFullPath.yaml`
      exists, `TIER1-FIXTURE`, address `0x6374c0`.
- [ ] `docs/packets/registry/gms_v92.yaml` and the `gms_v92` seed template both route
      `MOB_ESCORT_FULL_PATH` at `0x128` to writer `MobEscortFullPath`.
- [ ] `STATUS.md` shows `MOB_ESCORT_FULL_PATH` ✅ at `gms_v92`, and a branch diff of
      `STATUS.md` shows no other row's cell changing.
- [ ] `go run ./tools/packet-audit matrix --check` exits 0 with a clean `git status` after.
- [ ] `MobEscortFullPathBody` compiles against the new constructor; `atlas-channel` builds.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before the PR is opened.
