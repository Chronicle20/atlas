# PR Docker Build Job Collapse + Bootstrap Image Slim — Design

Version: v1
Status: Approved
Created: 2026-05-21
PRD: `docs/tasks/task-073-pr-build-job-collapse/prd.md`

---

## 1. Summary

Two largely independent levers land together as a single PR:

- **Lever 1 — Workflow collapse.** Merge `build-docker` and `build-docker-pr` in `.github/workflows/pr-validation.yml` into one matrix job. Always build (preserve Dockerfile validation for unlabeled PRs); push only when the `deploy-env` label is set. Rewire `update-pr-overlay` and `pr-validation-complete` to `needs: build-docker`.
- **Lever 2 — Bootstrap slim.** Replace the Apache Kafka tarball + `openjdk17-jre-headless` in `services/atlas-pr-bootstrap/Dockerfile` with two small native binaries: `kcat` (apk, for listing/metadata) and `rpk` (single static Go binary, for topic + consumer-group delete). Add `--mount=type=cache` on the apk install layer. Rewrite `cleanup.sh`'s list/delete blocks.

Both levers are decoupled at runtime: lever 1 changes only `.github/`, lever 2 changes only `services/atlas-pr-bootstrap/`. The single shared concern is that `build-docker`'s new cache scope (per-service) must give the slimmed `atlas-pr-bootstrap` image a warm path on the first labeled-PR replay.

---

## 2. Open-question resolutions

These are settled before plan-task starts. Each disposition feeds a concrete architecture decision below.

| PRD § | Question | Resolution |
|---|---|---|
| 9.1 | Is `build-docker-pr` a required check on `main`? | **No.** `gh api repos/Chronicle20/atlas/branches/main/protection` returns required contexts `["gitleaks", "PR Validation Complete"]`. Dropping `build-docker-pr` requires no branch-protection coordination. `pr-validation-complete` must keep passing — that is the only externally-required job and the collapse preserves it. |
| 9.2 | Choose A (replace Java tools) or B (slim Java tools)? | **Option A using `rpk`.** `rpk` is a single statically-linked Go binary (~30 MB) that speaks the Kafka admin protocol natively and covers `topic list/delete` and `group list/delete`. It is not in Alpine's apk index but ships as an official Linux/amd64 release asset; we fetch it once during build via `curl` + a layer cache. `kafka-python` would still pull in Python runtime (~30 MB) and pip install; the Apache CLI would still require JRE (~150 MB). `kcat` remains the listing path for completeness and because it is the canonical Alpine tool, but the actual cleanup script can lean on `rpk` alone — `kcat` is not strictly required and is dropped to keep the dependency surface minimal. See §4.1 for the final tool choice. |
| 9.3 | Where do the bats tests run in CI? | **They don't.** No bats job exists in `pr-validation.yml`. They are local-only. This task does **not** add a CI hook; it adds new bats cases that the contributor runs locally during plan execution and that the reviewer can rerun. Adding a CI bats job is a future task. |
| 9.4 | Does `docker/build-push-action@v6` (via `./.github/actions/docker-build`) honor `RUN --mount=type=cache`? | **Yes.** The composite action's first step is `docker/setup-buildx-action@v3`, which installs the buildkit-backed builder. Buildkit honors cache mounts unconditionally when the Dockerfile syntax frontend supports them — verified by adding `# syntax=docker/dockerfile:1.4` at the top of the slimmed Dockerfile. |
| 9.5 | Empty-matrix edge case after collapse | **No regression.** `build-docker` already has `needs.detect-changes.outputs.docker-services-matrix != '[]'` in its `if:` gate; when the matrix is empty the job is `skipped`. The aggregator's `[ "$DOCKER_RESULT" == "failure" ]` check correctly treats `skipped` as pass-through. |

---

## 3. Lever 1 — Workflow collapse

### 3.1 Final job topology

Before:

```
detect-changes → test-go-* / test-ui ┬→ build-docker      (validation, no push, no cache)
                                     └→ build-docker-pr   (push on deploy-env, cached)
                                            └→ update-pr-overlay
                                            └→ pr-validation-complete (needs both)
```

After:

```
detect-changes → test-go-* / test-ui ─→ build-docker      (validation + conditional push, cached)
                                            └→ update-pr-overlay  (gated on deploy-env)
                                            └→ pr-validation-complete (single Docker row)
```

`build-docker-pr` is **deleted in full** (the whole job stanza). Its permissions, ghcr login, buildx setup, and cache config migrate into the surviving `build-docker` job.

### 3.2 Conditional push expression

GHA's expression engine coerces boolean expressions to the literal strings `'true'`/`'false'` when used as a step input. The composite action `./.github/actions/docker-build` already accepts `push` as a string input and threads it through to `docker/build-push-action@v6`.

```yaml
push: ${{ github.event_name == 'pull_request' && contains(github.event.pull_request.labels.*.name, 'deploy-env') }}
```

This evaluates to:
- `'true'` on PR events with the `deploy-env` label → image pushed
- `'false'` on every other case (PR without label, workflow_dispatch) → image built locally only

Both branches still execute the full `docker/build-push-action@v6` step, so a broken Dockerfile fails the job regardless of push state. **C-1 is satisfied.**

### 3.3 Tag computation

`docker-build`'s composite action takes `image-name` and a comma-separated `tags` list (without `image:` prefix), then concatenates them inside. The action does **not** support an empty tag list; it always materializes at least one tag for the local build.

Both branches need a tag, but the value differs:

- Pushed: `pr-${PR_NUMBER}-${SHORT_SHA}` (matches today's `build-docker-pr` output verbatim — C-2).
- Local-only: `pr-${PR_NUMBER}` (matches today's `build-docker` — innocuous local label, never pushed).

Short-SHA computation moves into the `build-docker` job:

```yaml
- name: Compute short SHA
  id: sha
  run: |
    SHA="${{ github.event.pull_request.head.sha || github.sha }}"
    echo "short=$(git rev-parse --short=7 "$SHA")" >> $GITHUB_OUTPUT
```

The fallback to `github.sha` covers `workflow_dispatch` (no PR context). On `workflow_dispatch` the `contains()` check returns false, so push is always false and the tag falls into the local-only branch — `pr-${PR_NUMBER}` would be empty there, so guard:

```yaml
tags: ${{ (github.event_name == 'pull_request' && contains(github.event.pull_request.labels.*.name, 'deploy-env')) && format('pr-{0}-{1}', github.event.pull_request.number, steps.sha.outputs.short) || format('pr-{0}', github.event.pull_request.number || 'dispatch') }}
```

Cleaner alternative — pre-compute tag in a `Compute tag` step and reference its output. Plan-task should implement the cleaner alternative; the inline ternary is shown above to make the contract explicit.

### 3.4 Cache wiring

The composite action accepts `cache-from`/`cache-to` as inputs. They get wired the same way `build-docker-pr` does today:

```yaml
cache-from: type=gha,scope=${{ matrix.service.name }}-amd64
cache-to: type=gha,mode=max,scope=${{ matrix.service.name }}-amd64
```

Scope key `${service}-amd64` is **identical** to today's `build-docker-pr` scope. The first run after merge against a labeled-PR replay hits the existing cache. The unlabeled-PR cache (today: nothing — `build-docker` had no cache) is forfeit by design — not in scope to migrate, and the post-collapse warmth on unlabeled PRs comes from the same single scope, which fills in on the first labeled-PR run anyway.

### 3.5 ghcr login

The composite action `docker-build` gates its login step internally on `inputs.push == 'true' && inputs.registry-username != ''`. That contradicts the PRD §4.2 wording ("login runs unconditionally"), but the behavior is strictly better: an unlabeled PR with no need to push also runs zero login. Plan-task adopts the composite action's existing gate — **the workflow does not call `docker/login-action@v3` outside the composite action.**

Login credentials are passed in unconditionally:

```yaml
registry-username: ${{ github.actor }}
registry-password: ${{ secrets.GHCR_TOKEN }}
```

The composite action will skip the login step when `push: false`. No new secret scopes (NF-1).

### 3.6 Permissions

The collapsed job needs `packages: write` only when push fires, but GHA permissions are per-job, not per-step. Set unconditionally:

```yaml
permissions:
  contents: read
  packages: write
```

This is wider than today's `build-docker` job (which had only `contents: read`) but matches `build-docker-pr`. Acceptable per NF-1 — no new secrets, just a wider scope on the existing GITHUB_TOKEN for the build-docker job.

### 3.7 `update-pr-overlay` rewiring

Three mechanical edits:

| Field | Before | After |
|---|---|---|
| `needs:` | `[detect-changes, build-docker-pr]` | `[detect-changes, build-docker]` |
| `if:` last clause | `needs.build-docker-pr.result == 'success'` | `needs.build-docker.result == 'success'` |
| Inline comment text | "After build-docker-pr completes…" | "After build-docker completes…" |

The `contains(github.event.pull_request.labels.*.name, 'deploy-env')` clause remains — overlay resolution only fires for `deploy-env` PRs, independent of how `build-docker` decided to push. **C-3 satisfied.**

### 3.8 `pr-validation-complete` rewiring

```yaml
needs: [detect-changes, test-go-libraries, test-go-services, test-ui, build-docker, update-pr-overlay]
```

Drops `build-docker-pr` from `needs:`. The summary-table block drops the "Docker PR Push" row; "Docker Builds" row reports `needs.build-docker.result`. The failure check reduces to a single `DOCKER_RESULT` term. Aggregator preserves its existing `skipped`-tolerant logic.

### 3.9 Observability one-liner

NF-3 requires the step summary to surface whether a push fired. The composite action's existing "Image summary" step already writes `**Pushed**: ${{ inputs.push }}` to `$GITHUB_STEP_SUMMARY`. That's adequate. No new code; verify on first run.

### 3.10 Alternatives considered (Lever 1)

| Option | Why rejected |
|---|---|
| Keep two jobs but de-duplicate via job-level `if: contains(labels, 'deploy-env')` flip | Doesn't solve the wall-clock cost. Both jobs would still spin up runners. |
| Use a re-usable workflow (`workflow_call`) per service | Adds indirection; matrix already handles per-service iteration. |
| Inline `docker/build-push-action@v6` directly in pr-validation.yml, bypassing composite action | Duplicates buildx + login wiring in two places (also in main-publish if it uses the composite action — it does). Loses DRY benefit. PRD §4.2 explicitly accepts composite-action call. |
| Single composite call with three-step internal logic (validate-only step + push step) | Composite action is shared with main-publish; widening its interface for one caller is over-fit. The current "single docker/build-push-action with conditional push" satisfies the PRD literally. |

---

## 4. Lever 2 — Bootstrap slim

### 4.1 Tool choice (resolves PRD §9.2)

**Replace Apache Kafka CLI + `openjdk17-jre-headless` with `rpk` (single static Go binary).**

`rpk` is Redpanda's CLI; it is fully Kafka-protocol-compatible against any Kafka broker (including the Apache 3.x cluster Atlas runs). The relevant subcommands map 1:1:

| cleanup.sh today | After |
|---|---|
| `kafka-topics.sh --bootstrap-server X --list` | `rpk topic list -X brokers=X --format json` |
| `kafka-topics.sh --bootstrap-server X --delete --topic T` | `rpk topic delete -X brokers=X T` |
| `kafka-consumer-groups.sh --bootstrap-server X --list` | `rpk group list -X brokers=X --format json` |
| `kafka-consumer-groups.sh --bootstrap-server X --delete --group G` | `rpk group delete -X brokers=X G` |

Footprint comparison (cold):

| Stack | Image-size contribution | Per-call latency |
|---|---|---|
| Apache Kafka tarball + JRE | ~235 MB (85 + 150) | 2–3 s JVM startup per invocation |
| `rpk` static binary | ~30 MB | <50 ms |
| `kcat` (apk) | ~2 MB | <50 ms |

Choosing `rpk` alone covers every operation cleanup.sh needs. `kcat` is **dropped from the design** — including it would add 2 MB of binary and a second tool to teach the script. The PRD §4.4 listed `kcat` as an addition, but PRD §9.2 explicitly allowed picking `rpk` "if it solves both listing and deletes in one tool"; it does. The acceptance criterion "Contains `kcat` (`/usr/bin/kcat`)" in PRD §10 is therefore **superseded by this design**; the acceptance criterion becomes "Contains `rpk` (`/usr/local/bin/rpk`) and not `openjdk17-jre-headless`." Plan-task will update its task list to reflect this; the audit step records the deviation from PRD §10's literal text.

`rpk` is not in Alpine's apk index. The Dockerfile fetches the official Linux/amd64 release asset:

```dockerfile
ARG RPK_VERSION=24.3.1
RUN curl -fsSL --retry 3 --retry-delay 5 \
        "https://github.com/redpanda-data/redpanda/releases/download/v${RPK_VERSION}/rpk-linux-amd64.zip" \
        -o /tmp/rpk.zip && \
    unzip /tmp/rpk.zip -d /usr/local/bin && \
    chmod +x /usr/local/bin/rpk && \
    rm /tmp/rpk.zip
```

`unzip` is available in Alpine via `apk add unzip`; add it to the build-time-only set. Pin `RPK_VERSION` so cache busts are explicit; plan-task chooses the latest stable at implementation time.

### 4.2 Dockerfile shape (after)

```dockerfile
# syntax=docker/dockerfile:1.4
FROM alpine:3.23

ARG RPK_VERSION=24.3.1

RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    apk add --no-cache \
        bash \
        curl \
        jq \
        postgresql-client \
        redis \
        ca-certificates \
        github-cli \
        kubectl \
        unzip

RUN curl -fsSL --retry 3 --retry-delay 5 \
        "https://github.com/redpanda-data/redpanda/releases/download/v${RPK_VERSION}/rpk-linux-amd64.zip" \
        -o /tmp/rpk.zip && \
    unzip /tmp/rpk.zip -d /usr/local/bin && \
    chmod +x /usr/local/bin/rpk && \
    rm /tmp/rpk.zip

WORKDIR /atlas
COPY scripts/lib.sh /atlas/lib.sh
COPY scripts/bootstrap.sh /atlas/bootstrap.sh
COPY scripts/cleanup.sh /atlas/cleanup.sh
COPY canonical/ /atlas/canonical/

RUN chmod +x /atlas/bootstrap.sh /atlas/cleanup.sh

ENTRYPOINT ["/atlas/bootstrap.sh"]
```

Key shape differences vs today:

1. **Single-stage.** Stage 1 (Kafka tarball) is gone. The `--from=kafka` COPYs are gone.
2. **`# syntax=docker/dockerfile:1.4`** at top — required for cache mounts.
3. **`--mount=type=cache,target=/var/cache/apk,sharing=locked`** on the apk install layer. `sharing=locked` matters because the build is matrix-parallel only at the workflow level (different services run on different runners); within a single runner the lock is uncontended but the flag prevents subtle multi-build interleaving on shared runners.
4. **`openjdk17-jre-headless` removed**, `unzip` added (build-time + tiny — ~200 KB).
5. **`COPY --from=kafka …` removed.** `ENV PATH="/opt/kafka/bin:…"` removed.

### 4.3 cleanup.sh diff (semantic)

Two blocks change. Everything else (psql drop dbs, redis drop keys, ghcr image delete, Pi-hole DNS) is untouched.

**drop-topics:**

```bash
ATLAS_STEP=drop-topics log info "deleting per-env Kafka topics"
rpk topic list -X brokers="$BOOTSTRAP_SERVERS" --format json \
    | jq -r '.topics[].name' \
    | grep -E -- "-${ATLAS_ENV}\$" \
    | xargs -r -n 1 rpk topic delete -X brokers="$BOOTSTRAP_SERVERS"
```

**drop-groups:**

```bash
ATLAS_STEP=drop-groups log info "deleting per-env consumer groups"
# Atlas group names contain spaces (e.g. "Party Quest Service [1756]");
# preserve the xargs -d '\n' delimiter handling — same reason as before.
rpk group list -X brokers="$BOOTSTRAP_SERVERS" --format json \
    | jq -r '.groups[].name' \
    | grep -E -- "\\[${ATLAS_ENV}\\]\$" \
    | xargs -r -d '\n' -n 1 rpk group delete -X brokers="$BOOTSTRAP_SERVERS"
```

C-4 invariants preserved:

- `-d '\n'` delimiter retained — group-name whitespace handling is unchanged.
- `grep -E -- "-${ATLAS_ENV}\$"` and `grep -E -- "\\[${ATLAS_ENV}\\]\$"` regexes unchanged.
- The `xargs -r` flag (don't run command if input is empty) is retained — prevents `rpk delete` with no arg.
- All other cleanup phases (drop-dbs, drop-redis, drop-images, drop-dns) are untouched.

`rpk`'s JSON output schema for `topic list` is `{"topics":[{"name":"…",…}]}` (verified against rpk v24.x docs). The `--format json` flag is part of `rpk`'s output contract; pin `RPK_VERSION` so the schema does not drift silently.

### 4.4 bootstrap.sh & lib.sh

Neither references `kafka-topics.sh`, `kafka-consumer-groups.sh`, `kafka-run-class.sh`, `openjdk`, or `/opt/kafka` (verified by `grep -rn` in the worktree). **No changes.**

### 4.5 Bats tests

Both existing test files (`bootstrap_test.bats`, `cleanup_test.bats`) only assert `require_env` behavior — they don't invoke the kafka path. They pass unchanged on the slimmed image.

New cases added under `cleanup_test.bats`:

1. `cleanup.sh dispatches rpk topic list when reaching drop-topics` — stub `rpk` on PATH to print a known JSON list and assert the script calls `rpk topic delete` with the expected `-${ATLAS_ENV}`-suffixed name.
2. `cleanup.sh dispatches rpk group list with -d '\n' xargs` — stub `rpk group list` to emit a group name containing a space, assert the delete invocation receives the full quoted name in one argv element.
3. `cleanup.sh skips drop-topics with no matches` — stub `rpk topic list` to emit a list with zero matches; assert no `rpk topic delete` invocation (xargs -r behavior).

These are stub-based unit tests, not integration tests against a real broker. The end-to-end correctness check (C-4) is exercised manually on a real PR-env replay during plan-task acceptance — captured in `audit.md` post-implementation per PRD §10.

### 4.6 README update

`services/atlas-pr-bootstrap/README.md` lists runtime deps. Replace the "openjdk17-jre-headless + apache-kafka CLI" lines with "rpk (Kafka admin CLI; vendored as a static binary from redpanda-data/redpanda releases)". Mention the `RPK_VERSION` build arg.

### 4.7 Alternatives considered (Lever 2)

| Option | Why rejected |
|---|---|
| **B (preferred fallback in PRD).** Keep `kafka-topics.sh` + `kafka-consumer-groups.sh` for deletes; replace `openjdk17-jre-headless` with `eclipse-temurin-17-jre-alpine`. | Alpine 3.23 does not ship `eclipse-temurin-17-jre-alpine` in `community/`; would require an Eclipse Adoptium tarball. Net image size still ~150–200 MB. Adds JVM startup latency for every cleanup invocation. No upside vs rpk. |
| **`kafka-python`** as the admin client. | Pulls `python3` runtime (~30 MB) + `kafka-python` package via pip. Total ~50 MB. Python startup ~250 ms. No advantage over rpk; adds a second runtime. |
| **Self-built minimal Go admin binary** using `franz-go` or `IBM/sarama`. | Solves the problem cleanly but introduces a new code asset (a binary built in-repo and shipped). New CI surface for that binary. rpk already exists and is maintained. |
| **`kaf`** (single-binary Kafka CLI by birdayz). | Smaller than rpk (~10 MB) and apk-friendly via a static binary. Less feature-complete around admin ops; lacks consumer-group delete in some versions. rpk's coverage is unambiguous. |
| **Keep `kcat` for listing, add a Go admin client for delete.** | Two tools where one suffices. PRD §9.2 explicitly endorses single-tool rpk if it covers both. |

### 4.8 Risks (Lever 2)

| Risk | Mitigation |
|---|---|
| `rpk` JSON schema drifts between minor versions | Pin `RPK_VERSION` as a build arg. Bumping is a deliberate Dockerfile change. |
| `rpk` Linux/amd64 zip URL changes | URL has been stable since rpk 21.x; absorbed via the official redpanda-data/redpanda release tag scheme. If it breaks, build fails fast in CI — no silent regression. |
| `rpk delete` differs from `kafka-topics.sh --delete` in retention behavior on broker that has `delete.topic.enable=false` | Atlas cluster has `delete.topic.enable=true` (current cleanup.sh works). rpk uses the same `DeleteTopics` admin RPC; identical broker-side semantics. |
| Static Go binary on musl-Alpine | `rpk-linux-amd64` is statically linked; runs on musl. Verified by the Redpanda team's own published Docker images using Alpine bases. |
| Network dependency on github.com release asset at build time | Already a build-time dependency for `archive.apache.org` today (Kafka tarball). Net: same risk profile, smaller artifact. |

---

## 5. Cross-cutting concerns

### 5.1 Verification matrix

| Acceptance criterion | How verified during plan-task |
|---|---|
| PRD §10: single `build-docker` job | `grep -c 'build-docker-pr' .github/workflows/pr-validation.yml` returns 0. |
| PRD §10: unlabeled PR fails on broken Dockerfile | Locally introduce a deliberately broken Dockerfile in a side branch; run `gh workflow run` (or push without label); confirm CI red. Not added to the commit. |
| PRD §10: labeled PR pushes exactly one tag per service | `gh api /users/chronicle20/packages/container/<svc>/<svc>/versions` before/after diff on a fixture PR. |
| PRD §10: unlabeled PR pushes nothing | Same ghcr API check on an unlabeled-PR replay. |
| PRD §10: `update-pr-overlay` produces unchanged `bot/pr-<N>-resolved` shape | `git diff` the bot branch before/after against a saved snapshot. |
| PRD §10: `pr-validation-complete` single Docker row | Inspect the rendered $GITHUB_STEP_SUMMARY on a labeled-PR run. |
| PRD §10: branch protection update | **Not needed** — verified §2 row 9.1. Recorded in audit.md as "no change required". |
| PRD §10: image contains rpk, not openjdk | `docker run --rm <image> rpk version` succeeds; `docker run --rm <image> which java` returns non-zero. |
| PRD §10: cleanup.sh e2e on real env | Deploy bootstrap into atlas-pr-461's replay env and run cleanup against it; capture log lines in audit.md. |
| PRD §10: bats suite passes + new cases | `bats services/atlas-pr-bootstrap/test/` local run; capture output in audit.md. |
| PRD §10: ≥40% wall-clock reduction | Time the docker phase on the pre-merge baseline and on a labeled-PR replay post-merge; report ratio in audit.md. |
| PRD §10: no out-of-scope file changes | `git diff --stat main...task-073-pr-build-job-collapse` — assert paths match `.github/` or `services/atlas-pr-bootstrap/`. |

### 5.2 Rollback plan

Lever 1 is two coordinated workflow edits in a single file plus comment changes in `update-pr-overlay`. Rollback = revert the merge commit; the previous topology is byte-identical to what was there.

Lever 2 changes the Dockerfile and cleanup.sh. If `rpk` proves unreliable in production:
1. Revert `services/atlas-pr-bootstrap/Dockerfile` and `cleanup.sh` to pre-merge.
2. The next bootstrap-image rebuild picks up the reverted Dockerfile.
3. PostDelete-hook ArgoCD Application reconciles the new image on its next sync; no data migration.

Both levers are independently revertable because they touch disjoint paths.

### 5.3 Sequencing within the PR

Plan-task implements in this order:

1. Lever 1 first — pure workflow edit, lowest blast radius (only affects future PRs, not running services).
2. Lever 2 second — Dockerfile + script + bats.
3. Combined Docker build verification on the worktree (CLAUDE.md mandates `docker build -f services/atlas-pr-bootstrap/Dockerfile .` from worktree root after Dockerfile change).
4. PR-level smoke: open the task PR, observe `build-docker` exercise itself on its own change, including the slimmed bootstrap Dockerfile (meta-validation — the PR that changes Docker behavior is also validated by it).

Order matters: if Lever 1 ships first and Lever 2 has a Dockerfile bug, the new `build-docker` catches it cleanly without phantom `build-docker-pr` noise.

---

## 6. Files touched (final inventory)

| Path | Change |
|---|---|
| `.github/workflows/pr-validation.yml` | Collapse build-docker + build-docker-pr; rewire update-pr-overlay and pr-validation-complete |
| `.github/actions/docker-build/action.yml` | **No change** — interface already supports needed inputs (verified §3.2–3.5) |
| `services/atlas-pr-bootstrap/Dockerfile` | Drop kafka stage + JRE; add rpk fetch; add --mount=type=cache |
| `services/atlas-pr-bootstrap/scripts/cleanup.sh` | Rewrite drop-topics and drop-groups blocks to use rpk |
| `services/atlas-pr-bootstrap/scripts/bootstrap.sh` | **No change** (verified §4.4) |
| `services/atlas-pr-bootstrap/scripts/lib.sh` | **No change** (verified §4.4) |
| `services/atlas-pr-bootstrap/test/cleanup_test.bats` | Add 3 new cases for rpk dispatch + xargs preservation (§4.5) |
| `services/atlas-pr-bootstrap/test/bootstrap_test.bats` | **No change** |
| `services/atlas-pr-bootstrap/README.md` | Update runtime-deps section to list rpk |
| `docs/tasks/task-073-pr-build-job-collapse/audit.md` | (Created during code-review phase; documents §5.1 verification evidence and the PRD §10 `kcat` → `rpk` deviation) |
