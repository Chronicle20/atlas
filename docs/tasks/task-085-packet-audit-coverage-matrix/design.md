# Task-085: Evidence-Graded Packet Coverage Matrix + Tiered Byte-Fixture Oracle

Date: 2026-06-12
Status: Draft for review
Predecessor analysis: `retrospective.md` (same folder)

## 1. Problem

The packet-audit workstream (tasks 027–081) produced verdicts nobody trusts
without re-deriving them by hand, closure-by-prose-deferral instead of
verification, verdicts from four tool generations mixed in one baseline, and no
single artifact showing what is actually verified per packet × version. See
`retrospective.md` for the full evidence.

## 2. Goals

1. **One legible artifact**: a generated coverage matrix — packet operation ×
   version, split by direction — where every cell is `verified / partial /
   incomplete / n-a`, derivable by machine and readable by a human at a glance.
2. **Evidence-graded verdicts**: cell states are computed from machine-checkable
   evidence rules. Prose-only deferrals stop counting as done.
3. **Tiered oracle**: packets the flat diff cannot verify (opaque/mode-driven)
   require a byte-fixture test to reach `verified`; the diff verdict alone can
   never promote them.
4. **Automatic re-baselining**: any change to `tools/packet-audit`, the IDA
   exports, or `libs/atlas-packet` regenerates the matrix; drift appears as cell
   diffs in the PR.
5. **Version-applicability awareness**: packets absent from a region/version
   render as `n-a`, distinct from `incomplete`; packets Atlas routes that the
   version's client doesn't have become an explicit finding.
6. **Own the operation universe**: applicability lives in a repo-owned,
   IDA-evidenced operation registry. The hand-maintained Google-Sheets CSVs
   seed it once and then retire as a source of truth; per-version operation
   discovery comes from IDB review, which is exhaustive where hand-maintenance
   has tail-end gaps.

## 3. Non-Goals

- Rewriting the flat diff into a semantic/branch-aware diff engine. The tier-1
  byte-fixture oracle sidesteps that limitation instead.
- Live-wire capture / replay infrastructure (Direction C from brainstorming).
  Deferred to a future task; the evidence model below leaves room for a
  `capture` evidence kind.
- Handler/semantic verification (the NPC-discriminator and monster-book bug
  classes). Out of scope here; tracked by the matrix only insofar as scope
  boundaries become visible.
- Auditing versions beyond the five baselines (gms_v83, gms_v84, gms_v87,
  gms_v95, jms_v185). The matrix supports adding columns (e.g. gms_v92)
  cheaply, but populating them is a separate version-pass task. gms_v84 is in
  the baseline because a v84 tenant ships (task-083), but it has no IDA
  export or audit reports yet — its column starts honestly `incomplete` and
  is populated during rollout (registry discovery + export harvest + audit
  run). Task-083's finding that v84 is byte-identical to v83 predicts the
  column will largely mirror v83 once populated; the matrix verifies that
  rather than assuming it.

## 4. Architecture Overview

Three new pieces, all inside `tools/packet-audit`, plus one data migration:

```
op registry ──┐   (seeded once from the opcode CSVs; grown by IDA discovery)
audit runs ───┤
evidence/ ────┼──► matrix subcommand ──► docs/packets/audits/STATUS.md (+ status.json)
test-link ────┤
tier file ────┘
```

1. **Operation registry** (`docs/packets/registry/`): the per-version universe
   of operations — the rows and the applicability of every cell. Seeded from
   the CSVs, grown/corrected by IDA discovery (§5.2).
2. **Evidence ledger** (`docs/packets/evidence/`): structured per-packet,
   per-version records replacing prose acceptance in `_pending.md`.
3. **Byte-test linkage scanner**: discovers byte-fixture tests in
   `libs/atlas-packet` via machine-readable marker comments and binds them to
   matrix cells.
4. **`matrix` subcommand**: joins registry applicability, latest audit
   verdicts, evidence records, and test linkage into `STATUS.md` /
   `status.json`, applying the grading rules below. A `matrix --check` mode
   exits non-zero on evidence drift or grading regressions for CI.

## 5. Cell-State Semantics

For each (operation, direction, version) cell, evaluated in order:

| State | Symbol | Rule |
|-------|--------|------|
| **n-a** | ⬜ | The operation registry (§5.1) marks the packet absent in this version, AND the version's template does not route it, AND no Atlas encoder/decoder claims it for this version. |
| **conflict** | 🟥 | Applicability disagreement: the registry says absent but the template routes it (or vice versa), or Atlas code version-gates it into a version the registry excludes. Always a finding; never silently rendered as n-a or incomplete. |
| **verified** | ✅ | Tier 0: latest-tool ✅ verdict AND a linked byte-fixture test citing IDA evidence for this version. Tier 1: linked byte-fixture test citing IDA evidence (the diff verdict is advisory and cannot gate). |
| **partial** | 🟡 | Latest-tool ✅ verdict without a byte-test; OR an evidence-pinned deferral whose pinned decompile hash still matches the current IDA export. |
| **incomplete** | ❌ | Everything else — including every prose-only deferral existing today, ❌/🔍 verdicts without pinned evidence, and stale evidence (hash drift). |

"Latest-tool" means the verdict was produced by the current `tools/packet-audit`
commit against the current IDA exports. The matrix records the tool version
(git SHA of `tools/packet-audit` tree) and export hashes used per run; a verdict
produced by an older generation grades as if absent.

### 5.1 The operation registry

Applicability is owned by a repo artifact, not by the hand-maintained CSVs:

```
docs/packets/registry/
  gms_v83.yaml
  gms_v84.yaml
  gms_v87.yaml
  gms_v95.yaml
  jms_v185.yaml
```

(The CSVs have no v84 column; `gms_v84.yaml` is seeded as a copy of the v83
seed flagged `provenance: csv-import` with a v84 note, then corrected by
`discover-ops` against the v84 IDB — consistent with task-083's v84≡v83
finding while still verifying it.)

One file per version; one entry per operation present in that version:

```yaml
- op: LOGIN_STATUS                  # canonical op name, stable across versions
  direction: clientbound
  opcode: 0x000
  fname: "CLogin::OnCheckPasswordResult"
  provenance: csv-import | ida-discovered | manual
  ida:                              # required when provenance is ida-discovered
    address: 0x5e1230               # handler address / dispatch-table site
```

Applicability(op, direction, version) is then simply: entry present →
`present`; version file exists but no entry → `absent`; no registry file for
the version → `unknown` (cells render `incomplete` with an "applicability
unknown" note — never guessed).

The matrix cross-checks applicability against the per-version tenant template
(`services/atlas-configurations/seed-data/templates/`) and against Atlas
version gates discovered by the existing analyzer. Disagreement → `conflict`.
This generalizes the task-067/068 "template coverage gap" findings into a
standing check and catches the reverse case (Atlas emitting packets a version's
client cannot parse).

**CSV seeding (one-time)**: a `registry seed` subcommand parses
`docs/packets/MapleStory Ops - ClientBound.csv` / `... - ServerBound.csv`
(per-version index + hex-opcode pairs; absence encoded as empty index +
`0x000`) and emits registry files with `provenance: csv-import`. After seeding,
the CSVs are frozen as historical reference; corrections and additions go to
the registry, and STATUS.md is the artifact of record. The CSVs are known to be
reasonably accurate but missing operations at the tail end — which is exactly
what discovery (§5.2) repairs.

### 5.2 Operation discovery from the IDB

A `discover-ops` capability (exporter subcommand + playbook step) enumerates
the operation universe for a version directly from its IDA database:

- **Clientbound**: walk the client's packet dispatch (`CClientSocket::
  ProcessPacket` switch / handler registration table) and enumerate every
  handled opcode with its handler address and demangled name.
- **Serverbound**: enumerate `COutPacket` constructions / send-op constant
  sites to recover the set of opcodes the client can emit.

Discovery output reconciles against the registry: new ops are appended with
`provenance: ida-discovered` + the IDA address; ops the registry has that
discovery cannot find are flagged for review (CSV transcription error vs.
discovery blind spot — resolved by a human, recorded as `manual` provenance).
Each reconciliation run is a reviewable diff to `docs/packets/registry/`.

This makes a new version pass start from the binary itself: discover the op
universe first, then audit/verify against it — instead of trusting that a
hand-maintained sheet happens to cover the version.

## 6. Evidence Ledger

### 6.1 Layout

```
docs/packets/evidence/
  gms_v83/
    buddy.clientbound.Invite.yaml
    monster.clientbound.Spawn.yaml
  gms_v84/ …
  gms_v87/ …
  gms_v95/ …
  jms_v185/ …
```

One file per (packet, version) that needs evidence beyond a tool ✅. Schema:

```yaml
packet: buddy/clientbound/Invite        # pkg path + struct, matches audit report
direction: clientbound
version: gms_v83
category: OPAQUE | TRUNCATION | REPRESENTATION | OP-MODE-PREFIX |
          LOOP-EXCLUSIVE-BRANCH | VERSION-ABSENT | TIER1-FIXTURE
ida:
  function: "CWvsContext::OnFriendResult#Invite"
  address: 0xa3f2e8
  decompile_sha256: "ab12…"             # hash of the decompile text in the export
verifies:
  - libs/atlas-packet/buddy/clientbound/invite_test.go#TestInviteByteOutput_v83
notes: >
  Optional human context. Never consulted by grading.
```

### 6.2 Drift checking

`matrix --check` recomputes `decompile_sha256` from the current IDA export
(`docs/packets/ida-exports/*.json`) for every evidence record. Hash mismatch →
the record is stale → the cell degrades to `incomplete` and the check fails.
This is what makes a re-export or exporter improvement automatically invalidate
acceptance decisions that were based on the old dump — fixing retrospective
finding #4 (no re-audit loop).

### 6.3 Migration

The existing `_pending.md` entries, `OPAQUE_LEDGER.md` rows, and
`_unimplemented.json` allowlists are migrated into evidence records **only where
the original IDA citation is recoverable** (function + address present in the
prose). Entries without recoverable citations are NOT migrated — their cells
start as `incomplete`, which is the honest state. `_pending.md` and
`OPAQUE_LEDGER.md` are then frozen with a banner pointing at the evidence dir.
The allowlists remain consumed by the existing `validate` subcommand but no
longer influence grading.

### 6.4 Scaffolding: `evidence pin`

`packet-audit evidence pin --packet <pkg/dir/Struct> --version <key> --ida <fn-or-addr>`
scaffolds the evidence YAML: resolves the function in the version's IDA export,
computes `decompile_sha256` from the export text, and writes the record with
empty `verifies:`/`notes:` for the author to fill. Humans and agents never
hand-compute hashes; the same code path that pins is the one `--check` uses to
verify, so they cannot disagree.

## 7. Byte-Test Linkage

Byte-fixture tests are bound to cells via a marker comment scanned by the tool
(simple line scan, no AST dependency):

```go
// packet-audit:verify packet=buddy/clientbound/Invite version=gms_v83 ida=0xa3f2e8
func TestInviteByteOutput_v83(t *testing.T) { … }
```

Rules:

- One marker per (packet, version); a test may carry multiple markers (the
  common 4-variant sweep test carries four).
- The scanner verifies the marker's `ida` address appears in the matching
  evidence record (or, for tier-0 packets, in the audit report); orphan markers
  fail `matrix --check`.
- `go test ./...` passing in `libs/atlas-packet` is a precondition recorded by
  CI, not by the scanner — the scanner only proves linkage exists.

Existing byte-tests from tasks 065–069/080 are retrofitted with markers during
migration (they already cite IDA addresses in comments or commit messages).

## 8. Tier Definition

`docs/packets/evidence/tiers.yaml` enumerates tier-1 membership explicitly —
no inference:

- The 8 opaque type families from `OPAQUE_LEDGER.md` (mob temporary-stat,
  movement path, CPet body, AvatarLook blob, Asset/ItemSlotBase, GUILDMEMBER
  array, interaction Visitor/Room, GW_CharacterStat mask) — every packet that
  recurses into them.
- Mode-driven dispatcher families: party, guild, buddy, messenger, note, NPC
  conversation, interaction, storage, cash, memo.
- Any packet whose latest audit report carries `FlatInvalid`.

Everything else is tier 0. Moving a packet between tiers is a reviewed edit to
`tiers.yaml`. Tier 1 cells can only reach `verified` through a byte-fixture
test; a tool ✅ on a tier-1 packet renders at most `partial`.

## 9. STATUS.md Rendering

Generated, never hand-edited. Per direction, one table; rows grouped by domain
package, columns = versions:

```
## Clientbound

| Op | Packet | v83 | v84 | v87 | v95 | JMS185 |
|----|--------|-----|-----|-----|-----|--------|
| LOGIN_STATUS | login/AuthResult | ✅ | 🟡 | ✅ | ✅ | 🟡 |
| ACCOUNT_INFO | login/AccountInfo | 🟡 | ❌ | 🟡 | ✅ | ⬜ |
| SPAWN_MONSTER | monster/Spawn (T1) | 🟡 | ❌ | ❌ | 🟡 | ❌ |
```

Plus a per-version totals block (counts + percentages per state), a conflicts
section listing every 🟥 with its disagreement, and a generation stamp (tool
SHA, export hashes, date). `status.json` carries the same data for tooling.

The summary block is the human deliverable: at a glance, per version, how much
of the protocol surface is verified / partial / incomplete / absent.

## 10. Process Rules (replacing closure-by-deferral)

1. **Task-close gate**: a future audit/version task is done when every cell in
   its declared scope is `verified`, `partial`-with-evidence, or `n-a` — and
   the scope declaration itself is a list of matrix cells in the PRD. No prose
   acceptance.
2. **Re-baseline on change**: CI runs `matrix --check` on every PR touching
   `tools/packet-audit/`, `libs/atlas-packet/`, `docs/packets/ida-exports/`,
   or `docs/packets/evidence/`. The regenerated STATUS.md is committed in the
   same PR; cell regressions (any cell degrading) fail unless the PR
   description owns them.
3. **New version pass**: run `discover-ops` against the new IDB to produce the
   version's registry file, add the template and IDA export; the matrix
   auto-grows with cells pre-filled from applicability (⬜/❌). The pass's job
   is turning ❌ into ✅/🟡 — progress is the matrix diff itself.
   `STARTING_A_NEW_VERSION_PASS.md` is updated to this workflow, including the
   task-081 subcommand invocations it currently omits.
4. **Shared sub-structs get rows**: registry types audited in their own right
   (GW_CharacterStat, stat-registry, AvatarLook, …) appear in a separate
   sub-struct matrix section with the same grading, so cross-domain structures
   have an owner (fixes the v87 stat-registry escape class).

## 11. Repeatable Verification Workflow (playbook, skill, agent)

The matrix defines *what done means*; this section codifies *how a cell gets
promoted*, because the verification loop is the part of tasks 027–081 that
worked but lived only as tribal knowledge. Three layers, each building on the
previous:

### 11.1 Playbook: `docs/packets/audits/VERIFYING_A_PACKET.md`

The canonical single-packet × single-version procedure, written for a human or
any agent. Steps:

1. **Resolve scope**: look up the op in the operation registry; confirm
   applicability for the target version (absent → the work is confirming `n-a`
   or filing a `conflict`, then stop).
2. **Check current state**: the cell in `STATUS.md`, any existing evidence
   record, the latest audit report for the packet.
3. **Decompile the client side**: connect to the version's IDA instance
   (enumerate live instances and `select_instance` the one whose loaded IDB
   matches the target version — ports vary by IDA launch order, never
   hardcode them), decompile the FName from the registry entry, descend into helper
   reads/writes (address-based, same rule as the exporter), and write down the
   full ordered read/write list including guards and loop bounds.
4. **Compare against Atlas**: the encoder/decoder in `libs/atlas-packet`,
   including version gates. Divergence → wire fix first (own commit, own
   review), then continue.
5. **Derive expected bytes**: construct a concrete model fixture, hand-compute
   the expected byte sequence from the client read order (one fixture per mode
   for mode-driven packets).
6. **Write the byte-test** with the `packet-audit:verify` marker (§7).
7. **Pin evidence**: `packet-audit evidence pin …` (§6.4), fill `verifies:`.
8. **Regenerate**: run `matrix`; confirm the cell promoted; commit test +
   evidence + STATUS.md together.

`STARTING_A_NEW_VERSION_PASS.md` becomes a thin orchestration doc: set up the
new column/template/export, then apply this playbook per cell, hottest tier
first.

### 11.2 Skill: `/verify-packet <packet> <version>`

A project skill (same pattern as `convert-npc` etc.) that walks a Claude
session through the playbook with exact tool mechanics: registry lookup, the
ida-pro-mcp call sequence (batch decompile + callee descent), the marker and
evidence formats, and the §13 failure modes (unresolvable citation, orphan
marker). The skill enforces stop-points: it never fabricates expected bytes
from MapleStory knowledge — every byte in a fixture must trace to a decompile
line (Verification Over Memory rule).

### 11.3 Agent: `packet-verifier`

An agent definition wrapping the skill for fan-out during phase-4 fixture
campaigns: a family sub-task dispatches one `packet-verifier` per packet ×
version, each producing the three artifacts (byte-test, evidence record,
promoted cell) on the task branch. Multi-instance ida-pro-mcp makes concurrent
per-version agents viable; the dispatcher batches per IDB to respect what the
IDA server can load. Agent output is reviewable because the artifacts are
machine-checked (`matrix --check`) — a verifier that handwaves produces a cell
that stays ❌, not a prose claim.

## 12. Implementation Phases

Each phase lands independently and is useful on its own.

1. **Registry + matrix generator (read-only)**: `registry seed` (one-time CSV
   import → `docs/packets/registry/*.yaml`), applicability model, `matrix`
   subcommand joining registry + existing audit JSON; STATUS.md with
   everything graded honestly (mostly 🟡/❌ at first; no evidence ledger yet —
   deferrals all render ❌). Includes the conflict check; freezes the CSVs.
2. **Evidence ledger + drift check**: schema, loader, `--check` hash
   verification; migrate the recoverable subset of `_pending.md` /
   `OPAQUE_LEDGER.md`; freeze the prose files.
3. **Byte-test linkage**: marker scanner; retrofit markers onto existing
   byte-tests; `verified` promotion goes live.
4. **Playbook + skill + agent** (§11): write VERIFYING_A_PACKET.md, the
   `/verify-packet` skill, and the `packet-verifier` agent; validate by
   verifying 2–3 packets end-to-end (one tier-0, one tier-1 mode-driven, one
   opaque-family member) before any campaign starts.
5. **Operation discovery** (§5.2): `discover-ops` against the five baseline
   IDBs (including v84); reconcile against the seeded registry. This both
   repairs the CSV tail-end gaps for current versions and proves the workflow
   that future version passes will start with. For v84 this phase also
   harvests the IDA export and runs the first audit pass, bringing its column
   to parity with the other four.
6. **Tier-1 fixture campaign**: per-family byte-fixture work via
   `packet-verifier` fan-out, ordered by risk (character stat →
   spawn/movement → inventory/asset → dispatcher families). Each family is its
   own bounded sub-task with matrix-cell scope.
7. **Process wiring**: CI `matrix --check` job, STARTING_A_NEW_VERSION_PASS.md
   rewrite, task-close-gate documentation in the audit task template.

Phases 1–5 + 7 are tool/process work (bounded). Phase 6 is the long tail and
intentionally split into family-sized sub-tasks so no single task can "close
with deferrals" — an unfinished family is simply still ❌ in the matrix.

## 13. Error Handling

- CSV rows with unparseable opcode pairs → `registry seed` fails loudly with
  row numbers; seeding must be corrected, not skipped.
- Registry entries with invalid schema or duplicate (op, direction) per
  version → `matrix` fails loudly with file/entry identified.
- `discover-ops` reconciliation conflicts (registry entry discovery can't
  find, or discovered op colliding with an existing entry's opcode) → emitted
  as a review worklist; never auto-resolved.
- Evidence file referencing a packet/version with no audit report → `--check`
  failure (dangling evidence).
- Marker comment referencing a missing evidence record or mismatched address →
  `--check` failure (orphan marker).
- IDA export missing a function an evidence record cites → cell degrades to
  `incomplete` with a "citation unresolvable" note; `--check` fails.
- Two Atlas structs claiming the same (op, direction, version) → `conflict`.

## 14. Testing

- `registry seed`: table-driven tests over real CSV excerpts covering
  present/absent, the `Index`-column quirk on the first version pair, and
  multiline FName cells (e.g. `CLogin::Init` rows).
- Registry loader: applicability semantics (present/absent/unknown),
  duplicate detection, provenance round-trip.
- `discover-ops` reconciliation: synthetic discovery output vs seeded
  registry covering append, missing-at-discovery flag, and opcode collision.
- Grading: unit tests per rule in §5, including precedence (conflict beats
  n-a, stale evidence degrades, tier-1 ✅-without-fixture caps at partial).
- Drift: fixture test where an export hash changes and the cell degrades.
- Marker scanner: golden-file tests over real test-file excerpts.
- End-to-end: a small synthetic registry + audit JSON + evidence dir + test
  tree producing a golden STATUS.md / status.json.
- Determinism: two consecutive runs produce byte-identical output.

## 15. Risks

- **Migration honesty shock**: the first STATUS.md will show far less green
  than the task-080 "closeout" implied (most 🔍 acceptances lack recoverable
  citations). This is the point, but it should be expected in review.
- **Marker discipline**: marker comments can rot if tests are renamed; the
  orphan-marker check catches deletion but not semantic drift of the test body.
  Mitigated by keeping fixtures (expected bytes) inside the linked test.
- **Seed inaccuracy**: the CSVs are hand-maintained and known to miss
  tail-end operations, so the seeded registry inherits those gaps until
  phase-5 discovery repairs them. The conflict state surfaces
  registry-vs-template-vs-code disagreements rather than trusting any side
  blindly, but seed errors will generate conflict noise until discovery runs.
- **Discovery blind spots**: `discover-ops` depends on locating the dispatch
  table / send-op sites per binary; obfuscated or optimized dispatch may hide
  ops. Reconciliation treats discovery as additive evidence, not as authority
  to delete registry entries — removal is always a human decision recorded as
  `manual` provenance.
