# PR Docker Build Job Collapse + Bootstrap Image Slim — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Collapse `build-docker` and `build-docker-pr` in `pr-validation.yml` into a single matrix job that always builds and conditionally pushes, and replace the Apache Kafka tarball + JRE in `services/atlas-pr-bootstrap/Dockerfile` with a single static `rpk` binary, rewriting `cleanup.sh` to use it.

**Architecture:** Two decoupled levers in disjoint file trees. Lever 1 edits `.github/`; Lever 2 edits `services/atlas-pr-bootstrap/`. Cache scope key (`${service}-amd64`) is preserved so the first post-merge labeled-PR run hits the existing GHA cache warm. The composite action `.github/actions/docker-build/action.yml` already supports the inputs the merged job needs; no composite changes.

**Tech Stack:** GitHub Actions (`actions/checkout@v4`, `docker/setup-buildx-action@v3`, `docker/login-action@v3`, `docker/build-push-action@v6`), Alpine 3.23, `rpk` v24.3.1 static binary, bash + bats-core for cleanup.sh tests.

---

## File Structure

| File | Responsibility | Tasks touching it |
|---|---|---|
| `.github/workflows/pr-validation.yml` | One Docker matrix job, rewired aggregator + overlay deps | T1, T2, T3 |
| `services/atlas-pr-bootstrap/test/cleanup_test.bats` | Bats unit tests with `rpk`/`psql`/`redis-cli` stubs on PATH | T4, T5, T6 |
| `services/atlas-pr-bootstrap/scripts/cleanup.sh` | Drop-topics + drop-groups blocks switched to `rpk` | T7 |
| `services/atlas-pr-bootstrap/Dockerfile` | Single-stage; apk cache mount; `rpk` fetched at build time; no JRE | T8 |
| `services/atlas-pr-bootstrap/README.md` | Runtime-deps section mentions `rpk` and `RPK_VERSION` build arg | T9 |

`update-pr-overlay`'s overlay-substitution logic is unchanged; only its `needs:`, `if:`, and one comment line change (folded into T2).

---

## Task 1: Collapse the two Docker jobs into a single `build-docker`

**Files:**
- Modify: `.github/workflows/pr-validation.yml:124-213` (the existing `build-docker` and `build-docker-pr` blocks)

The new merged job always builds (Dockerfile validation for every PR — PRD C-1), and conditionally pushes when the PR carries the `deploy-env` label (PRD C-2). Push tag matches today's `build-docker-pr` output (`pr-<N>-<sha>`); local-only tag matches today's `build-docker` (`pr-<N>`). Cache scope `${service}-amd64` is unchanged from today's `build-docker-pr`. The composite action gates ghcr login internally on `push == 'true'`, so we pass credentials unconditionally and never set up a separate `docker/login-action@v3` step in the workflow.

- [ ] **Step 1: Replace lines 124-213 with the merged job**

Open `.github/workflows/pr-validation.yml`. Delete the entire `# Build Docker Images (validation only, no push)` section (line 124 banner through line 213, end of `build-docker-pr` step list). Replace with:

```yaml
  # ============================================
  # Build Docker Images
  #
  # Single matrix job. Always builds (Dockerfile validation on every PR);
  # pushes to ghcr only when the PR carries the `deploy-env` label, since
  # the Argo ApplicationSet filters out PRs without that label and any
  # pushed pr-<N>-<sha> tag would be orphaned. Cache scope key matches
  # the prior `build-docker-pr` job so the first post-collapse labeled-PR
  # run hits the existing GHA cache warm.
  # ============================================
  build-docker:
    name: Build Docker - ${{ matrix.service.name }}
    needs: [detect-changes, test-go-services, test-go-libraries, test-ui]
    if: |
      always() &&
      needs.detect-changes.outputs.docker-services-matrix != '[]' &&
      (needs.test-go-services.result == 'success' || needs.test-go-services.result == 'skipped') &&
      (needs.test-go-libraries.result == 'success' || needs.test-go-libraries.result == 'skipped') &&
      (needs.test-ui.result == 'success' || needs.test-ui.result == 'skipped')
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    strategy:
      fail-fast: false
      matrix:
        service: ${{ fromJson(needs.detect-changes.outputs.docker-services-matrix) }}

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Compute short SHA
        id: sha
        run: |
          SHA="${{ github.event.pull_request.head.sha || github.sha }}"
          echo "short=$(git rev-parse --short=7 "$SHA")" >> $GITHUB_OUTPUT

      - name: Compute push flag and tag
        id: pushtag
        env:
          PR_NUMBER: ${{ github.event.pull_request.number }}
          SHORT_SHA: ${{ steps.sha.outputs.short }}
          IS_PR: ${{ github.event_name == 'pull_request' }}
          HAS_LABEL: ${{ github.event_name == 'pull_request' && contains(github.event.pull_request.labels.*.name, 'deploy-env') }}
        run: |
          set -euo pipefail
          if [ "$HAS_LABEL" = "true" ]; then
            echo "push=true" >> "$GITHUB_OUTPUT"
            echo "tag=pr-${PR_NUMBER}-${SHORT_SHA}" >> "$GITHUB_OUTPUT"
          else
            echo "push=false" >> "$GITHUB_OUTPUT"
            if [ "$IS_PR" = "true" ]; then
              echo "tag=pr-${PR_NUMBER}" >> "$GITHUB_OUTPUT"
            else
              echo "tag=pr-dispatch" >> "$GITHUB_OUTPUT"
            fi
          fi

      - name: Build Docker image
        uses: ./.github/actions/docker-build
        with:
          context: ${{ matrix.service.docker_context }}
          dockerfile: ${{ matrix.service.path }}/Dockerfile
          image-name: ${{ matrix.service.docker_image }}
          tags: ${{ steps.pushtag.outputs.tag }}
          push: ${{ steps.pushtag.outputs.push }}
          registry-username: ${{ github.actor }}
          registry-password: ${{ secrets.GHCR_TOKEN }}
          cache-from: type=gha,scope=${{ matrix.service.name }}-amd64
          cache-to: type=gha,mode=max,scope=${{ matrix.service.name }}-amd64
```

Do not commit yet. T2 and T3 in the same file follow.

- [ ] **Step 2: Verify only one Docker job remains**

Run: `grep -nE '^  (build-docker|build-docker-pr):' .github/workflows/pr-validation.yml`
Expected output: exactly one line, `  build-docker:`.

Also run: `grep -c 'build-docker-pr' .github/workflows/pr-validation.yml`
Expected: a small non-zero number (T2 and T3 still need to clean up `update-pr-overlay` and `pr-validation-complete`).

---

## Task 2: Rewire `update-pr-overlay` to depend on `build-docker`

**Files:**
- Modify: `.github/workflows/pr-validation.yml` — the `update-pr-overlay` job (currently lines 239-356) and its preceding banner comment

The overlay job's `needs:` and `if:` clauses still reference `build-docker-pr`; rename to `build-docker`. The banner comment ("After build-docker-pr completes...") is updated. The `contains(... 'deploy-env')` clause is preserved — overlay resolution remains label-gated, independent of how `build-docker` decided to push.

- [ ] **Step 1: Update the `needs:` array**

In the `update-pr-overlay` job, change:

```yaml
    needs: [detect-changes, build-docker-pr]
```

to:

```yaml
    needs: [detect-changes, build-docker]
```

- [ ] **Step 2: Update the `if:` `result` clause**

Change:

```yaml
      needs.build-docker-pr.result == 'success'
```

to:

```yaml
      needs.build-docker.result == 'success'
```

- [ ] **Step 3: Update the banner comment**

In the banner just above `update-pr-overlay:` (around line 215-238), change every occurrence of `build-docker-pr` to `build-docker`. Specifically:

- `# After build-docker-pr completes, create/update a derived branch` → `# After build-docker completes, create/update a derived branch`

- [ ] **Step 4: Update the inline comment inside the "Bump image tags" step**

Inside `update-pr-overlay`'s `- name: Bump image tags for built services` step (around line 310-315), change:

```bash
          # The detect-changes matrix is the exact list of services
          # that build-docker-pr pushed pr-<N>-<sha> tags for. For each,
```

to:

```bash
          # The detect-changes matrix is the exact list of services
          # that build-docker pushed pr-<N>-<sha> tags for. For each,
```

- [ ] **Step 5: Verify**

Run: `grep -n 'build-docker-pr' .github/workflows/pr-validation.yml`
Expected: only matches remaining are inside the `pr-validation-complete` job (cleaned up in T3). No matches in `update-pr-overlay` or its banner.

---

## Task 3: Collapse the Docker rows in `pr-validation-complete`

**Files:**
- Modify: `.github/workflows/pr-validation.yml` — the `pr-validation-complete` job (currently lines 358-401)

Drop `build-docker-pr` from `needs:`. Drop the `DOCKER_PR_RESULT` variable, the "Docker PR Push" summary row, and the `DOCKER_PR_RESULT` term from the failure check. The single `DOCKER_RESULT` term covers both validation and push outcomes (one job now reports both). Aggregator's `skipped`-tolerant logic is unchanged.

- [ ] **Step 1: Update the `needs:` array**

Change:

```yaml
    needs: [detect-changes, test-go-libraries, test-go-services, test-ui, build-docker, build-docker-pr, update-pr-overlay]
```

to:

```yaml
    needs: [detect-changes, test-go-libraries, test-go-services, test-ui, build-docker, update-pr-overlay]
```

- [ ] **Step 2: Drop the `DOCKER_PR_RESULT` capture**

Delete the line:

```bash
          DOCKER_PR_RESULT="${{ needs.build-docker-pr.result }}"
```

- [ ] **Step 3: Drop the Docker PR Push summary row**

Delete the line:

```bash
          echo "| Docker PR Push | $DOCKER_PR_RESULT |" >> $GITHUB_STEP_SUMMARY
```

- [ ] **Step 4: Drop the `DOCKER_PR_RESULT` term from the failure check**

Change:

```bash
          if [ "$LIBS_RESULT" == "failure" ] || [ "$SERVICES_RESULT" == "failure" ] || [ "$UI_RESULT" == "failure" ] || [ "$DOCKER_RESULT" == "failure" ] || [ "$DOCKER_PR_RESULT" == "failure" ] || [ "$OVERLAY_RESULT" == "failure" ]; then
```

to:

```bash
          if [ "$LIBS_RESULT" == "failure" ] || [ "$SERVICES_RESULT" == "failure" ] || [ "$UI_RESULT" == "failure" ] || [ "$DOCKER_RESULT" == "failure" ] || [ "$OVERLAY_RESULT" == "failure" ]; then
```

- [ ] **Step 5: Verify no `build-docker-pr` reference remains anywhere**

Run: `grep -n 'build-docker-pr' .github/workflows/pr-validation.yml`
Expected: zero matches.

- [ ] **Step 6: Validate workflow YAML parses**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/pr-validation.yml'))" && echo OK`
Expected: prints `OK` with no Python traceback.

- [ ] **Step 7: Commit Lever 1**

```bash
git add .github/workflows/pr-validation.yml
git commit -m "ci(pr-validation): collapse build-docker + build-docker-pr into one job

Single matrix Docker job always builds (Dockerfile validation on every
PR) and conditionally pushes when the PR carries the 'deploy-env' label.
Push tag and cache scope key unchanged from the prior build-docker-pr.

- update-pr-overlay.needs and if rewired to build-docker.
- pr-validation-complete drops the Docker PR Push row; single
  DOCKER_RESULT term covers both validation and push.

Task: task-073"
```

---

## Task 4: Bats setup helper — stubs for `rpk`, `psql`, `redis-cli`, and `gh`

**Files:**
- Modify: `services/atlas-pr-bootstrap/test/cleanup_test.bats`

`cleanup.sh` shells out to `psql`, `rpk` (after our changes), `redis-cli`, `gh`, and `curl` (Pi-hole). Bats hosts won't have a real broker, DB, or registry; we stub each binary onto a per-test PATH dir. The stubs record argv to a log file so tests can assert which calls were made and with what arguments. This task installs the harness; T5 and T6 add the actual assertions; T7 swaps `cleanup.sh` to call `rpk`.

The existing `cleanup_test.bats` has a `setup()` that only computes `PROJECT_ROOT`. We extend it with a shared `make_stubs` helper that the new cases call.

- [ ] **Step 1: Replace `cleanup_test.bats` with the harness scaffold**

Overwrite the file contents with:

```bash
#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    STUB_BIN="$BATS_TEST_TMPDIR/bin"
    STUB_LOG="$BATS_TEST_TMPDIR/calls.log"
    mkdir -p "$STUB_BIN"
}

# make_stubs writes shell-script stubs for every external binary cleanup.sh
# invokes. Each stub appends its argv to "$STUB_LOG" and exits 0 unless the
# caller passes per-binary overrides.
#
# Args (optional, in order):
#   $1 — topic_list_json (default: empty topic list)
#   $2 — group_list_json (default: empty group list)
make_stubs() {
    local topic_json="${1:-{\"topics\":[]\}}"
    local group_json="${2:-{\"groups\":[]\}}"
    printf '%s\n' "$topic_json" > "$BATS_TEST_TMPDIR/topic_list.json"
    printf '%s\n' "$group_json" > "$BATS_TEST_TMPDIR/group_list.json"

    cat > "$STUB_BIN/rpk" <<'EOF'
#!/usr/bin/env bash
echo "rpk $*" >> "$STUB_LOG"
if [ "$1" = "topic" ] && [ "$2" = "list" ]; then
    cat "$BATS_TEST_TMPDIR/topic_list.json"
elif [ "$1" = "group" ] && [ "$2" = "list" ]; then
    cat "$BATS_TEST_TMPDIR/group_list.json"
fi
exit 0
EOF
    cat > "$STUB_BIN/psql" <<'EOF'
#!/usr/bin/env bash
echo "psql $*" >> "$STUB_LOG"
exit 0
EOF
    cat > "$STUB_BIN/redis-cli" <<'EOF'
#!/usr/bin/env bash
echo "redis-cli $*" >> "$STUB_LOG"
# When invoked with --scan, emit no keys so the xargs delete is a no-op.
exit 0
EOF
    cat > "$STUB_BIN/gh" <<'EOF'
#!/usr/bin/env bash
echo "gh $*" >> "$STUB_LOG"
exit 0
EOF
    chmod +x "$STUB_BIN"/*
}

# run_cleanup runs cleanup.sh with the standard test env vars and the
# stubs on PATH. Caller may set ATLAS_ENV before calling; default is "test".
run_cleanup() {
    PATH="$STUB_BIN:$PATH" \
    STUB_LOG="$STUB_LOG" \
    BATS_TEST_TMPDIR="$BATS_TEST_TMPDIR" \
    ATLAS_ENV="${ATLAS_ENV:-test}" \
    DB_HOST=h DB_PORT=5432 DB_USER=u DB_PASSWORD=p \
    ATLAS_DB_NAMES="foo bar" \
    BOOTSTRAP_SERVERS=kafka:9093 \
    REDIS_URL=redis:6379 \
    PR_NUMBER=99 \
    bash "$PROJECT_ROOT/scripts/cleanup.sh"
}

@test "cleanup.sh fails without ATLAS_ENV" {
    run env -u ATLAS_ENV bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_ENV"* ]]
}

@test "cleanup.sh fails without ATLAS_DB_NAMES" {
    run env ATLAS_ENV=test DB_HOST=h DB_PORT=5432 DB_USER=u DB_PASSWORD=p \
        BOOTSTRAP_SERVERS=k REDIS_URL=r PR_NUMBER=1 \
        -u ATLAS_DB_NAMES bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_DB_NAMES"* ]]
}
```

- [ ] **Step 2: Run bats to confirm the existing two cases still pass**

Run: `bats services/atlas-pr-bootstrap/test/cleanup_test.bats`
Expected:
```
✓ cleanup.sh fails without ATLAS_ENV
✓ cleanup.sh fails without ATLAS_DB_NAMES
2 tests, 0 failures
```

(If `bats` is not on the host, install via `apk add bats` / `brew install bats-core` / Ubuntu `apt-get install bats`. Plan-task assumes `bats-core` is available in the dev env.)

- [ ] **Step 3: Commit the bats harness scaffold**

```bash
git add services/atlas-pr-bootstrap/test/cleanup_test.bats
git commit -m "test(atlas-pr-bootstrap): add bats stub harness for cleanup.sh

setup() helper writes PATH-overriding stubs for rpk, psql, redis-cli, gh.
Stub argv is recorded to STUB_LOG so subsequent cases can assert call
shapes. Stubs land on PATH only; no real binary is required to run the
suite.

Task: task-073"
```

---

## Task 5: Bats case — `cleanup.sh` dispatches `rpk topic delete` only for `-${ATLAS_ENV}` topics

**Files:**
- Modify: `services/atlas-pr-bootstrap/test/cleanup_test.bats`

The first new case asserts the topic-list grep filter and delete dispatch:
- `rpk topic list` is invoked exactly once.
- Topics whose names end with `-test` (where `test` = `ATLAS_ENV`) are passed to `rpk topic delete`, one at a time.
- Topics whose names do NOT end with `-test` are NOT passed to delete.

This case will FAIL initially because `cleanup.sh` still calls `kafka-topics.sh`, not `rpk`. T7 makes it pass.

- [ ] **Step 1: Append the case to `cleanup_test.bats`**

Append at the end of the file:

```bash
@test "cleanup.sh deletes only -ATLAS_ENV-suffixed topics via rpk" {
    make_stubs '{"topics":[{"name":"foo-test"},{"name":"bar"},{"name":"baz-test"}]}'
    run run_cleanup
    [ "$status" -eq 0 ]

    # rpk topic list was invoked once
    grep -c '^rpk topic list ' "$STUB_LOG" >/dev/null
    [ "$(grep -c '^rpk topic list ' "$STUB_LOG")" -eq 1 ]

    # rpk topic delete was invoked for foo-test and baz-test, and not for bar
    grep -F 'rpk topic delete' "$STUB_LOG" | grep -F 'foo-test'
    grep -F 'rpk topic delete' "$STUB_LOG" | grep -F 'baz-test'
    if grep -F 'rpk topic delete' "$STUB_LOG" | grep -wF 'bar'; then
        echo "ERROR: topic 'bar' (no ATLAS_ENV suffix) was deleted" >&2
        return 1
    fi
}
```

- [ ] **Step 2: Run the case and confirm it FAILS**

Run: `bats services/atlas-pr-bootstrap/test/cleanup_test.bats -f "deletes only"`
Expected: failure. The exact failure mode depends on whether `kafka-topics.sh` is on the bats host:
- If absent: cleanup.sh exits non-zero with `kafka-topics.sh: command not found` → assertion `[ "$status" -eq 0 ]` fails.
- If present: still fails because the STUB_LOG has zero `rpk topic list` lines.

Either way the test must be red before T7. Do **not** commit yet — the new failing case lives unstaged on disk until T7 makes it green.

---

## Task 6: Bats case — `cleanup.sh` dispatches `rpk group delete` preserving group names with spaces

**Files:**
- Modify: `services/atlas-pr-bootstrap/test/cleanup_test.bats`

The second new case asserts the xargs `-d '\n'` invariant (PRD C-4): Atlas consumer-group names contain spaces (e.g. `"Party Quest Service [1756]"`). Default xargs would word-split into multiple `--group` args; `-d '\n'` keeps each name intact. Stub `rpk group list` to emit a group whose name contains spaces, assert the deletion arg has the full quoted name.

- [ ] **Step 1: Append the case to `cleanup_test.bats`**

Append at the end of the file:

```bash
@test "cleanup.sh deletes consumer groups with spaces in their names" {
    # Group list has one name matching [test] suffix (with spaces) and
    # one not matching. Only the matching one should be deleted.
    make_stubs \
        '{"topics":[]}' \
        '{"groups":[{"name":"Party Quest Service [test]"},{"name":"Other [other]"}]}'
    run run_cleanup
    [ "$status" -eq 0 ]

    # rpk group list invoked once
    [ "$(grep -c '^rpk group list ' "$STUB_LOG")" -eq 1 ]

    # rpk group delete was called for the spaced name as one argument
    grep -F 'rpk group delete' "$STUB_LOG" | grep -F 'Party Quest Service [test]'

    # The other-env group must not be deleted
    if grep -F 'rpk group delete' "$STUB_LOG" | grep -F 'Other [other]'; then
        echo "ERROR: group with non-matching env suffix was deleted" >&2
        return 1
    fi
}
```

- [ ] **Step 2: Run the case and confirm it FAILS**

Run: `bats services/atlas-pr-bootstrap/test/cleanup_test.bats -f "spaces in their names"`
Expected: failure for the same reason as T5 — cleanup.sh still calls `kafka-consumer-groups.sh`, not `rpk group ...`. Do not commit yet.

---

## Task 7: Switch `cleanup.sh` drop-topics + drop-groups to `rpk`

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/cleanup.sh:44-58`

Replace the two Kafka shell-script invocations with `rpk` calls. The JSON output is parsed by `jq`; the topic JSON is `{"topics":[{"name":"…"}…]}`, the group JSON is `{"groups":[{"name":"…"}…]}` (rpk v24.x contract). All other phases (drop-dbs, drop-redis, drop-images, drop-dns) are untouched. PRD C-4 invariants preserved:
- `-d '\n'` retained on the groups xargs (group names contain spaces).
- `grep -E -- "-${ATLAS_ENV}\$"` and `grep -E -- "\\[${ATLAS_ENV}\\]\$"` regexes unchanged.
- `xargs -r` retained (no-op on empty input).

- [ ] **Step 1: Replace the drop-topics block**

Replace lines 44-47 (the `ATLAS_STEP=drop-topics` comment through the `xargs -r -n 1 kafka-topics.sh ...` line) with:

```bash
ATLAS_STEP=drop-topics log info "deleting per-env Kafka topics"
rpk topic list -X brokers="$BOOTSTRAP_SERVERS" --format json \
    | jq -r '.topics[].name' \
    | grep -E -- "-${ATLAS_ENV}\$" \
    | xargs -r -n 1 rpk topic delete -X brokers="$BOOTSTRAP_SERVERS"
```

- [ ] **Step 2: Replace the drop-groups block**

Replace lines 49-58 (the `ATLAS_STEP=drop-groups` comment block through the `kafka-consumer-groups.sh ... --group` line) with:

```bash
ATLAS_STEP=drop-groups log info "deleting per-env consumer groups"
# Atlas consumer-group names contain spaces (e.g. "Party Quest Service [1756]",
# "Channel Service - %s [1756]"). xargs's default delimiter is whitespace, which
# would word-split each group name into 3-5 separate `--group` invocations and
# nothing would match. -d '\n' restricts splitting to newlines, so each group
# is passed intact. Observed 2026-05-16 cleaning up atlas-pr-461's leftover
# 1756-suffixed groups after the PostDelete hook had previously failed.
rpk group list -X brokers="$BOOTSTRAP_SERVERS" --format json \
    | jq -r '.groups[].name' \
    | grep -E -- "\\[${ATLAS_ENV}\\]\$" \
    | xargs -r -d '\n' -n 1 rpk group delete -X brokers="$BOOTSTRAP_SERVERS"
```

- [ ] **Step 3: Verify no kafka-* shell tool reference remains in cleanup.sh**

Run: `grep -nE 'kafka-topics|kafka-consumer-groups|kafka-run-class' services/atlas-pr-bootstrap/scripts/cleanup.sh`
Expected: zero matches.

- [ ] **Step 4: Re-run the full bats suite — all four cases must PASS**

Run: `bats services/atlas-pr-bootstrap/test/cleanup_test.bats`
Expected:
```
✓ cleanup.sh fails without ATLAS_ENV
✓ cleanup.sh fails without ATLAS_DB_NAMES
✓ cleanup.sh deletes only -ATLAS_ENV-suffixed topics via rpk
✓ cleanup.sh deletes consumer groups with spaces in their names
4 tests, 0 failures
```

- [ ] **Step 5: Add a third case — empty topic list yields no delete invocation**

This case proves the `xargs -r` "no input → no invocation" invariant survives the rewrite. Append to `cleanup_test.bats`:

```bash
@test "cleanup.sh skips rpk topic delete when no topic matches" {
    make_stubs '{"topics":[{"name":"prod-foo"},{"name":"prod-bar"}]}'
    run run_cleanup
    [ "$status" -eq 0 ]
    [ "$(grep -c '^rpk topic list ' "$STUB_LOG")" -eq 1 ]
    # No delete because no topic name ends with -test
    if grep -F 'rpk topic delete' "$STUB_LOG"; then
        echo "ERROR: rpk topic delete invoked despite no matching topics" >&2
        return 1
    fi
}
```

- [ ] **Step 6: Re-run the bats suite — all five cases must PASS**

Run: `bats services/atlas-pr-bootstrap/test/cleanup_test.bats`
Expected:
```
✓ cleanup.sh fails without ATLAS_ENV
✓ cleanup.sh fails without ATLAS_DB_NAMES
✓ cleanup.sh deletes only -ATLAS_ENV-suffixed topics via rpk
✓ cleanup.sh deletes consumer groups with spaces in their names
✓ cleanup.sh skips rpk topic delete when no topic matches
5 tests, 0 failures
```

- [ ] **Step 7: Commit cleanup.sh + new bats cases together**

```bash
git add services/atlas-pr-bootstrap/scripts/cleanup.sh services/atlas-pr-bootstrap/test/cleanup_test.bats
git commit -m "feat(atlas-pr-bootstrap): switch cleanup.sh drop-topics/groups to rpk

Replace kafka-topics.sh / kafka-consumer-groups.sh shell scripts with a
single static rpk binary. JSON output is parsed by jq; grep regexes and
the xargs -d '\\n' delimiter on the groups path are unchanged so
PRD C-4 invariants (env-suffix filter, group-name space handling) are
preserved.

Bats cases assert:
- topic delete called only for -ATLAS_ENV-suffixed names
- group delete preserves names containing spaces
- topic delete skipped when no topic matches

Task: task-073"
```

---

## Task 8: Rewrite `services/atlas-pr-bootstrap/Dockerfile`

**Files:**
- Modify: `services/atlas-pr-bootstrap/Dockerfile` (replace entire contents)

Single-stage Alpine 3.23. Adds `# syntax=docker/dockerfile:1.4` so buildkit honors `RUN --mount=type=cache`. Drops `openjdk17-jre-headless` and the entire Stage 1 (Kafka tarball). Fetches `rpk` v24.3.1 from the official Redpanda release asset, unzipped to `/usr/local/bin/rpk`. The `unzip` apk package is needed at build time. The apk install layer uses `--mount=type=cache,target=/var/cache/apk,sharing=locked` so repeated builds reuse downloaded `.apk` files.

The `ENV PATH="/opt/kafka/bin:${PATH}"` line is removed (no `/opt/kafka` exists anymore).

- [ ] **Step 1: Overwrite the Dockerfile**

Replace the entire file with:

```dockerfile
# syntax=docker/dockerfile:1.4
# Atlas PR-env bootstrap/cleanup image. Two entrypoints share this image:
#   /atlas/bootstrap.sh (PostSync) and /atlas/cleanup.sh (PostDelete).
# Single-stage Alpine. rpk (Redpanda CLI; Kafka admin protocol compatible)
# replaces the Apache Kafka tarball + JRE — single static Go binary,
# ~30 MB vs. ~235 MB and no JVM startup latency per cleanup invocation.
FROM alpine:3.23

ARG RPK_VERSION=24.3.1

RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    apk add \
        bash \
        curl \
        jq \
        postgresql-client \
        redis \
        ca-certificates \
        github-cli \
        kubectl \
        unzip

# Fetch rpk static binary. Pin version so JSON output schema and admin
# RPC semantics are deterministic; bumping is a deliberate Dockerfile
# change. Cache busts when RPK_VERSION ARG changes.
RUN curl -fsSL --retry 3 --retry-delay 5 \
        "https://github.com/redpanda-data/redpanda/releases/download/v${RPK_VERSION}/rpk-linux-amd64.zip" \
        -o /tmp/rpk.zip && \
    unzip /tmp/rpk.zip -d /usr/local/bin && \
    chmod +x /usr/local/bin/rpk && \
    rm /tmp/rpk.zip

WORKDIR /atlas
COPY services/atlas-pr-bootstrap/scripts/lib.sh /atlas/lib.sh
COPY services/atlas-pr-bootstrap/scripts/bootstrap.sh /atlas/bootstrap.sh
COPY services/atlas-pr-bootstrap/scripts/cleanup.sh /atlas/cleanup.sh
COPY services/atlas-pr-bootstrap/canonical/ /atlas/canonical/

RUN chmod +x /atlas/bootstrap.sh /atlas/cleanup.sh

ENTRYPOINT ["/atlas/bootstrap.sh"]
```

NOTE on the `COPY` paths: the existing Dockerfile uses `COPY scripts/...` (paths relative to the build context). The `detect-changes` action sets `docker_context` per service; for `atlas-pr-bootstrap` that context is `services/atlas-pr-bootstrap/` (verified by checking `.github/actions/detect-changes`). Keep `COPY scripts/...` to match the existing context. Override the snippet above accordingly:

```dockerfile
WORKDIR /atlas
COPY scripts/lib.sh /atlas/lib.sh
COPY scripts/bootstrap.sh /atlas/bootstrap.sh
COPY scripts/cleanup.sh /atlas/cleanup.sh
COPY canonical/ /atlas/canonical/
```

(The original Dockerfile already uses these short paths; preserve them.)

Final Dockerfile to write:

```dockerfile
# syntax=docker/dockerfile:1.4
# Atlas PR-env bootstrap/cleanup image. Two entrypoints share this image:
#   /atlas/bootstrap.sh (PostSync) and /atlas/cleanup.sh (PostDelete).
# Single-stage Alpine. rpk (Redpanda CLI; Kafka admin protocol compatible)
# replaces the Apache Kafka tarball + JRE — single static Go binary,
# ~30 MB vs. ~235 MB and no JVM startup latency per cleanup invocation.
FROM alpine:3.23

ARG RPK_VERSION=24.3.1

RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    apk add \
        bash \
        curl \
        jq \
        postgresql-client \
        redis \
        ca-certificates \
        github-cli \
        kubectl \
        unzip

# Fetch rpk static binary. Pin version so JSON output schema and admin
# RPC semantics are deterministic; bumping is a deliberate Dockerfile
# change.
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

- [ ] **Step 2: Verify no Kafka or JRE references remain**

Run: `grep -nE 'kafka|openjdk|/opt/kafka' services/atlas-pr-bootstrap/Dockerfile`
Expected: zero matches.

- [ ] **Step 3: Docker build from the worktree root (mandatory per CLAUDE.md)**

The composite action computes the path as `${docker_context}/Dockerfile`. For local validation we run with the worktree root as the build context (the CLAUDE.md prescription):

```bash
docker build -f services/atlas-pr-bootstrap/Dockerfile services/atlas-pr-bootstrap
```

Expected: build succeeds in ≤4 min cold (PRD P-2). Note the URL for `rpk-linux-amd64.zip` must be reachable from the build host.

- [ ] **Step 4: Verify the built image contains `rpk` and not `java`**

```bash
docker run --rm $(docker build -q -f services/atlas-pr-bootstrap/Dockerfile services/atlas-pr-bootstrap) sh -c 'rpk version && (which java && exit 1 || exit 0)'
```

Expected: prints an `rpk version v24.3.x` line and exits 0 (no `java` on PATH).

- [ ] **Step 5: Commit the Dockerfile rewrite**

```bash
git add services/atlas-pr-bootstrap/Dockerfile
git commit -m "feat(atlas-pr-bootstrap): replace Kafka tarball + JRE with rpk static binary

Single-stage Alpine. rpk (Redpanda CLI; Kafka admin protocol compatible)
covers topic and consumer-group list/delete in cleanup.sh — same RPCs
as the Apache Kafka CLIs we're replacing.

- # syntax=docker/dockerfile:1.4 at top so buildkit honors cache mounts
- --mount=type=cache on the apk install layer
- openjdk17-jre-headless and the Stage 1 Kafka tarball stage removed
- rpk fetched from official redpanda-data/redpanda release asset
- RPK_VERSION pin so JSON output schema is deterministic

Task: task-073"
```

---

## Task 9: Update `services/atlas-pr-bootstrap/README.md`

**Files:**
- Modify: `services/atlas-pr-bootstrap/README.md`

Add a short "Runtime dependencies" section listing the apk packages and the vendored `rpk` binary. Mention the `RPK_VERSION` build arg so the next contributor knows how to bump it.

- [ ] **Step 1: Append a runtime-deps section**

Append to the end of the file:

```markdown

## Runtime dependencies

The image is single-stage Alpine 3.23 and contains:

- apk: `bash`, `curl`, `jq`, `postgresql-client`, `redis`, `ca-certificates`, `github-cli`, `kubectl`, `unzip` (build-time only)
- `rpk` (Redpanda CLI; Kafka admin protocol compatible) — vendored as a
  static binary from `redpanda-data/redpanda` releases. Used by
  `cleanup.sh` for topic and consumer-group list/delete. Pin via the
  `RPK_VERSION` Dockerfile build arg.

Earlier revisions of the image baked the Apache Kafka tarball + an
OpenJDK 17 JRE for the same operations. `rpk` is a single ~30 MB static
Go binary and removes the JVM startup latency from every cleanup call.
```

- [ ] **Step 2: Commit the README update**

```bash
git add services/atlas-pr-bootstrap/README.md
git commit -m "docs(atlas-pr-bootstrap): document rpk and apk runtime deps

Task: task-073"
```

---

## Task 10: Final verification (no commit)

This is the verify-before-PR checklist. Each step is a check, not a code change.

- [ ] **Step 1: `grep` for stale `build-docker-pr` references**

Run: `grep -rn 'build-docker-pr' .github/ services/atlas-pr-bootstrap/`
Expected: zero matches.

- [ ] **Step 2: `grep` for stale Apache Kafka tool references**

Run: `grep -rnE 'kafka-topics|kafka-consumer-groups|kafka-run-class|openjdk' services/atlas-pr-bootstrap/`
Expected: zero matches.

- [ ] **Step 3: Bats suite final pass**

Run: `bats services/atlas-pr-bootstrap/test/`
Expected:
```
✓ bootstrap_test.bats: …
✓ cleanup.sh fails without ATLAS_ENV
✓ cleanup.sh fails without ATLAS_DB_NAMES
✓ cleanup.sh deletes only -ATLAS_ENV-suffixed topics via rpk
✓ cleanup.sh deletes consumer groups with spaces in their names
✓ cleanup.sh skips rpk topic delete when no topic matches
```
All cases pass.

- [ ] **Step 4: Docker build final pass**

Run: `docker build -f services/atlas-pr-bootstrap/Dockerfile services/atlas-pr-bootstrap`
Expected: build succeeds; final image layer count and ENV summary contain no `/opt/kafka` or `JAVA_HOME` references.

- [ ] **Step 5: Workflow YAML final parse**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/pr-validation.yml'))" && echo OK`
Expected: prints `OK`.

- [ ] **Step 6: Scope confirmation**

Run: `git diff --stat main...HEAD`
Expected: only paths under `.github/workflows/pr-validation.yml`, `services/atlas-pr-bootstrap/` (plus the planning docs).

- [ ] **Step 7: Hand off to code review**

Per CLAUDE.md "Code Review Before PR": run `superpowers:requesting-code-review`. It will dispatch `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no frontend changes). Capture findings under `docs/tasks/task-073-pr-build-job-collapse/audit.md`; include the explicit deviation from PRD §10's literal `kcat` text (we ship `rpk` only, per design §4.1).

---

## Spec coverage map

| PRD §        | Covered by |
|---|---|
| §4.1 Single matrix Docker job | T1 |
| §4.2 Conditional build/push within the job | T1 |
| §4.3 Downstream wiring (`update-pr-overlay`, `pr-validation-complete`) | T2, T3 |
| §4.4 Slimmed `atlas-pr-bootstrap` Dockerfile | T8 |
| §4.5 Bats / shell test re-verification | T4, T5, T6, T7 (steps 4-6) |
| §5.1 Workflow contract surface | T1, T2, T3 |
| §5.3 Bootstrap image contract | T8 (rpk path, RPK_VERSION arg), T9 (README) |
| §8.1 P-1, P-2, P-3 performance | T8 step 3 (timing); audit step (P-1 measurement) |
| §8.2 C-1, C-2, C-3, C-4 correctness | T1 (C-1/C-2), T2/T3 (C-3), T5/T6/T7 step 5 (C-4) |
| §8.3 Observability (Pushed: yes/no in step summary) | T1 (composite action already prints `**Pushed**: ${{ inputs.push }}`; no code change) |
| §10 Acceptance criteria | T1–T8 implement; T10 verifies; the `kcat`→`rpk` deviation is documented in audit at code-review time |

## Notes on rigid skill discipline (TDD)

T5 and T6 deliberately leave failing tests on disk uncommitted; T7 makes them green and commits the cleanup.sh + bats cases together. This is the TDD rhythm — red, green, refactor (no refactor needed here). T8 and T9 are not TDD-shaped (they're a Dockerfile and a README); validation comes from `docker build` and `grep` checks.
