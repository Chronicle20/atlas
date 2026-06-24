# Summon Clientbound Packet-Fixture Verification Campaign — Design

Task: task-106-summon-packet-fixtures
Status: Approved-pending
Created: 2026-06-23
Inputs: `prd.md` (approved)

---

## 1. Summary

Drive every `incomplete`/`partial` cell in the `summon` clientbound family of the
coverage matrix (`docs/packets/audits/STATUS.md`) to `verified` (✅) across all five
versions, by adding a `packet-audit:verify` byte-fixture per cell and the supporting
evidence the grader requires.

The PRD framed this as a near-mechanical "port the verified v95 read order across
versions." Grounding the campaign in the actual grader source and the existing test
files shows it is **genuine per-cell live re-verification**, not a blind port, for two
concrete reasons surfaced below (§3). The shape of each packet is largely known, but
the per-version read order must be confirmed field-by-field against each live IDB, and
the existing comments in the test files explicitly flag the current fixtures as
unconfirmed inference pointing at the wrong dispatch path.

This design **corrects three load-bearing assumptions in the PRD** — the grading model,
the per-cell artifact set, and the "v95 is already pinned" claim — and specifies the
exact, version-stratified promotion recipe.

## 2. Scope (the cells)

Six clientbound packets in `libs/atlas-packet/summon/clientbound/`:
`SummonSpawn`, `SummonRemove`, `SummonMove`, `SummonAttack`, `SummonDamage`,
`SummonSkill`. Production writers wrap these in
`services/atlas-channel/atlas.com/channel/socket/writer/summon.go`.

Current matrix state (regenerated and confirmed in this worktree):

| Packet | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|
| SummonSpawn | ❌ | ❌ | ❌ | ✅ | ❌ |
| SummonRemove | ❌ | ❌ | ❌ | ✅ | ❌ |
| SummonMove | ❌ | ❌ | ❌ | 🟡 | ❌ |
| SummonAttack | ❌ | ❌ | ❌ | ✅ | ❌ |
| SummonDamage | ❌ | ❌ | ❌ | ✅ | ❌ |
| SummonSkill | ❌ | ❌ | ❌ | ✅ | ❌ |

**25 cells to promote**: 24 `incomplete` (v83/v84/v87/jms × 6) + 1 `partial`
(v95 SummonMove). v95 for the other five packets is already ✅ and out of scope.

Out of scope (PRD non-goals, reconfirmed): the serverbound summon handlers
(`SummonMoveHandle`, `SummonDamageHandle`, `SummonAttackHandle`) — already verified on
all five versions; new summon gameplay; opcode reshifts.

## 3. The two findings that change the work

### 3.1 The grading model is version-stratified, not uniform (corrects PRD §4.3)

The grader (`tools/packet-audit/internal/matrix/grade.go`) does **not** treat the
summon family as a single tier:

- `summon/` is **not** in `docs/packets/evidence/tiers.yaml` (`packets: []`, and the
  `packet_prefixes` list covers `monster/`, `pet/`, `interaction/`, … but not
  `summon/`). The `(T1)` text in STATUS.md is a human label, **not** the grader's tier.
- Therefore the grader's tier for a summon clientbound cell is decided entirely by the
  audit report's `FlatInvalid` flag: `tier1 = in.Tier1[pkt] || rep.FlatInvalid`
  (grade.go:117). `in.Tier1["summon/clientbound/*"]` is **false**.

Inspecting the committed reports (`docs/packets/audits/<v>/Summon*.json`):

| Report | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|
| `Verdict` | 3 (🔍) | 3 | 3 | 0 (Match) | 3 |
| `FlatInvalid` | true | true | true | **false** | true |

`FlatInvalid` (`internal/report/report.go:22`) means the static analyzer could not
reduce a writer branch to a version predicate, so the flat positional diff is not
authoritative — the verdict is *capped* to 🔍. It is **a modeling limitation, not a
wire bug**; the report's own prose says "confirm per-branch via byte-level tests." A
byte-fixture is exactly what resolves it.

Consequence — two distinct promotion recipes:

- **v95 SummonMove (tier-0):** report is `Verdict=Match, FlatInvalid=false`. The
  grader's tier-0 verified rule is `toolPass && marker.Found` (grade.go:215). It needs
  **only a `packet-audit:verify` marker** on the existing byte-fixture. Per
  `VERIFYING_A_PACKET.md` §7, a tier-0 cell **must NOT** pin an evidence record.
- **v83/v84/v87/jms × 6 (tier-1 via `FlatInvalid`):** the grader's tier-1 verified rule
  is `marker.Found && hasEvidence && evidence.Fresh` (grade.go:199) — the flat report is
  fine (tier-1 does not require `toolPass`). Each needs **a marker AND a fresh pinned
  evidence record**.

This directly corrects the PRD, which said every promotion pins an evidence record. The
lone v95 cell must not; the other 24 must.

### 3.2 The existing fixtures are unconfirmed inference at the wrong dispatch address

`spawn_test.go` already carries v83 (`TestSummonSpawnBytesV83`) and jms
(`TestSummonSpawnBytesJMS185`) byte-fixtures — **without** verify markers — and its own
comments flag them as not trustworthy as-is:

- The v83 read order was corrected (via live x32dbg, task-088) to the **active** field
  dispatch `CSummonedPool::OnCreated @0x95ADEC`, whose dispatcher pre-reads `cid`. The
  committed export/report for v83 SummonSpawn points at the **inactive**
  `OnCreated @0x938F61` (no `cid` pre-read) — "the wrong path."
- v84/v87/jms "inherit this correction by inference … but have NOT been re-confirmed
  live — their coverage-matrix cells need re-verification against the cid-pre-reading
  dispatcher (the old `ida=` markers below point at the wrong path)."

This is a landmine: a marker's `ida=` address must agree with the evidence record /
report address, or `matrix --check` raises an **orphan-marker** failure (grade.go
failure modes). So for any cell where the committed export entry references the inactive
path, promotion requires re-pointing the export entry (surgical splice, §6.4) to the
active function so marker + report + evidence all agree — then pinning. Blindly adding a
marker over the existing fixture would either point at the wrong address (orphan) or
ratify an unverified byte layout (a false ✅ — the exact "spot-check presented as a full
sweep" / dispatcher-false-pass anti-pattern the project memory warns against).

Net: each of the 24 tier-1 cells is a real `VERIFYING_A_PACKET.md` pass against the live
IDB, not a copy of v95.

## 4. Approach decision

**Chosen: Approach A — strict per-cell live re-verification.** Every cell is decompiled
on its matching live IDB (all five are reachable: v83=13341, v84=13337, v87=13340,
v95=13339, jms=13338), the read order written down, the byte-fixture confirmed or
corrected field-by-field with decompile-line citations, and the marker/export/evidence
addresses made to agree on the **active** function. This matches CLAUDE.md ("full sweep,
not spot-checking"; "Verification Over Memory"; "no blind port") and resolves the
wrong-path markers the test comments already flag.

Alternatives considered and rejected:

- **B — trust-the-inference port** (add markers + pin evidence over the existing
  fixtures without live re-confirmation). Rejected: the fixtures' own comments say they
  are unconfirmed and point at the wrong dispatch path; this manufactures false ✅ cells
  and trips the orphan-marker check.
- **C — hybrid** (live-verify only spawn + the v95 partial, port the rest). Rejected:
  the marginal cost of full live verification is low (6 read functions × ≤5 live IDBs,
  all reachable) and the confidence gain is high; a partial sweep recorded as a full
  family flip is the false-pass failure mode.

## 5. Per-cell promotion recipe

For each cell, follow `docs/packets/audits/VERIFYING_A_PACKET.md`. Batch **by IDB**:
`select_instance` one version, do all six packets for it, then move on — never two IDBs
in parallel (the IDA instance selection is shared global state; project memory).

### 5.1 Tier-0: v95 SummonMove (the lone 🟡)
1. `select_instance(13339)`; decompile `CSummonedPool::OnMove`; write the read order
   (header + raw `CMovePath` movement blob — the codec rebroadcasts `rawMovement` byte-
   faithfully; the start position lives inside the blob, see `summon.go` comment).
2. Confirm the existing `TestSummonMoveBytesV95` bytes against that read order.
3. Add `// packet-audit:verify packet=summon/clientbound/SummonMove version=gms_v95 ida=<0xaddr>`
   above it. **Do not** pin evidence (tier-0; §7 of the playbook).
4. `packet-audit matrix`; cell → ✅; `matrix --check` exit 0. Commit fixture + matrix.

### 5.2 Tier-1: v83/v84/v87/jms × 6 packets
For each `<packet> × <version>`:
1. `select_instance(<port>)` for the version; verify by binary **name** (memory: ports
   are stable here but confirm the loaded IDB).
2. Decompile the **active** read function (`CSummonedPool::OnCreated`/`OnRemoved`/
   `OnMove`/`OnAttack`/`OnHit`/`OnSkill`), descending into helper reads
   (`CSummoned::Init`, AvatarLook, etc.). Resolve the active-vs-inactive dispatch trap
   (§3.2) — record the active address.
3. Compare against the codec in `libs/atlas-packet/summon/clientbound/<pkt>.go`,
   including its version gates (`spawnHasAvatarLook`, `MajorAtLeast(95)`/`(185)`, the
   attack target loop). Confirm v84 takes the v83-shaped clientbound path (the v84
   off-by-one class: gates must be `MajorAtLeast(95)`/region-correct, never a bare
   `>83`). **If the decompile contradicts the codec → it is a wire bug**: fix the codec
   in its own commit first (§4 of the playbook), then continue (PRD non-goal allows a
   fix only when a fixture proves a byte error).
4. Add/correct the per-version byte-fixture (extend the existing `*_test.go` table; one
   `func Test…Bytes<VER>` per version), citing the decompile line for every field.
5. If the committed export entry for the function references the wrong (inactive) path,
   **surgically splice** the active function into `docs/packets/ida-exports/<v>.json`
   (§6.4) and regenerate that one report (`-output /tmp`, copy the single report in) so
   report address = active address.
6. Add the `// packet-audit:verify packet=summon/clientbound/<P> version=<v> ida=<active 0xaddr>`
   marker. Marker address must equal the report/evidence address.
7. Pin evidence:
   `packet-audit evidence pin --packet summon/clientbound/<P> --version <v> --ida "<active FName>" --category TIER1-FIXTURE`,
   then add the `verifies: [<testfile>#<TestName>]` field by hand.
8. `packet-audit matrix`; cell → ✅; `matrix --check` exit 0.
9. Commit the coupled artifacts together (fixture + evidence + [export/report if
   re-pointed] + regenerated STATUS.md/status.json).

## 6. Mechanics & constraints

1. **Artifact set (corrected from PRD).** tier-0 (v95 SummonMove) = fixture+marker +
   regenerated matrix, **no evidence**. tier-1 (24 cells) = fixture+marker + fresh
   evidence record + regenerated matrix, **plus** a re-pointed export entry/report
   wherever the committed one references the inactive dispatch path.
2. **Test pattern.** Use the `test.Variants` table + `test.Encode`/`RoundTrip` helpers
   (`libs/atlas-packet/test/context.go`); reference `party/clientbound/invite_test.go`
   and the existing `spawn_test.go`. No `*_testhelpers.go`; Builder/table pattern only.
   Adding fixtures must not rename existing test funcs (tests reference internals).
3. **Export hygiene (§10 of the playbook).** The export is **non-idempotent** — never
   re-run a full `export`. To re-point/deepen one function: harvest to a temp file and
   splice **only** that entry into the committed export. Mismatched marker vs
   report/evidence address = orphan-marker failure; keep all three in sync.
4. **Acceptance bar.** `matrix --check` currently exits **0** at baseline (verified in
   this worktree). The bar is therefore a strict clean exit 0 after the work — no new
   orphan/dangling/stale/drift lines, conflict count stays 0.
5. **Build/verify gates (CLAUDE.md).** Changed module is `libs/atlas-packet` (fixtures)
   and possibly `tools/packet-audit` consumers / `services/atlas-channel` (only if a
   wire bug forces a writer fix). Run `go test -race ./...`, `go vet ./...`,
   `go build ./...` in each changed module; `tools/redis-key-guard.sh` clean. A
   `docker buildx bake` is required only if a service `go.mod` is touched — expected
   **not** to be (test-only changes in a lib), but mandatory if a codec fix lands in
   atlas-channel.

## 7. Risks

- **jms IDB is the SCY retail dump** (`MapleStory_dump_SCY.exe` @13338), not a
  `*_U_DEVM` build. Project memory warns the jms retail dump is SMC/control-flow-
  virtualized for some sends. The summon **read** functions decompiled before (the
  committed jms reports carry real addresses, e.g. `0x9f80f8`, `0x823aed`), so they are
  likely fine — but if a jms read function is genuinely undecompilable, that is a real
  blocker → escalate per the playbook (do not fabricate the read order).
- **Active-vs-inactive dispatch** (§3.2) beyond spawn: confirm per packet which
  `CSummonedPool::On*` is the live field-path target before pinning.
- **Latent wire bug.** The spawn codec already encodes `oid` on all versions and gates
  AvatarLook by region/version. If a live decompile (esp. v87/jms) contradicts an
  inferred gate, that surfaces a wire bug to fix-first (§5.2 step 3) — possible, not
  expected.
- **Marker/evidence/report address drift** → orphan/dangling `matrix --check` failures;
  mitigated by syncing all three to the active address in the same commit.

## 8. Sequencing

1. v95 SummonMove (tier-0, simplest — clears the lone 🟡; validates the marker→matrix
   loop end-to-end before the heavier tier-1 work).
2. v83 (active dispatch best understood from task-088; IDB currently active) — all six
   packets.
3. v84, then v87 — all six each; watch the v84 off-by-one gate class.
4. jms — all six; jms SMC risk handled last so a blocker there doesn't stall GMS cells.

Each version's six cells are one IDA session and a small batch of commits; the campaign
is ~6–8 commits. The eventual PR branch is produced by rebase at PR time (one worktree,
no mid-task forks).

## 9. Acceptance criteria (from PRD, made precise)

- [ ] All six summon clientbound packets show ✅ for v83, v84, v87, v95, jms in the
      regenerated matrix.
- [ ] No 🟡 cell remains in the summon family (v95 SummonMove resolved).
- [ ] Every tier-1 promoted cell has a `packet-audit:verify` byte-fixture **and** a
      fresh pinned evidence record (with `verifies:`) committed together; the v95
      SummonMove cell has a marker and **no** evidence record (tier-0).
- [ ] Every marker `ida=` address agrees with its report/evidence address (no orphan
      markers); any export re-point is a surgical splice, never a full re-export.
- [ ] `packet-audit matrix --check` exits 0 (the baseline is already 0).
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
      `libs/atlas-packet` (and any other module touched); redis-key-guard clean;
      `docker buildx bake` only if a service `go.mod` changed.
