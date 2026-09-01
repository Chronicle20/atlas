# Plan Audit — task-286-build-verify-concurrency (Tasks 1-5)

**Plan Path:** docs/tasks/task-286-build-verify-concurrency/plan.md
**Audit Date:** 2026-08-29
**Branch:** task-286-build-verify-concurrency
**Base Branch:** main
**Scope:** Tasks 1-5 only (Tasks 6-9 and the 8b interstitial are audited separately)

## Executive Summary

Tasks 1-4 are fully implemented and match the plan's design corrections
(C1/C2/C3), with honest, non-fabricated measurement notes in `measurements.md`
(notably Task 1's transparent explanation that the byte-count reduction is not
observable from inside a worktree checkout, and Task 2's explicit "not applied
at implementation time" note for the host-tuning after-figure). Task 5's code
and tests are complete and correct — `buildx-bootstrap.sh`, its test suite,
`build-services.sh --load`, and `verify.sh`'s `buildx builder` gate are all
present and passing — but Task 5 Step 6's required `measurements.md` entry
(`## Layer 4 — builder`, `docker system df` before/after a full
`all-go-services` bake) was never written; no commit touches
`measurements.md` for Task 5. This is a real, unrecorded gap against the
plan's own acceptance criterion 4 and the "Final gate" requirement that
`measurements.md` carry all five criteria. All shell-guard and suite-level
tests for these five tasks pass.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Starve the docker build context (Layer 1) | DONE | `.dockerignore` rewritten as allowlist matching plan's exact content (`.dockerignore:1-23`, commit `19fc7580c`); `measurements.md` "## Layer 1" carries before/after `transferring context` figures with an honest caveat that both read `11.30MB` from inside a worktree checkout, plus the required `all-services` cacheonly bake exiting 0 (1829s, commit `583be745e`); `docs/verification.md` "## The docker layer" states the allowlist behavior. |
| 2 | Disk-backed scratch root and its sweeper (Layer 0) | DONE | `tools/scratch-sweep.sh` (119 lines) implements the full contract incl. dangerous-root guards, `--dry-run`, `--now`, `--age-days`, relative-root refusal (an added hardening beyond the plan, from a later review round, commit `8410746d2`/`42a59c45d`); `tools/scratch-sweep_test.sh` — 28/28 assertions pass (verified live); `docs/verification.md:100-186` "## Host tuning (WSL2)" carries `.wslconfig`, `/etc/fstab`, `TMPDIR`, and the systemd timer sections, all with placeholder paths (no literal home path); `measurements.md` "## Layer 0 — scratch" records a real before-figure and explicitly declines to fabricate an after-figure since host tuning was not applied at implementation time — consistent with the plan's own instruction to record that fact explicitly instead. |
| 3 | One bake solve instead of one per target (Layer 2, docker half) | DONE | `tools/verify.sh:566-569` (as wired by Task 7's slot layer) is a single `docker buildx bake ... "${TARGETS[@]}"` invocation, no per-target loop (`grep -c 'for t in .\{0,4\}TARGETS' tools/verify.sh` → 0, confirmed by the passing "no per-target bake loop remains" test); `tools/verify_test.sh` bake-selection assertions ("two changed go.mods select two bake targets", "two bake targets produce exactly one bake gate", "the gate names the target count", "no probes, no bake gate") all pass; `docs/verification.md:329-345` updated accordingly. |
| 4 | Parallel Go layer with ordered reporting (Layer 2, Go half) | DONE | `tools/verify.sh:40-42` (`GO_JOBS`/`ATLAS_VERIFY_GO_JOBS` validation), `tools/verify.sh:395-441` (`go_layer`, `launch_go_layers`, `replay_go_layer`) match the plan's two-phase launch/replay shape verbatim, including the `if ... else echo $? ...` rc-capture idiom the plan calls load-bearing; driver at `tools/verify.sh:448-476` uses the `if [ "$FACTS" -eq 0 ]` guard form (not the bare `&&`) as required. `tools/verify_test.sh` job-count-invariance, invalid-`GO_JOBS` rejection, and log-dir-cleanup assertions pass. `measurements.md` "## Layer 2 — parallelism (Go half)" records a real (not fabricated) serial-vs-pooled wall-time comparison with a documented methodology and honestly notes it substitutes for a full-`verify.sh` timing that was out of session scope. |
| 5 | A governed BuildKit builder (Layer 4) | PARTIAL | Code and tests are all correct and complete: `deploy/buildkit/buildkitd.toml` matches the plan verbatim (`max-parallelism = 8`, the two-tier gcpolicy); `tools/buildx-bootstrap.sh` implements the full `--check`/`--force`/create/idempotent contract; `tools/buildx-bootstrap_test.sh` — 15/15 assertions pass including the docker-gated idempotency cases (verified live); `tools/build-services.sh` passes `--load`; `tools/verify.sh:563,567-569` adds the `buildx builder` gate before the bake step, exactly as specified; `docs/verification.md:372-402` "### The builder" documents all required content. **Missing:** Step 6's required `measurements.md` entry — a `## Layer 4 — builder` heading with `docker system df` before/after a full `all-go-services` bake — does not exist anywhere in `measurements.md` (confirmed: `grep -n -i "layer 4" measurements.md` returns nothing, and `git log --oneline -- measurements.md` shows no commit for Task 5's SHA `19187a6b7`, whose diff touches only `deploy/`, `docs/verification.md`, `tools/build-services.sh`, `tools/buildx-bootstrap.sh`, `tools/buildx-bootstrap_test.sh`, and `tools/verify.sh` — not `measurements.md`). This is acceptance criterion 4 from Task 5's own list and one of the five criteria the plan's "Final gate" section requires `measurements.md` to carry. |

**Completion Rate:** 4/5 tasks fully DONE, 1/5 PARTIAL (4.5/5 informal weight ≈ 90%)
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 5 — missing measurement record only; code/tests complete)

## Skipped / Deferred Tasks

**Task 5 — missing `## Layer 4 — builder` measurement.** The plan's Task 5
Step 6 and its acceptance criterion both require recording `docker system df`
before and after a full `all-go-services` bake in `measurements.md`, as the
evidence that the build cache stays under the 60 GB ceiling the pinned
`buildkitd.toml` enforces. No commit on this branch writes that section. The
underlying claim (cache bounded) is plausible given the checked-in GC policy,
but it is unverified by the branch's own evidence trail — exactly the kind of
unrecorded gap the plan's "Final gate" section flags as required before Task
9 lands (`measurements.md must by then carry all five recorded
measurements`). Impact: low functional risk (the code path is otherwise
fully tested), but it is a genuine, unexplained divergence from the plan's
explicit acceptance criterion — not something marked deferred or waived
anywhere in the branch (contrast with Task 7's honestly-recorded "WAIVED BY
USER DECISION" entry for its own acceptance criterion 4, which this task has
no equivalent of).

## Build & Test Results

| Suite | Result | Notes |
|---|---|---|
| `tools/shell-guard.sh --require-shellcheck` | PASS | "85 script(s) OK (syntax + shellcheck -S error)." |
| `tools/scratch-sweep_test.sh` | PASS | 28/28 assertions, including all dangerous-root and relative-root guards. |
| `tools/buildx-bootstrap_test.sh` | PASS | 15/15 assertions including docker-gated create/check/idempotent cases (real `docker buildx` calls succeeded). |
| `tools/verify_test.sh` | PASS (flaky once) | One full run exited 0 with all assertions `ok`. An earlier run in this same session reported "1 failure(s)" with no visible `FAIL -` line in the tail of that run's output; a second full run immediately after was clean (0 failures, exit 0). Most likely explanation: this worktree has a flagless `tools/verify.sh` running concurrently per this audit's own brief, and `tools/verify_test.sh` exercises the machine-wide build-slot broker (`tools/lib/build-slot.sh`) that a concurrent real run also contends for — plausible transient interference, not a structural defect in Tasks 1-5's code. Not re-run repeatedly per this audit's read-only/single-shot instruction; flagged here rather than resolved. |
| `docker buildx ls` | PASS | Shows `atlas*` on the `docker-container` driver, running. |

No `go build`/`go test` verification was run: this branch touches no Go
module (per the plan's own "Module root" note), so Go build/test is not
applicable to Tasks 1-5.

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE
- **Recommendation:** NEEDS_FIXES (one missing measurement entry; all code and tests for Tasks 1-5 are otherwise complete and passing)

## Action Items

1. Add a `## Layer 4 — builder` section to `measurements.md` recording
   `docker system df` build-cache figures before and after a full
   `docker buildx bake --set '*.output=type=cacheonly' all-go-services` (or
   equivalent `all-services`) run against the governed `atlas` builder, per
   Task 5 Step 6 and its acceptance criterion 4 — or, if that measurement is
   deliberately being waived (as Task 7's criterion 4 was), record that
   decision explicitly in `measurements.md` rather than leaving it silently
   absent.
2. (Non-blocking, worth a note for whoever audits Tasks 6-9) Re-run
   `tools/verify_test.sh` once more in isolation (no concurrent `verify.sh`)
   to confirm the single observed "1 failure(s)" run was in fact contention
   noise and not a latent race in the build-slot test fixtures.
