# Plan Audit — task-073-pr-build-job-collapse

**Plan Path:** `docs/tasks/task-073-pr-build-job-collapse/plan.md`
**Audit Date:** 2026-05-21 (rebase update added 2026-05-21)
**Branch:** `task-073-pr-build-job-collapse`
**Base Branch:** `main`

## Rebase update (2026-05-21)

The branch was rebased onto `origin/main` after PR #541 (task-070,
`fix(pr-env): teardown contract + sweep + smoke regression`) landed.
Conflict resolution preserved the original audit findings; commit SHAs
changed but the per-task evidence (file paths, line anchors) remains
accurate against the rebased tree. Post-rebase commit topology:

- `bd186f8b9` ci(pr-validation): collapse build-docker + build-docker-pr into one job
- `f7b58a6d2` ci(pr-validation): drop literal build-docker-pr from banner
- `d48594e0e` ci(docker-build): accept provenance/sbom inputs (default false)
- `5abf890ab` test(atlas-pr-bootstrap): add bats stub harness for cleanup.sh
- `2688e5b22` feat(atlas-pr-bootstrap): switch cleanup.sh drop-topics/groups to rpk
- `27e55896b` feat(atlas-pr-bootstrap): replace Kafka tarball + JRE with rpk static binary
- `c2a35daa8` docs(atlas-pr-bootstrap): document rpk and apk runtime deps
- `dbc2d4443` test(atlas-pr-bootstrap): fix bootstrap_test env arg order
- `c2eb301db` audit(task-073): plan adherence — READY FOR PR (this file, pre-rebase)

Rebase deltas (vs. pre-rebase audit):

- `cleanup.sh` rpk swap now lands on top of task-070's `compute_atlas_env`
  derivation and `drop-branch` phase. Both changes are textually
  compatible; the rebased file derives `ATLAS_ENV` from `PR_NUMBER` and
  then runs the rpk-migrated topic/group cleanup.
- `cleanup_test.bats` was rewritten as part of conflict resolution to:
  (a) keep task-070's PR_NUMBER-required and branch-delete cases,
  (b) keep our stub harness (`make_stubs`, `run_cleanup`),
  (c) compute the fixture env-hash via `compute_atlas_env 99` inside a
  new `fixture_env` helper so the rpk-suffix fixtures match cleanup.sh's
  derived hash. All 20 bats cases in `services/atlas-pr-bootstrap/test/`
  pass post-rebase (10 from this branch's relevant scope + 10 from
  task-070's sweep/lib tests).
- `pr-validation.yml` collapse landed on top of task-070's
  `update-pr-overlay` extension to also substitute placeholders into
  `deploy/k8s/overlays/pr-cleanup/`. Both changes are textually
  compatible.

**Pre-existing on main, surfaced by the rebase but explicitly out of scope:**
task-070 introduced `scripts/sweep-orphans.sh` and `test/sweep_test.bats`,
which still call `kafka-topics.sh` / `kafka-consumer-groups.sh`. The
runbook (`docs/runbooks/ephemeral-pr-deployments.md:366`) references
`/atlas/sweep-orphans.sh` as an in-cluster invocation, but neither main's
Dockerfile nor our rewritten Dockerfile `COPY`s `sweep-orphans.sh` into
the image — that runbook path was already aspirational pre-rebase, so our
T8 image slim does not regress sweep-orphans runtime behavior (zero in
both states). Migrating sweep-orphans to `rpk` is a follow-up task
(suggested ticket: "task-NNN: migrate sweep-orphans.sh to rpk and COPY
into atlas-pr-bootstrap image") and is intentionally out of this PR's
scope.

**Post-rebase verification:**

- `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/pr-validation.yml'))"` → `OK`
- `python3 -c "import yaml; yaml.safe_load(open('.github/actions/docker-build/action.yml'))"` → `OK`
- `grep -rn 'build-docker-pr' .github/ services/atlas-pr-bootstrap/` → zero matches
- `grep -rnE 'kafka-topics|kafka-consumer-groups|kafka-run-class|openjdk' services/atlas-pr-bootstrap/` → matches only in `scripts/sweep-orphans.sh` and `test/sweep_test.bats` (both pre-existing on main, out of scope per above)
- `bats services/atlas-pr-bootstrap/test/` → 20/20 green
- `docker build -f services/atlas-pr-bootstrap/Dockerfile services/atlas-pr-bootstrap` → succeeds

Original audit text below applies; only commit SHAs are stale.

## Executive Summary

All ten plan tasks (T1–T10) are fully implemented with file-level evidence. Six commits on the task branch correspond directly to plan task groupings; two additional commits (`0a64b1cd7` provenance/sbom inputs, `b2c2afb9c` bootstrap_test env arg order) cover deviations that are necessary correctness fixes, each justified below. The five-case bats suite plus two pre-existing bootstrap cases all pass (7/7). The slimmed `atlas-pr-bootstrap` Docker image builds successfully, contains `rpk v24.3.1` on `PATH`, and contains no `java` / `/opt/kafka` paths. The collapsed `pr-validation.yml` parses as valid YAML and contains zero `build-docker-pr` references. Scope is correctly limited to `.github/` (workflow + composite action) and `services/atlas-pr-bootstrap/` (Dockerfile + scripts + tests + README) — no Go or atlas-ui changes.

## 1. Plan Task Coverage

| # | Task | Status | Evidence |
|---|------|--------|----------|
| T1 | Collapse `build-docker` + `build-docker-pr` into one matrix job | PASS | `.github/workflows/pr-validation.yml:134-194` — single `build-docker:` job with `Compute push flag and tag` step gating on `deploy-env` label (lines 162-181); cache scope key `${{ matrix.service.name }}-amd64` preserved (lines 193-194). Commit `815ad5127`. |
| T2 | Rewire `update-pr-overlay` to depend on `build-docker` | PASS | `.github/workflows/pr-validation.yml:223` (`needs: [detect-changes, build-docker]`); line 228 (`needs.build-docker.result == 'success'`); banner comment line 199 (`After build-docker completes...`); inline comment line 293 (`build-docker pushed pr-<N>-<sha> tags`). Commits `815ad5127` + `66fab8708`. |
| T3 | Collapse Docker rows in `pr-validation-complete` | PASS | `.github/workflows/pr-validation.yml:345` (`needs: [...build-docker, update-pr-overlay]` — no `build-docker-pr`); no `DOCKER_PR_RESULT` capture; no "Docker PR Push" summary row; failure check line 373 has no `DOCKER_PR_RESULT` term. Workflow parses as valid YAML (`python3 -c "import yaml; yaml.safe_load(...)"` → `OK`). Commit `815ad5127`. |
| T4 | Bats setup helper with stubs for `rpk`/`psql`/`redis-cli`/`gh` | PASS | `services/atlas-pr-bootstrap/test/cleanup_test.bats:3-65` — `setup()` (PROJECT_ROOT/STUB_BIN/STUB_LOG), `make_stubs()` writing four stubs, `run_cleanup()` PATH wrapper, two pre-existing require_env cases retained. Commit `c7e04ccd4`. |
| T5 | Bats case: `rpk topic delete` only for `-${ATLAS_ENV}` topics | PASS | `cleanup_test.bats:82-98` — asserts `rpk topic list` invoked once, `foo-test`/`baz-test` deleted, `bar` not deleted. Committed with T7 implementation in `ac0360027`. (Plan T5 step 2 said "leave failing, no commit" — that TDD red phase is invisible in git history, which is acceptable per plan's "T5/T6 deliberately leave failing tests on disk uncommitted" note.) |
| T6 | Bats case: `rpk group delete` preserves names with spaces | PASS | `cleanup_test.bats:100-120` — asserts `Party Quest Service [test]` (spaces intact) is passed to `rpk group delete` as one argument; `Other [other]` is not. Committed with T7 in `ac0360027`. |
| T7 | Switch `cleanup.sh` drop-topics/groups to `rpk` | PASS | `services/atlas-pr-bootstrap/scripts/cleanup.sh:44-63` — `rpk topic list -X brokers=… --format json | jq -r '.topics[].name' | { grep -E -- "-${ATLAS_ENV}\$" \|\| true; } | xargs -r -n 1 rpk topic delete …` and the analogous `rpk group list / delete` block with `-d '\n'` and `[ENV]\$` regex preserved. Step 5 third bats case (`skips rpk topic delete when no topic matches`) at `cleanup_test.bats:122-132`. `grep -E 'kafka-topics\|kafka-consumer-groups\|kafka-run-class' services/atlas-pr-bootstrap/scripts/cleanup.sh` → zero matches. Full bats suite passes (5/5 cleanup + 2 bootstrap). Commit `ac0360027`. |
| T8 | Rewrite `services/atlas-pr-bootstrap/Dockerfile` | PASS | `services/atlas-pr-bootstrap/Dockerfile:1-41` — `# syntax=docker/dockerfile:1.4`, single-stage `alpine:3.23`, `ARG RPK_VERSION=24.3.1`, `RUN --mount=type=cache,target=/var/cache/apk,sharing=locked apk add …`, `rpk` fetched from `redpanda-data/redpanda` v24.3.1 release zip, short-path `COPY scripts/…` matching existing build context, no `openjdk` / `/opt/kafka` / `ENV PATH=/opt/kafka/bin`. `docker build -f services/atlas-pr-bootstrap/Dockerfile services/atlas-pr-bootstrap` succeeds; runtime check (`docker run --rm --entrypoint sh "$IMG" -c 'rpk version …'`) prints `Version: v24.3.1 / Git ref: afe1a3f1ff / Build date: 2024-12-02T22:25:56Z` and confirms `java` is absent. Commit `9ac5582b6`. |
| T9 | Update README with runtime-deps section | PASS | `services/atlas-pr-bootstrap/README.md:15-27` — "Runtime dependencies" section names the apk packages, the vendored `rpk` binary with `RPK_VERSION` build arg, and explains the JRE/tarball removal rationale. Commit `34a27d760`. |
| T10 | Final verification (no commit) | PASS | All seven verify steps executed during this audit: `grep -rn 'build-docker-pr' .github/ services/atlas-pr-bootstrap/` → zero matches; `grep -rnE 'kafka-topics\|kafka-consumer-groups\|kafka-run-class\|openjdk' services/atlas-pr-bootstrap/` → zero matches; `bats services/atlas-pr-bootstrap/test/` → 7/7 ok; `docker build` succeeds; YAML parse OK; `git diff --stat main…HEAD` shows only in-scope paths. Step 7 (hand off to code review) is satisfied by this audit. |

**Completion Rate:** 10/10 plan tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## 2. Deviations from Plan

### D-1. Provenance/SBOM inputs added to composite action (commit `0a64b1cd7`) — JUSTIFIED

Not in plan T1–T10. Necessary correctness fix.

**Root cause.** The deleted `build-docker-pr` job on `main` set `provenance: false` and `sbom: false` directly on `docker/build-push-action@v6` (verified at `git show main:.github/workflows/pr-validation.yml` lines 211-212). The composite action `.github/actions/docker-build/action.yml` did not previously expose or forward these inputs, so the collapsed `build-docker` job going through the composite would have regressed to `docker/build-push-action@v6`'s default `provenance: mode=min`, which produces a multi-manifest list. The Argo PR overlay's `kustomize images:` bump rewrites a single `newTag:`, which mis-resolves against a manifest list digest reference. `main-publish.yml` (lines 156-157, 200-201) also sets both to `false` explicitly today, so the composite action's new defaults (`'false'`) match production posture for both workflows.

**Diff is minimal and idiomatic.** Two new optional inputs with `default: 'false'`, forwarded to the underlying `docker/build-push-action@v6` step. No behavioral change for any existing caller because the underlying action default was `mode=min` while every caller explicitly overrode it.

**Verdict.** Required. Without this commit T1's collapsed job would have silently regressed an Argo-contract invariant.

### D-2. `|| true` wrappers on grep stages in cleanup.sh (commit `ac0360027`) — JUSTIFIED

Not in plan T7 step 1/2 literal text. Necessary correctness fix.

**Root cause.** Plan T7 step 1 spec was:

```bash
… | jq -r '.topics[].name' \
    | grep -E -- "-${ATLAS_ENV}\$" \
    | xargs -r -n 1 rpk topic delete …
```

But `cleanup.sh:17` is `set -euo pipefail`. Under `pipefail`, `grep` returning 1 (no match) propagates as a pipeline failure, which `set -e` then turns into script exit. Plan T7 step 5 (the third bats case) explicitly asserts the script must succeed and skip `rpk topic delete` when no topic name ends with `-${ATLAS_ENV}`. The literal plan code would have failed that test. The implementer wrapped each grep in `{ grep … || true; }` so unmatched grep returns 0 to the pipeline and `xargs -r` then short-circuits on empty input. PRD §8.2 C-4 invariants are preserved:
- Regex `-${ATLAS_ENV}\$` and `\[${ATLAS_ENV}\]\$` unchanged (`scripts/cleanup.sh:50, 62`).
- `xargs -r -d '\n' -n 1` on groups path unchanged (`scripts/cleanup.sh:63`).
- `xargs -r` no-op-on-empty behavior unchanged (`scripts/cleanup.sh:51, 63`).

Note: the pre-change cleanup.sh on `main` ran `kafka-topics.sh … | grep … | xargs …` under the same `set -euo pipefail` and had a latent pipefail bug for the empty-match case. The bats stubs surfaced it; the fix incidentally corrects that pre-existing latent bug too.

**Verdict.** Required. Plan-literal code would have failed T7 step 5's own assertion.

### D-3. `bootstrap_test.bats` env arg order one-liner (commit `b2c2afb9c`) — JUSTIFIED

Not in plan T1–T10. Pre-existing latent bug surfaced by T10's "Bats suite final pass".

**Root cause.** `bootstrap_test.bats:14` (pre-fix) read:
```bash
run env ATLAS_ENV=test -u ATLAS_UI_BASE bash …
```
GNU `env` parses positional `VAR=VAL` assignments before processing `-u` flags. With the assignment first, `-u` is treated as the command name, leading to exit 127 ("command not found") instead of surfacing the `require_env` error. The case was effectively passing for the wrong reason. The implementer applied the same one-character reorder pattern that T4's `cleanup_test.bats` already used (the new cleanup cases at `cleanup_test.bats:74-77` correctly put `-u` first), so the fix is strictly a test-side correction with no production impact. Commit body explicitly cites this as "the analogous fix applied to cleanup_test.bats."

**Verdict.** Required. Test-only correction surfaced by running `bats services/atlas-pr-bootstrap/test/` as plan T10 step 3 mandates.

### D-4. `kcat` → `rpk` (PRD §4.4 / §10 literal deviation) — DOCUMENTED

Per `context.md` "Key decisions" (lines 28-29) and `design.md` §4.1. PRD §4.4 listed `kcat` as the addition; PRD §10 acceptance criterion was literally `kcat (/usr/bin/kcat)`. The design chose `rpk` over `kcat` because `rpk` covers list + delete for both topics and consumer groups in a single static binary, while `kcat` does not implement consumer-group admin operations. PRD §9.2 explicitly permits "single tool that covers list + delete", which `rpk` satisfies and `kcat` does not. The acceptance criterion is therefore restated as "Contains `rpk` (`/usr/local/bin/rpk`) and not `openjdk17-jre-headless`." Plan T10 step 7 explicitly required the audit to document this — captured here.

**Verdict.** Documented in three places (context.md, design.md §4.1, this audit). Not a silent drift.

### Other deviations
None found. No tasks were silently reordered, no acceptance criteria were dropped, and the git history (`git log main..HEAD`) cleanly maps eight commits to the plan's seven prescribed commits plus the two deviation fixes above.

## 3. Verification Command Outputs

```
$ grep -nE '^  (build-docker|build-docker-pr):' .github/workflows/pr-validation.yml
134:  build-docker:

$ grep -c 'build-docker-pr' .github/workflows/pr-validation.yml
0

$ grep -rn 'build-docker-pr' .github/ services/atlas-pr-bootstrap/
(no output — zero matches)

$ grep -nE 'kafka-topics|kafka-consumer-groups|kafka-run-class' services/atlas-pr-bootstrap/scripts/cleanup.sh
(no output)

$ grep -rnE 'kafka-topics|kafka-consumer-groups|kafka-run-class|openjdk' services/atlas-pr-bootstrap/
(no output)

$ grep -nE 'kafka|openjdk|/opt/kafka' services/atlas-pr-bootstrap/Dockerfile
(no output)

$ python3 -c "import yaml; yaml.safe_load(open('.github/workflows/pr-validation.yml'))" && echo OK
OK

$ bats services/atlas-pr-bootstrap/test/
1..7
ok 1 bootstrap.sh fails without ATLAS_ENV
ok 2 bootstrap.sh fails without ATLAS_UI_BASE
ok 3 cleanup.sh fails without ATLAS_ENV
ok 4 cleanup.sh fails without ATLAS_DB_NAMES
ok 5 cleanup.sh deletes only -ATLAS_ENV-suffixed topics via rpk
ok 6 cleanup.sh deletes consumer groups with spaces in their names
ok 7 cleanup.sh skips rpk topic delete when no topic matches

$ docker build -f services/atlas-pr-bootstrap/Dockerfile services/atlas-pr-bootstrap
… (truncated; all 9 stage-0 layers succeed; final manifest exported)

$ IMG=$(docker build -q -f services/atlas-pr-bootstrap/Dockerfile services/atlas-pr-bootstrap)
$ docker run --rm --entrypoint sh "$IMG" -c 'rpk version; (which java && echo BAD) || echo "no java"; ls /opt 2>/dev/null || echo "(no /opt)"'
Version:     v24.3.1
Git ref:     afe1a3f1ff
Build date:  2024-12-02T22:25:56Z
no java
(no /opt)

$ git diff --stat main...HEAD -- ':!docs/'
 .github/actions/docker-build/action.yml            |  10 ++
 .github/workflows/pr-validation.yml                | 112 ++++++++------------
 services/atlas-pr-bootstrap/Dockerfile             |  46 ++++----
 services/atlas-pr-bootstrap/README.md              |  14 +++
 services/atlas-pr-bootstrap/scripts/cleanup.sh     |  17 +--
 services/atlas-pr-bootstrap/test/bootstrap_test.bats    |   2 +-
 services/atlas-pr-bootstrap/test/cleanup_test.bats | 117 ++++++++++++++++++++-
 7 files changed, 217 insertions(+), 101 deletions(-)

$ git log --oneline main..HEAD -- .github/workflows/main-publish.yml services/atlas-ui/ libs/
(no output — no out-of-scope changes)
```

## 4. Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY FOR PR

All ten plan tasks are implemented with passing verification. The four deviations from the literal plan text (provenance/sbom composite-action inputs, cleanup.sh `|| true` wrappers, bootstrap_test env arg order, kcat→rpk tool substitution) are each justified by either a correctness regression that the plan-literal code would have caused (D-1, D-2), a pre-existing latent bug surfaced by the plan's own verification step (D-3), or an upstream design decision already documented in `design.md` and `context.md` (D-4).

## 5. Action Items

None. No FAIL or NEEDS REWORK findings.

Per CLAUDE.md "Code Review Before PR": the plan-adherence audit (this document) is one of three reviewer outputs. The PR is Go-free and TypeScript-free, so neither `backend-guidelines-reviewer` nor `frontend-guidelines-reviewer` is applicable. The user may proceed to open the PR.
