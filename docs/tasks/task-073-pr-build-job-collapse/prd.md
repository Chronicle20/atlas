# PR Docker Build Job Collapse + Bootstrap Image Slim — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-21
---

## 1. Overview

The PR validation pipeline at `.github/workflows/pr-validation.yml` currently performs two Docker builds per labeled PR per affected service:

1. `build-docker` — matrix build, no push, runs on every PR that touches a service.
2. `build-docker-pr` — matrix build, push to `ghcr.io/.../*:pr-<N>-<sha>`, gated on the `deploy-env` label.

For most services this duplication is mildly wasteful (~30 s per service). For `atlas-pr-bootstrap` it is severe: the image bakes Apache Kafka 3.7.2 tarball (~85 MB, network-bound), `openjdk17-jre-headless` (~150 MB), `github-cli`/`kubectl`/`postgresql-client`/`redis`/`bash`/`jq`/`curl` (~50 MB), and the entire Kafka `libs/` jar directory. The result is roughly 5–8 minutes of wall-clock per build, and we pay it twice on labeled PRs because GHA's build cache is scoped per-job and the two jobs do not share warmth. On the day this issue surfaced, an `atlas-pr-bootstrap`-touching PR added ~7–8 min to `build-docker-pr` and ~15+ min to `build-docker` before completing.

This task addresses both layers of the problem. **Lever 1 (collapse jobs)** merges `build-docker` and `build-docker-pr` into a single matrix job that always builds (preserving Dockerfile validation for unlabeled PRs) and conditionally pushes (only when the `deploy-env` label is set). **Lever 2 (slim bootstrap)** trims `services/atlas-pr-bootstrap/Dockerfile` by replacing the Apache Kafka tarball + JRE with `kcat` and the minimum admin tooling, and by adding `--mount=type=cache` for the apk machine cache.

The combined wins target: ~5–8 min/service wall-clock reduction across every labeled PR (lever 1), plus another ~3–5 min/build of hygiene on the bootstrap image specifically (lever 2).

## 2. Goals

Primary goals:
- Eliminate the duplicate Docker build per labeled PR per service.
- Preserve Dockerfile validation on every PR (labeled or not) — a PR that breaks a Dockerfile must still fail CI.
- Preserve push behavior on `deploy-env`-labeled PRs (pushed tag `pr-<N>-<sha>`, same image name, same cache scope key) so `update-pr-overlay` and downstream Argo flows continue to work without changes.
- Slim `services/atlas-pr-bootstrap` so it builds in materially less time and is smaller on disk, without losing any tool that `bootstrap.sh` or `cleanup.sh` depends on.

Non-goals:
- Changing the `deploy-env` label semantics or the criteria for ephemeral-env deployment.
- Touching `.github/workflows/main-publish.yml` or main-branch publish flow.
- Re-writing the bot-branch overlay resolution in `update-pr-overlay`.
- Multi-arch (linux/arm64) builds.
- Re-scoping or renaming GHA caches across the rest of the repo.

## 3. User Stories

- As a backend engineer pushing a PR with a `deploy-env` label, I want my Docker images built once per service, not twice, so PR feedback arrives ~half as fast in wall-clock for the docker phase.
- As a backend engineer pushing an unlabeled PR, I want my Dockerfile to still be validated so I don't merge a broken image.
- As a DevOps engineer maintaining `atlas-pr-bootstrap`, I want its image to build in <3 min cold and rely on standard apk packages so I can iterate on `cleanup.sh` without waiting on a Kafka tarball download.
- As an oncall engineer reading PR status, I want the `pr-status` (`PR Validation Complete`) aggregator to report a single, unambiguous Docker result rather than two parallel ones.

## 4. Functional Requirements

### 4.1 Single matrix Docker job (`build-docker`)

- Exactly one Docker-build job remains. Working name: `build-docker`. The old `build-docker-pr` job is removed.
- `needs:` includes `detect-changes, test-go-services, test-go-libraries, test-ui` — unchanged from today.
- `if:` gate is the union of today's two gates, simplified:
  ```
  always() &&
  needs.detect-changes.outputs.docker-services-matrix != '[]' &&
  (needs.test-go-services.result == 'success' || needs.test-go-services.result == 'skipped') &&
  (needs.test-go-libraries.result == 'success' || needs.test-go-libraries.result == 'skipped') &&
  (needs.test-ui.result == 'success' || needs.test-ui.result == 'skipped')
  ```
  No `deploy-env` label check at the job gate — the gate is now inside the build step's `push:` input.
- Strategy: `fail-fast: false`, matrix from `needs.detect-changes.outputs.docker-services-matrix` — unchanged.
- Permissions: `contents: read, packages: write` (matches today's `build-docker-pr`; required for the conditional push).

### 4.2 Conditional build/push within the job

- Single `docker/build-push-action@v6` step (not two `if:`-gated steps; one composite-action call with computed inputs).
- `push` input is the literal string `'true'` when the PR carries `deploy-env`, `'false'` otherwise. Expression:
  ```yaml
  push: ${{ github.event_name == 'pull_request' && contains(github.event.pull_request.labels.*.name, 'deploy-env') }}
  ```
  (GHA coerces booleans to `'true'`/`'false'` strings.)
- `tags` input:
  - When `push == true`: `${{ matrix.service.docker_image }}:pr-${{ github.event.pull_request.number }}-${{ steps.sha.outputs.short }}` — identical to today's `build-docker-pr` output.
  - When `push == false`: a non-pushed tag is still required for the local build to succeed, but it does not need to be unique or stable across PRs. Use `${{ matrix.service.docker_image }}:pr-${{ github.event.pull_request.number }}` (same as today's `build-docker`).
  - A single expression evaluating to one of the two tags is acceptable; ternary via `${{ X && A || B }}` is fine.
- `cache-from`/`cache-to`:
  - **Must** stay scoped per-service (`scope=${{ matrix.service.name }}-amd64`) so per-service caches don't collide.
  - The scope **must match today's `build-docker-pr` scope key** so the first run after this change is a warm hit, not a cold miss. (Today's `build-docker` job had no cache at all; that cache is forfeit and not in scope to migrate.)
- `provenance: false`, `sbom: false` — unchanged from today's `build-docker-pr`.
- Login to ghcr (`docker/login-action@v3`) runs unconditionally inside the job; an unused login on unlabeled PRs is cheap and avoids step-skipping complexity. The login step's `if:` may be omitted.
- `Set up Docker Buildx` (`docker/setup-buildx-action@v3`) runs unconditionally.
- `Compute short SHA` runs unconditionally (the tag input may reference its output regardless of push state; ternary uses it only on the push branch).

### 4.3 Downstream wiring

- `update-pr-overlay`:
  - `needs:` updated from `[detect-changes, build-docker-pr]` to `[detect-changes, build-docker]`.
  - `if:` updated from `needs.build-docker-pr.result == 'success'` to `needs.build-docker.result == 'success'`.
  - Other clauses (`contains(... 'deploy-env')`, `github.event_name == 'pull_request'`) unchanged.
  - All inline comments mentioning `build-docker-pr` are updated to read `build-docker`.
- `pr-validation-complete`:
  - `needs:` drops `build-docker-pr` (single Docker job now).
  - The summary table collapses "Docker Builds" and "Docker PR Push" rows into a single "Docker Builds" row reporting `needs.build-docker.result`.
  - The failure check `if [ "$DOCKER_RESULT" == "failure" ] || [ "$DOCKER_PR_RESULT" == "failure" ]` reduces to a single `DOCKER_RESULT` check.
- No required-status-check name renames are introduced **unless** GitHub branch protection currently references `build-docker-pr` by name. (See §9.1.)

### 4.4 Slimmed `atlas-pr-bootstrap` Dockerfile

- The two-stage Kafka tarball stage is removed entirely.
- `kafka-topics.sh`, `kafka-consumer-groups.sh`, and `kafka-run-class.sh` are removed from the runtime image.
- `openjdk17-jre-headless` is removed from the apk install list.
- `kcat` is added to the apk install list. (`kcat` is in `community/` on Alpine 3.23.)
- `cleanup.sh` is rewritten so all listing operations go through `kcat -L -J` (JSON metadata) instead of `kafka-topics.sh --list` and `kafka-consumer-groups.sh --list`.
- Deletes must be preserved. Approach: use the Confluent-style Kafka admin REST via Kafka's `AdminClient` protocol. Since the cluster does not expose a REST proxy today, the deletes still need a Kafka admin client. Two acceptable options:
  - **Option A (preferred):** add the Apache `kafka-python` package (or `kafka-go` CLI binary) via apk to provide topic/group delete. Validate that the chosen tool's runtime cost (size, install time) is materially less than the Kafka tarball + JRE.
  - **Option B (fallback):** keep `kafka-topics.sh` + `kafka-consumer-groups.sh` for deletes only, but slim the JRE from `openjdk17-jre-headless` to `eclipse-temurin-17-jre-alpine` (or another smaller JRE), and replace the tarball stage with a minimal `COPY` of just the two `.sh` scripts plus the subset of `libs/` they actually load.
  - The design phase chooses between A and B based on apk-package availability and proven correctness, but the PRD acceptance criteria are the same: deletes still work, kafka-topics.sh tarball stage is gone or significantly smaller.
- `lib.sh` is updated only if its log labels reference the removed Kafka tool names; otherwise unchanged.
- `--mount=type=cache,target=/var/cache/apk,sharing=locked` is added to the apk install layer so re-runs reuse downloaded `.apk` files. (Requires `# syntax=docker/dockerfile:1.4` or later at top of Dockerfile, which Buildx supports by default.)

### 4.5 Bats / shell test re-verification

- The existing `services/atlas-pr-bootstrap/test/` bats suite must pass against the slimmed image.
- New test cases are added for the kcat-based listing path (golden output parsing) and for the new delete path (mocked admin call exit code 0 on success).
- The test job that runs bats (or however bootstrap tests run today — see §9.3) does not need to be re-wired.

## 5. API Surface

This task does not introduce HTTP APIs. Surface changes are:

### 5.1 Workflow contract

| Surface | Before | After |
|---|---|---|
| Job `build-docker` (validation) | runs always | **renamed concept**; now the only docker job, runs always |
| Job `build-docker-pr` (push) | runs on `deploy-env` PRs | **removed** |
| Output: pushed image tag | `pr-<N>-<sha>` (from `build-docker-pr`) | `pr-<N>-<sha>` (from `build-docker`) — unchanged for consumers |
| Cache scope key | `${service}-amd64` | `${service}-amd64` — unchanged |
| `update-pr-overlay.needs` | `build-docker-pr` | `build-docker` |
| `pr-validation-complete.needs` | both jobs | only `build-docker` |

### 5.2 GitHub branch protection / required checks

If `build-docker-pr` is currently listed as a required status check, this task **must** include a coordinated branch-protection update so the new `build-docker` check is required and the old one is removed. See §10 acceptance criteria.

### 5.3 Bootstrap image contract

- Image name `ghcr.io/chronicle20/atlas-pr-bootstrap` and PostDelete-hook entrypoint `/atlas/cleanup.sh` are unchanged.
- All env vars consumed by `cleanup.sh` (§9.1 in cleanup.sh header) remain consumed in the same way.
- `kcat` binary becomes a documented runtime dependency of `cleanup.sh`.

## 6. Data Model

Not applicable — no database changes.

## 7. Service Impact

| Service / area | Change |
|---|---|
| `.github/workflows/pr-validation.yml` | `build-docker-pr` job removed; `build-docker` job rewritten to single-step conditional push; `update-pr-overlay.needs` re-wired; `pr-validation-complete` aggregator simplified. |
| `.github/actions/docker-build/action.yml` | **No interface change required.** The composite action already accepts `push` and `tags` as inputs. Internal review only — confirm the action threads `push` through to `build-push-action` without surprises. |
| `services/atlas-pr-bootstrap/Dockerfile` | Stage 1 (Kafka tarball) removed (option A) or shrunk (option B). JRE removed (A) or replaced with smaller variant (B). `kcat` added. `--mount=type=cache` added. |
| `services/atlas-pr-bootstrap/scripts/cleanup.sh` | Topic/group listing switched to `kcat -L -J | jq`. Deletes switched to chosen admin client (A) or kept via Kafka shell (B). |
| `services/atlas-pr-bootstrap/scripts/bootstrap.sh` | Only changes if it currently shells `kafka-topics.sh`/`kafka-consumer-groups.sh` (verify in design phase). |
| `services/atlas-pr-bootstrap/scripts/lib.sh` | Only changes if it references removed binaries. |
| `services/atlas-pr-bootstrap/test/*.bats` | New cases for kcat listing path + new delete path. |
| `services/atlas-pr-bootstrap/README.md` | Update runtime-deps section. |
| `.github/workflows/main-publish.yml` | **Not touched.** Out of scope. |

## 8. Non-Functional Requirements

### 8.1 Performance

- **P-1**: Total wall-clock for the Docker phase on a labeled PR touching a single service must drop by ≥40% versus today's two-job baseline (warm-cache run). Measured by re-running an identical-content PR before and after on the same runner class.
- **P-2**: `atlas-pr-bootstrap` cold-cache image build must complete in ≤4 min (today: ~7–8 min). Warm-cache must complete in ≤90 s.
- **P-3**: No regression in any non-`atlas-pr-bootstrap` service's build time. (Lever 1's cache scope key is unchanged for those services.)

### 8.2 Correctness

- **C-1**: Every PR (labeled or not) that introduces a Dockerfile syntax/build error must fail `build-docker`. A PR that merges without ever exercising its Dockerfile is a regression.
- **C-2**: Pushed images on `deploy-env` PRs continue to land at `ghcr.io/.../<svc>:pr-<N>-<sha>` with the same content layout as today.
- **C-3**: `update-pr-overlay` continues to receive the matrix-of-built-services and produce the same `bot/pr-<N>-resolved` branch contents.
- **C-4**: `cleanup.sh` on a slimmed bootstrap image must still delete: per-env Postgres DBs, per-env Kafka topics matching `-<env>$`, per-env consumer groups matching `\[<env>\]$`, per-env Redis keys, per-PR ghcr image tags, and Pi-hole A records. Whitespace handling of group names (the `xargs -d '\n'` line documented in cleanup.sh) must be preserved.

### 8.3 Observability

- GHA step summaries for the new `build-docker` job must clearly indicate whether the push occurred (one-liner: `"Pushed: yes"` or `"Pushed: no (no deploy-env label)"`). Today's UX surfaces this via the presence/absence of `build-docker-pr`; after collapse we lose that signal unless we add it back inside the job summary.
- `pr-validation-complete` summary table must list the build outcome unambiguously (single row).

### 8.4 Multi-tenancy / security

- ghcr login credentials (`secrets.GHCR_TOKEN`) are only consumed when push actually fires (the `build-push-action` step gates on its own `push` input; login runs always but never authenticates against ghcr unless the subsequent push attempts it). Acceptable per existing posture.
- No new secret scopes introduced.

## 9. Open Questions

### 9.1 Branch-protection coupling

Does `https://github.com/Chronicle20/atlas/settings/branches` currently list `build-docker-pr` as a required status check on `main`? If yes, dropping the job without first updating branch protection will leave PRs that don't touch any service stuck waiting for a check that never runs. **Resolve before plan-task: query `gh api repos/Chronicle20/atlas/branches/main/protection`.**

### 9.2 Kcat coverage of admin operations

`kcat` does **not** support topic delete or consumer-group delete operations — it is a producer/consumer/metadata tool. The PRD assumes one of:
- (A) replace Java tools with a Python or Go admin client that's smaller than JRE+Kafka jars, **or**
- (B) keep the Java tools for deletes but minimize their footprint.

The design phase must pick A or B based on package size measurement. Open question: is there an Alpine package that ships `rpk` (Redpanda's CLI, fully Kafka-protocol-compatible, single Go binary)? If so it solves both listing and deletes in one tool.

### 9.3 Bats test execution location

Where does the bootstrap bats suite run in CI today? Inspection of `.github/workflows/pr-validation.yml` lines 1–400 found no bats job. The bats files at `services/atlas-pr-bootstrap/test/*.bats` may only run locally. The design phase should confirm and decide whether this task adds a CI hook for them, given correctness risk of the cleanup.sh rewrite.

### 9.4 `--mount=type=cache` portability

Does `docker/build-push-action@v6` configured the way it is in `.github/actions/docker-build/action.yml` honor `RUN --mount=type=cache,target=...` from the Dockerfile? Buildx does by default; non-Buildx legacy builder does not. Need to confirm the composite action explicitly uses Buildx (it does — `docker/setup-buildx-action@v3` is in `build-docker-pr`; need to confirm the new merged job inherits this; it should since we keep that step).

### 9.5 Empty-matrix edge case

When `docker-services-matrix == '[]'`, today both jobs are skipped, and `pr-validation-complete` treats `skipped` as non-failing. After collapse only one job is skipped — the aggregator logic must still treat that single `skipped` value correctly. (Spot-check, but should be fine since `[ "skipped" == "failure" ]` is false.)

## 10. Acceptance Criteria

- [ ] `.github/workflows/pr-validation.yml` contains a single Docker build job named `build-docker` and no `build-docker-pr` job.
- [ ] On an unlabeled PR that modifies a service Dockerfile, CI fails when the Dockerfile is broken (validation preserved). Verified by deliberately breaking a Dockerfile on a no-`deploy-env` PR.
- [ ] On a labeled PR (`deploy-env`), the per-PR tag `ghcr.io/chronicle20/<svc>:pr-<N>-<sha>` is pushed exactly once per affected service. Verified by ghcr UI or `gh api`.
- [ ] On an unlabeled PR, **no** `pr-<N>-*` tag is pushed to ghcr. Verified by ghcr tag list before/after.
- [ ] `update-pr-overlay` continues to produce `bot/pr-<N>-resolved` with correctly bumped image tags on labeled PRs. Verified by diffing the bot branch before and after on a single fixture PR.
- [ ] `pr-validation-complete` reports a single Docker row and passes on labeled and unlabeled PRs.
- [ ] If `build-docker-pr` was a required branch-protection check on `main`, branch protection is updated to require `build-docker` and not `build-docker-pr`. Documented in the task's `audit.md` post-implementation.
- [ ] `services/atlas-pr-bootstrap/Dockerfile` produces an image that:
  - Builds cold in ≤4 min on a standard GHA `ubuntu-latest` runner.
  - Contains `kcat` (`/usr/bin/kcat`) and not `openjdk17-jre-headless`.
  - Contains the admin tool chosen in design phase (option A) or a slimmed Kafka CLI stage (option B).
- [ ] `cleanup.sh` end-to-end pass on a real Atlas PR-env (or a faithfully mocked harness): drops dbs, topics, groups, redis keys, ghcr tags, dns records. Documented evidence in `audit.md`.
- [ ] Existing bats suite for `atlas-pr-bootstrap` passes; new cases added for the kcat listing path and new delete path.
- [ ] Measured before/after wall-clock for the Docker phase on a labeled-PR replay shows ≥40% reduction. Captured in `audit.md`.
- [ ] No changes outside of: `.github/workflows/pr-validation.yml`, `.github/actions/docker-build/` (if any), `services/atlas-pr-bootstrap/**`. Confirmed by `git diff --stat main...task-073`.
