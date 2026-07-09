# Task 138 — Mount Food Packet Verification: Context

Companion to `plan.md`. Key files, decisions, and dependencies an implementer
needs, verified against the repo at planning time (2026-07-09).

## What this task is

Verification campaign only: promote the serverbound `USE_MOUNT_FOOD` row
(fname `CWvsContext::SendTamingMobFoodItemUseRequest`) from ❌ to ✅ in all
five coverage-matrix columns. The feature itself (task-086) is fully
implemented; no behavior changes expected.

## Key files

| File | Role |
|---|---|
| `libs/atlas-packet/mount/serverbound/food.go` | The codec under verification. `Food{updateTime uint32, slot int16, itemId uint32}`, unconditional LE `ts(4)+slot(2)+itemId(4)`. Changes only if a version's IDB proves a different order (branch b). |
| `libs/atlas-packet/mount/serverbound/food_test.go` | Gains `TestFoodByteFixture` with five `packet-audit:verify` markers. Existing `TestFoodDecode` stays. |
| `tools/packet-audit/cmd/run.go` | `candidatesFromFName` switch — the missing case is why every cell reads "no audit report". Task 1 adds `{name: "Food", pkg: "mount", dir: csvpkg.DirServerbound}` (model: the `CWvsContext::SendPetFoodItemUseRequest` case at ~1069). |
| `tools/packet-audit/cmd/disambiguation_test.go` | Regression tables for `qualifiedWriterName` / `locateAtlasFile` — gets `{"mount","Food","MountFood"}` entries. |
| `docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_v95,gms_jms_185}.json` | Committed exports; fname absent from all five → each gets one surgically-spliced sender entry (+ absent-only helpers). NON-idempotent: never regenerate wholesale. |
| `docs/packets/registry/<v>.yaml` | Registry rows: opcode 77/77/80/83/69, all `provenance: csv-import` (lines 2181/2844/2311/2529/2306). Edited only under branch (a). |
| `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_95,jms_185}_1.json` | `MountFoodHandle` + `LoggedInValidator` at 0x4D/0x4D/0x50/0x53/0x45 (lines 405/409/358/238/325). Already agree with registry. Edited only under branch (a). |
| `docs/packets/audits/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}/` | Each gets a copied `MountFood.{json,md}` report. |
| `docs/packets/audits/STATUS.md`, `status.json` | Regenerated only via `go run ./tools/packet-audit matrix`. Row at STATUS.md:565. |
| `docs/packets/audits/VERIFYING_A_PACKET.md` | The governing playbook (§6 marker, §8 regen, §9 serverbound flow, §10 export hygiene). |

## Decisions locked in design (do not re-litigate)

1. **No evidence pinning.** `USE_MOUNT_FOOD` is `tier1: false`; tier-0 cells
   promote on audit report + marker alone (grade.go tier-0 branch). A pinned
   record would be a standing freshness liability.
2. **Direct fname→codec linkage** (one `candidatesFromFName` case), not a thin
   wrapper struct — `Food` is a single-op codec already shaped correctly.
   Report/marker name: `qualifiedWriterName("mount","Food")` = `MountFood` →
   `packet=mount/serverbound/MountFood`, report file `MountFood.{json,md}`.
3. **Strict serialization** of the five version passes (v83→v84→v87→v95→jms):
   shared IDA `select_instance` state, shared test file, global matrix. No
   subagent fan-out for this row.
4. **Per-version atomic commits**: marker + export splice + report +
   regenerated STATUS/status.json together, so `matrix --check` never sees an
   orphan marker and any mid-campaign stop leaves a green tree.
5. **v84 evidence must come from the v84 binary.** No assumed byte-identity
   with v83. No live v84 IDA instance ⇒ stop-and-ask (genuine blocker).
6. **v92 stays parked** — no template edit, no matrix column, no opcode
   inference from the v87→v95 shift. Blocked solely on a v92 IDB.

## Tool mechanics (gotchas already hit by past tasks)

- **Harvest**: `packet-audit export -version <key> -prior-export "" -pending
  <roster.md> -descent-depth 12 -output <tmp>` = targeted harvest of only the
  roster fnames. Splice result into the committed export; keys are NOT sorted
  (append at end = minimal diff); 2-space indent + trailing newline.
- **Strip the `{op: "Delegate", ref: "COutPacket..."}` artifact** from the
  spliced sender entry — it kills report-gen descent.
- **Report gen** is the ROOT command (deterministic, no live IDA) with
  `-ida-source <export>.json -template template_<v>_1.json -output <tmp>`;
  copy only `MountFood.{json,md}` in. `triage`/`decompose` cannot create new
  reports.
- **jms naming**: version key `gms_jms_185`, export `gms_jms_185.json`, audit
  dir `jms_v185` (root.go:204–208 maps it), marker version `jms_v185`. Only
  the `*_U_DEVM` IDB decompiles — the retail dump is SMC.
- **Opcode ground truth** is the integer in `COutPacket::COutPacket(&pkt, OP)`
  — distrust IDB symbol names and the csv-seeded registry alike.
- **Unnamed sender**: byte signature `6A <op> 8D 8D ?? ?? ?? ?? E8` + 
  structure-match to a named twin (pet food: `Encode4(get_update_time) +
  Encode2(nPOS) + Encode4(nItemID)` — same expected shape); rename + idb_save.
- **`matrix --check` bar**: pre-existing 🟥 conflicts may keep exit ≠ 0; the
  bar is zero NEW orphan/dangling/stale/drift lines for this packet and no
  conflict-count increase.
- **`pt.Variants` indices** (`libs/atlas-packet/test/context.go`): [0]=v28,
  [1]=v83, [2]=v87, [3]=v95, [4]=JMS v185, [5]=v84, [6]=v86.

## Dependencies

- Live ida-pro-mcp server (default `http://192.168.20.3:13337/mcp`; ports vary
  by launch order — always `list_instances` first) with IDBs for GMS v83,
  v84.1, v87, v95, and JMS v185 `*_U_DEVM`.
- Task 1 (fname linkage) must land before any version pass can grade.
- No service `go.mod` changes expected ⇒ no `docker buildx bake` gate; final
  gates are `go test -race` / `go vet` in `libs/atlas-packet` +
  `tools/packet-audit`, plus `tools/redis-key-guard.sh`.

## Results

(Filled in by Task 7 at execution time: per-version instance, function
address, decompiled opcode, encode order, and any discrepancy branches fired.)
