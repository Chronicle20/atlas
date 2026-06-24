# Door Clientbound Packet-Fixture Verification Campaign — Design

Task: task-107-door-packet-fixtures
Phase: 2 (Design)
Created: 2026-06-23
Status: Draft for review

---

## 1. Problem & Framing

The `door` family in the coverage matrix (`docs/packets/audits/STATUS.md`) is the
lowest-percentage implemented family (~20%). The three clientbound door packets are
`verified` only for `gms_v83`; they are `incomplete` for `gms_v84`, `gms_v87`,
`gms_v95`, and `jms_v185`. The goal is to drive all 12 `incomplete` cells to `verified`
(✅), landing each as the three coupled artifacts the playbook
(`docs/packets/audits/VERIFYING_A_PACKET.md`) requires.

The PRD framed this as a *port-the-verified-read-order-across-versions* campaign on the
assumption that "only per-version opcodes and any version-shifted offsets differ." That
framing is **half right**. Investigation surfaced one fact that materially reshapes the
work, documented in §3.

### 1.1 What was confirmed from source

- **Writer ownership (PRD Open Question 1, resolved).** The door clientbound writers
  live in `libs/atlas-packet/door/clientbound/`:
  - `spawn.go` → `SpawnDoor` (op `SPAWN_DOOR`, client recv `CTownPortalPool::OnTownPortalCreated`)
  - `remove.go` → `RemoveDoor` (op `REMOVE_DOOR`, client recv `CTownPortalPool::OnTownPortalRemoved`)
  - `remove_town.go` → `RemoveTownDoor` (op `SPAWN_PORTAL`, client recv `CWvsContext::OnTownPortal`, town=true branch)
  - `spawn_portal.go` → `SpawnPortal` (op `SPAWN_PORTAL`, same recv, live-portal branch) — **not a matrix cell**; see §9.

  Production usage is in `services/atlas-channel/.../socket/writer/door.go` and the door
  Kafka consumer. The encoders take no version branch — a single layout for all tenants.

- **`RemoveDoor` vs `RemoveTownDoor` (PRD Open Question 2, resolved).** They are **two
  distinct opcodes**, not one opcode with a flag:
  - `REMOVE_DOOR` (field-side despawn) → `writeByte(0)` + `writeInt(ownerId)` = 5 bytes.
  - `SPAWN_PORTAL` town-removal → `writeInt(NONE)` + `writeInt(NONE)` = 8 bytes (no
    position; the client's `OnTownPortal` guards the x/y reads on `id != 999999999`).
  `SpawnPortal` (live portal) shares the `SPAWN_PORTAL` opcode but writes 12 bytes
  (adds position). The byte-distinction is already pinned by the v83 fixtures.

- **Genuinely version-absent? (PRD Open Question 3).** No. All three ops appear in every
  version's registry (`docs/packets/registry/<v>.yaml`) with non-`n-a` opcodes, so every
  target cell is a real gap to close, not an `n-a`.

---

## 2. Promotion Mechanism (how a cell becomes ✅)

Grounded against the in-repo reference `reactor/clientbound/ReactorDestroy` (tier-0,
clientbound, `verified` on all five versions) and the sibling
`field/serverbound/FieldUseDoor` (verified on all five). A clientbound cell promotes when
**all** of the following exist for that `packet × version`:

1. **A per-version audit report** at `docs/packets/audits/<version>/<Writer>.{json,md}`.
   Its absence is exactly the `"note": "no audit report"` recorded on every incomplete
   door cell in `status.json`. This is the dominant missing artifact.
2. **A `packet-audit:verify` marker** above the byte-test, citing that version's IDA
   address:
   `// packet-audit:verify packet=door/clientbound/<Struct> version=<key> ida=0x<addr>`
3. **(Tier-1 only) a pinned evidence record** at
   `docs/packets/evidence/<version>/door.clientbound.<Struct>.yaml` with a `verifies:`
   field naming the test.

`ReactorDestroy` confirms the tier-0 shape: 5 stacked markers, 5 per-version reports,
**no evidence records**. `FieldUseDoor` confirms the serverbound/tier-1 shape: stacked
markers **plus** a per-version evidence record.

### 2.1 Tier classification (drives whether evidence is pinned)

From `status.json`:

| Packet | Tier | Per-version evidence pin? |
|---|---|---|
| `door/clientbound/SpawnDoor` | tier-0 (`tier1:false`) | No |
| `door/clientbound/RemoveDoor` | tier-0 (`tier1:false`) | No |
| `door/clientbound/RemoveTownDoor` | **tier-1 (`tier1:true`)** | **Yes** |

(The PRD's `(T1)` annotations are inconsistent with `status.json`; `status.json` is
authoritative. `RemoveTownDoor` is the only tier-1 packet in scope.)

---

## 3. The Reshaping Finding: the receiver functions are absent from 4 exports

`grep` of `docs/packets/ida-exports/` shows the three client receiver functions
(`CTownPortalPool::OnTownPortalCreated`, `CTownPortalPool::OnTownPortalRemoved`,
`CWvsContext::OnTownPortal`) exist **only in `gms_v83.json`**. They are **not present in
`gms_v84.json`, `gms_v87.json`, `gms_v95.json`, or `gms_jms_185.json`**.

Consequences:

- **Export-only verification is impossible.** Report generation (playbook §9 step 3)
  descends from the registry `fname` and fails with "not in export" when the function is
  absent. We cannot synthesize the 12 reports without first getting these functions into
  each export.
- **A fresh full re-export is forbidden** (playbook §10: re-running `export` drifts
  ~150 unrelated function keys and degrades other cells). The only sanctioned path is a
  **surgical, absent-only splice** of the harvested function entries into each committed
  export.
- **The functions may be unnamed in the v84/v87/v95/jms IDBs.** Absent-from-export does
  not tell us whether they are named-but-unharvested or genuinely unnamed. Per memory,
  v95 is well-named, v87 had naming groundwork, v84 is byte-identical to v83, and jms
  must use the `*_U_DEVM` build. Where a receiver is unnamed we **name it** (playbook §10
  byte-signature + twin-match against the v83 named twin) — naming is a producible step,
  not a blocker.

This turns task-107 from "add a marker line per cell" into a **per-version IDA-harvest →
export-splice → report-gen** campaign. It is still the smallest family campaign (3
packets, 4 versions), but the per-cell cost is the harvest/splice, not the fixture.

---

## 4. Architecture: the per-cell pipeline

Each of the 12 cells flows through the same deterministic pipeline. Cells are grouped
**by IDB** (one `select_instance` target at a time) because the IDA instance is shared
global state and the export is per-version — never interleave two versions.

For each version V ∈ {gms_v84, gms_v87, gms_v95, jms_v185}, for each of the 3 receivers:

1. **Select the instance.** Enumerate `mcp__ida-pro__list_instances`, `select_instance`
   the one whose loaded IDB matches V. **Never hardcode ports** (playbook §3; the PRD's
   port list is a hint, not a contract — confirm the loaded version per session).
2. **Locate the receiver.** Find `CTownPortalPool::OnTownPortalCreated` /
   `...OnTownPortalRemoved` / `CWvsContext::OnTownPortal`. If unnamed, name it via the
   send/recv signature and structure-match to the v83 twin; record the address.
3. **Decompile & confirm read order.** Decompile the receiver (descend into helper
   reads, address-based). Write the full ordered read list and compare to the v83
   reference read order (SpawnDoor: `Decode1(launched) Decode4(ownerId) Decode2(x)
   Decode2(y)`; RemoveDoor: `Decode1 Decode4`; OnTownPortal: `Decode4 Decode4` then
   guarded `Decode2 Decode2`). **A divergence is a wire bug** (see §6).
4. **Splice the export.** Harvest the receiver (+ any deep helpers) to a temp file
   (`-prior-export "" -pending <roster> -descent-depth 12`), then **surgically splice
   only the needed entries** into `docs/packets/ida-exports/<V export>.json`
   (absent-only for helpers; this is an *add*, never an overwrite of existing keys).
5. **Generate the report.** Run the root `packet-audit` command for V with its csv +
   `template_<V>.json` + the now-complete `-ida-source` to a temp `-output`; copy
   `<Writer>.{json,md}` into `docs/packets/audits/<V>/`.
6. **Add the marker.** Append a `packet-audit:verify ... version=<V> ida=0x<addr>` line
   above the existing test in the matching `*_test.go` (stacked, mirroring
   `ReactorDestroy`). The existing cross-version equality loop already proves byte
   identity; no new test body is needed unless §6 finds a delta.
7. **(RemoveTownDoor / tier-1 only) pin evidence.**
   `packet-audit evidence pin --packet door/clientbound/RemoveTownDoor --version <V>
   --ida "CWvsContext::OnTownPortal" --category TIER1-FIXTURE`, then hand-add the
   `verifies:` line.
8. **Regenerate & check.** `packet-audit matrix` then `matrix --check` (+ `fname-doc
   --check`, `operations --check`). Confirm the cell flipped to ✅ and no new
   problems mention a door packet (§7 bar).

**Commit granularity.** One commit per `packet × version` cell, carrying its three (or
two, for tier-0) coupled artifacts — export splice, audit report, marker (+ evidence) —
plus the regenerated `STATUS.md`/`status.json`. This keeps each promotion atomic and
reviewable, matching the playbook's "commit the artifacts together" rule. (The
`STATUS.md`/`status.json` regen is shared state; sequence commits so each regen reflects
exactly the cells landed so far.)

### 4.1 Per-version opcode / receiver reference

| Op (recv fn) | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|
| `SPAWN_DOOR` (`OnTownPortalCreated`) | 0x113 ✅ | 0x11A | 0x124 | 0x14A | 0x128 |
| `REMOVE_DOOR` (`OnTownPortalRemoved`) | 0x114 ✅ | 0x11B | 0x125 | 0x14B | 0x129 |
| `SPAWN_PORTAL` (`OnTownPortal`) | 0x043 ✅ | 0x045 | 0x045 | 0x045 | 0x03D |

(Opcodes from `STATUS.md` / per-version `registry/<v>.yaml`, cross-checked. These are the
**dispatch** opcodes; the receiver function address per version comes from step 2/3, not
from the opcode.)

---

## 5. Wire shape is already proven byte-identical

The existing v83 tests (`spawn_test.go`, `remove_test.go`, `remove_town_test.go`,
`spawn_portal_test.go`) already loop over `pt.Variants` (which includes v83, v84, v87,
v95, jms_v185, v86) and assert **byte-equality with v83** for every variant. So the
encoder output is already machine-proven identical across all target versions. The IDA
work in §4 is therefore primarily about producing the *evidence the grader consumes*
(report + marker + tier-1 evidence) and **independently confirming** via decompile that
each version's client truly reads that v83 layout — not about expecting a code change.

This is also the campaign's main correctness guard against a false pass: the cross-version
equality assertion is real byte verification, not mode-byte enumeration. We are not
shortcutting; we are confirming the already-pinned bytes against four more clients.

---

## 6. If a decompile reveals a structural delta (wire fix path)

If step 3 finds a version whose client read order differs from v83 (e.g. an inserted
field, a changed guard, a different discriminant), that is a genuine wire bug in the
unbranched encoder. Per PRD non-goals and playbook §4, the fix path is:

1. **Surface it** (do not silently patch). The encoder currently asserts "layout
   identical across all tenant versions" in its doc comment — a delta falsifies that.
2. Land the **wire fix first** as its own commit (add the version branch to the encoder,
   update the cross-version test to expect the divergence), with its own review.
3. Then resume the verification pipeline for that cell against the corrected encoder.

No delta is expected (the v84≡v83 rule and the simple fixed-width bodies make one
unlikely), but the pipeline must not assume identity — it must read each client.

---

## 7. `matrix --check` exit-code bar

Per playbook §8, `matrix --check` currently exits 1 from a pre-existing 🟥 registry-seed
conflict backlog unrelated to door. The acceptance bar for this task is therefore **"no
new problems"**, not a clean exit 0:

- Zero orphan / dangling / stale / drift lines mentioning any `door/clientbound/*` packet.
- The global conflict count must **not increase**.
- Every door cell in scope reads ✅ after regen.

`fname-doc --check` and `operations --check` must likewise introduce no new failures.

---

## 8. Alternatives considered

1. **Export-only verification (rejected — infeasible).** Derive addresses and reports
   purely from committed exports. Impossible: the receiver functions are absent from four
   of five exports (§3). Report-gen would fail "not in export."
2. **Fresh full re-export per version (rejected).** Would surface the functions but
   drifts ~150 unrelated keys and degrades other cells (playbook §10). Surgical
   absent-only splice is the only sanctioned path.
3. **One test function per version vs stacked markers on the existing test
   (chosen: stacked markers).** The repo idiom (`ReactorDestroy`, `FieldUseDoor`,
   `PartyLeft`) stacks per-version `packet-audit:verify` lines above a single
   table-driven test that already iterates `pt.Variants`. Adding per-version test
   functions would duplicate the cross-version loop for no grading benefit. Stacked
   markers it is.
4. **Skip the audit reports and rely on markers alone (rejected).** The `"no audit
   report"` note proves the grader requires a per-version report for these clientbound
   cells; markers alone won't promote them.

---

## 9. The `SpawnPortal` wrinkle (out of scope, documented)

`spawn_portal.go` defines a fourth writer, `SpawnPortal` (live town portal, 12 bytes),
which shares the `SPAWN_PORTAL` opcode with `RemoveTownDoor`. It has a v83 evidence
record (`docs/packets/evidence/gms_v83/door.clientbound.SpawnPortal.yaml`) and a v83
audit report, **but no op row in `status.json`** — the `SPAWN_PORTAL` matrix row is
mapped to `RemoveTownDoor`. `SpawnPortal` is therefore not a tracked matrix cell and is
**out of scope** for task-107 (PRD lists only three packets). We will:

- Not add `SpawnPortal` cells or markers for v84/v87/v95/jms.
- When confirming `OnTownPortal` for `RemoveTownDoor`, the same decompile also covers the
  live-portal (SpawnPortal) branch — note it in the read-order write-up but do not pin it.
- Flag in the PR description that `SpawnPortal` is an untracked-but-evidenced writer, in
  case the matrix should later gain a row for it (a separate task).

This is surfaced, not silently skipped.

---

## 10. Service / module impact

- **`libs/atlas-packet`** — only `*_test.go` files change (added markers; new test bodies
  only if §6 triggers). **No `go.mod` is touched** → per CLAUDE.md the `docker buildx
  bake` gate is conditional on a `go.mod` change and does **not** apply here. Verification
  is `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `libs/atlas-packet`
  (plus `tools/redis-key-guard.sh` from repo root, which is unaffected).
- **`docs/packets/`** — per-version export splices, audit reports, evidence records (for
  RemoveTownDoor), and regenerated `STATUS.md`/`status.json`.
- **`tools/packet-audit`** — used as-is; no code change expected (the door receivers link
  by `fname`, which already matches across versions, so no new `candidatesFromFName` case
  is needed — that switch is serverbound-only per playbook §9).
- **Production Go code** — unchanged unless §6 finds a wire delta.

---

## 11. Verification plan (acceptance mapping)

| PRD Acceptance Criterion | How this design satisfies it |
|---|---|
| All 3 door clientbound packets ✅ on v83–jms | §4 pipeline run for all 12 cells; §4.1 opcode table |
| Each promoted cell has a `packet-audit:verify` fixture + (tier-1) fresh pinned evidence committed together | §2 mechanism; §4 steps 6–7; §2.1 tier table (evidence for RemoveTownDoor only) |
| `matrix --check` / `fname-doc` / `operations` exit cleanly | §7 "no new problems" bar |
| Affected module test/vet/build clean; bake for touched `go.mod` | §10 — only `libs/atlas-packet` test files change; no `go.mod` touched → no bake required |

Definition of done: all 12 cells ✅ in regenerated `STATUS.md`; per-version reports +
markers (+ RemoveTownDoor evidence) committed atomically per cell; `libs/atlas-packet`
green on `go test -race`/`vet`/`build`; `matrix --check` introduces no new door-related
problems and the conflict count does not increase.

---

## 12. Risks & mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Export splice corrupts a committed export (non-idempotent `export`) | Medium | Absent-only surgical splice of named entries; never run full `export` over a committed file; diff the splice before commit (playbook §10) |
| Receiver unnamed in v84/v87/v95/jms IDB | Medium | Name it via send/recv byte-signature + v83-twin structure match (§3); naming is producible, not a blocker |
| jms retail IDB is SMC / undecompilable | Low (door is a simple recv) | Use the clean `*_U_DEVM` jms build (playbook §10) |
| A version's read order genuinely diverges from v83 | Low | §6 wire-fix-first path; cross-version test updated to expect the divergence |
| `matrix --check` pre-existing conflicts mask a new door regression | Low | §7 bar checks door-specific lines + conflict-count delta, not raw exit code |
| `COutPacket`-delegate harvest artifact blocks report-gen | Low | Strip the delegate ctor call from the spliced entry (playbook §10) |

---

## 13. Open questions — all resolved in design

- Writer ownership → `libs/atlas-packet/door/clientbound/` (§1.1).
- `RemoveDoor` vs `RemoveTownDoor` → two distinct opcodes, distinct bodies (§1.1).
- Any genuinely version-absent door packet → no, all present in every registry (§1.1).

No open questions remain for the planning phase.
