# Mount Food Packet Verification — Design

Task: task-138-mount-food-verification
Status: Approved PRD → design
Created: 2026-07-09
Updated: 2026-07-25 (expanded from 5 to 9 version columns; v48 false-n-a correction)

## 1. What this is

A **nine-cell** verification campaign for the serverbound `USE_MOUNT_FOOD` packet (fname
`CWvsContext::SendTamingMobFoodItemUseRequest`), promoting the STATUS.md row to ✅ across
gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v95, jms_v185. No feature
code changes are expected: the codec (`libs/atlas-packet/mount/serverbound/food.go`), handler,
and (for v61+) templates already exist. The deliverables are IDA-derived evidence artifacts
(export splices, audit reports, byte-fixture markers), one small tooling change that makes the
packet gradeable, one legacy-variant addition to the test context, and the **v48 correction**
(registry op + template registration + matrix `n-a → verified`).

The v1 design covered five columns; the row has since grown to nine on main. See PRD §1.1.

## 2. Grounding captured at scope-expansion (2026-07-25)

All four legacy senders are **symbol-named** (`has_type: true`) in the DEVM IDBs, so the "if
unnamed, byte-signature + rename" contingency from the v1 design does **not** apply to them:

| Version | IDB session | Send fn address | Opcode | Evidence |
|---|---|---|---|---|
| gms_v48 | `0bb5f11a` (`GMS_v48_1_DEVM`) | `0x70e00b` | **0x3D (61)** | decompiled this turn: `COutPacket(v9, 61)` then `Encode4(update_time)·Encode2(slot)·Encode4(itemId)`, guard `nItemID/10000==226` |
| gms_v61 | `965202bf` (`GMS_v61.1_U_DEVM`) | `0x831f44` | 0x4C (76) | registry `ida-discovered`; symbol present |
| gms_v72 | `90e36cb0` (`GMS_v72.1_U_DEVM`) | `0x904419` | 0x4C (76) | registry `ida-discovered`; symbol present |
| gms_v79 | `9a7d3642` (`GMS_v79_1_DEVM`) | `0x955781` | 0x4B (75) | registry `ida-discovered`; symbol present |

Session ids are from this turn's `idb_list` and can change across relaunches — re-enumerate at
execution, never hardcode.

### 2.1 The codec is version-invariant

The v48 body is byte-for-byte the v83+ order (`update_time u32, slot i16, itemId u32`). Every
version differs only in the **opcode**, and the opcode is config-resolved (seed template +
registry), never in the codec. Consequence: `food.go` needs no `MajorVersion` gating and is
not expected to change; every byte-fixture variant asserts the **same 10-byte body**, each
citing its own version's decompile. Discrepancy branch (b) (§5) is a contingency only.

## 3. Resolved open questions

### 3.1 Evidence pinning: NOT required — grader-proven, not remembered

`status.json` records `USE_MOUNT_FOOD` as `tier1: false`. The tier-0 grader promotes a cell to
`StateVerified` on exactly two conditions: an audit report with `Verdict == Match` and
`!FlatInvalid`, plus a `packet-audit:verify` marker linking packet × version. Evidence records
never participate in tier-0 promotion. **Decision: pin nothing.** Each cell needs the audit
report + the marker, and the marker's `ida=` address must match the audit report address.

### 3.2 The v48 `n-a` is a false negative — correct it

The matrix records gms_v48 `USE_MOUNT_FOOD` as `n-a` (opcode -1) because the op is **absent
from `docs/packets/registry/gms_v48.yaml`** (the selective v48 export never harvested the
sender). n-a with opcode -1 = "op absent from the registry for this version." The fix is to
**add the op to the registry** (opcode `0x3D`, `ida-discovered @0x70e00b`); the matrix
generator then reclassifies the cell from `n-a` to `incomplete`, after which the standard
verify pass (report + marker) promotes it to `verified`. Because a same-feature sibling
(`SET_TAMING_MOB_INFO`, clientbound) exists on v48, this also clears any feature-family
n-a-consistency exposure (`na_consistency.go`) — verifying the cell is the clean resolution,
no `feature-na-evidence.yaml` entry needed.

### 3.3 v84 IDB availability

Present this turn (session `79511a2a`, `GMS_v84.1_U_DEVM`). Evidence must be v84-derived even
if byte-identical to v83; never copy v83's address/bytes.

## 4. Structural gap: tooling linkage (must-do before anything grades)

`CWvsContext::SendTamingMobFoodItemUseRequest` has **no case in `candidatesFromFName`**
(`tools/packet-audit/cmd/run.go`), so report generation cannot map the fname to the Atlas
codec — this is why every cell reads "no audit report" today. Add the case:

```go
case "CWvsContext::SendTamingMobFoodItemUseRequest":
    // USE_MOUNT_FOOD — taming-mob (mount) food. Codec mount/serverbound/Food
    // (handler MountFoodHandle). update_time u32 + slot i16 + itemId u32.
    return []candidate{{name: "Food", pkg: "mount", dir: csvpkg.DirServerbound}}
```

- `qualifiedWriterName("mount", "Food")` = `MountFood` → report `MountFood.{json,md}`, marker
  `packet=mount/serverbound/MountFood`. No `reportName` override needed (no clientbound `mount`
  pkg, so no name collision).

## 5. Per-version pipeline (×9, shared mutating steps serialized)

Order: **v48 → v61 → v72 → v79 → v83 → v84 → v87 → v95 → jms_v185** (ascending; legacy first so
the v48 correction — the only structural change — lands before the plain verifications).

Per version:

1. **Select + locate.** Select the version's IDB session; verify the loaded binary matches.
   `func_query name_regex` for `SendTamingMobFoodItemUseRequest`. Legacy four are pre-named at
   the §2 addresses. If unnamed (possible only for some of the original five): name via the
   `6A <op> 8D 8D ?? ?? ?? ?? E8` signature + structure-match, then `idb_save`. If genuinely
   unlocatable: stop-and-ask.
2. **Derive send order.** Decompile; record the `COutPacket(&pkt, OPCODE)` integer (opcode
   ground truth) and the ordered field writes.
3. **Harvest + splice.** Harvest to a temp export, surgically splice the new entries into the
   committed `docs/packets/ida-exports/<version>.json` (jms uses `gms_jms_185.json`). Strip any
   `{op: Delegate, ref: COutPacket}` artifact. Never regenerate wholesale.
4. **(v48 only) Registry + template.** Add `USE_MOUNT_FOOD` to `gms_v48.yaml`; register
   `MountFoodHandle` at `0x3D` in `template_gms_48_1.json` (verified no collision — 0x3D unused;
   78 existing handler entries). v61/v72/v79 already have both.
5. **Generate the report.** Root command with `-ida-source .../<export>.json` and
   `-template .../template_<v>_1.json` to a temp `-output`; copy `MountFood.{json,md}` into
   `docs/packets/audits/<version dir>/`. Must grade `Verdict: Match`; a non-Match is a real
   divergence → §6 before proceeding.
6. **Fixture + marker + regen + commit.** Append this version's variant case + marker to
   `food_test.go`, run `packet-audit matrix` then `matrix --check`, confirm the cell flipped ✅
   with no new orphan/dangling/stale/drift lines, and commit the version's artifacts together.

Alternative considered — fan-out to nine `packet-verifier` subagents: rejected. The shared test
file, shared matrix regen, and (for v48) shared registry/template are serialization points; the
session-scoped IDA reads could parallelize but the commit unit (cell) cannot.

## 6. Discrepancy branches (decision table, from FR-4)

Registry/template opcodes 0x4C/0x4C/0x4B/0x4D/0x4D/0x50/0x53/0x45 already match `MountFoodHandle`
registrations for v61–jms; v48 has no registration yet (added in step 4, not a discrepancy).

| Finding in IDB | Action (same task, same branch) |
|---|---|
| (a) `COutPacket` opcode ≠ registry/template value | Fix registry YAML + seed template to the IDB value; PR calls out that existing tenants need a live config patch. |
| (b) Send order ≠ `update_time u32, slot i16, itemId u32` | Wire fix FIRST as its own commit: version-gate the codec via `MajorAtLeast(N)` per atlas-packet patterns; update `TestFoodDecode`. **Not expected** — body is version-invariant (§2.1). |
| (c) Function absent from an IDB entirely | Stop-and-ask with the search evidence. Never fake a hash or borrow another version's address. |

## 7. Test design

Two changes to `libs/atlas-packet`:

1. **`test/context.go`** — append four legacy variants (never insert; positional refs must
   hold). After the existing `[6]`=v86, add `[7]`=GMS v48, `[8]`=GMS v61, `[9]`=GMS v72,
   `[10]`=GMS v79 (Region "GMS", the respective MajorVersion, MinorVersion 1).
2. **`mount/serverbound/food_test.go`** — one `TestFoodByteFixture` carrying all nine markers
   stacked above it, and a `cases` table with one entry per variant. Each case asserts the
   exact 10-byte body for a concrete model and decode round-trips it. Because the body is
   version-invariant, the `want` bytes are identical per variant — the per-version evidence is
   the marker/comment address, not a differing layout. The existing `TestFoodDecode` stays.

Markers are added incrementally — each version's marker lands in the same commit as its report,
so `matrix --check` never sees a marker whose address has no matching report.

## 8. Verification gates (exit criteria)

- `go test -race ./...` and `go vet ./...` clean in `libs/atlas-packet` and `tools/packet-audit`.
- `tools/lint.sh --check` clean (gofumpt/goimports/standard) — new-code-gated.
- `go run ./tools/packet-audit matrix --check`: all nine `USE_MOUNT_FOOD` cells ✅; v48 no
  longer `n-a`; zero new orphan/dangling/stale/drift/n-a-consistency lines; conflict count not
  increased.
- No `docker buildx bake` needed (no `go.mod` touched — registry YAML and template JSON don't
  touch `go.mod`; branch (b), if it ever fired, touches only `libs/atlas-packet`).
- FR-6 bookkeeping: update project memory — all nine cells verified; gms_12 parked on IDB;
  gms_92 reduced to a one-line template registration (out of matrix scope).

## 9. Out of scope (restated hard lines)

- **gms_12**: no template edit, no matrix column, no opcode inference. Parked on the v12 IDB.
- **gms_92 matrix work**: not a matrix column; no cell to promote. (Template registration is a
  separate, optional fold-in — PRD §9.)
- Clientbound `SetTamingMobInfo` (separate matrix row).
- Handler/processor/Kafka behavior in atlas-channel and atlas-mounts.

## 10. Risk register

| Risk | Mitigation |
|---|---|
| Export splice drifts unrelated keys | Temp-file harvest + surgical splice only; diff before commit. |
| `COutPacket` delegate artifact blocks report-gen | Strip it from the spliced entry. |
| v48 opcode 0x3D collides with an existing handler | Verified no collision (0x3D absent from `template_gms_48_1.json`); re-confirm at exec before adding. |
| Report grades non-Match on a version | That IS the finding — branch (b): wire fix first, own commit, then re-generate. |
| Legacy variant inserted (not appended) in test context | Append after `[6]`; break existing positional `Variants[N]` references otherwise. |
| Marker committed before its report | Per-version atomic commits make this unrepresentable. |
| v48 n-a not clearing after registry add | The generator derives n-a from registry absence; adding the op reclassifies to incomplete — verify via `matrix` regen, not by hand-editing status.json. |
