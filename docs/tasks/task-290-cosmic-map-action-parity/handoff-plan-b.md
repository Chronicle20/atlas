# Handoff — Plan B complete, pre-PR gates remain

## State

Plan A (`plan.md`, A1–A12) and Plan B (`plan-b.md`, B1–B7) are both fully implemented on
branch `task-290-cosmic-map-action-parity`. PR #1629 is open against `main`.

Plan B landed as three batched dispatches (see Ruling 2 below), 8 commits,
`e4b1acf63..3dc057036`:

| commit | task | files |
|---|---|---|
| `c31817b5f` | B1 seven plain map-entry effects | 77 |
| `ceae390d9` | B2 UI-lock and tutorial-effect scripts | 44 |
| `7282368c9` | B3 four Evan dragon cutscenes | 44 |
| `8603682f5` | B4 `startEreb` (`jobId` condition, FR-1.1) | 11 |
| `b634410` + `7718a454f` | B5 six Demon boss spawns (+ staging repair) | 66 |
| `c0257f371` | B6 `100000006` quest-gated spawn | 11 |
| `3dc0570` | B7 four job-instructor duplicates | 44 |

27 scripts × 11 version roots = 297 new seed documents. `gms/83_1/map-actions/onUserEnter/`
now holds 35 documents (8 pre-existing + 27); 385 repo-wide.

Every batch was reviewed (`task-reviewer`, sonnet) and gated
(`tools/verify.sh --quick`): batch 1 APPROVED, batch 2 APPROVED_WITH_FINDINGS (0 blocking),
batch 3 APPROVED. Three gates, all exit 0. **Last-gated commit: `3dc057036`.** Nothing is
pending a gate.

## What remains — all of it controller-shaped

1. **Final whole-branch review.** `superpowers:requesting-code-review` over the full merge
   base `9613e7259..HEAD`. `plan-adherence-reviewer` should be pointed at BOTH `plan.md` and
   `plan-b.md` — 19 tasks total, still fine unsharded. `frontend_review=false`
   (`ts_changed=false`). Plan B added no Go, so the backend audit's earlier verdict stands
   for the Go surface.
2. **The flagless `tools/verify.sh`** against merge base `9613e7259`. It last passed at
   `7275e3a61`, which predates all of Plan B. Per CLAUDE.md only that run counts as verified,
   so it must be re-run. Budget for it: `fanout_reason=shared-lib:go.work` fans out to 93
   modules and it will NOT finish in a single 600s window — run it backgrounded with a long
   timeout and do not misread a timeout as a failure.
3. **`superpowers:finishing-a-development-branch`.** PR #1629 already exists and is in sync,
   so this is a re-push plus letting CI re-run; the user has already ruled that Plan B stacks
   onto this PR rather than opening a new one.

## Rulings made during Plan B — the user should read these

**Ruling 1 — Plan B's quest-status shift is superseded; B6 emits `"1"`, not `"2"`.**
`plan-b.md`'s Global Constraints and Task B6 both require the PRD FR-1.4 `n + 1` shift, with
B6 concretely specifying `"value": "2"`. That premise is the same one task A12 overturned on
this branch; `prd.md:66-74` carries the correction. Re-verified against live source
(`query-aggregator/quest/model.go:11-14` — `StateStarted = 1`; `validation/model.go:402-413`
— no offset applied) and then against the ORIGINAL Cosmic script
(`scripts/map/onUserEnter/100000006.js`: `ms.getQuestStatus(2175) == 1`). The value is `"1"`.
`plan-b.md` was deliberately NOT edited — it stays a historical phase artifact.
*Cost if wrong:* one condition value in one document ×11 roots; the spawn fires in the wrong
quest state. One-line fix.

**Ruling 2 — B1–B7 batched into three dispatches, not seven.** All seven tasks are the same
transcribe → `cp` → catalog-lint → commit shape, and the mechanical checks already prove
replication, schema validity and spawn guards, leaving only literal values for review.
*Cost if wrong:* coarser review granularity — a defect in one document shares a verdict with
its batch-mates. Recoverable by splitting a failing batch into per-task fix rounds.

## Carried-forward items the next controller must not lose

- **Deferred minor (B1):** the seven B1 documents use rule `id` `enter_effect`, while the
  in-tree exemplar `map-go1010100.json` uses `show_field_effect`. The implementer followed the
  plan's literal template; the plan's prose disagrees with its own template. Rule ids are
  document-local with no cross-document contract and both lint and schema accept either.
  The final review should triage whether the catalog wants internal consistency before merge.
- **Unexplained (B5):** the first B5 commit silently failed to stage five of six `gms/83_1`
  source files. `git check-ignore` confirms they were never ignored; the implementer's
  root-cause note ("possibly a stale index state") in `batch-3-report.md` is unresolved.
  `catalog-lint` caught it — the A10 investment paying off. Worth watching for a recurrence.
- **Process note:** a task-reviewer reading the live working tree observed another agent's
  half-finished `cp` loop. It correctly reported it as out-of-range rather than attributing it
  to its own diff, and it turned out to be a real defect. Keep reviewers on the packaged diff
  for in-range judgments, but do not suppress out-of-range observations.
- **`context.md` §5 gap:** it says the Cosmic checkout is at "the path recorded in the session
  that produced these plans" without naming it. It is at `~/source/Cosmic`. All of Plan B's
  B5/B6/B7 literals were re-derived from `~/source/Cosmic/scripts/map/onUserEnter/` and
  confirmed exact. Recording the path there would save the next session a search.
- **Still open from Plan A:** `plan-c.md`/`plan-c2.md`/`plan-c3.md` (Tasks C1–C23) are
  unstarted. Their stated precondition — `plan.md` green and `plan-b.md` landed — is now met.

## Resumption

`.superpowers/sdd/plan-b/progress.md` is the full ledger (git-ignored; this file is its
durable summary). Briefs, reports and review artifacts live beside it.
