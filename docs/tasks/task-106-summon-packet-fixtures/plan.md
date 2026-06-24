# Summon Clientbound Packet-Fixture Verification Campaign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Drive every `incomplete`/`partial` cell in the `summon` clientbound family of the coverage matrix (`docs/packets/audits/STATUS.md`) to `verified` (✅) across v83/v84/v87/v95/jms by adding a `packet-audit:verify` byte-fixture per cell — and, for the 24 tier-1 cells, a fresh pinned evidence record — each derived field-by-field from the live client read order.

**Architecture:** This is a *verification* campaign, not a feature. Each matrix cell is promoted by following `docs/packets/audits/VERIFYING_A_PACKET.md`: decompile the client's read order on the matching live IDB, confirm or correct the byte-fixture in `libs/atlas-packet/summon/clientbound/`, add a `packet-audit:verify` marker, pin evidence where the grader requires it, regenerate the matrix, and commit the coupled artifacts together. Work is batched **by IDB** — `select_instance` one version, do all six packets, then move on; never two IDBs in parallel (the IDA instance selection is shared global state).

**Tech Stack:** Go (`libs/atlas-packet` test fixtures; `tools/packet-audit` CLI), ida-pro-mcp (live decompile), the packet-audit evidence/matrix ledger under `docs/packets/`.

---

## CRITICAL — read before any task

These rules are load-bearing. Violating any of them produces a *false* ✅ (the exact anti-pattern project memory and CLAUDE.md warn against) or trips `matrix --check`.

1. **No invented bytes.** Every byte in every fixture must trace to a specific decompile line. Do **not** ratify an existing fixture or copy expected bytes from this plan, from another version, or from MapleStory knowledge without confirming the layout against the *live* decompile of the cell under test. The existing `*_test.go` fixtures are a **starting point to confirm or correct**, not ground truth — their own comments flag v84/v87/jms as "NOT re-confirmed live … the old `ida=` markers point at the wrong path."

2. **Two grading recipes (the grader is version-stratified, not uniform).** `summon/` is **not** in `docs/packets/evidence/tiers.yaml`, so a summon cell's tier is decided by the audit report's `FlatInvalid` flag (`tools/packet-audit/internal/matrix/grade.go:117` → `tier1 = in.Tier1[pkt] || rep.FlatInvalid`):
   - **v95 SummonMove only** — report `Verdict=Match (0)`, `FlatInvalid=false` → **tier-0**. Verified rule is `toolPass && marker.Found` (grade.go:215). It needs a marker **and NO evidence record** (`VERIFYING_A_PACKET.md` §7 — a tier-0 evidence record is a standing freshness liability).
   - **all 24 other cells** (v83/v84/v87/jms × 6) — reports have `FlatInvalid=true` → **tier-1**. Verified rule is `marker.Found && hasEvidence && evidence.Fresh` (grade.go:199). Each needs a marker **AND** a fresh pinned evidence record. `FlatInvalid=true` is a static-analyzer modeling limitation ("could not reduce a writer branch to a version predicate"), not a wire bug; a byte-fixture is exactly what resolves it.

3. **Marker `ida=` MUST equal the report/evidence address**, or `matrix --check` raises an **orphan-marker** failure. The evidence record's address is resolved from the export by the FName you pass to `evidence pin --ida` (`tools/packet-audit/cmd/evidence.go:45` `functionAddress`). So: marker `ida=` = the address the FName resolves to in `docs/packets/ida-exports/<export>.json` = the audit report `Address`. Keep all three in sync.

4. **Active-vs-inactive dispatch trap.** The v83 `SummonSpawn` report points at `CSummonedPool::OnCreated @0x938f61` — the **inactive** twin (dispatcher does NOT pre-read `cid`). The live-confirmed **active** field-path target is `OnCreated @0x95ADEC` (task-088 x32dbg). Where the committed report/export address ≠ the live active read function, the export entry must be re-pointed (surgical splice, § R5 of the Recipe) and the report regenerated so marker+report+evidence all agree on the **active** address. Confirm per cell which `CSummonedPool::On*` is the live target before pinning.

5. **The export is NOT idempotent.** Never run a full `packet-audit export`. To re-point/deepen ONE function, harvest to a temp file and surgically splice only that entry into the committed export (`VERIFYING_A_PACKET.md` §10).

6. **Acceptance bar is strict exit 0.** Baseline `go run ./tools/packet-audit matrix --check` exits **0** in this worktree (confirmed). After every commit it must stay 0 — zero new orphan/dangling/stale/drift lines, conflict count stays 0.

7. **jms is the retail SCY dump** — SMC / control-flow-virtualized for some sends. The summon *read* functions were decompiled before (reports carry real addresses), so they are likely fine. If a jms read function is genuinely undecompilable, that is a real blocker → **escalate, do not fabricate** the read order.

8. **Worktree discipline.** All work happens in `.worktrees/task-106-summon-packet-fixtures` on branch `task-106-summon-packet-fixtures`. After each commit verify `git rev-parse --show-toplevel` ends with `/.worktrees/task-106-summon-packet-fixtures` and `git branch --show-current` is `task-106-summon-packet-fixtures`.

---

## Files

All paths relative to the worktree root `.worktrees/task-106-summon-packet-fixtures/`.

**Fixtures (modify — add per-version `Test…Bytes<VER>` funcs + markers; never rename existing funcs, tests reference internals):**
- `libs/atlas-packet/summon/clientbound/spawn_test.go`
- `libs/atlas-packet/summon/clientbound/remove_test.go`
- `libs/atlas-packet/summon/clientbound/move_test.go`
- `libs/atlas-packet/summon/clientbound/attack_test.go`
- `libs/atlas-packet/summon/clientbound/damage_test.go`
- `libs/atlas-packet/summon/clientbound/skill_test.go`

**Codecs (READ for comparison; modify ONLY if a live decompile proves a byte error — that is a wire-fix in its own commit, §R3):**
- `libs/atlas-packet/summon/clientbound/{spawn,remove,move,attack,damage,skill}.go`

**Evidence ledger (create — tier-1 only):**
- `docs/packets/evidence/{gms_v83,gms_v84,gms_v87,jms_v185}/summon.clientbound.<Packet>.yaml`

**Audit reports / export (modify ONLY where a re-point is needed, §R5):**
- `docs/packets/audits/{gms_v83,gms_v84,gms_v87,jms_v185}/Summon<Packet>.json` (+ `.md`)
- `docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_jms_185}.json`

**Regenerated each promotion (commit alongside):**
- `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

**Reference only (do not edit):**
- `docs/packets/audits/VERIFYING_A_PACKET.md` — the canonical playbook
- `tools/packet-audit/internal/matrix/grade.go` — the grading rules
- `libs/atlas-packet/test/{context.go,roundtrip.go}` — `test.Variants`, `test.Encode`, `test.RoundTrip`, `test.CreateContext`
- `libs/atlas-packet/party/clientbound/invite_test.go` — the reference `pt.Variants`-table byte-fixture pattern

---

## IDB instances (confirm by binary NAME, never hardcode the port)

Per project memory (IDBs_v9) the current ports are below, but launch order can shift them. **Always** enumerate (`mcp__ida-pro__list_instances`) and `select_instance` the one whose loaded IDB **name** matches the target version before reading.

| Version key | Expected IDB / binary | Expected port | Export file | Audit dir | Evidence dir |
|---|---|---|---|---|---|
| `gms_v83` | GMS v83 | 13341 | `gms_v83.json` | `gms_v83/` | `gms_v83/` |
| `gms_v84` | GMS v84 (`GMS_v84.1_U_DEVM`) | 13337 | `gms_v84.json` | `gms_v84/` | `gms_v84/` |
| `gms_v87` | GMS v87 | 13340 | `gms_v87.json` | `gms_v87/` | `gms_v87/` |
| `gms_v95` | GMS v95 | 13339 | `gms_v95.json` | `gms_v95/` | `gms_v95/` |
| `jms_v185` | JMS v185 (SCY retail dump) | 13338 | `gms_jms_185.json` | `jms_v185/` | `jms_v185/` |

Note: the jms **export file** keeps the historical name `gms_jms_185.json` while its **version key / audit dir / evidence dir** are `jms_v185` (`tools/packet-audit/internal/matrix/model.go:17-20`). `matrix.ExportPath("jms_v185")` resolves this for you; `evidence pin --version jms_v185` does too.

---

## The Verification Recipe (referenced by every task)

Two variants. Apply **R-T0** to the single tier-0 cell (Task 1); apply **R-T1** to each of the 24 tier-1 cells (Tasks 2–5). Substitute the `<…>` placeholders from each task's parameter table.

### R-T0 — tier-0 cell (v95 SummonMove only)

- **Step a — select the IDB.** `mcp__ida-pro__list_instances`; `select_instance(<port>)` for `gms_v95`; confirm the loaded IDB name is the v95 binary.
- **Step b — decompile the active read function.** Decompile `<FName>` at `<address>` (descend into helper reads as needed). Write down the full ordered read list (header reads + body + loop bounds + guards).
- **Step c — compare to the codec** `libs/atlas-packet/summon/clientbound/move.go`. Confirm every field of the existing `TestSummonMoveBytesV95` fixture maps to a decompile read. If a field disagrees → it is a wire bug → STOP and apply §R3 (fix the codec first, own commit) before continuing.
- **Step d — add the marker** immediately above `TestSummonMoveBytesV95`:
  `// packet-audit:verify packet=summon/clientbound/SummonMove version=gms_v95 ida=<address>`
  where `<address>` equals the v95 SummonMove report `Address` (`docs/packets/audits/gms_v95/SummonMove.json` → `0x759830`) — confirm it matches the live decompile address; if the live active address differs, re-point per §R5 first.
  **Do NOT pin an evidence record** (tier-0).
- **Step e — regenerate + verify.** `go run ./tools/packet-audit matrix` then `go run ./tools/packet-audit matrix --check` (exit 0). Confirm the SummonMove × gms_v95 cell flipped 🟡 → ✅ in `docs/packets/audits/STATUS.md`.
- **Step f — run the Go test.** `go test ./libs/atlas-packet/summon/clientbound/ -run TestSummonMoveBytesV95 -v` → PASS.
- **Step g — commit** fixture + STATUS.md + status.json together (no evidence file).

### R-T1 — tier-1 cell (each of v83/v84/v87/jms × {Spawn,Remove,Move,Attack,Damage,Skill})

- **Step a — select the IDB.** `mcp__ida-pro__list_instances`; `select_instance(<port>)` for `<version>`; confirm the loaded IDB **name** matches.
- **Step b — decompile the ACTIVE read function.** Decompile `<FName>` at the **committed report address** `<report_addr>`, AND resolve whether that is the live active field-path target (§4 trap). The dispatcher chain is `CUserPool::OnUserCommonPacket` (reads `cid`) → `CSummonedPool::OnPacket` (reads `oid`) → the per-op leaf (`On*`). Confirm: which `On*` the dispatcher actually routes this opcode to, and that it pre-reads `cid`+`oid` upstream. Record the **active leaf address** `<active_addr>`. Descend into helper reads (`CSummoned::Init`, `AvatarLook::Decode`, `CMovePath::Decode`, etc.).
- **Step c — write the full ordered read list** for the cell: the upstream `cid`,`oid` pre-reads + the leaf body, with every field's decompile line. This is the authority for the bytes.
- **Step d — compare to the codec** `libs/atlas-packet/summon/clientbound/<pkg>.go`, including its version gates (`spawnHasAvatarLook`, `t.IsRegion("GMS") && t.MajorAtLeast(95)`, the attack target loop). **v84 off-by-one check:** confirm v84 takes the v83-shaped clientbound path — gates must be region/`MajorAtLeast`-correct, never a bare `>83`. If the decompile contradicts the codec → **wire bug → §R3 first** (own commit), then continue.
- **Step e — write/correct the byte-fixture.** Add `func TestSummon<Packet>Bytes<VER>(t *testing.T)` to the cell's `*_test.go` (extend the existing table; reuse the shared body var where present, e.g. `summonSpawnV83Body`). Use `test.Encode(t, ctx, in.Encode, nil)` with `ctx := test.CreateContext(<Region>, <Major>, <Minor>)`. Cite the decompile line for every byte in comments. Do NOT rename existing funcs.
- **Step f — re-point the export/report IF NEEDED (§R5).** If `<active_addr>` ≠ `<report_addr>` (or the export's `<FName>` resolves to the inactive twin): surgically splice the active function entry into `docs/packets/ida-exports/<export>.json` and regenerate that one report (§R5). Otherwise skip.
- **Step g — add the marker** above the new test func:
  `// packet-audit:verify packet=summon/clientbound/Summon<Packet> version=<version> ida=<active_addr>`
  `<active_addr>` MUST equal the report `Address` and what `<FName>` resolves to in the export.
- **Step h — pin evidence:**
  ```bash
  go run ./tools/packet-audit evidence pin \
    --packet summon/clientbound/Summon<Packet> \
    --version <version> \
    --ida "<FName>" \
    --category TIER1-FIXTURE
  ```
  This writes `docs/packets/evidence/<version>/summon.clientbound.Summon<Packet>.yaml`. Then **manually add** the `verifies:` field:
  ```yaml
  verifies:
    - libs/atlas-packet/summon/clientbound/<pkg>_test.go#TestSummon<Packet>Bytes<VER>
  ```
- **Step i — regenerate + verify.** `go run ./tools/packet-audit matrix`; `go run ./tools/packet-audit matrix --check` (exit 0, no new orphan/dangling/stale lines mentioning summon). Confirm the cell flipped ❌ → ✅ in STATUS.md.
- **Step j — run the Go test.** `go test ./libs/atlas-packet/summon/clientbound/ -run TestSummon<Packet>Bytes<VER> -v` → PASS.
- **Step k — commit** the coupled artifacts together: fixture + evidence YAML + (export/report if re-pointed) + STATUS.md + status.json.

### R3 — wire-fix sub-procedure (only if a decompile contradicts a codec)

PRD non-goal allows a writer fix **only when a fixture proves a byte error**. If step d finds a real divergence:
1. Fix the codec in `libs/atlas-packet/summon/clientbound/<pkg>.go` (and its `Decode` mirror).
2. `go test -race ./libs/atlas-packet/...` clean.
3. Commit the wire fix **alone** with a message explaining the decompile evidence (`fix(summon): <field> on <version> per <FName>@<addr>`).
4. **If the fix lands in a codec consumed by `services/atlas-channel`** (it is — `socket/writer/summon.go` wraps these), then per CLAUDE.md you must additionally: `go build ./...`, `go vet ./...`, `go test -race ./...` in `services/atlas-channel`, and `docker buildx bake atlas-channel` from the worktree root. (Test-only fixture changes do NOT trigger a bake; a codec change does.)
5. Return to the cell's recipe at step e.

### R5 — export re-point sub-procedure (surgical splice; only if `<active_addr>` ≠ `<report_addr>`)

The export is non-idempotent — never re-run a full `export`. To re-point ONE function (`VERIFYING_A_PACKET.md` §10):
1. Harvest the active function to a temp file (one IDB selected):
   ```bash
   go run ./tools/packet-audit export \
     --ida-url http://<host>:<port>/mcp --ida-port <port> \
     -prior-export "" -pending /tmp/summon_roster.md -descent-depth 12 \
     -output /tmp/summon_export.json
   ```
   (`/tmp/summon_roster.md` lists the active `<FName>`; mirror the format of `docs/packets/ida-exports/_pending.md`.)
2. Surgically splice ONLY the active `<FName>` entry (and any newly-needed deep helper, absent-only) from `/tmp/summon_export.json` into `docs/packets/ida-exports/<export>.json`. Overwrite the single stale sender entry; never bulk-replace.
3. Regenerate the one report to a temp dir, then copy it in:
   ```bash
   go run ./tools/packet-audit \
     -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
     -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
     -template services/atlas-configurations/seed-data/templates/template_<tmpl>.json \
     -ida-source docs/packets/ida-exports/<export>.json \
     -output /tmp/rpt
   cp /tmp/rpt/<version>/Summon<Packet>.json docs/packets/audits/<version>/
   cp /tmp/rpt/<version>/Summon<Packet>.md   docs/packets/audits/<version>/
   ```
   (`<tmpl>` = `gms_83_1` / `gms_84_1` / `gms_87_1` / `jms_185_1`.)
4. Confirm the new report `Address` = `<active_addr>`, then proceed to the marker (step g) and evidence pin (step h) so all three agree.
5. Commit the export + report changes **with** the fixture/evidence for that cell.

---

## Task 1: v95 SummonMove (tier-0 — clears the lone 🟡)

**Why first:** simplest cell; validates the marker→matrix loop end-to-end before the heavier tier-1 work.

**Files:**
- Modify: `libs/atlas-packet/summon/clientbound/move_test.go`
- Regenerate: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

**Parameters:** version `gms_v95`, port 13339, FName `CSummonedPool::OnMove`, report addr `0x759830` (`docs/packets/audits/gms_v95/SummonMove.json`), test `TestSummonMoveBytesV95` (already exists), region/major/minor = `GMS,95,1`.

- [x] **Step 1: Pre-flight.** Confirm cwd is the worktree; `go run ./tools/packet-audit matrix --check` exits 0; STATUS.md shows `MOVE_SUMMON … summon/clientbound/SummonMove … 0x118 🟡` for v95.
- [x] **Step 2: Apply Recipe R-T0** with the parameters above. The SummonMove wire is `int cid, int oid, raw CMovePath blob` (the blob already begins with start x,y — confirm via `CMovePath::Decode` reached from `CSummonedPool::OnMove`; the start position must NOT be written separately — see `move.go` comment).
- [x] **Step 3: Confirm promotion.** STATUS.md v95 SummonMove cell now ✅; `matrix --check` exit 0; `go test ./libs/atlas-packet/summon/clientbound/ -run TestSummonMoveBytesV95 -v` PASS.
- [x] **Step 4: Verify NO evidence file was created** for v95 SummonMove (`ls docs/packets/evidence/gms_v95/ | grep -i summon.clientbound.SummonMove` → no output; tier-0 carries none).
- [x] **Step 5: Commit.**
  ```bash
  git add libs/atlas-packet/summon/clientbound/move_test.go docs/packets/audits/STATUS.md docs/packets/audits/status.json
  git commit -m "task-106: verify summon/clientbound/SummonMove gms_v95 (tier-0 marker; clears lone 🟡)"
  ```

---

## Task 2: v83 — all six summon clientbound packets (tier-1)

**Why second:** the active dispatch is best understood here (task-088 x32dbg). This task also resolves the `SummonSpawn` inactive-path re-point (§4) that v84/v87/jms may inherit.

**Files:**
- Modify: all six `libs/atlas-packet/summon/clientbound/*_test.go`
- Create: `docs/packets/evidence/gms_v83/summon.clientbound.Summon{Spawn,Remove,Move,Attack,Damage,Skill}.yaml`
- Possibly modify (re-point): `docs/packets/ida-exports/gms_v83.json`, `docs/packets/audits/gms_v83/SummonSpawn.json` (+`.md`)
- Regenerate: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

**Parameter table (version `gms_v83`, port 13341, ctx `GMS,83,1`; report addrs from `docs/packets/audits/gms_v83/`):**

| Packet | Codec file | Existing test to extend / add | FName (registry/report) | Committed report addr | Re-point? |
|---|---|---|---|---|---|
| SummonSpawn | spawn.go | `TestSummonSpawnBytesV83` (exists, **no marker**) | `CSummonedPool::OnCreated` | `0x938f61` | **LIKELY** → live active `0x95ADEC` (§4). Confirm live which `OnCreated` the dispatcher routes 0xAF to and whether it pre-reads cid; align all three artifacts to the active addr. |
| SummonRemove | remove.go | add `TestSummonRemoveBytesV83` (only `TestSummonRemoveBytes` exists, unmarked) | `CSummonedPool::OnRemoved` | `0x7a64eb` | confirm live; re-point only if active ≠ committed |
| SummonMove | move.go | add `TestSummonMoveBytesV83` (rename-free: `TestSummonMoveBytes` exists, unmarked — add a distinct `…V83` func) | `CSummonedPool::OnMove` | `0x7a6861` | confirm live |
| SummonAttack | attack.go | add `TestSummonAttackBytesV83` (`TestSummonAttackBytes` exists, unmarked) | `CSummonedPool::OnAttack` | `0x7a6882` | confirm live |
| SummonDamage | damage.go | `TestSummonDamageBytes` exists (unmarked) → add `…V83` marked func | `CSummonedPool::OnHit` | `0x7a6ebe` | confirm live (note: body comments call this the "OnSkill"-named leaf — the *address* is authoritative for the marker, not the name; see context.md) |
| SummonSkill | skill.go | add `TestSummonSkillBytesV83` (`TestSummonSkillBytes` exists, unmarked) | `CSummonedPool::OnSkill` | `0x7a6e5a` | confirm live |

> The existing `*_test.go` already carry unmarked v83 byte bodies. Treat them as a hypothesis to **confirm against the live decompile**, then add a **new** marked `…BytesV83` test (or add the marker above the existing one only after live confirmation). Do NOT delete/rename the existing funcs.

- [x] **Step 1: Select the v83 IDB.** `mcp__ida-pro__list_instances`; `select_instance(13341)` (confirm name).
- [x] **Step 2: Resolve the dispatch chain once.** Decompile `CUserPool::OnUserCommonPacket` (cid pre-read) and `CSummonedPool::OnPacket @0x938dd7` (oid pre-read + the per-opcode vtable dispatch for 0xAF..0xB4). Record which leaf each opcode routes to and the active address of each. **Resolve the `SummonSpawn` active-vs-inactive question here** (`0x938f61` vs `0x95ADEC`).
- [x] **Step 3: SummonSpawn** — apply Recipe R-T1. If the dispatcher routes 0xAF to an `OnCreated` whose address ≠ `0x938f61`, apply §R5 re-point (splice the active `OnCreated` into `gms_v83.json`, regen `gms_v83/SummonSpawn.json`) so report+marker+evidence = active addr. Confirm the spawn Init tail (charLevel, foothold) and the `spawnHasAvatarLook` gate (v83 = no avatar byte). Commit per step k.
- [x] **Step 4: SummonRemove** — apply Recipe R-T1. Wire = `ownerId, oid, animated byte (4|1)`. Commit.
- [x] **Step 5: SummonMove** — apply Recipe R-T1. Wire = `cid, oid, raw CMovePath blob` (no separate start pos). Commit.
- [x] **Step 6: SummonAttack** — apply Recipe R-T1. Wire = `cid, oid, byte charLevel(0), byte direction, byte count, per target{int monsterOid, byte 6, int damage}`; confirm v83 has **no** trailing flag byte (the trailing byte is the v95-only delta). Commit.
- [x] **Step 7: SummonDamage** — apply Recipe R-T1. Wire = `cid, oid, byte attackIdx(12), int damage, int monsterIdFrom, byte bLeft(0)`; confirm v83 stops at bLeft (no trailing dir byte). Commit.
- [x] **Step 8: SummonSkill** — apply Recipe R-T1. Wire = `cid, oid, byte (stance&0x7F)`; confirm there is **no** summonSkillId int. Commit.
- [x] **Step 9: Full-family check for v83.** `go run ./tools/packet-audit matrix && go run ./tools/packet-audit matrix --check` → exit 0; STATUS.md shows all six v83 summon clientbound cells ✅. `go test ./libs/atlas-packet/summon/clientbound/ -v` all PASS.

---

## Task 3: v84 — all six summon clientbound packets (tier-1)

**Watch:** the v84 off-by-one gate class. v84 clientbound summon packets are **v83-shaped** (no avatar byte, no trailing attack flag); the only summon gate that fires at v84 is the **serverbound** attack envelope (`MajorAtLeast(84)`), which is out of scope here. Confirm every clientbound gate evaluates to the v83 path for `GMS,84,1`.

**Files:**
- Modify: all six `libs/atlas-packet/summon/clientbound/*_test.go` (add `…BytesV84` funcs)
- Create: `docs/packets/evidence/gms_v84/summon.clientbound.Summon{Spawn,Remove,Move,Attack,Damage,Skill}.yaml`
- Possibly modify (re-point): `docs/packets/ida-exports/gms_v84.json`, `docs/packets/audits/gms_v84/SummonSpawn.json` (+`.md`)
- Regenerate: STATUS.md, status.json

**Parameter table (version `gms_v84`, port 13337 — `GMS_v84.1_U_DEVM`, ctx `GMS,84,1`; report addrs from `docs/packets/audits/gms_v84/`):**

| Packet | FName | Committed report addr | Re-point? |
|---|---|---|---|
| SummonSpawn | `CSummonedPool::OnCreated` | `0x97038b` | confirm live (apply §R5 if the dispatcher routes to an active twin ≠ committed, mirroring the v83 finding) |
| SummonRemove | `CSummonedPool::OnRemoved` | `0x7cbfa1` | confirm live |
| SummonMove | `CSummonedPool::OnMove` | `0x7cc317` | confirm live |
| SummonAttack | `CSummonedPool::OnAttack` | `0x7cc338` | confirm live |
| SummonDamage | `CSummonedPool::OnHit` | `0x7cc984` | confirm live |
| SummonSkill | `CSummonedPool::OnSkill` | `0x7cc920` | confirm live |

- [x] **Step 1: Select the v84 IDB.** `select_instance(13337)`; confirm name `GMS_v84.1_U_DEVM`.
- [x] **Step 2: Resolve the v84 dispatch chain** (`CSummonedPool::OnPacket` + cid/oid pre-reads); record active leaf addresses; resolve any spawn active-vs-inactive twin.
- [x] **Step 3–8: Apply Recipe R-T1** to Spawn, Remove, Move, Attack, Damage, Skill (one commit per cell). For each, confirm the clientbound layout is byte-identical to the v83 active layout AND that every codec gate resolves to the v83 path under `GMS,84,1` (off-by-one guard). `…BytesV84` may assert equality to the shared body var (e.g. `summonSpawnV83Body`, `summonAttackV83Body`) **after** live confirmation, mirroring `TestSummonDamageBytesV87`'s pattern.
- [x] **Step 9: Full-family check for v84.** matrix + `matrix --check` exit 0; six v84 cells ✅; `go test` PASS.

---

## Task 4: v87 — all six summon clientbound packets (tier-1)

**Files:**
- Modify: all six `*_test.go` (add `…BytesV87`; note `damage_test.go` already has `TestSummonDamageBytesV87` — confirm-and-mark rather than duplicate)
- Create: `docs/packets/evidence/gms_v87/summon.clientbound.Summon{Spawn,Remove,Move,Attack,Damage,Skill}.yaml`
- Possibly modify (re-point): `docs/packets/ida-exports/gms_v87.json`, `docs/packets/audits/gms_v87/SummonSpawn.json` (+`.md`)
- Regenerate: STATUS.md, status.json

**Parameter table (version `gms_v87`, port 13340, ctx `GMS,87,1`; report addrs from `docs/packets/audits/gms_v87/`):**

| Packet | FName | Committed report addr | Re-point? |
|---|---|---|---|
| SummonSpawn | `CSummonedPool::OnCreated` | `0x9b3749` | confirm live (registry note: 0xBC → spawn vtable+0x30 target `sub_9B3749`) |
| SummonRemove | `CSummonedPool::OnRemoved` | `0x7f8cb0` | confirm live |
| SummonMove | `CSummonedPool::OnMove` | `0x7f902b` | confirm live |
| SummonAttack | `CSummonedPool::OnAttack` | `0x7f904c` | confirm live (codec comment: v87 OnAttack has NO trailing read — the trailing flag is v95-only) |
| SummonDamage | `CSummonedPool::OnHit` | `0x7f969f` | confirm live (existing `TestSummonDamageBytesV87` asserts v87 ≡ v83) |
| SummonSkill | `CSummonedPool::OnSkill` | `0x7f963b` | confirm live |

- [x] **Step 1: Select the v87 IDB.** `select_instance(13340)`; confirm name.
- [x] **Step 2: Resolve the v87 dispatch chain**; record active leaf addresses; resolve any spawn twin.
- [x] **Step 3–8: Apply Recipe R-T1** to all six (one commit per cell). For SummonDamage, confirm the existing `TestSummonDamageBytesV87` body against the live decompile, then add its marker + pin evidence (do not duplicate the func). Confirm v87 SummonAttack has no trailing flag byte.
- [x] **Step 9: Full-family check for v87.** matrix + `matrix --check` exit 0; six v87 cells ✅; `go test` PASS.

---

## Task 5: jms_v185 — all six summon clientbound packets (tier-1)

**Watch:** jms is the retail SCY dump (SMC for some sends). Summon *read* functions were decompiled before (reports carry real addresses), so expect them to decompile. If any read function is genuinely undecompilable → **escalate per §7, do not fabricate**. jms `SummonSpawn` carries the avatar-look tail (`spawnHasAvatarLook` returns true for JMS ≥185) — confirm the trailing `bAvatarLook` byte against `sub_823AED@0x823aed` (`Decode1 bAvatarLook @0x823b99`, then `if (v8) AvatarLook::Decode @0x823bb0`).

**Files:**
- Modify: all six `*_test.go` (spawn_test.go already has `TestSummonSpawnBytesJMS185` — confirm-and-mark; add `…BytesJMS185` for the other five)
- Create: `docs/packets/evidence/jms_v185/summon.clientbound.Summon{Spawn,Remove,Move,Attack,Damage,Skill}.yaml`
- Possibly modify (re-point): `docs/packets/ida-exports/gms_jms_185.json`, `docs/packets/audits/jms_v185/Summon*.json` (+`.md`)
- Regenerate: STATUS.md, status.json

**Parameter table (version `jms_v185`, port 13338, ctx `JMS,185,1`; export `gms_jms_185.json`; report addrs from `docs/packets/audits/jms_v185/`):**

| Packet | FName | Committed report addr | Notes |
|---|---|---|---|
| SummonSpawn | `CSummonedPool::OnCreated` | `0x9f80f8` | avatar-look tail present; descend into `sub_823AED@0x823aed` |
| SummonRemove | `CSummonedPool::OnRemoved` | `0x828502` | confirm live |
| SummonMove | `CSummonedPool::OnMove` | `0x8286e4` | confirm live |
| SummonAttack | `CSummonedPool::OnAttack` | `0x828707` | confirm live; confirm trailing-flag gate is GMS-only (no jms trailing byte) |
| SummonDamage | `CSummonedPool::OnHit` | `0x828d16` | confirm live |
| SummonSkill | `CSummonedPool::OnSkill` | `0x828cb2` | confirm live |

- [x] **Step 1: Select the jms IDB.** `select_instance(13338)`; confirm name is the JMS v185 binary.
- [x] **Step 2: Resolve the jms dispatch chain** (`CSummonedPool::OnPacket @0x9F7F6E` + cid/oid pre-reads); record active leaf addresses. If a read function is SMC/undecompilable, escalate (§7).
- [x] **Step 3–8: Apply Recipe R-T1** to all six (one commit per cell). For SummonSpawn, confirm the existing `TestSummonSpawnBytesJMS185` body (avatar-look tail = +1 byte over the shared body) against the live decompile, then add its marker + pin evidence.
- [x] **Step 9: Full-family check for jms.** matrix + `matrix --check` exit 0; six jms cells ✅; `go test` PASS.

---

## Task 6: Final acceptance gate

**Files:** none new — verification only.

- [x] **Step 1: Whole-family matrix check.** `go run ./tools/packet-audit matrix && go run ./tools/packet-audit matrix --check` → exit 0. In STATUS.md, all six summon clientbound rows show ✅ for v83/v84/v87/v95/jms; **no 🟡 anywhere in the summon family**.
  ```bash
  grep -iE "SummonSpawn|SummonRemove|SummonMove|SummonAttack|SummonDamage|SummonSkill|SPAWN_SPECIAL|REMOVE_SPECIAL|MOVE_SUMMON|SUMMON_ATTACK|DAMAGE_SUMMON|SUMMON_SKILL" docs/packets/audits/STATUS.md
  ```
  Confirm zero `🟡` and zero `❌` on the six clientbound summon rows.
- [x] **Step 2: Marker/evidence parity.** Confirm 25 markers exist (24 tier-1 + 1 tier-0) and 24 evidence files exist (no v95-SummonMove evidence):
  ```bash
  grep -rn "packet-audit:verify packet=summon/clientbound" libs/atlas-packet/summon/clientbound/ | wc -l   # expect 25
  find docs/packets/evidence -name 'summon.clientbound.Summon*.yaml' | wc -l                                # expect 24
  ```
- [x] **Step 3: Go module gates (CLAUDE.md §Build & Verification).** In `libs/atlas-packet`:
  ```bash
  go test -race ./... && go vet ./... && go build ./...
  ```
  all clean. If §R3 forced a codec fix consumed by `services/atlas-channel`, additionally run the same three in `services/atlas-channel` **and** `docker buildx bake atlas-channel` from the worktree root.
- [x] **Step 4: Redis key guard.** From the repo root: `GOWORK=off tools/redis-key-guard.sh` → clean (no summon changes touch redis, but the gate is mandatory).
- [x] **Step 5: No stray artifacts.** `git status` shows only intended changes (fixtures, evidence YAMLs, optional re-pointed export/report, STATUS.md, status.json, plan/context docs). No `// TODO`, no stubbed tests.
- [x] **Step 6: Acceptance criteria sign-off.** Re-read `design.md` §9 and confirm each box is satisfied with evidence (matrix output, test output). Then proceed to code review (`superpowers:requesting-code-review`) before opening a PR, per CLAUDE.md.

---

## Self-Review (completed by plan author)

**Spec coverage** — every design.md item maps to a task:
- design §2 (25 cells): Task 1 (v95 SummonMove tier-0) + Tasks 2–5 (24 tier-1 cells). ✅
- design §3.1 (version-stratified grading, tier-0 vs tier-1 artifact sets): CRITICAL §2; R-T0 (no evidence) vs R-T1 (evidence). ✅
- design §3.2 (unconfirmed fixtures at wrong dispatch address; active-vs-inactive): CRITICAL §1, §4; Task 2 Step 2–3; §R5. ✅
- design §4 (Approach A strict per-cell live re-verification): the Recipe forbids blind ports; CRITICAL §1. ✅
- design §5.1/§5.2 (per-cell recipes): R-T0 / R-T1. ✅
- design §6.1 (artifact set corrected): R-T0/R-T1 + Task 6 Step 2 parity check. ✅
- design §6.2 (test pattern, no rename, no `*_testhelpers.go`): Files note + R-T1 step e. ✅
- design §6.3 (export non-idempotent, surgical splice): CRITICAL §5; §R5. ✅
- design §6.4 (acceptance bar strict 0): CRITICAL §6; Task 6 Step 1. ✅
- design §6.5 (build/verify gates; bake only if go.mod touched): Task 6 Steps 3–4; §R3 step 4. ✅
- design §7 risks (jms SMC, active-vs-inactive, latent wire bug, address drift): §7, §4; §R3; §R5; Tasks 2/5 notes. ✅
- design §8 sequencing (v95→v83→v84→v87→jms): Tasks 1→2→3→4→5. ✅
- design §9 acceptance criteria: Task 6. ✅
- PRD §4 (six packets to verified), §10 acceptance: Tasks 1–6. ✅

**Placeholder scan:** the `<…>` tokens are intentional parameter slots, each resolved by a per-task parameter table; no "TBD"/"implement later"/"add error handling" placeholders. Byte values are deliberately NOT pre-baked — they are derived live per CLAUDE.md "no inventing" (the whole point of verification). ✅

**Type/name consistency:** marker format, `evidence pin` flag names (`--packet/--version/--ida/--category`), version keys (`gms_v83/gms_v84/gms_v87/gms_v95/jms_v185`), export filename exception (`gms_jms_185.json` for key `jms_v185`), template names (`template_gms_83_1.json` etc.), and test-func naming (`TestSummon<Packet>Bytes<VER>`) are consistent across all tasks and match the live source read during planning. ✅
