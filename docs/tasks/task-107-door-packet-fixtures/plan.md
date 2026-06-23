# Door Clientbound Packet-Fixture Verification Campaign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Drive all 12 `incomplete` cells of the `door` clientbound family (3 packets × gms_v84/gms_v87/gms_v95/jms_v185) to `verified` (✅) in the packet coverage matrix, landing each cell as its coupled artifacts (IDA export splice + per-version audit report + `packet-audit:verify` marker, plus a pinned evidence record for the one tier-1 packet).

**Architecture:** This is a *verification* campaign, not a feature. The encoders in `libs/atlas-packet/door/clientbound/` are already byte-proven identical across all versions by the existing cross-version equality tests, and `gms_v83` is a fully verified reference. The work per cell is: confirm the version's client read order in IDA, surgically splice the receiver function into that version's committed IDA export, generate the per-version audit report from that export, stack a `packet-audit:verify` marker on the existing test, (tier-1 only) pin an evidence record, regenerate the matrix, and commit the cell's artifacts together. Cells are grouped **by version/IDB** (one `select_instance` target at a time — never interleave two versions against the shared IDA instance).

**Tech Stack:** Go (`libs/atlas-packet`), the `tools/packet-audit` CLI (`export`, root report-gen, `evidence pin`, `matrix`, `matrix --check`, `fname-doc --check`, `operations --check`), and the `mcp__ida-pro__*` MCP tools (decompile / rename / address lookup against the live IDBs).

> **Path convention in this plan:** `<worktree-root>` = the task worktree directory (`.worktrees/task-107-door-packet-fixtures/`). All other paths are repo-relative from there.

---

## Reference facts (read once before any task)

### The reshaping finding (design §3) — confirmed during planning

The three client receiver functions exist **only** in `docs/packets/ida-exports/gms_v83.json`:

- `CTownPortalPool::OnTownPortalCreated` (SpawnDoor)
- `CTownPortalPool::OnTownPortalRemoved` (RemoveDoor)
- `CWvsContext::OnTownPortal` (RemoveTownDoor)

They are **absent** from `gms_v84.json`, `gms_v87.json`, `gms_v95.json`, `gms_jms_185.json`. Therefore export-only report generation is impossible and a fresh full re-export is forbidden (it drifts ~150 unrelated keys — playbook §10). Every cell requires a **surgical, absent-only splice** of the harvested receiver into the version's committed export. Where the receiver is unnamed in a version's IDB, **name it** (playbook §10 byte-signature + v83-twin structure match) — naming is producible, not a blocker.

### Per-packet reference (the v83 read order is the structural template every version must match)

| Struct (= Writer = report filename stem) | Receiver fn (registry `fname`) | v83 addr | v83 client read order | Body bytes | Tier | Test file |
|---|---|---|---|---|---|---|
| `SpawnDoor` | `CTownPortalPool::OnTownPortalCreated` | `0x7bd6c6` | `Decode1(launched)` `Decode4(ownerId)` `Decode2(x)` `Decode2(y)` | 9 | 0 | `libs/atlas-packet/door/clientbound/spawn_test.go` |
| `RemoveDoor` | `CTownPortalPool::OnTownPortalRemoved` | `0x7be064` | `Decode1(flag)` `Decode4(ownerId)` | 5 | 0 | `libs/atlas-packet/door/clientbound/remove_test.go` |
| `RemoveTownDoor` | `CWvsContext::OnTownPortal` | `0xa226a6` | `Decode4(townId)` `Decode4(targetId)` then **guarded** `Decode2(x)` `Decode2(y)` (skipped when both ids == 999999999) | 8 (removal path: two NONE ids, no position) | **1** | `libs/atlas-packet/door/clientbound/remove_town_test.go` |

> `SpawnPortal` (live-portal, 12 bytes, shares `SPAWN_PORTAL` opcode) is **out of scope** (design §9): it has no `status.json` op row. When confirming `OnTownPortal` for `RemoveTownDoor`, the same decompile also covers the live-portal branch — note it in the read-order write-up but do **not** add a marker, report, or evidence for it. Flag it in the PR description as an untracked-but-evidenced writer.

### Per-version mapping (template / export / audit dir / IDA port)

| Version key (marker `version=`) | Seed template | Committed export json | Audit report dir | IDA port (per memory IDBs_v9 — **confirm by binary name, never trust the number**) |
|---|---|---|---|---|
| `gms_v84` | `services/atlas-configurations/seed-data/templates/template_gms_84_1.json` | `docs/packets/ida-exports/gms_v84.json` | `docs/packets/audits/gms_v84/` | 13337 |
| `gms_v87` | `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` | `docs/packets/ida-exports/gms_v87.json` | `docs/packets/audits/gms_v87/` | 13340 |
| `gms_v95` | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` | `docs/packets/ida-exports/gms_v95.json` | `docs/packets/audits/gms_v95/` | 13339 |
| `jms_v185` | `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` | `docs/packets/ida-exports/gms_jms_185.json` | `docs/packets/audits/jms_v185/` | 13338 (use the clean `*_U_DEVM` build, not the SMC retail dump) |

### Per-version dispatch opcodes (use to locate the OnPacket case when the receiver is unnamed)

These are the **dispatch** opcodes (from `status.json` / `registry/<v>.yaml`), used only to find the case label in the version's dispatcher (`CTownPortalPool::OnPacket` for SpawnDoor/RemoveDoor; the `CWvsContext` opcode handler for OnTownPortal). The **marker address** is the receiver function's own address found in step 2/3, NOT the opcode.

| Op (receiver) | gms_v84 | gms_v87 | gms_v95 | jms_v185 |
|---|---|---|---|---|
| `SPAWN_DOOR` (`OnTownPortalCreated`) | `0x11A` (282) | `0x124` (292) | `0x14A` (330) | `0x128` (296) |
| `REMOVE_DOOR` (`OnTownPortalRemoved`) | `0x11B` (283) | `0x125` (293) | `0x14B` (331) | `0x129` (297) |
| `SPAWN_PORTAL` (`OnTownPortal`) | `0x45` (69) | `0x45` (69) | `0x45` (69) | `0x3D` (61) |

### Promotion mechanism (what makes a cell ✅ — design §2)

- **Tier-0** (`SpawnDoor`, `RemoveDoor`): per-version **audit report** + stacked **`packet-audit:verify` marker**. **No evidence record** (playbook §7 — an evidence record is a standing freshness liability for tier-0).
- **Tier-1** (`RemoveTownDoor`): audit report + marker + **pinned evidence record** with a `verifies:` line.

The dominant missing artifact today is the per-version audit report (every incomplete door cell carries `"note": "no audit report"` in `status.json`).

### `matrix --check` acceptance bar (design §7, playbook §8)

`matrix --check` currently exits 1 from a pre-existing registry-seed conflict backlog unrelated to door. The bar is **"no new problems"**, not exit 0:
- Zero orphan / dangling / stale / drift lines mentioning any `door/clientbound/*` packet.
- The global conflict count must **not increase**.
- Every door cell in scope reads ✅ after regen.
- `fname-doc --check` and `operations --check` introduce no new failures.

### Wire-delta contingency (design §6) — not expected, but the pipeline must not assume identity

If a version's decompiled read order diverges from the v83 template (inserted field, changed guard, different discriminant), that is a genuine wire bug in the unbranched encoder. Do **not** silently patch and do **not** continue the cell. STOP, surface it, and escalate: the fix is a separate prior commit (add the version branch to the encoder + update the cross-version test to expect the divergence, with its own review), then resume the cell against the corrected encoder. No delta is expected (v84≡v83; simple fixed-width bodies), but each client must actually be read.

---

## THE PER-CELL PIPELINE (canonical procedure — every cell task references this)

Each of the 12 cell tasks below is the same 9-step procedure with different parameters. The task's parameter block supplies: `VERSION`, `PORT`, `STRUCT`, `RECEIVER`, `OPCODE`, `READ_ORDER`, `BYTES`, `TIER`, `TEMPLATE`, `EXPORT`, `AUDIT_DIR`, `TEST_FILE`. Substitute them literally.

**P1 — Select the instance.** `mcp__ida-pro__list_instances`; identify the instance whose loaded **binary name** matches `VERSION` (do not trust `PORT` — it is a hint); `mcp__ida-pro__select_instance(<that port>)`. Record the confirmed binary name + port in the commit message.

**P2 — Locate / name the receiver.** Find `RECEIVER` via `mcp__ida-pro__func_query` with `name_regex`. If it resolves, record its address. If it does **not** resolve (unnamed in this IDB): find the dispatcher (`CTownPortalPool::OnPacket`, or the `CWvsContext` opcode handler for `OnTownPortal`), locate the case for `OPCODE`, descend to the handler function, confirm it structurally matches the v83 twin (`RECEIVER` @ the v83 addr in the reference table), then `mcp__ida-pro__rename` it to `RECEIVER`. Record the final address as `IDA_ADDR`.

**P3 — Decompile & confirm read order.** `mcp__ida-pro__decompile` the receiver; descend into helper reads (address-based). Write the full ordered read list and compare to `READ_ORDER`. **If it matches**, proceed. **If it diverges**, STOP and follow the Wire-delta contingency above (escalate; do not continue this cell).

**P4 — Splice the export (absent-only).** Harvest the receiver (+ deep helpers) to a temp file via the targeted-roster form, pointed at this version's live IDB:
```bash
go run ./tools/packet-audit export \
  --version VERSION \
  --prior-export "" \
  --pending /tmp/door-roster-VERSION.md \
  --descent-depth 12 \
  --ida-url http://192.168.20.3:PORT/mcp --ida-port PORT \
  --output /tmp/door-harvest-VERSION-STRUCT.json
```
where `/tmp/door-roster-VERSION.md` contains the single line `RECEIVER`. Then **surgically splice** only the `RECEIVER` entry (and any of its deep helpers that are absent) from `/tmp/door-harvest-VERSION-STRUCT.json` into the committed `EXPORT` — **add, never overwrite** an existing key (absent-only for helpers). If the harvested entry carries a `COutPacket` delegate ctor artifact, strip that one call before splicing (playbook §10). Diff the splice (`git diff EXPORT`) and confirm it only **adds** keys.

**P5 — Generate the report.** Run the root report-gen to a temp output, then copy the one report:
```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template TEMPLATE \
  -ida-source EXPORT \
  -output /tmp/door-rpt-VERSION
cp /tmp/door-rpt-VERSION/VERSION/STRUCT.json AUDIT_DIR/STRUCT.json
cp /tmp/door-rpt-VERSION/VERSION/STRUCT.md   AUDIT_DIR/STRUCT.md
```
Confirm `AUDIT_DIR/STRUCT.json` has `"Verdict": 0`, `"FlatInvalid": false`, the expected `Rows` count for `READ_ORDER`, and an `"Address"` equal to `IDA_ADDR`. (Report-gen must succeed; a "not in export" / "delegate to COutPacket" failure means P4 was incomplete — fix the splice, do not fake the report.)

**P6 — Add the marker.** Append one line to the stacked marker block above the test function in `TEST_FILE` (mirroring `reactor/clientbound/destroy_test.go`):
```go
// packet-audit:verify packet=door/clientbound/STRUCT version=VERSION ida=IDA_ADDR
```
`IDA_ADDR` MUST equal the report's `Address` (an orphan-marker `matrix --check` failure means they disagree). Do not add a new test body — the existing cross-version equality loop already proves byte identity.

**P7 — (TIER==1 only) pin evidence.** Only for `RemoveTownDoor`:
```bash
go run ./tools/packet-audit evidence pin \
  --packet door/clientbound/STRUCT --version VERSION \
  --ida "RECEIVER" --category TIER1-FIXTURE
```
Then open `docs/packets/evidence/VERSION/door.clientbound.STRUCT.yaml` and hand-add:
```yaml
verifies:
    - TEST_FILE#TestRemoveTownDoor
```
(Tier-0 cells: skip this step entirely — do **not** pin.)

**P8 — Regenerate & verify promotion.**
```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```
Confirm in `docs/packets/audits/status.json` that the `STRUCT` cell for `VERSION` is now `"state": "verified"`. Confirm the `matrix --check` output mentions **no** `door/clientbound/*` orphan/dangling/stale/drift line and the conflict count did not increase vs the Task 1 baseline. `fname-doc`/`operations` introduce no new failures.

**P9 — Build/vet/test the module, then commit the cell's artifacts together.**
```bash
cd <worktree-root>
( cd libs/atlas-packet && go test -race ./door/... && go vet ./... && go build ./... )
git add EXPORT AUDIT_DIR/STRUCT.json AUDIT_DIR/STRUCT.md TEST_FILE \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
# TIER==1 only, also add:  docs/packets/evidence/VERSION/door.clientbound.STRUCT.yaml
git commit -m "task-107: verify door/clientbound/STRUCT for VERSION (<binary>@<port>)"
git rev-parse --show-toplevel   # must end with /.worktrees/task-107-door-packet-fixtures
git branch --show-current       # must be task-107-door-packet-fixtures
```

---

## Task 1: Preflight — baseline, tooling, IDA liveness, v83 reference capture

**Files:** none modified (verification only; no commit).

- [ ] **Step 1: Confirm worktree and clean baseline**

Run:
```bash
cd <worktree-root>
git rev-parse --show-toplevel   # must end with /.worktrees/task-107-door-packet-fixtures
git branch --show-current       # must be task-107-door-packet-fixtures
git status --porcelain          # expect clean (only the planning docs, already committed)
```
Expected: on branch `task-107-door-packet-fixtures`, working tree clean.

- [ ] **Step 2: Confirm the `door` module is green today (the encoders already pass cross-version)**

Run:
```bash
( cd libs/atlas-packet && go test ./door/... && go vet ./door/... && go build ./... )
```
Expected: PASS (the existing v83 golden + cross-version equality tests pass for all `pt.Variants`).

- [ ] **Step 3: Confirm `packet-audit` builds and capture the conflict-count baseline**

Run:
```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check ; echo "exit=$?"
```
Expected: `matrix` regenerates `STATUS.md`/`status.json` with no diff (baseline already current); `matrix --check` exits 1 from the pre-existing backlog. **Record the total conflict/problem count printed** — this is the "do not increase" baseline for every cell's P8. Confirm NO door-related line appears in the output today.

- [ ] **Step 4: Verify the reshaping finding still holds (receivers absent from 4 exports)**

Run:
```bash
for f in gms_v84 gms_v87 gms_v95 gms_jms_185; do
  echo "=== $f ===";
  grep -o -E 'CTownPortalPool::OnTownPortalCreated|CTownPortalPool::OnTownPortalRemoved|CWvsContext::OnTownPortal' \
    docs/packets/ida-exports/$f.json | sort -u;
done
```
Expected: each prints an empty result (functions absent). This confirms the splice path in P4 is required for every cell. If any function IS already present, that version's report-gen for that struct may work without a splice — note it and skip only that struct's P4 splice (still verify the export entry resolves).

- [ ] **Step 5: Confirm the IDA instances are live and map ports to binaries**

Run `mcp__ida-pro__list_instances`. Record the port→binary mapping for the v84, v87, v95, and jms IDBs (memory says 13337/13340/13339/13338 respectively, but **confirm by binary name**). If an instance is missing, that version's tasks are blocked on a live IDB — surface it (genuine external blocker) rather than guessing addresses.

- [ ] **Step 6: Capture the v83 reference read orders (sanity anchor)**

`select_instance` the v83 IDB and `mcp__ida-pro__decompile` `CTownPortalPool::OnTownPortalCreated` (`0x7bd6c6`), `CTownPortalPool::OnTownPortalRemoved` (`0x7be064`), `CWvsContext::OnTownPortal` (`0xa226a6`). Write the three read orders into your working notes; confirm they match the Reference-facts table. These are the structural templates each target version's P3 compares against. (No file change, no commit.)

---

## Task 2: gms_v84 — `SpawnDoor` (tier-0)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v84.json` (splice `CTownPortalPool::OnTownPortalCreated`)
- Create: `docs/packets/audits/gms_v84/SpawnDoor.json`, `docs/packets/audits/gms_v84/SpawnDoor.md`
- Modify: `libs/atlas-packet/door/clientbound/spawn_test.go` (add stacked marker)
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9) with these parameters**

```
VERSION    = gms_v84
PORT       = 13337            # confirm binary name in P1
STRUCT     = SpawnDoor
RECEIVER   = CTownPortalPool::OnTownPortalCreated
OPCODE     = 0x11A            # SPAWN_DOOR dispatch case (locate handler if unnamed)
READ_ORDER = Decode1(launched) Decode4(ownerId) Decode2(x) Decode2(y)   # 4 rows, 9 bytes
BYTES      = 9
TIER       = 0               # NO evidence pin
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_gms_84_1.json
EXPORT     = docs/packets/ida-exports/gms_v84.json
AUDIT_DIR  = docs/packets/audits/gms_v84
TEST_FILE  = libs/atlas-packet/door/clientbound/spawn_test.go
```
Marker line to add (with the address found in P2): `// packet-audit:verify packet=door/clientbound/SpawnDoor version=gms_v84 ida=0x<addr>`

- [ ] **Step 2: Confirm promotion**

Expected after P8: `status.json` `SpawnDoor` → `gms_v84` → `"state": "verified"`; no door line in `matrix --check`; conflict count == Task 1 baseline.

- [ ] **Step 3: Commit** — per P9 (`go.mod` untouched → no bake).

---

## Task 3: gms_v84 — `RemoveDoor` (tier-0)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v84.json` (splice `CTownPortalPool::OnTownPortalRemoved`)
- Create: `docs/packets/audits/gms_v84/RemoveDoor.json`, `docs/packets/audits/gms_v84/RemoveDoor.md`
- Modify: `libs/atlas-packet/door/clientbound/remove_test.go`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9) with these parameters**

```
VERSION    = gms_v84
PORT       = 13337
STRUCT     = RemoveDoor
RECEIVER   = CTownPortalPool::OnTownPortalRemoved
OPCODE     = 0x11B            # REMOVE_DOOR dispatch case
READ_ORDER = Decode1(flag) Decode4(ownerId)   # 2 rows, 5 bytes
BYTES      = 5
TIER       = 0
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_gms_84_1.json
EXPORT     = docs/packets/ida-exports/gms_v84.json
AUDIT_DIR  = docs/packets/audits/gms_v84
TEST_FILE  = libs/atlas-packet/door/clientbound/remove_test.go
```
Marker: `// packet-audit:verify packet=door/clientbound/RemoveDoor version=gms_v84 ida=0x<addr>`

- [ ] **Step 2: Confirm promotion** — `RemoveDoor` → `gms_v84` → `verified`; no new problems.
- [ ] **Step 3: Commit** — per P9.

---

## Task 4: gms_v84 — `RemoveTownDoor` (tier-1, evidence pinned)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v84.json` (splice `CWvsContext::OnTownPortal`)
- Create: `docs/packets/audits/gms_v84/RemoveTownDoor.json`, `docs/packets/audits/gms_v84/RemoveTownDoor.md`
- Create: `docs/packets/evidence/gms_v84/door.clientbound.RemoveTownDoor.yaml`
- Modify: `libs/atlas-packet/door/clientbound/remove_town_test.go`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9, INCLUDING P7) with these parameters**

```
VERSION    = gms_v84
PORT       = 13337
STRUCT     = RemoveTownDoor
RECEIVER   = CWvsContext::OnTownPortal
OPCODE     = 0x45            # SPAWN_PORTAL dispatch case
READ_ORDER = Decode4(townId) Decode4(targetId) [guard: skip Decode2(x) Decode2(y) when both == 999999999]
BYTES      = 8              # removal path: two NONE ids, no position
TIER       = 1             # PIN EVIDENCE (P7)
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_gms_84_1.json
EXPORT     = docs/packets/ida-exports/gms_v84.json
AUDIT_DIR  = docs/packets/audits/gms_v84
TEST_FILE  = libs/atlas-packet/door/clientbound/remove_town_test.go
```
Marker: `// packet-audit:verify packet=door/clientbound/RemoveTownDoor version=gms_v84 ida=0x<addr>`
P7 evidence `verifies:` line: `- libs/atlas-packet/door/clientbound/remove_town_test.go#TestRemoveTownDoor`
While in `OnTownPortal`, also note the live-portal (SpawnPortal) branch in your read-order notes but do NOT pin/marker it (out of scope, design §9).

- [ ] **Step 2: Confirm promotion** — `RemoveTownDoor` → `gms_v84` → `verified`; the evidence record exists with `verifies:`; `matrix --check` reports no "dangling evidence" for door and no new door problems.
- [ ] **Step 3: Commit** — per P9, **including** `docs/packets/evidence/gms_v84/door.clientbound.RemoveTownDoor.yaml`.

---

## Task 5: gms_v87 — `SpawnDoor` (tier-0)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v87.json`
- Create: `docs/packets/audits/gms_v87/SpawnDoor.{json,md}`
- Modify: `libs/atlas-packet/door/clientbound/spawn_test.go`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9) with these parameters**

```
VERSION    = gms_v87
PORT       = 13340
STRUCT     = SpawnDoor
RECEIVER   = CTownPortalPool::OnTownPortalCreated
OPCODE     = 0x124
READ_ORDER = Decode1(launched) Decode4(ownerId) Decode2(x) Decode2(y)   # 4 rows, 9 bytes
BYTES      = 9
TIER       = 0
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_gms_87_1.json
EXPORT     = docs/packets/ida-exports/gms_v87.json
AUDIT_DIR  = docs/packets/audits/gms_v87
TEST_FILE  = libs/atlas-packet/door/clientbound/spawn_test.go
```
Marker: `// packet-audit:verify packet=door/clientbound/SpawnDoor version=gms_v87 ida=0x<addr>`
(Note: per memory, v87 had naming groundwork but is not fully named — expect to name the receiver in P2.)

- [ ] **Step 2: Confirm promotion** — `SpawnDoor` → `gms_v87` → `verified`; no new problems.
- [ ] **Step 3: Commit** — per P9.

---

## Task 6: gms_v87 — `RemoveDoor` (tier-0)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v87.json`
- Create: `docs/packets/audits/gms_v87/RemoveDoor.{json,md}`
- Modify: `libs/atlas-packet/door/clientbound/remove_test.go`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9) with these parameters**

```
VERSION    = gms_v87
PORT       = 13340
STRUCT     = RemoveDoor
RECEIVER   = CTownPortalPool::OnTownPortalRemoved
OPCODE     = 0x125
READ_ORDER = Decode1(flag) Decode4(ownerId)   # 2 rows, 5 bytes
BYTES      = 5
TIER       = 0
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_gms_87_1.json
EXPORT     = docs/packets/ida-exports/gms_v87.json
AUDIT_DIR  = docs/packets/audits/gms_v87
TEST_FILE  = libs/atlas-packet/door/clientbound/remove_test.go
```
Marker: `// packet-audit:verify packet=door/clientbound/RemoveDoor version=gms_v87 ida=0x<addr>`

- [ ] **Step 2: Confirm promotion** — `RemoveDoor` → `gms_v87` → `verified`; no new problems.
- [ ] **Step 3: Commit** — per P9.

---

## Task 7: gms_v87 — `RemoveTownDoor` (tier-1, evidence pinned)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v87.json`
- Create: `docs/packets/audits/gms_v87/RemoveTownDoor.{json,md}`
- Create: `docs/packets/evidence/gms_v87/door.clientbound.RemoveTownDoor.yaml`
- Modify: `libs/atlas-packet/door/clientbound/remove_town_test.go`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9, INCLUDING P7) with these parameters**

```
VERSION    = gms_v87
PORT       = 13340
STRUCT     = RemoveTownDoor
RECEIVER   = CWvsContext::OnTownPortal
OPCODE     = 0x45
READ_ORDER = Decode4(townId) Decode4(targetId) [guard: skip Decode2(x) Decode2(y) when both == 999999999]
BYTES      = 8
TIER       = 1
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_gms_87_1.json
EXPORT     = docs/packets/ida-exports/gms_v87.json
AUDIT_DIR  = docs/packets/audits/gms_v87
TEST_FILE  = libs/atlas-packet/door/clientbound/remove_town_test.go
```
Marker: `// packet-audit:verify packet=door/clientbound/RemoveTownDoor version=gms_v87 ida=0x<addr>`
P7 `verifies:` line: `- libs/atlas-packet/door/clientbound/remove_town_test.go#TestRemoveTownDoor`

- [ ] **Step 2: Confirm promotion** — `RemoveTownDoor` → `gms_v87` → `verified`; evidence record present; no new door problems.
- [ ] **Step 3: Commit** — per P9, **including** `docs/packets/evidence/gms_v87/door.clientbound.RemoveTownDoor.yaml`.

---

## Task 8: gms_v95 — `SpawnDoor` (tier-0)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v95.json`
- Create: `docs/packets/audits/gms_v95/SpawnDoor.{json,md}`
- Modify: `libs/atlas-packet/door/clientbound/spawn_test.go`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9) with these parameters**

```
VERSION    = gms_v95
PORT       = 13339
STRUCT     = SpawnDoor
RECEIVER   = CTownPortalPool::OnTownPortalCreated
OPCODE     = 0x14A
READ_ORDER = Decode1(launched) Decode4(ownerId) Decode2(x) Decode2(y)   # 4 rows, 9 bytes
BYTES      = 9
TIER       = 0
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_gms_95_1.json
EXPORT     = docs/packets/ida-exports/gms_v95.json
AUDIT_DIR  = docs/packets/audits/gms_v95
TEST_FILE  = libs/atlas-packet/door/clientbound/spawn_test.go
```
Marker: `// packet-audit:verify packet=door/clientbound/SpawnDoor version=gms_v95 ida=0x<addr>`
(Note: per memory, v95 is well-named — the receiver likely resolves directly in P2.)

- [ ] **Step 2: Confirm promotion** — `SpawnDoor` → `gms_v95` → `verified`; no new problems.
- [ ] **Step 3: Commit** — per P9.

---

## Task 9: gms_v95 — `RemoveDoor` (tier-0)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v95.json`
- Create: `docs/packets/audits/gms_v95/RemoveDoor.{json,md}`
- Modify: `libs/atlas-packet/door/clientbound/remove_test.go`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9) with these parameters**

```
VERSION    = gms_v95
PORT       = 13339
STRUCT     = RemoveDoor
RECEIVER   = CTownPortalPool::OnTownPortalRemoved
OPCODE     = 0x14B
READ_ORDER = Decode1(flag) Decode4(ownerId)   # 2 rows, 5 bytes
BYTES      = 5
TIER       = 0
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_gms_95_1.json
EXPORT     = docs/packets/ida-exports/gms_v95.json
AUDIT_DIR  = docs/packets/audits/gms_v95
TEST_FILE  = libs/atlas-packet/door/clientbound/remove_test.go
```
Marker: `// packet-audit:verify packet=door/clientbound/RemoveDoor version=gms_v95 ida=0x<addr>`

- [ ] **Step 2: Confirm promotion** — `RemoveDoor` → `gms_v95` → `verified`; no new problems.
- [ ] **Step 3: Commit** — per P9.

---

## Task 10: gms_v95 — `RemoveTownDoor` (tier-1, evidence pinned)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v95.json`
- Create: `docs/packets/audits/gms_v95/RemoveTownDoor.{json,md}`
- Create: `docs/packets/evidence/gms_v95/door.clientbound.RemoveTownDoor.yaml`
- Modify: `libs/atlas-packet/door/clientbound/remove_town_test.go`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9, INCLUDING P7) with these parameters**

```
VERSION    = gms_v95
PORT       = 13339
STRUCT     = RemoveTownDoor
RECEIVER   = CWvsContext::OnTownPortal
OPCODE     = 0x45
READ_ORDER = Decode4(townId) Decode4(targetId) [guard: skip Decode2(x) Decode2(y) when both == 999999999]
BYTES      = 8
TIER       = 1
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_gms_95_1.json
EXPORT     = docs/packets/ida-exports/gms_v95.json
AUDIT_DIR  = docs/packets/audits/gms_v95
TEST_FILE  = libs/atlas-packet/door/clientbound/remove_town_test.go
```
Marker: `// packet-audit:verify packet=door/clientbound/RemoveTownDoor version=gms_v95 ida=0x<addr>`
P7 `verifies:` line: `- libs/atlas-packet/door/clientbound/remove_town_test.go#TestRemoveTownDoor`

- [ ] **Step 2: Confirm promotion** — `RemoveTownDoor` → `gms_v95` → `verified`; evidence record present; no new door problems.
- [ ] **Step 3: Commit** — per P9, **including** `docs/packets/evidence/gms_v95/door.clientbound.RemoveTownDoor.yaml`.

---

## Task 11: jms_v185 — `SpawnDoor` (tier-0)

**Files:**
- Modify: `docs/packets/ida-exports/gms_jms_185.json`
- Create: `docs/packets/audits/jms_v185/SpawnDoor.{json,md}`
- Modify: `libs/atlas-packet/door/clientbound/spawn_test.go`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9) with these parameters**

```
VERSION    = jms_v185
PORT       = 13338            # use the clean *_U_DEVM build, NOT the SMC retail dump (playbook §10)
STRUCT     = SpawnDoor
RECEIVER   = CTownPortalPool::OnTownPortalCreated
OPCODE     = 0x128
READ_ORDER = Decode1(launched) Decode4(ownerId) Decode2(x) Decode2(y)   # 4 rows, 9 bytes
BYTES      = 9
TIER       = 0
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_jms_185_1.json
EXPORT     = docs/packets/ida-exports/gms_jms_185.json
AUDIT_DIR  = docs/packets/audits/jms_v185
TEST_FILE  = libs/atlas-packet/door/clientbound/spawn_test.go
```
Marker: `// packet-audit:verify packet=door/clientbound/SpawnDoor version=jms_v185 ida=0x<addr>`
Note for P4/P5: the marker `version=` and audit dir are `jms_v185`, but the committed export file is `gms_jms_185.json`. The report-gen `-output` subdir for the jms template is `jms_v185` (Variant `JMS/v185`).

- [ ] **Step 2: Confirm promotion** — `SpawnDoor` → `jms_v185` → `verified`; no new problems. If the jms binary is genuinely undecompilable on the receiver (SMC), surface it as a real blocker (playbook §10) rather than faking — but door receivers are simple, so this is unlikely on the `*_U_DEVM` build.
- [ ] **Step 3: Commit** — per P9.

---

## Task 12: jms_v185 — `RemoveDoor` (tier-0)

**Files:**
- Modify: `docs/packets/ida-exports/gms_jms_185.json`
- Create: `docs/packets/audits/jms_v185/RemoveDoor.{json,md}`
- Modify: `libs/atlas-packet/door/clientbound/remove_test.go`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9) with these parameters**

```
VERSION    = jms_v185
PORT       = 13338
STRUCT     = RemoveDoor
RECEIVER   = CTownPortalPool::OnTownPortalRemoved
OPCODE     = 0x129
READ_ORDER = Decode1(flag) Decode4(ownerId)   # 2 rows, 5 bytes
BYTES      = 5
TIER       = 0
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_jms_185_1.json
EXPORT     = docs/packets/ida-exports/gms_jms_185.json
AUDIT_DIR  = docs/packets/audits/jms_v185
TEST_FILE  = libs/atlas-packet/door/clientbound/remove_test.go
```
Marker: `// packet-audit:verify packet=door/clientbound/RemoveDoor version=jms_v185 ida=0x<addr>`

- [ ] **Step 2: Confirm promotion** — `RemoveDoor` → `jms_v185` → `verified`; no new problems.
- [ ] **Step 3: Commit** — per P9.

---

## Task 13: jms_v185 — `RemoveTownDoor` (tier-1, evidence pinned)

**Files:**
- Modify: `docs/packets/ida-exports/gms_jms_185.json`
- Create: `docs/packets/audits/jms_v185/RemoveTownDoor.{json,md}`
- Create: `docs/packets/evidence/jms_v185/door.clientbound.RemoveTownDoor.yaml`
- Modify: `libs/atlas-packet/door/clientbound/remove_town_test.go`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Run THE PER-CELL PIPELINE (P1–P9, INCLUDING P7) with these parameters**

```
VERSION    = jms_v185
PORT       = 13338
STRUCT     = RemoveTownDoor
RECEIVER   = CWvsContext::OnTownPortal
OPCODE     = 0x3D
READ_ORDER = Decode4(townId) Decode4(targetId) [guard: skip Decode2(x) Decode2(y) when both == 999999999]
BYTES      = 8
TIER       = 1
TEMPLATE   = services/atlas-configurations/seed-data/templates/template_jms_185_1.json
EXPORT     = docs/packets/ida-exports/gms_jms_185.json
AUDIT_DIR  = docs/packets/audits/jms_v185
TEST_FILE  = libs/atlas-packet/door/clientbound/remove_town_test.go
```
Marker: `// packet-audit:verify packet=door/clientbound/RemoveTownDoor version=jms_v185 ida=0x<addr>`
P7 `verifies:` line: `- libs/atlas-packet/door/clientbound/remove_town_test.go#TestRemoveTownDoor`

- [ ] **Step 2: Confirm promotion** — `RemoveTownDoor` → `jms_v185` → `verified`; evidence record present; no new door problems.
- [ ] **Step 3: Commit** — per P9, **including** `docs/packets/evidence/jms_v185/door.clientbound.RemoveTownDoor.yaml`.

---

## Task 14: Final sweep, full module gate, and PR prep

**Files:** none modified (verification only).

- [ ] **Step 1: Confirm all 12 (15 incl. v83) door cells are ✅**

Run:
```bash
go run ./tools/packet-audit matrix
git diff --stat docs/packets/audits/STATUS.md docs/packets/audits/status.json   # expect no diff (already committed per cell)
grep -n -A20 '"packet": "door/clientbound/SpawnDoor"' docs/packets/audits/status.json
grep -n -A20 '"packet": "door/clientbound/RemoveDoor"' docs/packets/audits/status.json
grep -n -A20 '"packet": "door/clientbound/RemoveTownDoor"' docs/packets/audits/status.json
```
Expected: for all three packets, every `gms_v83/gms_v84/gms_v87/gms_v95/jms_v185` cell reads `"state": "verified"`; no `"note": "no audit report"` remains on any door cell.

- [ ] **Step 2: Confirm marker + report + evidence counts**

Run:
```bash
grep -rc "packet-audit:verify packet=door/clientbound" libs/atlas-packet/door/clientbound/spawn_test.go libs/atlas-packet/door/clientbound/remove_test.go libs/atlas-packet/door/clientbound/remove_town_test.go
ls docs/packets/audits/gms_v84 docs/packets/audits/gms_v87 docs/packets/audits/gms_v95 docs/packets/audits/jms_v185 | grep -E 'SpawnDoor|RemoveDoor|RemoveTownDoor'
ls docs/packets/evidence/gms_v84 docs/packets/evidence/gms_v87 docs/packets/evidence/gms_v95 docs/packets/evidence/jms_v185 | grep RemoveTownDoor   # tier-1 only
```
Expected: each test file has 5 markers (1 pre-existing v83 + 4 new); each of the 4 target audit dirs has all three `*.{json,md}` report pairs; each of the 4 target evidence dirs has exactly the `RemoveTownDoor` record (tier-0 packets NOT pinned).

- [ ] **Step 3: Full `matrix --check` / `fname-doc` / `operations` "no new problems" check**

Run:
```bash
go run ./tools/packet-audit matrix --check ; echo "exit=$?"
go run ./tools/packet-audit fname-doc --check ; echo "exit=$?"
go run ./tools/packet-audit operations --check ; echo "exit=$?"
```
Expected: no orphan/dangling/stale/drift line mentions any `door/clientbound/*` packet; total conflict count == the Task 1 baseline (did not increase); `fname-doc`/`operations` show no new failures.

- [ ] **Step 4: Full module verification gate (CLAUDE.md "Build & Verification")**

Run from the worktree root:
```bash
( cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./... )
tools/redis-key-guard.sh   # repo-root guard; door changes cannot affect it — expect clean
```
Expected: `go test -race`/`vet`/`build` clean in `libs/atlas-packet`; redis-key-guard clean. **No `docker buildx bake`** is required — no `go.mod` was touched (design §10; only `*_test.go` + docs changed).

- [ ] **Step 5: Confirm git state and prepare for review**

Run:
```bash
git log --oneline -15
git rev-parse --show-toplevel   # must end with /.worktrees/task-107-door-packet-fixtures
git branch --show-current       # must be task-107-door-packet-fixtures
git status --porcelain          # clean
```
Expected: 12 cell commits (Tasks 2–13) on `task-107-door-packet-fixtures`, working tree clean.

- [ ] **Step 6: Code review before PR**

Per CLAUDE.md "Code Review Before PR", invoke `superpowers:requesting-code-review` (it dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer` since Go files changed). Address findings before opening the PR. In the PR description, flag the `SpawnPortal` wrinkle (design §9): it is an untracked-but-evidenced writer (v83 evidence + report exist, no `status.json` op row) — a candidate for a future matrix row, deliberately out of scope here.

---

## Self-Review (completed by planner)

**Spec coverage:** PRD §4 (3 packets × 4 versions) → Tasks 2–13 (12 cells); PRD acceptance "all ✅" → Task 14 Step 1; "marker + fresh evidence committed together" → P6/P7/P9 + Task 14 Step 2; "`matrix --check`/`fname-doc`/`operations` clean (no new problems)" → P8 + Task 14 Step 3; "module test/vet/build clean, bake for touched go.mod" → Task 14 Step 4 (no go.mod touched → no bake). Design §3 reshaping finding → P2/P4 (name + splice); §6 wire-delta → P3 contingency; §7 check bar → P8/Task 14 Step 3; §9 SpawnPortal → Task 4/7/10/13 notes + Task 14 Step 6.

**Placeholder scan:** The only `0x<addr>` tokens are per-version receiver addresses *discovered* in P2 (genuinely unknowable until the live decompile) — inputs to the cell, not unfilled placeholders. Every command, file path, opcode, struct name, read order, byte count, and tier is concrete.

**Type consistency:** `STRUCT` names (`SpawnDoor`/`RemoveDoor`/`RemoveTownDoor`) match the Go structs, the report filenames, the marker `packet=door/clientbound/<STRUCT>` paths, and the evidence `door.clientbound.<STRUCT>.yaml` names throughout. Tier assignment (0/0/1) is consistent across the reference table, the per-task TIER param, and the evidence steps. Export-file vs version-key mismatch for jms (`gms_jms_185.json` file, `jms_v185` key/dir) is called out explicitly in Task 11.
