# Task 169 — Design

Technical approach + key decisions per requirement. Ordered by the plan's phases. The guiding
constraint throughout: **no existing verified-cell count moves** except deliberate ⬜→n-a /
🟡-disambiguation, and every tool-behavior change is TDD with a "fires on violation, passes
clean" pair.

---

## P1 — De-drift & single-source docs (FR-2.1/2.2)

**Decision: canonical-doc + thin-pointer.** One playbook owns each procedure; commands/agents
link to it and stop restating rules.
- Canonical owners: `VERIFYING_A_PACKET.md` (single-cell verify), `IMPLEMENTING_A_PACKET.md`
  (new codec), `STARTING_A_NEW_VERSION_PASS.md` (version bring-up), `DISPATCHER_FAMILY.md`
  (dispatcher family). `.claude/commands/verify-packet.md` + `agents/packet-verifier.md` drop
  their re-listed "5 rules"/"steps 1-8" and say "follow `VERIFYING_A_PACKET.md`
  §0-10 verbatim; do not paraphrase."
- Stale-fact fixes (mechanical, enumerated in the plan as a checklist so the freshness lint
  can later assert them): version universe 5→9 (name v48/v61/v72/v79); baselines are empty
  (party/guild/buddy graduated); `matrix --check` is a hard gate (delete continue-on-error
  language); add `🧩` to the state legend everywhere ✅/🟡/❌/⬜/🟥 is listed; correct the
  README `export` flags to the real `runExport` set.
- **Single source of truth for the "true facts" a doc asserts:** introduce
  `docs/packets/PROCESS.md` — a short top-of-tree index that states the current version set,
  baseline status, CI gate list, and links each playbook to its entry point. The freshness
  lint (FR-2.3) validates the machine-checkable subset of this file. CLAUDE.md +
  superpowers-integration link here (FR-1.4).

## P2 — Matrix expressiveness & visibility

### FR-4.1 Sub-struct n-a (the load-bearing render fix)
Today `gradeSubStructCell` (`internal/matrix/build.go`) hardcodes `applicability=Present,
routed=true`, so a sub-struct can never grade `StateNA`; a genuinely-inapplicable shared type
is `⬜` only by *absence of a report* — indistinguishable from unaudited.

**Decision: drive sub-struct applicability from `_unimplemented.json`.** That file already
holds per-version n-a dispositions for ops; extend its consumption to sub-structs.
- Loader already reads `_unimplemented.json` per version; expose a
  `Unimplemented[version][packetID] bool` set to `Build`.
- `gradeSubStructCell`: if `(packetID, version) ∈ Unimplemented`, grade `StateNA` (with the
  disposition note) instead of forcing `Present`. This is the *only* behavioral change; a
  sub-struct with no report and no disposition stays exactly as today.
- Migration: the existing `_unimplemented.json` entries authored during task-113 (v48 MTS,
  etc.) that name sub-structs will now render n-a instead of ❌ — this is the intended
  ⬜/❌→n-a reclassification called out in AC-3. Capture the before/after delta explicitly.
- Test: `build_test.go` case — sub-struct in Unimplemented → NA; not in → unchanged.

### FR-4.2 Partial disambiguation
`gradeCore` emits 🟡 for three notes: `tier-1: needs byte-fixture`, `tool ✅ without byte-test`,
`evidence-pinned deferral`. The `State` enum must stay stable (status.json contract; model.go
forbids reorder).

**Decision: render-only differentiation via the existing `Note`.** No new State. `render.go`
maps the three known partial notes to a disambiguating glyph/suffix in STATUS.md
(e.g. `🟡ᶠ` needs-fixture / `🟡ᵈ` diff-only / `🟡ᵖ` pinned-deferral) and a legend row. `status.json`
is unchanged (Note already carries the distinction). Pure render layer → zero grading risk.

### FR-4.3 Per-version support summary
**Decision: a `matrix` render mode, not a new pipeline.** Add `matrix --summary <version>` (or
a `support-summary` subcommand) that reads the already-built `status.json` and emits
`docs/packets/audits/support/<version>.md`: totals, verified%, and a gap table split into
`n-a (deliberate)` vs `unverified (open)` vs `conflict`. Deterministic (no date), committed,
regenerated alongside the matrix. Drives FR-4.4.

### FR-4.4 `status <version>` query
**Decision: thin CLI over status.json.** `packet-audit status <version>` prints to stdout: the
summary numbers + the open-gap list + any stale-evidence rows (from the same freshness data
`matrix --check` computes). Read-only, no files written. Reuses the FR-4.3 aggregation.

### FR-4.5 Per-arm rollup (stretch)
If time permits: `status <version> --family <name>` cross-references `dispatchers/<family>.yaml`
arms against evidence/markers and prints "k/n arms verified". Non-blocking.

### FR-5.1 Family-cap guard
`evidence/families.yaml` is empty (all graduated), so `🧩` caps nothing; a new dispatcher
family would silently escape capping.
**Decision: a completeness check, not re-populating the baseline.** Add to `dispatcher-lint`
(or a new `--check`): every `docs/packets/dispatchers/*.yaml` family must be either (a)
represented by discrete per-mode structs that pass INV-1..5, or (b) explicitly listed in
`families.yaml`/baseline. A brand-new `dispatchers/*.yaml` with no discrete implementation and
no baseline entry fails. This preserves "graduated = fine" while closing the silent-loss hole.

## P3 — Executable entry points (FR-1)

**Decision: agents wrap playbooks; commands wrap agents/orchestration.** Mirror the existing
`packet-verifier` / `dispatcher-family-implementer` shape (`.claude/agents/*.md` +
`.claude/commands/*.md`).
- **FR-1.1 `packet-implementer`** (agent) + `/implement-packet` (command): drives
  `IMPLEMENTING_A_PACKET.md` §0-4, owns the shared-codec-wrapper Step-0 decision, ends by
  handing each cell to `packet-verifier`. Encodes: validator-mandatory handlers, route all 9
  templates, config-resolved mode/message bytes (DOM-25), no wire change to existing versions.
- **FR-1.2 version bring-up** — `/bringup-version <region> <major> <minor>` command that
  scripts the `STARTING` pipeline as ordered, resumable stages, encoding the serial
  constraints (shared run.go/families.yaml/global IDA) and export-hygiene (§10). It dispatches
  `packet-verifier` fan-out for the campaign. Prefer a **command that narrates + delegates**
  (like `/execute-task`) over a monolith agent, so a human stays in the loop between stages.
- **FR-1.3 `family-auditor`** (read-only agent): enumerates a family from
  `dispatchers/<family>.yaml` + registry + matrix, reports per-version verified/orphan/
  unverified, writes `docs/tasks/.../family-audit-<name>.md`. Never mutates codecs — this is
  the bug-triage driver that `dispatcher-family-implementer` (do-mode) lacks.
- **FR-1.4** CLAUDE.md + superpowers-integration gain a "Packet work" section: a 3-row table
  (task type → entry point → canonical playbook) pointing at `docs/packets/PROCESS.md`.

## P4 — Churn guards (FR-3)

### FR-3.1 Boundary-fixture pair + gate lint
**Decision: two mechanisms.**
1. *Gate-idiom lint* (cheap, high value): a `dispatcher-lint`-style or `go vet`-adjacent check
   flagging raw `t.MajorVersion() > N` / `< N` on wire-encoding paths in `libs/atlas-packet`,
   recommending `MajorAtLeast(N)` / a named boundary helper. Allowlist the legitimate
   non-boundary uses. This directly attacks the `>83` footgun class.
2. *Boundary-fixture assertion*: a new `packet-audit gate-check` reads a **machine-readable
   gate registry** (`docs/packets/gates.yaml` — packet, field, boundary version, the two
   straddling version keys) and asserts a verified byte-fixture exists at both straddling
   versions. Seed `gates.yaml` from the existing gate inventory. CI-gated. Start with the
   ~60 divergence gates already enumerated in task-113's `code-gate-audit.md` as the seed.
   Pragmatic scope: assert the *pair exists and is verified*, not that the bytes differ.

### FR-3.2 Export non-destructive default
**Decision: guard in `runExport`.** Default: if `--output` exists and differs, write
`<output>.new` + a summary of added/removed/changed function keys, and **exit non-zero without
overwriting**. `--force` restores today's overwrite. A `--splice <fname>` mode harvests a single
function and merges one entry (the §10 surgical path) — codifying the manual splice the
campaigns did by hand. Regression test: existing export + no --force → refuses, file unchanged.

### FR-3.3 Coverage manifest + completeness critic
**Decision: lightweight manifest + critic agent, not a tool subcommand.**
- Manifest: `docs/tasks/<task>/coverage-manifest.yaml` — `{ops:[], versions:[], fields:[],
  out_of_scope:[]}` the task declares up front.
- Critic: a `packet-completeness-critic` agent (or a checklist appended to the reviewers) that
  diffs the manifest against the git delta (touched `libs/atlas-packet` structs/gates) and the
  matrix delta, flagging **changed-but-unclaimed** (a codec moved but isn't in the manifest —
  the scope hole) and **claimed-but-unverified** (manifest op with no verified cell). Runs in
  the pre-PR review step. This is process tooling; the "guard" is the critic firing in review.

## P5 — Housekeeping

- **FR-5.2** Decide `verify-serverbound`: it produces `registry/verify_serverbound_*.md` and is
  referenced nowhere. Preference: **document + wire it** into the serverbound section of
  `IMPLEMENTING`/`STARTING` (it is genuinely the serverbound send-site worklist) rather than
  delete. Add a short "re-audit an existing column after export drift" playbook covering the
  diagnostic toolkit (`validate`/`decompose`/`triage`/`diff-shape`) for maintenance use.

## Cross-cutting

- **Order:** P1 first (docs become the lint's reference baseline), then P2 (visibility is
  self-contained + low-risk), then P3 (agents consume the now-true docs), then P4 (guards), P5.
- **CI wiring:** new checks join `.github/workflows/packet-matrix.yml` as additional steps
  (doc-freshness, gate-check, family-cap). Each ships with a seeded-violation test proving it
  fails, so we don't ship a green-only guard.
- **Risk register:** the only code-behavior change touching grading is FR-4.1 (sub-struct
  applicability). Everything else is render-only, new-subcommand, doc, agent-def, or CI. FR-4.1
  gets the most test scrutiny + an explicit before/after count diff in the PR.
