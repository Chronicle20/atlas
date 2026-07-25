# Task 138 — Mount Food Packet Verification: Context

Companion to `plan.md`. Key files, decisions, and dependencies an implementer
needs, verified against the repo at planning time (2026-07-09; scope expanded
2026-07-25).

## What this task is

Verification campaign: promote the serverbound `USE_MOUNT_FOOD` row
(fname `CWvsContext::SendTamingMobFoodItemUseRequest`) to ✅ in **all nine**
coverage-matrix columns (gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84,
gms_v87, gms_v95, jms_v185). The feature itself (task-086) is fully
implemented; no behavior changes expected. Two exceptions to "verification
only": **gms_v48** additionally needs its registry op + a `MountFoodHandle`
seed-template registration (its cell is a false `n-a` because the op is absent
from `gms_v48.yaml`), and `test/context.go` gains four legacy tenant variants.
Scope grew from five to nine columns because main brought up the legacy
version columns after the v1 plan (2026-07-09). See PRD §1.1.

Grounded per-version facts (addresses from the DEVM IDBs; legacy four are
symbol-named — no rename needed):

| Version | Address | Opcode | Template `MountFoodHandle` | Registry op |
|---|---|---|---|---|
| gms_v48 | `0x70e00b` (7397387) | **0x3D** decompile-verified | add at `0x3D` | add |
| gms_v61 | `0x831f44` | 0x4C | present | present |
| gms_v72 | `0x904419` | 0x4C | present | present |
| gms_v79 | `0x955781` | 0x4B | present | present |
| gms_v83 | derive | 0x4D | present | present |
| gms_v84 | derive | 0x4D | present | present |
| gms_v87 | derive | 0x50 | present | present |
| gms_v95 | derive | 0x53 | present | present |
| jms_v185 | derive | 0x45 | present | present |

## Key files

| File | Role |
|---|---|
| `libs/atlas-packet/mount/serverbound/food.go` | The codec under verification. `Food{updateTime uint32, slot int16, itemId uint32}`, unconditional LE `ts(4)+slot(2)+itemId(4)`. Changes only if a version's IDB proves a different order (branch b). |
| `libs/atlas-packet/test/context.go` | Gains four appended legacy variants `[7]`v48/`[8]`v61/`[9]`v72/`[10]`v79 (Task 2). Never insert — positional refs must hold. |
| `libs/atlas-packet/mount/serverbound/food_test.go` | Gains `TestFoodByteFixture` with **nine** `packet-audit:verify` markers. Existing `TestFoodDecode` stays. |
| `tools/packet-audit/cmd/run.go` | `candidatesFromFName` switch — the missing case is why every cell reads "no audit report". Task 1 adds `{name: "Food", pkg: "mount", dir: csvpkg.DirServerbound}` (model: the `CWvsContext::SendPetFoodItemUseRequest` case at ~1069). |
| `tools/packet-audit/cmd/disambiguation_test.go` | Regression tables for `qualifiedWriterName` / `locateAtlasFile` — gets `{"mount","Food","MountFood"}` entries. |
| `docs/packets/ida-exports/{gms_v48,gms_v61,gms_v72,gms_v79,gms_v83,gms_v84,gms_v87,gms_v95,gms_jms_185}.json` | Committed exports; fname absent from all nine → each gets one surgically-spliced sender entry (+ absent-only helpers). NON-idempotent: never regenerate wholesale. |
| `docs/packets/registry/gms_v48.yaml` | **No `USE_MOUNT_FOOD` op yet** — Task 3 adds it (opcode 61/`0x3D`, `ida-discovered @0x70e00b`). This is what clears the false `n-a`. |
| `docs/packets/registry/<v>.yaml` (others) | Registry rows: v61/v72=0x4C, v79=0x4B, v83/v84=0x4D, v87=0x50, v95=0x53, jms=0x45. Legacy = `ida-discovered`; original five = `csv-import`. Edited only under branch (a). |
| `services/atlas-configurations/seed-data/templates/template_gms_48_1.json` | **No `MountFoodHandle` yet** — Task 3 registers it at `0x3D` (verified no collision; 78 existing handler entries). |
| `services/atlas-configurations/seed-data/templates/template_{gms_61,gms_72,gms_79,gms_83,gms_84,gms_87,gms_95,jms_185}_1.json` | `MountFoodHandle` + `LoggedInValidator` at 0x4C/0x4C/0x4B/0x4D/0x4D/0x50/0x53/0x45. Already agree with registry. Edited only under branch (a). |
| `docs/packets/audits/{gms_v48,gms_v61,gms_v72,gms_v79,gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}/` | Each gets a copied `MountFood.{json,md}` report. |
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
3. **Strict serialization** of the nine version passes
   (v48→v61→v72→v79→v83→v84→v87→v95→jms): shared `food_test.go`, global
   matrix, and (v48) shared registry/template. Session-based IDA reads are
   per-`database` but the mutating steps are not — no subagent fan-out.
4. **Per-version atomic commits**: marker + export splice + report +
   regenerated STATUS/status.json together (+ v48's registry/template), so
   `matrix --check` never sees an orphan marker and any mid-campaign stop
   leaves a green tree.
5. **v84 evidence must come from the v84 binary.** No assumed byte-identity
   with v83. No live v84 IDA instance ⇒ stop-and-ask (genuine blocker).
6. **Codec is version-invariant** — body `update_time u32·slot i16·itemId u32`
   on every version (v48 decompile confirms it matches v83+). `food.go` needs
   no version gate; every fixture asserts the same 10 bytes; only the marker
   address differs per version. Branch (b) is a contingency only.
7. **gms_12 stays parked** — no template edit, no matrix column, no opcode
   inference. Blocked solely on the absence of a v12 IDB.
8. **gms_92 matrix work excluded** — gms_92 is not a matrix column (no cell to
   promote). Its mount-food gap is now just a one-line `template_gms_92_1.json`
   registration (opcode `0x54`, already verified); optional fold-in per PRD §9.

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
  [1]=v83, [2]=v87, [3]=v95, [4]=JMS v185, [5]=v84, [6]=v86 — **then Task 2
  appends** [7]=v48, [8]=v61, [9]=v72, [10]=v79.

## Dependencies

- Live ida-pro-mcp server (default `http://192.168.20.3:13337/mcp`; session
  ids/ports vary by launch order — always `idb_list` first) with IDBs for GMS
  v48, v61, v72, v79, v83, v84.1, v87, v95 (all `*_DEVM`) and JMS v185
  `*_U_DEVM`. All present this turn.
- Task 1 (fname linkage) must land before any version pass can grade.
- No service `go.mod` changes expected ⇒ no `docker buildx bake` gate; final
  gates are `go test -race` / `go vet` in `libs/atlas-packet` +
  `tools/packet-audit`, plus `tools/redis-key-guard.sh`.

## Results

All nine `USE_MOUNT_FOOD` coverage-matrix cells are byte-verified (✅) against
live IDA decompiles read via the **session-based** IDA-MCP server
(`http://192.168.20.3:8745/mcp`, `-ida-database <session>`). The IDA setup
changed since planning: the old per-port / `select_instance` model
(`-ida-port`) is gone; each IDB is now addressed by a `database` session id.

Per-version verification facts (opcode = the integer in
`COutPacket::COutPacket(&pkt, N)`; body is version-invariant
`Encode4(update_time)·Encode2(slot)·Encode4(itemId)`, guarded by
`itemId/10000 == 226`):

| Version | IDB session | Func address | Opcode | Notes |
|---|---|---|---|---|
| gms_v48 | 0bb5f11a | 0x70e00b | 0x3D (61) | registry op + template `MountFoodHandle` ADDED — corrects the false `n-a` (op was absent from `gms_v48.yaml`) |
| gms_v61 | 965202bf | 0x831f44 | **0x48 (72)** | **opcode CORRECTED 0x4C→0x48** — the plan/registry/template `0x4C` was a stale mislabel of a distinct function (`sub_832680`, category 231, 2-field). Registry + template fixed (branch a), with a live-tenant config-patch callout in the fix commit |
| gms_v72 | 90e36cb0 | 0x904419 | 0x4C (76) | registry `fname` fixed (stale `sub_955781` placeholder → canonical) so the matrix could resolve the report |
| gms_v79 | 9a7d3642 | 0x955781 | 0x4B (75) | registry `fname` fixed (un-demangled `sub_955781` → canonical) |
| gms_v83 | ce4ff298 | 0xa09a64 | 0x4D (77) | pure verify |
| gms_v84 | 79511a2a | 0xa53e46 | 0x4D (77) | IDB function was UNNAMED (`sub_A53E46`) → renamed to the mangled `CWvsContext::SendTamingMobFoodItemUseRequest` + `idb_save`; own address used (no borrow from v83) |
| gms_v87 | 81f32170 | 0xa9f310 | 0x50 (80) | pure verify |
| gms_v95 | e4abcb98 | 0x9d63a0 | 0x53 (83) | pure verify |
| jms_v185 | 3c4bb8b1 | 0xaee70c | 0x45 (69) | SMC dump (`MapleStory_dump_SCY`) decompiled fine — no DEVM build needed; pure verify |

**No opcodes were inferred** — every value traces to a decompiled `COutPacket`
integer. The only opcode change (gms_v61) was a decompile-driven correction of
a pre-existing mislabel, not an inference.

### Tooling enabler added (this branch)

`packet-audit export` gained an `-ida-database <session>` flag that injects the
`database` argument on every session-scoped MCP tool call
(`lookup_funcs`/`func_query`/`decompile`/`callees`) via the single
`callStructured` chokepoint, so a harvest targets the intended IDB
deterministically on the session-based server. The dead `-ida-port` /
`select_instance` path is left intact but unused.

### Canonical variant set expanded

Task 2 added GMS v48/v61/v72/v79 to the shared `libs/atlas-packet/test`
`Variants` slice. Per maintainer decision these four legacy versions are now
first-class members of the canonical set, so six pre-existing byte-fixture
tests that range over `Variants` (npc/StartConversation, party/TownPortal,
field/WarpToMap, character/CharacterData monster-book, character/clientbound
CharacterInfo pets, model/CharacterStatistics gachaponExp) had their
legacy-variant expectations aligned to each codec's already-IDA-grounded
version gate. No codec was changed; the tests now assert real legacy behavior.

### Remaining narrowed gaps

- **gms_12** remains parked solely on the absence of a v12 IDB (no matrix
  column, no opcode inference).
- **gms_92** mount food is unblocked (v92 IDB present, opcode `0x54` verified)
  and reduced to a one-line `template_gms_92_1.json` registration that is out
  of this task's matrix scope (gms_92 is not a matrix column).
- **gms_v61 live-tenant follow-up**: the `0x4C`→`0x48` opcode correction
  (commit `3e9b52cd0`) fixes the seed template only — any already-deployed
  v61 tenant socket config still routing `MountFoodHandle` at `0x4C` needs an
  operational config PATCH to `0x48` to match.
