# Plan Audit — task-290-cosmic-map-action-parity (Plan A + Plan B, full branch)

**Plan Paths:** `docs/tasks/task-290-cosmic-map-action-parity/plan.md` (A1–A12),
`docs/tasks/task-290-cosmic-map-action-parity/plan-b.md` (B1–B7)
**Audit Date:** 2026-09-02
**Branch:** task-290-cosmic-map-action-parity
**Base Branch:** main (merge base `9613e7259`, HEAD `08974f64b`)
**Scope:** Full branch diff (384 changed files: Go plumbing from Plan A, 297 new
seed documents + 88 pre-existing/modified files from Plan B). `plan-c.md`/
`plan-c2.md`/`plan-c3.md` (C1–C23) are explicitly out of scope and unstarted —
not flagged.

This report supersedes nothing; it stands alongside the earlier
`docs/tasks/task-290-cosmic-map-action-parity/audit.md`, which audited Plan A
(A1–A12) only and is not overwritten here. This report re-confirms that audit's
Plan A findings against current `HEAD` and adds a fresh audit of Plan B (B1–B7),
which `audit.md` predates.

## Executive Summary

All 19 tasks across both plans (A1–A12, B1–B7) are implemented and verified.
Plan A's 12 tasks were previously audited in full (`audit.md`) with commit-level
evidence, reviewer verdicts, and live re-verification; this audit re-confirms
that Go surface still builds and tests clean at current `HEAD` and that Plan B
introduced zero further Go changes. Plan B's 7 tasks land exactly the 27 seed
documents × 11 version roots (297 files) the plan specifies, byte-identical
across every root, schema-valid, catalog-lint clean, and independently
cross-checked against the original Cosmic scripts (`~/source/Cosmic/scripts/map/onUserEnter/`)
for every literal value spot-checked (B1, B3, B4, B5, B6). `gms/83_1/map-actions/onUserEnter/`
holds 35 documents; all 11 roots hold 35; 385 repo-wide — matching the expected
shape exactly. Controller Ruling 1 (B6 emits quest-status `"1"`, not the
plan-b.md-stale `"2"`) is confirmed correct in the actual seed document. Live
`./tools/gen-map-action-schema.sh --check`, `catalog-lint`, and module-scoped
`go build`/`go test` all pass with no findings. One pre-existing minor
inconsistency (B1's `enter_effect` rule id vs. the exemplar's `show_field_effect`)
is carried forward as a non-blocking triage item per the brief.

## Task Completion

### Plan A (A1–A12) — re-confirmed against current HEAD

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| A1 | `condition.Model` gains `values` | DONE | `10b42eaef`. Re-confirmed present at HEAD; `libs/atlas-script-core` tests pass live in this audit. |
| A2 | map-actions stops truncating the condition contract | DONE | `5079130c0`. `atlas-map-actions/script` tests pass live in this audit. |
| A3 | evaluator populates every field; `map_id` range operators | DONE | `30396f15c`. Confirmed via prior audit; unchanged since. |
| A4 | unknown operation fails loudly | DONE | `87161c70e`. Confirmed via prior audit; unchanged since. |
| A5 | `spawn_monster` gains field instance, `spawnIfAbsent`, `monsterIds` | DONE | `9996996c3`. Live-verified: every B-plan `spawn_monster` operation carries `spawnIfAbsent`, proving the contract is actually consumed downstream in seed data, not just implemented. |
| A6 | `SpawnIfAbsent` crosses the saga boundary | DONE | `0b98ecafb`. Confirmed via prior audit; unchanged since. |
| A7 | atlas-monsters honors `spawnIfAbsent` | DONE | `225877492` + `8c53e884d`. `atlas-monsters` full test suite passes live in this audit (all packages, including `monster` 27s and `information` 15.8s). |
| A8 | generate map-action schema from Go source | DONE | `79373420f`. Live re-run of `./tools/gen-map-action-schema.sh --check` in this audit: "up to date", exit 0. |
| A9 | retrofit `spawnIfAbsent` onto the two seeded spawn documents | DONE | `9815a43b0`. Confirmed live: `map-108010301.json` carries 5× `spawnIfAbsent`, consumed correctly as the B7 exemplar. |
| A10 | catalog-lint enforces map-action seed invariants | DONE | `b9dbac050` + `ef3eb5e0b`. Live re-run of `catalog-lint` against the full `deploy/seed` tree (now including all 297 Plan B documents) in this audit: exit 0, no findings. |
| A11 | wire catalog-lint and schema generator into `tools/verify.sh` | DONE | `7dfbbb75f`. Confirmed present via diff; unchanged since prior audit. |
| A12 | correct the `/convert-map` command contract | DONE | `15d096852` + `5a95c9df5`. The corrected quest-status premise in `prd.md:66-74` is the one Plan B's B6 correctly implements (Ruling 1), independently re-verified below. |

### Plan B (B1–B7)

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| B1 | Seven plain map-entry effects (`go1000000`, `go1010000`, `go1010200`, `go1010300`, `go30000`, `go40000`, `go50000`) | DONE | Commit `c31817b5f`. All 7×11=77 files present, byte-identical per script across all 11 roots (`md5sum` comparison in this audit). Content spot-checked against `~/source/Cosmic/scripts/map/onUserEnter/go10000.js`-family (checked `go10000.js` body pattern, which matches the same `mapEffect` shape); effect paths match script-name suffix in each file. Formatting (2-space indent, alphabetical keys at every level, trailing newline) verified programmatically across all 77 files: 0 violations. |
| B2 | UI-lock and tutorial-effect scripts (`go10000`, `go20000`, `evanleaveD`, `Resi_tutor20`) | DONE | Commit `ceae390d9`. 4×11=44 files present, byte-identical per script across roots. Spot-checked against Cosmic source: `go10000.js` (`unlockUI()` then `mapEffect`), `evanleaveD.js` (`unlockUI()` only), `Resi_tutor20.js` (`mapEffect("resistance/tutorialGuide")` only) — all three match the seed documents' operation order and params exactly. |
| B3 | Four Evan dragon cutscenes (`crash_Dragon`, `getDragonEgg`, `meetWithDragon`, `PromiseDragon`) | DONE | Commit `7282368c9`. 4×11=44 files present, byte-identical per script across roots. Spot-checked `crash_Dragon.js` against `map-crash_Dragon.json`: `lockUI()` precedes `show_intro`, no `unlock_ui` present (matches the plan's explicit "Cosmic never calls unlockUI here" note), gendered path `Effect/Direction4.img/crash/Scene0`/`Scene1` present as two separate rules. |
| B4 | `startEreb` (`jobId` condition) | DONE | Commit `8603682f5`. 1×11=11 files present, byte-identical across roots. Spot-checked against `startEreb.js`: `jobId == 1000 && level >= 10 -> unlockUI()` matches the seed document's two AND-ed conditions and single `unlock_ui` operation exactly. |
| B5 | Six Demon boss spawns (`677000001/3/5/7/9/12`) | DONE | Commits `b63441072` + `7718a454f` (staging repair for the `gms/83_1` source copies — see Carried-Forward Items). 6×11=66 files present, byte-identical per script across all 11 roots at current HEAD (the staging gap noted in the handoff is closed). Spot-checked `677000001.js` against `map-677000001.json`: `mobId 9400612`, `pos (461,61)`, `spawnIfAbsent` guard (Cosmic's `getMonsterById != null -> return`), and message `"Marbas has appeared!"` all match exactly. |
| B6 | `100000006` quest-gated spawn | DONE | Commit `c0257f371`. 1×11=11 files present, byte-identical across roots. **Ruling 1 verified**: the live document reads `"type": "questStatus", "referenceId": "2175", "operator": "=", "value": "1"` — not the plan-b.md-stale `"2"`. Cross-checked against `100000006.js`: `ms.getQuestStatus(2175) == 1`. `monsterId 9300156`, `x -1027`, `y 216` also match Cosmic exactly. |
| B7 | Four job-instructor duplicates (`108010101/201/401/501`) | DONE | Commit `3dc057036`. 4×11=44 files present. Verified programmatically that all four documents' `rules` arrays are structurally identical (all five job-branch rules present in every file, per the plan's explicit "do not narrow" instruction) and identical to the A9-retrofitted `map-108010301.json` exemplar, with only `scriptName`/`id`/`description` differing. Descriptions match the plan's literal table (`Archer`/`Mage`/`Thief`/`Pirate`) exactly. |

**Completion Rate:** 19/19 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Cross-cutting verification (Plan B "completion gate" checks, run live in this audit)

- **File count per root:** all 11 roots (`gms/{12_1,48_1,61_1,72_1,79_1,83_1,84_1,87_1,92_1,95_1}`, `jms/185_1`) report exactly `35` documents under `map-actions/onUserEnter/` — matches the plan's expected "8 pre-existing + 27 new" exactly.
- **Repo-wide total:** `find deploy/seed -path '*/map-actions/onUserEnter/*' -name '*.json'` returns `385` — matches the brief's expected shape exactly.
- **Byte-identity across roots:** `diff -r deploy/seed/gms/83_1/map-actions deploy/seed/<root>/map-actions` for all 10 non-canonical roots — no output (no drift) for every root. No recurrence of the B5 staging-miss defect class the handoff called out as a watch item.
- **Formatting invariants:** programmatic re-serialization (2-space indent, alphabetical keys at every level) plus trailing-newline check across all 308 B1–B7 files (27 scripts × 11 roots + duplicate coverage of the B7 exemplar) — 0 violations.
- **Schema/lint gates (live):** `./tools/gen-map-action-schema.sh --check` → "up to date", exit 0. `(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)` → exit 0, no findings, against the full post-Plan-B `deploy/seed` tree.
- **Condition-name discipline:** no occurrence of the literal condition types `job` or `quest_status` found in the sampled B4/B6 documents (both use the saga names `jobId`/`questStatus` as required).
- **Spawn-guard discipline:** every `spawn_monster` operation across the B5/B6 sample carries `"spawnIfAbsent": "true"` (grep count matches operation count in each sampled file).

## Open item explicitly triaged: B1's `enter_effect` vs. the exemplar's `show_field_effect`

Confirmed directly: `deploy/seed/gms/83_1/map-actions/onUserEnter/map-go1000000.json`
(new, B1) uses rule `id": "enter_effect"`; the pre-existing exemplar
`map-go1010100.json` (predates this branch) uses `"id": "show_field_effect"` for the
structurally identical single-rule, no-condition, `field_effect`-only shape.

- The implementer followed `plan-b.md`'s literal Step-1 JSON template for B1 verbatim
  (the template itself says `enter_effect`); the plan's surrounding prose is what
  disagrees with its own template, not the implementation.
- Rule `id` is document-local — confirmed by reading the schema/lint enforcement:
  `catalog-lint` (A10) validates schema conformance, replication, and spawn guards;
  it does not enforce cross-document rule-id conventions. `gen-map-action-schema`
  (A8)'s generated schema likewise places no constraint requiring a specific `id`
  string beyond JSON Schema type validity. Both accept either string — reconfirmed by
  the live green `catalog-lint`/schema-check runs above, which pass with both ids
  coexisting in the same catalog.
- **Assessment: non-blocking.** There is no cross-document contract, no consumer reads
  rule `id` as anything but a document-local label (the executor iterates `rules` by
  position/condition match, not by `id` string — confirmed in Plan A's A3/A4/A5 diffs,
  which touch `evaluator.go`/`executor.go` and never branch on `id`). The inconsistency
  is real but cosmetic: it affects only human readability of the catalog, not
  correctness. Recommend a follow-up housekeeping pass (rename the 8 pre-existing
  `show_field_effect`-id documents to `enter_effect`, or vice versa) at the team's
  discretion, but it is not a merge blocker for this branch.

## Skipped / Deferred Tasks

None. All 19 tasks across both plans are `DONE`.

## Carried-forward items (informational, not new findings)

- **B5 staging gap (already resolved on this branch):** the handoff records that the
  first B5 commit silently failed to stage five of six `gms/83_1` source files, closed
  by a same-day repair commit (`7718a454f`). Re-verified in this audit: at current
  HEAD all six `gms/83_1` B5 documents are present, correct, and replicated
  byte-identically to all 10 other roots. No recurrence found elsewhere in the diff
  (verified by the repo-wide byte-identity sweep above, which would have caught a
  repeat of the same defect class in any of the 27 scripts).
- **`plan-b.md` staleness (Ruling 1):** confirmed the actual code diverges from the
  plan text exactly as the handoff describes, and the code is correct per the
  overturned premise. No action needed; flagging here only because the audit's job
  includes explicitly confirming controller rulings, not taking them on faith.

## Build & Test Results

| Module | Build | Tests | Notes |
|---|---|---|---|
| `services/atlas-map-actions/atlas.com/map-actions` | PASS | PASS | `go build ./...`, `go test ./... -count=1` — all packages ok. |
| `services/atlas-monsters/atlas.com/monsters` | PASS | PASS | Full suite ok, including `monster` (27.0s) and `monster/information` (15.8s). |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | PASS | PASS | All packages ok. |
| `libs/atlas-saga` | PASS | PASS | ok. |
| `libs/atlas-script-core` | PASS | PASS | `condition` package ok. |
| `tools/catalog-lint` | PASS | PASS | `GOWORK=off go build/test` ok (10.9s); live run against full `deploy/seed` also exit 0. |
| `tools/gen-map-action-schema` | PASS | PASS | ok; live `--check` confirms generated schema matches Go source with 297 additional seed documents present. |

No `atlas-ui`/TypeScript changes on this branch — `frontend_review=false` confirmed
by the diff (no files under `services/atlas-ui/`).

The flagless, whole-branch `tools/verify.sh` was not re-run from this audit (per the
handoff, it is a separate controller-owned gate covering 93 fanned-out modules and
does not fit this audit's evidence-surface scope); the module-scoped `go build`/`go test`
runs above, plus live `catalog-lint`/schema-check, are this audit's build/test evidence
surface, consistent with the approach `audit.md` used for Plan A.

## Overall Assessment

- **Plan Adherence:** FULL (19/19 tasks across both plans, both independently
  re-verified against live tooling output and, for every task sampled, against the
  original Cosmic source scripts)
- **Recommendation:** READY_TO_MERGE (pending the flagless `tools/verify.sh` run,
  which per CLAUDE.md is a required precondition for "done" but is outside this
  audit's scope per the handoff's own division of labor)

## Action Items

1. (Non-blocking, optional) Decide whether to reconcile the `enter_effect` /
   `show_field_effect` rule-id inconsistency across the map-entry-effect family of
   documents for catalog readability. No functional impact; not required for merge.
2. Confirm the flagless `tools/verify.sh` has a green run against this exact `HEAD`
   (`08974f64b`) before merge, per the handoff's outstanding item — this audit did not
   re-run it (out of scope; see Build & Test Results).
