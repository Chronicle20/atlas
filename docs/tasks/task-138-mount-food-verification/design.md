# Mount Food Packet Verification — Design

Task: task-138-mount-food-verification
Status: Approved PRD → design
Created: 2026-07-09

## 1. What this is

A five-cell verification campaign for the serverbound `USE_MOUNT_FOOD` packet
(fname `CWvsContext::SendTamingMobFoodItemUseRequest`), promoting the STATUS.md
row from ❌❌❌❌❌ to ✅✅✅✅✅ across gms_v83, gms_v84, gms_v87, gms_v95,
jms_v185. No feature code changes are expected: the codec
(`libs/atlas-packet/mount/serverbound/food.go`), handler, and templates already
exist. The deliverables are IDA-derived evidence artifacts (export splices,
audit reports, byte-fixture markers) plus one small tooling change that makes
the packet gradeable at all.

## 2. Resolved open questions (from PRD §9)

### 2.1 Evidence pinning: NOT required — grader-proven, not remembered

`status.json` records `USE_MOUNT_FOOD` as `tier1: false`. The grader
(`tools/packet-audit/internal/matrix/grade.go`, tier-0 branch at the end of
`gradeCore`) promotes a tier-0 cell to `StateVerified` on exactly two
conditions:

- `toolPass` — an audit report exists for the version with
  `Verdict == Match` and `!FlatInvalid`, and
- `marker.Found` — a `packet-audit:verify` marker links the packet × version.

Evidence records never participate in tier-0 promotion (they only enable the
`evidence-pinned deferral` partial state). Per VERIFYING_A_PACKET.md §7, a
pinned record is a standing freshness liability — export churn would degrade
the cell. **Decision: pin nothing.** Each cell needs exactly two artifacts:
the audit report and the marker. The playbook §9 "THREE artifacts" wording
applies to tier-1 serverbound cells; this row is tier-0.

Consequence for the marker check: `matrix --check`'s orphan-marker rule
requires the marker's `ida=` address to match the **audit report** address
(there is no evidence record to match). The address in each marker must be the
harvested function address for that version's export entry.

### 2.2 v84 IDB availability: resolve at execution, with a defined escalation

The v84 registry entry (opcode 77) carries a `discover-ops`-corrected note, so
a v84 IDB existed as recently as task-085. At execution time, enumerate live
instances (`mcp__ida-pro__list_instances`) and confirm one has GMS v84.1
loaded. If no v84 instance can be brought up, that is a **genuine blocker**
(missing IDB, per playbook "producible vs genuine"): stop and ask. Do not
derive the v84 cell from v83 bytes — the PRD forbids assumed byte-identity;
the evidence must come from the v84 binary even if the answer turns out
identical.

## 3. The one structural gap: tooling linkage (must-do before anything grades)

`CWvsContext::SendTamingMobFoodItemUseRequest` has **no case in
`candidatesFromFName`** (`tools/packet-audit/cmd/run.go`), so report
generation cannot map the fname to the Atlas codec — this is why every cell
reads "no audit report" today. Per playbook §9, a new serverbound op requires
its primary fname added as a case:

```go
case "CWvsContext::SendTamingMobFoodItemUseRequest":
    // USE_MOUNT_FOOD — taming-mob (mount) food. Codec mount/serverbound/Food
    // (handle MountFoodHandle). ts u32 + slot i16 + itemId u32 per <version notes>.
    return []candidate{{name: "Food", pkg: "mount"}}
```

- `locateAtlasFile` resolves `type Food struct` under `mount/serverbound/`
  (the only file in the pkg — no ambiguity).
- `qualifiedWriterName("mount", "Food")` = `MountFood` → report file
  `docs/packets/audits/<version>/MountFood.{json,md}` and marker path
  `packet=mount/serverbound/MountFood`.
- No `reportName` override needed: there is no clientbound `mount` pkg, so no
  cross-direction name collision (the SummonMove problem does not apply).

Alternative considered — thin wrapper struct per op (§9 shared-model
pattern): rejected. That pattern exists for *shared* decoders serving several
ops; `Food` is a single-op codec already shaped correctly. Direct linkage is
the minimal change.

## 4. Per-version pipeline (×5, strictly serialized)

The same five-step pass per version, in order **v83 → v84 → v87 → v95 →
jms_v185**. Serialization is mandatory: `select_instance` is shared global
state on the IDA server, all five markers land in the one `food_test.go`, and
the matrix regen is global — parallel per-version agents would collide on all
three. One owner, sequential passes.

Per version:

1. **Select + locate.** `select_instance` the version's IDB; verify the loaded
   binary matches. `func_query` with `name_regex` for
   `SendTamingMobFoodItemUseRequest`. If unnamed in that IDB, name it via the
   §10 byte-signature (`6A <op> 8D 8D ?? ?? ?? ?? E8`) + structure-match to a
   named twin — producible work, not a blocker. If genuinely unlocatable after
   attempting that: stop-and-ask (never substitute a fname).
2. **Derive send order.** Decompile; record the `COutPacket::COutPacket(&pkt,
   OPCODE)` integer (the opcode ground truth — distrust the symbol and the
   csv-seeded registry alike) and the ordered field writes with their widths.
   Descend into helper writes as the exporter would.
3. **Harvest + splice.** Harvest the send function to a temp export
   (`-prior-export "" -pending <roster> -descent-depth 12`), then surgically
   splice ONLY the new entries into the committed
   `docs/packets/ida-exports/<version>.json` (jms uses `gms_jms_185.json`;
   its retail dump is SMC — use the `*_U_DEVM` build's instance). Strip any
   `{op: Delegate, ref: COutPacket}` harvest artifact from the spliced entry
   (known report-gen killer, §10). Never regenerate an export wholesale.
4. **Generate the report.** Root command with
   `-ida-source docs/packets/ida-exports/<export>.json` and
   `-template services/atlas-configurations/seed-data/templates/template_<v>_1.json`
   to a temp `-output`, then copy only `MountFood.{json,md}` into
   `docs/packets/audits/<version dir>/` (jms dir is `jms_v185` — copy
   explicitly, don't trust default dir naming). The report must grade
   `Verdict: Match` against the codec; a non-Match is a real divergence →
   §5 discrepancy handling before proceeding.
5. **Fixture + marker + regen + commit.** Add/extend the byte-fixture test
   (§6 below) with this version's marker, run
   `go run ./tools/packet-audit matrix` then `matrix --check`, confirm the
   cell flipped ✅ with **no new** orphan/dangling/stale/drift lines and no
   conflict-count increase, and commit the version's artifacts together
   (test + export splice + report + STATUS.md/status.json).

Alternative considered — one big final commit after all five passes:
rejected. The playbook's unit of consistency is the cell (§8 "commit test +
evidence + STATUS together"); per-version commits keep every intermediate
tree state green for `matrix --check` and make a mid-campaign stop clean.

Alternative considered — fan-out to five `packet-verifier` subagents:
rejected for this row (shared IDA instance, shared test file, shared matrix —
three serialization points make fan-out all contention, no parallelism).

## 5. Discrepancy branches (decision table, from FR-4)

Current baseline, verified against the repo: registry opcodes 77/77/80/83/69
(0x4D/0x4D/0x50/0x53/0x45) **already match** all five seed-template
`MountFoodHandle` registrations, so branch (a) fires only if the IDB
contradicts both.

| Finding in IDB | Action (same task, same branch) |
|---|---|
| (a) `COutPacket` opcode ≠ registry/template value | Fix registry YAML + seed template; call out in PR that existing tenants need a live config patch (new-opcodes-not-in-live-config incident class). |
| (b) Send order ≠ `ts u32, slot i16, itemId u32` in some version | Wire fix FIRST as its own commit (playbook §4): version-gate the codec via `MajorVersion()` branches per existing atlas-packet patterns — and beware the v84 class: a gate that differs v83→v87 must be `>=87`, not `>83`, unless the v84 IDB says otherwise. Handler wiring in atlas-channel only changes if the decoded field set changes. |
| (c) Function absent from an IDB entirely | Stop-and-ask with the search evidence (regex + byte-signature attempts). Never fake a hash or borrow another version's address. |

Branch (b) is the only one that touches feature code; it stays in-task (no
follow-up split), and `services/atlas-channel` / `atlas-mounts` remain
untouched unless the decoded fields themselves change meaning.

## 6. Test design

Extend `libs/atlas-packet/mount/serverbound/food_test.go` following the
`storage/serverbound/operation_meso_test.go` convention: one test function
carrying all five markers stacked above it:

```go
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v83 ida=0x<addr>
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v84 ida=0x<addr>
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v87 ida=0x<addr>
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v95 ida=0x<addr>
// packet-audit:verify packet=mount/serverbound/MountFood version=jms_v185 ida=0x<addr>
func TestFoodByteFixture(t *testing.T) { ... }
```

Body: a `pt.Variants`-driven table (v83/v84/v87/v95/jms185 variants exist in
`libs/atlas-packet/test/context.go`) asserting, per variant:

- **exact expected raw bytes** for a concrete model (full body — ts, slot,
  itemId — hand-computed little-endian, each field's comment citing the
  decompile line/address it traces to, per §5 of the playbook; never
  opcode-only, per the mode-byte false-pass rule), and
- decode round-trip of those bytes back into the model fields.

If all five versions prove byte-identical, the expected bytes are the same
per variant but each variant's citation is its own version's decompile — the
evidence is per-version even when the layout isn't. The existing
`TestFoodDecode` stays (it pins the handler-facing decode contract); the new
fixture test carries the markers.

Markers are added **incrementally** — each version's marker lands in the same
commit as its report, so `matrix --check` never sees a marker whose address
has no matching report (orphan-marker failure mode).

## 7. Verification gates (exit criteria)

- `go test -race ./...` and `go vet ./...` clean in `libs/atlas-packet` (and
  `tools/packet-audit` for the run.go case + its tests, e.g. the
  disambiguation test table if it enumerates candidates).
- `go run ./tools/packet-audit matrix --check`: all five `USE_MOUNT_FOOD`
  cells ✅; zero new orphan/dangling/stale/drift lines; conflict count not
  increased (pre-existing 🟥 backlog may keep exit ≠ 0 — the bar is
  no-new-problems, per §8 note).
- No `docker buildx bake` needed unless a service `go.mod` changes (none
  expected — branch (b) touches only `libs/atlas-packet`; template JSON edits
  under branch (a) don't change `go.mod` either).
- FR-6 bookkeeping: update the parked-v92 note (project memory topic
  `project_v92_mount_food_parked`) to record that the only remaining
  mount-food gap is the v92 inbound registration, still blocked solely on a
  v92 IDB.

## 8. Out of scope (restated hard lines)

- Anything v92: no template edit, no matrix column, no opcode inference from
  the v87→v95 shift.
- Clientbound `SetTamingMobInfo` (separate matrix row).
- Handler/processor/Kafka behavior in atlas-channel and atlas-mounts.

## 9. Risk register

| Risk | Mitigation |
|---|---|
| Export splice drifts unrelated keys | Temp-file harvest + surgical splice only; diff the export before commit and confirm only the new entries changed. |
| `COutPacket` delegate artifact blocks report-gen | Strip it from the spliced entry (documented fix, §10). |
| Report grades non-Match on a version | That IS the finding — branch (b): wire fix first, own commit, then re-generate. |
| jms retail-dump SMC instance selected by mistake | Confirm the `*_U_DEVM` IDB via `list_instances` metadata before decompiling. |
| Marker committed before its report | Per-version atomic commits (test+report+matrix together) make this unrepresentable. |
