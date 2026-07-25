# Task 169 — Implementation Plan

Phased. Each task lists its files, the change, and its done-check. Execute phases in order;
within a phase, tasks marked `[parallel-ok]` touch disjoint files. Every tool-behavior change
is TDD (write the failing test first). Gate after each phase: `go test ./tools/packet-audit/...`,
`matrix --check` exit 0, existing verified counts unchanged (except the FR-4.1 reclassification,
tracked explicitly).

Baseline before starting: capture `go run ./tools/packet-audit matrix` totals for all 9
versions into `docs/tasks/task-169-packet-process-consistency/baseline-counts.md` (the frozen
reference for AC-3).

---

## Phase 1 — De-drift & single-source docs (FR-2.1, FR-2.2)

- **T1.1** Create `docs/packets/PROCESS.md` — top-of-tree index: current version set (9, named),
  baseline status (empty/graduated), CI gate list, and a task-type→entry-point→playbook table.
  This is the machine-lintable source of truth for the freshness check. Done: file exists,
  links resolve.
- **T1.2** Fix stale facts in `docs/packets/IMPLEMENTING_A_PACKET.md` +
  `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md`: 5→9 versions, name v48/v61/v72/v79,
  correct the `-versions` default string. Done: no "five"/"5 versions"/5-key literal remains
  where it means the version set. [parallel-ok]
- **T1.3** Fix `docs/packets/DISPATCHER_FAMILY.md` (baseline empty, not party/guild/buddy) and
  `docs/packets/audits/VERIFYING_A_PACKET.md` (§8 hard-gate, remove continue-on-error language;
  add `🧩` to the legend; correct step-range refs). [parallel-ok]
- **T1.4** Fix `tools/packet-audit/README.md` `export` invocation to the real `runExport` flags
  (`--version/--output/--ida-url/--ida-port/--descent-depth/--prior-export/--pending/
  --generated-at`); add the missing subcommands to any command table. [parallel-ok]
- **T1.5** De-duplicate: `.claude/commands/verify-packet.md` + `.claude/agents/packet-verifier.md`
  replace the restated "5 rules"/"steps 1–8" with a pointer to `VERIFYING_A_PACKET.md` §0–10.
  Same for `dispatcher-family-implementer` vs `DISPATCHER_FAMILY.md` (keep the def, cite the
  doc for the procedure). Done: no divergent step-count/rule copies remain.
- **Gate P1:** all playbooks internally consistent; links resolve; no tool/doc contradiction
  from RC-B remains (manual checklist in the phase-close doc).

## Phase 2 — Matrix expressiveness & visibility (FR-4, FR-5.1)

- **T2.1** [TDD] Sub-struct n-a (FR-4.1). `internal/matrix/load.go`+`build.go`: thread an
  `Unimplemented[version]set(packetID)` (from `_unimplemented.json`, already loaded) into
  `Build`; `gradeSubStructCell` grades `StateNA` when the (packetID,version) is dispositioned.
  Tests: dispositioned→NA, undispositioned→unchanged. Done: `build_test.go` green; regenerate
  matrix; record the ⬜/❌→n-a delta vs baseline-counts.md (expected, documented).
- **T2.2** [TDD] Partial disambiguation (FR-4.2). `internal/matrix/render.go`: map the three
  partial `Note` values to distinct glyphs + legend rows; `status.json` unchanged. Test:
  `render_test.go` asserts the three notes render distinctly and totals are unchanged.
- **T2.3** Support summary (FR-4.3). `cmd/`: `matrix --summary` (or `support-summary`
  subcommand) → `docs/packets/audits/support/<version>.md` per version (totals, verified%,
  gap table n-a/unverified/conflict). Deterministic. Done: 9 files generated, match status.json.
- **T2.4** `status <version>` query (FR-4.4). `cmd/status.go`: read status.json, print summary +
  open gaps + stale evidence to stdout. Test: golden-ish assertion on a fixture status.json.
- **T2.5** [TDD] Family-cap guard (FR-5.1). Extend `dispatcher-lint` (or add `--check-families`):
  every `dispatchers/*.yaml` family must be discrete-implemented (INV-clean) OR baseline-listed;
  a new unimplemented+unlisted family fails. Test: seeded phantom family fails; current tree
  passes.
- **T2.6 (stretch)** Per-arm rollup (FR-4.5): `status <version> --family <name>` prints
  k/n arms verified. Non-blocking.
- **Gate P2:** `matrix --check` exit 0; existing verified counts unchanged except the T2.1
  documented reclassification; new commands deterministic + committed outputs.

## Phase 3 — Executable entry points (FR-1)

- **T3.1** `.claude/agents/packet-implementer.md` (FR-1.1) — wraps `IMPLEMENTING_A_PACKET.md`
  §0–4; Step-0 shared-codec decision; route all 9 templates; DOM-25 config-resolved bytes;
  hands cells to `packet-verifier`; no existing-version wire change. [parallel-ok]
- **T3.2** `.claude/commands/implement-packet.md` (FR-1.1) — dispatches T3.1. [parallel-ok]
- **T3.3** `.claude/commands/bringup-version.md` (FR-1.2) — narrate-and-delegate orchestrator
  for the `STARTING` pipeline stages (registry→discover-ops→template→export→static-audit→
  matrix→verifier fan-out); encodes serial + export-hygiene constraints; resumable. [parallel-ok]
- **T3.4** `.claude/agents/family-auditor.md` (FR-1.3) — read-only family coverage audit →
  findings doc; never mutates codecs. [parallel-ok]
- **T3.5** CLAUDE.md + `docs/superpowers-integration.md` (FR-1.4): add "Packet work" section —
  task-type→entry-point→playbook table pointing at `docs/packets/PROCESS.md`.
- **Gate P3:** each task type executable from a documented entry; agent defs parse (dispatch a
  trivial dry-run of family-auditor on one small family to confirm it produces a findings doc).

## Phase 4 — Churn guards (FR-3)

- **T4.1** [TDD] Gate-idiom lint (FR-3.1a). New `cmd/gatelint.go` (or `dispatcher-lint` step):
  flag raw `MajorVersion() > N`/`< N` on wire-encode paths in `libs/atlas-packet` outside an
  allowlist. Test: seeded `>83` fails; the `MajorAtLeast` form passes.
- **T4.2** Gate registry + boundary-fixture check (FR-3.1b). Author `docs/packets/gates.yaml`
  (seed from task-113 `code-gate-audit.md`: packet, field, boundary, straddling version keys).
  `cmd/gate-check.go`: assert a verified fixture exists at both straddling versions. Test:
  seeded gate with a missing-side fixture fails; a complete pair passes. [depends T4.1 optional]
- **T4.3** [TDD] Export non-destructive default (FR-3.2). `runExport`: refuse overwrite of an
  existing differing `--output` without `--force`; write `<output>.new` + change summary; add
  `--splice <fname>` single-entry merge. Test: existing export + no --force → refuses, file
  byte-unchanged; --force overwrites; --splice merges one entry.
- **T4.4** Coverage manifest + critic (FR-3.3). Define `coverage-manifest.yaml` schema (in
  PROCESS.md); add `packet-completeness-critic` agent (or a checklist section to the
  reviewers) that diffs manifest vs git+matrix delta → changed-but-unclaimed /
  claimed-but-unverified. Wire into the pre-PR review step.
- **T4.5** Doc-freshness lint (FR-2.3). `cmd/doclint.go` (or a `go test` in the tool):
  parse `PROCESS.md`+playbooks for the asserted version count / baseline-family list, compare
  to `matrix.VersionKeys` + the baseline YAMLs; fail on divergence. Test: mutate a doc literal
  → fails; current tree passes.
- **T4.6** CI wiring: add doc-freshness, gate-check, family-cap steps to
  `.github/workflows/packet-matrix.yml`. Each must be green on the tree and shown to fail on a
  seeded violation (in the task's test evidence, not committed to the tree).
- **Gate P4:** every new check green locally; each has a fires-on-violation test.

## Phase 5 — Housekeeping, gate, review, PR (FR-5.2)

- **T5.1** `verify-serverbound`: wire + document it into the serverbound sections of
  `IMPLEMENTING`/`STARTING`; add a "re-audit an existing column after export drift" maintenance
  playbook covering `validate`/`decompose`/`triage`/`diff-shape`.
- **T5.2** Full gate: `go build`/`go vet`/`go test -race` on `tools/packet-audit` +
  `libs/atlas-packet`; `matrix --check` / `operations --check` / `fname-doc --check` /
  `dispatcher-lint` / new checks all exit 0; redis-key-guard clean; docker bake for any Go
  module whose go.mod changed (packet-audit is a module → bake its consumers only if go.mod
  touched — likely none).
- **T5.3** Code review: `plan-adherence-reviewer` + `backend-guidelines-reviewer` +
  the new completeness-critic on this branch. Resolve blocking findings.
- **T5.4** Reconciliation doc + PR. PR body: the RC→FR→AC trace, the FR-4.1 count-delta table,
  and the "prove both directions" evidence for each new guard.

## Verification matrix (AC → where satisfied)

- AC-1 → T3.1–3.5. AC-2 → T2.5,T4.1,T4.2,T4.5,T4.6 (+ fires-on-violation tests).
- AC-3 → baseline-counts.md diff at each gate; only T2.1 moves cells. AC-4 → T2.3,T2.4.
- AC-5 → T4.3. AC-6 → T1.*,T4.5. AC-7 → T5.2.
