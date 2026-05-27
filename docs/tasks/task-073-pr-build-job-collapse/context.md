# Task 073 — Context

## Goal

Halve PR Docker validation wall-clock and slim `atlas-pr-bootstrap` runtime image.

Two decoupled levers in one PR:

1. **Lever 1 — Workflow collapse.** Merge `build-docker` (validate) + `build-docker-pr` (push) into one matrix job that always builds and pushes only when `deploy-env` is set. Rewire `update-pr-overlay` and `pr-validation-complete`.
2. **Lever 2 — Bootstrap slim.** Replace Apache Kafka tarball + JRE in `services/atlas-pr-bootstrap/Dockerfile` with a single static `rpk` binary; add buildkit apk cache mount; rewrite `cleanup.sh` drop-topics / drop-groups blocks to use `rpk`.

## Key files

| Path | What it does today | What changes |
|---|---|---|
| `.github/workflows/pr-validation.yml` | Two matrix Docker jobs (validate + push); aggregator references both | Single matrix job; aggregator collapses |
| `.github/actions/docker-build/action.yml` | Composite that wraps `docker/build-push-action@v6` (already accepts `push`, `tags`, `cache-*`, login gated internally) | **No change.** Interface already supports the merged job. |
| `services/atlas-pr-bootstrap/Dockerfile` | Two-stage build (Apache Kafka tarball + JRE; runtime image ~400+ MB) | Single-stage Alpine + `rpk` static binary + apk cache mount; ~80 MB target |
| `services/atlas-pr-bootstrap/scripts/cleanup.sh` | Uses `kafka-topics.sh` and `kafka-consumer-groups.sh` for list/delete | Switch both to `rpk` |
| `services/atlas-pr-bootstrap/scripts/bootstrap.sh` | No Kafka shell deps | **No change.** Verified `grep` clean. |
| `services/atlas-pr-bootstrap/scripts/lib.sh` | Generic helpers | **No change.** Verified `grep` clean. |
| `services/atlas-pr-bootstrap/test/cleanup_test.bats` | Tests only `require_env` | Add cases stubbing `rpk`/`psql`/`redis-cli` |
| `services/atlas-pr-bootstrap/test/bootstrap_test.bats` | Tests `require_env` only | **No change.** |
| `services/atlas-pr-bootstrap/README.md` | Lists scripts; no explicit runtime-deps section | Add a "Runtime dependencies" note mentioning `rpk` (vendored), apk packages, and the `RPK_VERSION` build arg |

## Key decisions (locked in design)

- **`rpk` chosen over `kafka-python`, `eclipse-temurin-17-jre-alpine`, `kaf`, self-built Go binary** — single static binary, ~30 MB, no runtime needed, covers list + delete for both topics and consumer groups (1:1 with the Apache CLIs we're replacing). See design §4.7.
- **`kcat` is dropped from the design** — PRD §4.4 listed it as an addition, but PRD §9.2 explicitly permits "single tool that covers list + delete" which `rpk` does. PRD §10's literal "`kcat` (`/usr/bin/kcat`)" criterion is superseded; the audit step records this deviation. The acceptance criterion becomes "Contains `rpk` (`/usr/local/bin/rpk`) and not `openjdk17-jre-headless`."
- **`build-docker-pr` is not a required branch-protection check.** `gh api ... /branches/main/protection` returned required contexts `["gitleaks", "PR Validation Complete"]`. No branch-protection coordination needed; only `pr-validation-complete` is externally observable, and the collapse preserves it.
- **`docker/build-push-action@v6` via the composite action does honor `--mount=type=cache`** — composite's first step is `docker/setup-buildx-action@v3`. Required at the top of the Dockerfile: `# syntax=docker/dockerfile:1.4`.
- **Cache scope key stays `${service}-amd64`** — identical to today's `build-docker-pr`, so the first post-merge labeled-PR run hits the existing GHA cache warm. Today's `build-docker` job had no cache; that's forfeit by design.
- **ghcr login is gated inside the composite action** on `inputs.push == 'true' && inputs.registry-username != ''`. The collapsed `build-docker` job does **not** need a separate `docker/login-action@v3` step — the design's text overrides PRD §4.2 wording. An unlabeled PR runs zero login.
- **`pr-validation-complete` is the only branch-protection-required check.** Must keep passing.
- **bats tests are local-only.** No CI bats job exists today; this task does not add one. New bats cases will run during plan-task acceptance and at code-review time, manually.

## Open questions — resolved

All five PRD §9 questions are resolved in design §2. No remaining open questions for implementation.

## Verification posture (per CLAUDE.md)

- **Mandatory:** `docker build -f services/atlas-pr-bootstrap/Dockerfile .` from the worktree root after the Dockerfile changes.
- Go test/vet/build are N/A — no Go code changes.
- `bats services/atlas-pr-bootstrap/test/` must pass locally.
- YAML lint via `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/pr-validation.yml'))"` for syntactic correctness (CI-side semantic correctness will be the PR's own first build).

## Sequencing rationale

Lever 1 (workflow) lands before Lever 2 (Dockerfile + cleanup.sh) within the same PR. The order has no runtime coupling — both levers touch disjoint paths — but the design notes that landing the workflow collapse before the Dockerfile change means the new `build-docker` cleanly exercises the slimmed bootstrap Dockerfile on the task PR itself (meta-validation). Within the commit history this is purely cosmetic; we commit Lever 1, then Lever 2 piecewise (TDD), then Docker build verification, then README.

## Out of scope

- `.github/workflows/main-publish.yml` (PRD §2 non-goal)
- Multi-arch (linux/arm64) builds
- Adding a CI hook for bats
- Touching any other `libs/` or `services/`
- `update-pr-overlay`'s overlay-substitution logic — only its `needs:`/`if:`/comment text changes
