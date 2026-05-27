# PR-Env Teardown Regressions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the six independent fixes uncovered by PR 544's teardown incident — rpk jq schema, missing `sweep-orphans.sh` in image, sweep-orphans Kafka phases still on Kafka CLI, duplicated env-var defaults, abort-first failure policy, and the literal-`%s` consumer-group bug — plus regression tests pinning rpk output to committed fixtures.

**Architecture:** A shared bash mini-framework in `services/atlas-pr-bootstrap/scripts/lib.sh` (`record_error`, `run_phase`, `summarize_phases`) lets both `cleanup.sh` and `sweep-orphans.sh` run every phase regardless of any single failure. rpk JSON schemas are pinned to committed fixtures replayed by bats stubs. `consumergroup.Resolve` gains variadic args so atlas-channel / atlas-login interpolate the channel ID at runtime against the PR-overlay-supplied env var, not at compile time against the default. A new `gen-cleanup-env.sh` derives `ATLAS_SERVICES` from `.github/config/services.json` into a cluster-infra coordination artifact; `postdelete-cleanup.yaml` switches from inline `env:` to `envFrom: configMapRef:`.

**Tech Stack:** bash + bats; jq; rpk 24.3.1; Go (`libs/atlas-kafka/consumergroup`); kustomize; Docker buildx; GitHub Actions workflow.

---

## File Structure Overview

| Path | Type | Touched by tasks |
|---|---|---|
| `services/atlas-pr-bootstrap/scripts/lib.sh` | modify | 1, 2 |
| `services/atlas-pr-bootstrap/test/lib_test.bats` | modify | 1 |
| `services/atlas-pr-bootstrap/test/fixtures/rpk-topic-list.json` | create | 2 |
| `services/atlas-pr-bootstrap/test/fixtures/rpk-group-list.json` | create | 2 |
| `services/atlas-pr-bootstrap/test/fixtures/README.md` | create | 2 |
| `services/atlas-pr-bootstrap/scripts/cleanup.sh` | rewrite | 3 |
| `services/atlas-pr-bootstrap/test/cleanup_test.bats` | modify | 3 |
| `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` | modify | 4 |
| `services/atlas-pr-bootstrap/test/sweep_test.bats` | rewrite | 4 |
| `services/atlas-pr-bootstrap/Dockerfile` | modify | 5 |
| `services/atlas-pr-bootstrap/test/dockerfile_test.bats` | create | 5 |
| `libs/atlas-kafka/consumergroup/resolver.go` | modify | 6 |
| `libs/atlas-kafka/consumergroup/resolver_test.go` | modify | 6 |
| `services/atlas-channel/atlas.com/channel/main.go` | modify (1 line) | 7 |
| `services/atlas-login/atlas.com/login/main.go` | modify (1 line) | 7 |
| `deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh` | create | 8 |
| `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` | create (generated) | 8 |
| `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml` | modify | 9 |
| `deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh` | modify (comment) | 9 |
| `.github/workflows/pr-validation.yml` | modify | 10 |
| `docs/runbooks/ephemeral-pr-deployments.md` | modify | 11 |

---

## Task 1 — Add phase-runner helpers + unit tests to lib.sh

**Bugs addressed:** 5 (try-all failure policy) — also unblocks Tasks 3 & 4.

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/lib.sh`
- Modify: `services/atlas-pr-bootstrap/test/lib_test.bats`

The current header is `set -euo pipefail`. We need `lib.sh` to keep
`set -uo pipefail` semantics (catch unset vars + pipe failures)
without `-e` leaking into a sourcing script's top scope, because
`cleanup.sh` and `sweep-orphans.sh` source `lib.sh` and will drop
`-e` themselves. `lib.sh` is sourced, not executed, so the `set` in
its header applies to the sourcer's shell. We change it to
`set -uo pipefail` here too — there is no functional consumer of
`-e` inside `lib.sh`'s own function bodies.

- [ ] **Step 1.1: Write failing test for `record_error`**

Append to `services/atlas-pr-bootstrap/test/lib_test.bats`:

```bats
@test "record_error: appends phase to ATLAS_PHASE_ERRORS and logs error" {
    ATLAS_PHASE_ERRORS=()
    run bash -c '. "'"$PROJECT_ROOT"'/scripts/lib.sh"; ATLAS_PHASE_ERRORS=(); record_error drop-topics "rpk failed"; printf "%s\n" "${ATLAS_PHASE_ERRORS[@]}"'
    [ "$status" -eq 0 ]
    [[ "$output" == *"drop-topics"* ]]
    [[ "$output" == *"rpk failed"* ]]
}

@test "run_phase: success path emits start + complete, no error appended" {
    run bash -c '
        . "'"$PROJECT_ROOT"'/scripts/lib.sh"
        ATLAS_PHASE_ERRORS=()
        ok_phase() { return 0; }
        run_phase good_phase ok_phase
        echo "errors=${#ATLAS_PHASE_ERRORS[@]}"
    '
    [ "$status" -eq 0 ]
    [[ "$output" == *"phase start"* ]]
    [[ "$output" == *"phase complete"* ]]
    [[ "$output" == *"errors=0"* ]]
}

@test "run_phase: failure path records phase and returns 0 (orchestration continues)" {
    run bash -c '
        . "'"$PROJECT_ROOT"'/scripts/lib.sh"
        ATLAS_PHASE_ERRORS=()
        bad_phase() { return 7; }
        run_phase bad_phase bad_phase
        echo "rc=$?"
        echo "errors=${ATLAS_PHASE_ERRORS[*]}"
    '
    [ "$status" -eq 0 ]
    [[ "$output" == *"rc=0"* ]]
    [[ "$output" == *"errors=bad_phase"* ]]
}

@test "summarize_phases: phases_failed=0 success line, exit 0" {
    run bash -c '
        . "'"$PROJECT_ROOT"'/scripts/lib.sh"
        ATLAS_PHASE_ERRORS=()
        summarize_phases 7
        echo "rc=$?"
    '
    [ "$status" -eq 0 ]
    [[ "$output" == *"phases_run=7"* ]]
    [[ "$output" == *"phases_failed=0"* ]]
    [[ "$output" == *"rc=0"* ]]
}

@test "summarize_phases: error path lists failed phases as JSON array, exits 1" {
    run bash -c '
        . "'"$PROJECT_ROOT"'/scripts/lib.sh"
        ATLAS_PHASE_ERRORS=(drop-topics drop-redis)
        summarize_phases 7
        echo "rc=$?"
    '
    [ "$status" -eq 0 ]
    [[ "$output" == *"phases_run=7"* ]]
    [[ "$output" == *"phases_failed=2"* ]]
    [[ "$output" == *'["drop-topics","drop-redis"]'* ]]
    [[ "$output" == *"rc=1"* ]]
}
```

- [ ] **Step 1.2: Run lib_test.bats; the new tests must fail**

Run: `bats services/atlas-pr-bootstrap/test/lib_test.bats`
Expected: existing 4 `compute_atlas_env` tests pass; the 5 new ones fail with `record_error: command not found` / `run_phase: command not found` / `summarize_phases: command not found`.

- [ ] **Step 1.3: Edit `services/atlas-pr-bootstrap/scripts/lib.sh`**

Change the header line `set -euo pipefail` to `set -uo pipefail` and
append the new helpers to the end of the file. Final file contents:

```bash
#!/usr/bin/env bash
# Shared helpers for bootstrap.sh, cleanup.sh, sweep-orphans.sh.
#
# Header is `set -uo pipefail` (no -e) because cleanup.sh and
# sweep-orphans.sh implement try-all phase orchestration via
# run_phase; a top-level -e would short-circuit that.

set -uo pipefail

log() {
    local level="$1"; shift
    local step="${ATLAS_STEP:-init}"
    if command -v jq >/dev/null 2>&1; then
        printf '{"ts":"%s","level":"%s","atlas.env":"%s","atlas.step":"%s","msg":%s}\n' \
            "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$level" "${ATLAS_ENV:-}" "$step" \
            "$(printf '%s' "$*" | jq -Rs .)"
    else
        # Fallback for environments without jq (e.g., bats hosts without
        # the bootstrap image installed). Emits the same raw message
        # without JSON encoding so test assertions still match.
        printf '[%s] %s\n' "$level" "$*" >&2
    fi
}

require_env() {
    for v in "$@"; do
        if [ -z "${!v:-}" ]; then
            log error "missing required env: $v"
            exit 1
        fi
    done
}

retry() {
    local max=$1; shift
    local sleep_s=$1; shift
    local n=0
    while ! "$@"; do
        n=$((n+1))
        if [ "$n" -ge "$max" ]; then
            log error "retry exhausted after $n attempts: $*"
            return 1
        fi
        sleep "$sleep_s"
    done
}

http_ok() {
    local url=$1
    local status
    status=$(curl -s -o /dev/null -w '%{http_code}' "$url" || echo 000)
    [ "$status" = "200" ] || [ "$status" = "204" ]
}

http_ok_tenant() {
    local url=$1
    local status
    status=$(curl -s -o /dev/null -w '%{http_code}' \
        -H "TENANT_ID: $TENANT_ID" -H "REGION: $REGION" \
        -H "MAJOR_VERSION: $MAJOR_VERSION" -H "MINOR_VERSION: $MINOR_VERSION" \
        "$url" || echo 000)
    [ "$status" = "200" ] || [ "$status" = "204" ]
}

# compute_atlas_env: derive the 4-hex-char per-env hash from a PR number.
# MUST stay in sync with .github/workflows/pr-validation.yml's update-pr-overlay
# step and the cluster-infra ApplicationSet template. test/lib_test.bats pins
# the contract via the PR 491 / 522 recovery-log oracles.
compute_atlas_env() {
    local pr_number="$1"
    if [ -z "$pr_number" ]; then
        log error "compute_atlas_env: empty PR_NUMBER"
        return 1
    fi
    printf "pr-%d" "$pr_number" | sha256sum | cut -c1-4
}

# ----------------------------------------------------------------------------
# Phase-runner framework. See task-075/design.md §3.1.
#
# Used by cleanup.sh and sweep-orphans.sh to run every phase regardless
# of any single phase's outcome. ONLY run_phase appends to
# ATLAS_PHASE_ERRORS — phase functions emit detail logs via `log warn`
# / `log error` and return non-zero on failure; run_phase records the
# phase name exactly once.
# ----------------------------------------------------------------------------

# ATLAS_PHASE_ERRORS holds the names of phases that have failed. The
# caller initialises (or resets, for per-PR sweep loops) by assigning
# ATLAS_PHASE_ERRORS=() before run_phase is called.
declare -ga ATLAS_PHASE_ERRORS=()

# record_error <phase> <msg>
# Appends <phase> to ATLAS_PHASE_ERRORS and logs <msg> at level=error
# with ATLAS_STEP=<phase>.
record_error() {
    local phase="$1"; shift
    ATLAS_PHASE_ERRORS+=("$phase")
    ATLAS_STEP="$phase" log error "$*"
}

# run_phase <phase_name> <function_name>
# Emits a "phase start" info log, runs <function_name>, and either
# emits "phase complete" (on zero return) or appends <phase_name> to
# ATLAS_PHASE_ERRORS via record_error (on non-zero). Always returns 0
# so a caller without `set -e` continues running subsequent phases.
run_phase() {
    local phase="$1"; local fn="$2"
    ATLAS_STEP="$phase" log info "phase start"
    if "$fn"; then
        ATLAS_STEP="$phase" log info "phase complete"
    else
        record_error "$phase" "phase exited non-zero"
    fi
    return 0
}

# summarize_phases <total_phase_count>
# Emits one JSON summary line and returns 0 (success) or 1 (errors
# recorded). Callers typically `exit $?` after this.
summarize_phases() {
    local total="$1"
    local failed="${#ATLAS_PHASE_ERRORS[@]}"
    local failed_json
    if [ "$failed" -eq 0 ]; then
        ATLAS_STEP=done log info "cleanup complete phases_run=$total phases_failed=0"
        return 0
    fi
    failed_json=$(printf '%s\n' "${ATLAS_PHASE_ERRORS[@]}" \
        | jq -Rsc 'split("\n") | map(select(length>0))')
    ATLAS_STEP=done log error "cleanup completed with errors phases_run=$total phases_failed=$failed failed_phases=$failed_json"
    return 1
}
```

- [ ] **Step 1.4: Run lib_test.bats; all tests pass**

Run: `bats services/atlas-pr-bootstrap/test/lib_test.bats`
Expected: 9 tests pass (4 existing `compute_atlas_env` + 5 new).

- [ ] **Step 1.5: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/lib.sh \
        services/atlas-pr-bootstrap/test/lib_test.bats
git commit -m "task-075: add phase-runner framework to lib.sh

record_error / run_phase / summarize_phases let cleanup.sh and
sweep-orphans.sh run every phase regardless of any single failure.
Header drops -e so try-all orchestration isn't short-circuited."
```

---

## Task 2 — Add rpk fixtures + shared jq query constants

**Bugs addressed:** 1 (jq schema mismatch). Sets up the inputs for Tasks 3 & 4.

**Files:**
- Create: `services/atlas-pr-bootstrap/test/fixtures/rpk-topic-list.json`
- Create: `services/atlas-pr-bootstrap/test/fixtures/rpk-group-list.json`
- Create: `services/atlas-pr-bootstrap/test/fixtures/README.md`
- Modify: `services/atlas-pr-bootstrap/scripts/lib.sh` (append constants)

- [ ] **Step 2.1: Create the topic list fixture**

Create `services/atlas-pr-bootstrap/test/fixtures/rpk-topic-list.json`:

```json
[
  {"name": "boss-spawn-events", "partitions": 3, "replicas": 1},
  {"name": "boss-spawn-events-a1b2", "partitions": 3, "replicas": 1},
  {"name": "character-events-a1b2", "partitions": 6, "replicas": 1},
  {"name": "configurations-events", "partitions": 1, "replicas": 1}
]
```

The fixture is scoped to `ATLAS_ENV=a1b2`. It exercises: a
non-suffixed topic (must NOT delete), two env-suffixed topics
(must delete), and a second non-suffixed topic.

- [ ] **Step 2.2: Create the group list fixture**

Create `services/atlas-pr-bootstrap/test/fixtures/rpk-group-list.json`:

```json
[
  {"name": "Account Service", "members": 0},
  {"name": "Channel Service - 7e3a-0a1b [a1b2]", "members": 1},
  {"name": "Party Quest Service [a1b2]", "members": 1},
  {"name": "Party Quest Service [other]", "members": 1}
]
```

Exercises: a non-suffixed group (must NOT delete), an env-suffixed
group with a hyphenated channel-id segment (must delete), an
env-suffixed group with spaces (must delete intact), a
non-target-env-suffixed group (must NOT delete).

- [ ] **Step 2.3: Create the fixtures README**

Create `services/atlas-pr-bootstrap/test/fixtures/README.md`:

```markdown
# rpk JSON-output fixtures

Pinned to `ARG RPK_VERSION=24.3.1` in `../../Dockerfile`. Bumping
that version invalidates these files; regenerate against the new
rpk binary and re-run `bats services/atlas-pr-bootstrap/test/`.

## Regenerate

Run against any reachable Kafka broker (e.g. the cluster's
`kafka.home:9093`):

```
rpk topic list -X brokers=<broker> --format json \
  > services/atlas-pr-bootstrap/test/fixtures/rpk-topic-list.json
rpk group list -X brokers=<broker> --format json \
  > services/atlas-pr-bootstrap/test/fixtures/rpk-group-list.json
```

After regenerating, edit the files to keep the test scenarios
intact:

- One topic name ending in `-a1b2` plus one not ending in
  `-a1b2` (cleanup-side suffix test).
- One group name ending in `[a1b2]` containing spaces, one
  ending in `[a1b2]` without spaces, one ending in `[other]`
  (cleanup-side group suffix + spaced-name test).

`a1b2` is a literal `ATLAS_ENV` value the bats tests use directly
(via `make_stubs`); other tests compute their own env hash from
`PR_NUMBER` and sed-substitute fixture copies — see
`cleanup_test.bats::make_stubs`.
```

- [ ] **Step 2.4: Append jq-query constants to lib.sh**

Append to the end of `services/atlas-pr-bootstrap/scripts/lib.sh`:

```bash
# ----------------------------------------------------------------------------
# rpk JSON-output schema constants. See test/fixtures/rpk-*.json.
#
# rpk 24.3.1 emits a flat array for both topic list and group list:
#   [{"name":"…","partitions":…}, …]
# The pre-fix queries (.topics[].name / .groups[].name) assumed an
# object wrapping the array and failed with "Cannot index array with
# string …" — see prd.md §1 / Bug 1.
#
# Bumping ARG RPK_VERSION in the Dockerfile invalidates the fixtures.
# Regenerate against the new rpk and re-run bats; the schema may move
# again.
# ----------------------------------------------------------------------------
readonly RPK_TOPICS_JQ='.[].name'
readonly RPK_GROUPS_JQ='.[].name'
```

- [ ] **Step 2.5: Sanity-check fixtures against the constants**

Run: `jq -r '.[].name' services/atlas-pr-bootstrap/test/fixtures/rpk-topic-list.json`
Expected output:

```
boss-spawn-events
boss-spawn-events-a1b2
character-events-a1b2
configurations-events
```

Run: `jq -r '.[].name' services/atlas-pr-bootstrap/test/fixtures/rpk-group-list.json`
Expected output:

```
Account Service
Channel Service - 7e3a-0a1b [a1b2]
Party Quest Service [a1b2]
Party Quest Service [other]
```

- [ ] **Step 2.6: Commit**

```bash
git add services/atlas-pr-bootstrap/test/fixtures \
        services/atlas-pr-bootstrap/scripts/lib.sh
git commit -m "task-075: add rpk fixtures + shared jq query constants

rpk 24.3.1 topic/group list --format json emits a flat array, not
an object. RPK_TOPICS_JQ / RPK_GROUPS_JQ in lib.sh match that
schema; fixtures pin the schema so a future rpk bump fails bats
instead of leaking state in production."
```

---

## Task 3 — Refactor cleanup.sh: try-all + rpk jq fix

**Bugs addressed:** 1 (jq schema), 5 (try-all).

**Files:**
- Rewrite: `services/atlas-pr-bootstrap/scripts/cleanup.sh`
- Modify: `services/atlas-pr-bootstrap/test/cleanup_test.bats`

- [ ] **Step 3.1: Rewrite cleanup.sh with phases + try-all orchestration**

Replace `services/atlas-pr-bootstrap/scripts/cleanup.sh` with:

```bash
#!/usr/bin/env bash
# Atlas PR-env cleanup. Each phase is idempotent and runs through the
# shared run_phase orchestrator (lib.sh) so a single phase's failure
# does not skip subsequent phases. The Job exits non-zero iff at
# least one phase failed; the summary line names which.
#
# Required env:
#   PR_NUMBER              — PR number; ATLAS_ENV is derived as sha256("pr-N")[:4]
#   DB_HOST/PORT/USER/PASS — Postgres connection details
#   ATLAS_DB_NAMES    — space-separated list of base DB names
#   BOOTSTRAP_SERVERS — kafka.home:9093
#   REDIS_URL         — redis.home:6379
#   PIHOLE_API_BASE_1, PIHOLE_TOKEN_1, PIHOLE_API_BASE_2, PIHOLE_TOKEN_2
#   GHCR_TOKEN        — for image-tag delete + bot-branch delete
#   ATLAS_SERVICES    — comma-separated list of service names for image cleanup

set -uo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

# Phase 0 Task 0.1 finding: db-credentials secret values carry trailing
# whitespace (literal space + CR + LF). Strip BEFORE require_env so an
# all-whitespace value is caught by the empty check.
DB_USER="$(printf '%s' "${DB_USER:-}" | tr -d ' \r\n')"
DB_PASSWORD="$(printf '%s' "${DB_PASSWORD:-}" | tr -d ' \r\n')"

require_env PR_NUMBER DB_HOST DB_PORT DB_USER DB_PASSWORD ATLAS_DB_NAMES BOOTSTRAP_SERVERS REDIS_URL

# Derive ATLAS_ENV from PR_NUMBER. Bug #4 (env-hash annotation drift): the
# Application's atlas.env annotation can disagree with the formula's actual
# output (observed on PRs 491/522, see task-070 recovery-log.md). Deriving
# here guarantees cleanup targets the correct hash regardless. lib.sh's
# compute_atlas_env is pinned by test/lib_test.bats against the formula
# used by .github/workflows/pr-validation.yml and the ApplicationSet.
ATLAS_ENV="$(compute_atlas_env "$PR_NUMBER")"
export ATLAS_ENV
ATLAS_STEP=init log info "derived ATLAS_ENV=${ATLAS_ENV} for PR ${PR_NUMBER}"

# ----------------------------------------------------------------------------
# Phase functions. Each returns 0 on success, non-zero on failure;
# run_phase (lib.sh) records the phase name once on non-zero. Detail
# log lines inside a phase use log warn / log error.
# ----------------------------------------------------------------------------

do_drop_dbs() {
    ATLAS_STEP=drop-dbs log info "dropping per-env Postgres databases"
    # Probe connectivity before the per-DB loop. Postgres unreachable
    # means cleanup-targeting is broken and no other phase can be
    # trusted to reason about per-env state, so this is a hard exit.
    if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "SELECT 1" >/dev/null 2>&1; then
        ATLAS_STEP=drop-dbs log error "Postgres unreachable at $DB_HOST:$DB_PORT; aborting cleanup"
        exit 1
    fi
    # ATLAS_DB_NAMES is space-separated (matches kustomization.yaml's atlas-db-names
    # configMapGenerator and the create-dbs Job's for-loop). Use default IFS so the
    # `read -ra` splits on whitespace.
    local -a dbs
    read -ra dbs <<< "$ATLAS_DB_NAMES"
    local rc=0
    local db full
    for db in "${dbs[@]}"; do
        full="${db}-${ATLAS_ENV}"
        if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS \"$full\";" >/dev/null 2>&1; then
            ATLAS_STEP=drop-dbs log warn "drop $full failed"
            rc=1
        fi
    done
    return $rc
}

do_drop_topics() {
    ATLAS_STEP=drop-topics log info "deleting per-env Kafka topics"
    # rpk topic list --format json emits a flat array. RPK_TOPICS_JQ
    # (lib.sh) is '.[].name'. A jq schema error (rpk upgrade) is
    # surfaced as a non-zero phase return, not silently swallowed.
    local topics
    topics=$(rpk topic list -X brokers="$BOOTSTRAP_SERVERS" --format json \
        | jq -r "$RPK_TOPICS_JQ") || return 1
    # grep returning 1 when no match is fine and idempotent.
    local matched
    matched=$(printf '%s\n' "$topics" | { grep -E -- "-${ATLAS_ENV}\$" || true; })
    [ -z "$matched" ] && return 0
    printf '%s\n' "$matched" | xargs -r -n 1 rpk topic delete -X brokers="$BOOTSTRAP_SERVERS"
}

do_drop_groups() {
    ATLAS_STEP=drop-groups log info "deleting per-env consumer groups"
    # Atlas consumer-group names contain spaces (e.g. "Party Quest Service [1756]",
    # "Channel Service - <uuid> [1756]"). xargs's default delimiter is whitespace,
    # which would word-split each group name. -d '\n' restricts splitting to
    # newlines so each group is passed intact.
    local groups
    groups=$(rpk group list -X brokers="$BOOTSTRAP_SERVERS" --format json \
        | jq -r "$RPK_GROUPS_JQ") || return 1
    local matched
    matched=$(printf '%s\n' "$groups" | { grep -E -- "\\[${ATLAS_ENV}\\]\$" || true; })
    [ -z "$matched" ] && return 0
    printf '%s\n' "$matched" | xargs -r -d '\n' -n 1 rpk group delete -X brokers="$BOOTSTRAP_SERVERS"
}

do_drop_redis() {
    ATLAS_STEP=drop-redis log info "deleting per-env Redis keys"
    redis-cli -u "redis://$REDIS_URL" --scan --pattern "${ATLAS_ENV}:*" \
        | xargs -r -n 1000 redis-cli -u "redis://$REDIS_URL" DEL
}

do_drop_images() {
    if [ -z "${ATLAS_SERVICES:-}" ] || [ -z "${GHCR_TOKEN:-}" ]; then
        ATLAS_STEP=drop-images log info "no ATLAS_SERVICES/GHCR_TOKEN; skipping"
        return 0
    fi
    ATLAS_STEP=drop-images log info "deleting per-PR ghcr image tags"
    local -a svcs
    IFS=',' read -ra svcs <<< "$ATLAS_SERVICES"
    local svc vid rc=0
    for svc in "${svcs[@]}"; do
        while read -r vid; do
            gh api --method DELETE -H "Authorization: Bearer $GHCR_TOKEN" \
                "/users/chronicle20/packages/container/${svc}%2F${svc}/versions/${vid}" \
                >/dev/null 2>&1 || ATLAS_STEP=drop-images log warn "delete ${svc}/${vid} failed"
        done < <(gh api -H "Authorization: Bearer $GHCR_TOKEN" \
            "/users/chronicle20/packages/container/${svc}%2F${svc}/versions" \
            --jq ".[] | select(.metadata.container.tags[]? | startswith(\"pr-${PR_NUMBER}-\")) | .id" \
            2>/dev/null) || rc=1
    done
    return $rc
}

do_drop_dns() {
    if [ -z "${PIHOLE_API_BASE_1:-}" ] || [ -z "${PIHOLE_TOKEN_1:-}" ]; then
        ATLAS_STEP=drop-dns log info "no Pi-hole creds; skipping"
        return 0
    fi
    ATLAS_STEP=drop-dns log info "removing Pi-hole A records"
    local host="${PR_NUMBER}.atlas.home"
    local rc=0
    local i base_var token_var base token sid entry encoded_entry
    for i in 1 2; do
        base_var="PIHOLE_API_BASE_$i"
        token_var="PIHOLE_TOKEN_$i"
        base="${!base_var:-}"
        token="${!token_var:-}"
        [ -z "$base" ] && continue
        [ -z "$token" ] && continue
        sid=$(curl -k -fsS -X POST "$base/api/auth" \
            -H "Content-Type: application/json" \
            -d "{\"password\":\"$token\"}" 2>/dev/null \
            | jq -r '.session.sid // empty')
        if [ -z "$sid" ]; then
            ATLAS_STEP=drop-dns log warn "Pi-hole $i: auth failed, skipping host removal"
            rc=1
            continue
        fi
        entry=$(curl -k -fsS -H "X-FTL-SID: $sid" "$base/api/config/dns/hosts" \
            | jq -r ".config.dns.hosts[]? | select(endswith(\" $host\"))" | head -1)
        if [ -n "$entry" ]; then
            encoded_entry=$(printf '%s' "$entry" | sed 's/ /%20/g')
            curl -k -fsS -X DELETE -H "X-FTL-SID: $sid" \
                "$base/api/config/dns/hosts/$encoded_entry" || {
                    ATLAS_STEP=drop-dns log warn "Pi-hole $i delete failed for $host"
                    rc=1
                }
        fi
    done
    return $rc
}

do_drop_branch() {
    if [ -z "${PR_NUMBER:-}" ] || [ -z "${GHCR_TOKEN:-}" ]; then
        ATLAS_STEP=drop-branch log info "no GHCR_TOKEN; skipping"
        return 0
    fi
    ATLAS_STEP=drop-branch log info "deleting bot/pr-${PR_NUMBER}-resolved"
    # 404 is the branch-already-deleted case — treat as success. Other
    # errors log warn but still mark the phase as failed (return 1).
    local err
    if ! err=$(gh api --method DELETE \
        -H "Authorization: Bearer ${GHCR_TOKEN}" \
        "/repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-${PR_NUMBER}-resolved" \
        2>&1); then
        case "$err" in
            *"Reference does not exist"*|*"Branch not found"*|*"404"*)
                return 0
                ;;
            *)
                ATLAS_STEP=drop-branch log warn "branch delete: $err"
                return 1
                ;;
        esac
    fi
    return 0
}

# ----------------------------------------------------------------------------
# Orchestration. PHASES is interleaved <phase_name> <function_name>.
# ----------------------------------------------------------------------------
PHASES=(
    drop-dbs     do_drop_dbs
    drop-topics  do_drop_topics
    drop-groups  do_drop_groups
    drop-redis   do_drop_redis
    drop-images  do_drop_images
    drop-dns     do_drop_dns
    drop-branch  do_drop_branch
)
TOTAL=$(( ${#PHASES[@]} / 2 ))
ATLAS_PHASE_ERRORS=()
for ((i=0; i<${#PHASES[@]}; i+=2)); do
    run_phase "${PHASES[i]}" "${PHASES[i+1]}"
done
summarize_phases "$TOTAL"
exit $?
```

- [ ] **Step 3.2: Update cleanup_test.bats stubs to read from fixtures**

Replace lines 11-50 of `services/atlas-pr-bootstrap/test/cleanup_test.bats`
(the `make_stubs` function) with:

```bash
# make_stubs writes shell-script stubs for every external binary cleanup.sh
# invokes. Each stub appends its argv to "$STUB_LOG" and exits 0 unless the
# caller passes per-binary overrides.
#
# Args (optional, in order):
#   $1 — topic_list_json (default: rpk-topic-list.json fixture)
#   $2 — group_list_json (default: rpk-group-list.json fixture)
make_stubs() {
    local topic_json
    local group_json
    if [ "${1+set}" = set ]; then
        topic_json="$1"
    else
        topic_json="$(cat "$PROJECT_ROOT/test/fixtures/rpk-topic-list.json")"
    fi
    if [ "${2+set}" = set ]; then
        group_json="$2"
    else
        group_json="$(cat "$PROJECT_ROOT/test/fixtures/rpk-group-list.json")"
    fi
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
```

The two existing tests that hand-spelled inline JSON
(`cleanup.sh deletes only -ATLAS_ENV-suffixed topics via rpk` at
old line 139 and `cleanup.sh deletes consumer groups with spaces`
at old line 158) are computing their own env hash via
`fixture_env`. Replace those two `@test` blocks with versions that
sed the fixture into an env-hash-aware copy:

```bats
@test "cleanup.sh deletes only -ATLAS_ENV-suffixed topics via rpk" {
    local env_hash
    env_hash="$(fixture_env)"
    local topics
    topics=$(sed "s/a1b2/${env_hash}/g" \
        "$PROJECT_ROOT/test/fixtures/rpk-topic-list.json")
    make_stubs "$topics" '[]'
    run run_cleanup
    [ "$status" -eq 0 ]

    # rpk topic list was invoked once
    [ "$(grep -c '^rpk topic list ' "$STUB_LOG")" -eq 1 ]

    # rpk topic delete was invoked for the two env-suffixed topics and
    # not for the unsuffixed ones.
    grep -F 'rpk topic delete' "$STUB_LOG" | grep -F "boss-spawn-events-${env_hash}"
    grep -F 'rpk topic delete' "$STUB_LOG" | grep -F "character-events-${env_hash}"
    if grep -F 'rpk topic delete' "$STUB_LOG" | grep -wF 'configurations-events'; then
        echo "ERROR: unsuffixed topic was deleted" >&2
        return 1
    fi
}

@test "cleanup.sh deletes consumer groups with spaces in their names" {
    # Group list has one name matching [<env>] suffix (with spaces) and one
    # not matching. Only the matching one should be deleted.
    local env_hash
    env_hash="$(fixture_env)"
    local groups
    groups=$(sed "s/a1b2/${env_hash}/g" \
        "$PROJECT_ROOT/test/fixtures/rpk-group-list.json")
    make_stubs '[]' "$groups"
    run run_cleanup
    [ "$status" -eq 0 ]

    # rpk group list invoked once
    [ "$(grep -c '^rpk group list ' "$STUB_LOG")" -eq 1 ]

    # rpk group delete was called for the spaced + hyphenated names as
    # single arguments each.
    grep -F 'rpk group delete' "$STUB_LOG" | grep -F "Party Quest Service [${env_hash}]"
    grep -F 'rpk group delete' "$STUB_LOG" | grep -F "Channel Service - 7e3a-0a1b [${env_hash}]"

    # The other-env group must not be deleted
    if grep -F 'rpk group delete' "$STUB_LOG" | grep -F 'Party Quest Service [other]'; then
        echo "ERROR: group with non-matching env suffix was deleted" >&2
        return 1
    fi
}
```

Also replace the existing `cleanup.sh skips rpk topic delete when no topic matches` test (old line 182) with:

```bats
@test "cleanup.sh skips rpk topic delete when no topic matches" {
    make_stubs '[{"name":"prod-foo"},{"name":"prod-bar"}]' '[]'
    run run_cleanup
    [ "$status" -eq 0 ]
    [ "$(grep -c '^rpk topic list ' "$STUB_LOG")" -eq 1 ]
    # No delete because no topic name ends with -<env_hash>
    if grep -F 'rpk topic delete' "$STUB_LOG"; then
        echo "ERROR: rpk topic delete invoked despite no matching topics" >&2
        return 1
    fi
}
```

- [ ] **Step 3.3: Add the three new behavioural tests**

Append to `services/atlas-pr-bootstrap/test/cleanup_test.bats`:

```bats
@test "cleanup.sh runs every phase even when drop-topics fails" {
    # rpk emits non-JSON for `topic list`, so jq exits non-zero and
    # do_drop_topics returns 1. Every subsequent phase must still run.
    mkdir -p "$STUB_BIN"
    cat > "$STUB_BIN/rpk" <<'EOF'
#!/usr/bin/env bash
echo "rpk $*" >> "$STUB_LOG"
if [ "$1" = "topic" ] && [ "$2" = "list" ]; then
    echo "<not-json>"
    exit 0
elif [ "$1" = "group" ] && [ "$2" = "list" ]; then
    echo "[]"
    exit 0
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
exit 0
EOF
    cat > "$STUB_BIN/gh" <<'EOF'
#!/usr/bin/env bash
echo "gh $*" >> "$STUB_LOG"
exit 0
EOF
    chmod +x "$STUB_BIN"/*

    run run_cleanup
    # Exit 1 because at least one phase failed.
    [ "$status" -eq 1 ]
    # Every subsequent phase logged its "phase complete" line.
    [[ "$output" == *'"atlas.step":"drop-groups"'*'phase complete'* ]] \
        || [[ "$output" == *'drop-groups'*'phase complete'* ]]
    [[ "$output" == *'drop-redis'*'phase complete'* ]]
    [[ "$output" == *'drop-images'*'phase complete'* ]]
    [[ "$output" == *'drop-dns'*'phase complete'* ]]
    [[ "$output" == *'drop-branch'*'phase complete'* ]]
    # Summary names drop-topics as the failed phase.
    [[ "$output" == *'failed_phases'*'drop-topics'* ]]
    [[ "$output" == *'phases_failed=1'* ]]
}

@test "cleanup.sh exits 0 when all phases succeed" {
    make_stubs '[]' '[]'
    run run_cleanup
    [ "$status" -eq 0 ]
    [[ "$output" == *'phases_failed=0'* ]]
    [[ "$output" == *'phases_run=7'* ]]
}

@test "cleanup.sh fails fast on malformed rpk output" {
    # Same as the try-all test but asserts the specific jq-error
    # signal is recognisable in the log stream.
    mkdir -p "$STUB_BIN"
    cat > "$STUB_BIN/rpk" <<'EOF'
#!/usr/bin/env bash
echo "rpk $*" >> "$STUB_LOG"
if [ "$1" = "topic" ] && [ "$2" = "list" ]; then
    printf 'this is not json\n'
    exit 0
fi
echo "[]"
EOF
    cat > "$STUB_BIN/psql" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$STUB_BIN/redis-cli" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$STUB_BIN/gh" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "$STUB_BIN"/*

    run run_cleanup
    [ "$status" -eq 1 ]
    [[ "$output" == *'drop-topics'* ]]
    # phase exited non-zero recorded
    [[ "$output" == *'phase exited non-zero'* ]]
}
```

Replace `run_cleanup` (around line 56 of the existing file) with a version that doesn't `unset` STUB_LOG and that adds `ATLAS_SERVICES` so do_drop_images has something to operate on (it'll be a no-op without GHCR_TOKEN — which is the desired no-op-but-runs behaviour). Replace the existing `run_cleanup`:

```bash
run_cleanup() {
    PATH="$STUB_BIN:$PATH" \
    STUB_LOG="$STUB_LOG" \
    BATS_TEST_TMPDIR="$BATS_TEST_TMPDIR" \
    DB_HOST=h DB_PORT=5432 DB_USER=u DB_PASSWORD=p \
    ATLAS_DB_NAMES="foo bar" \
    BOOTSTRAP_SERVERS=kafka:9093 \
    REDIS_URL=redis:6379 \
    PR_NUMBER="${PR_NUMBER:-99}" \
    bash "$PROJECT_ROOT/scripts/cleanup.sh"
}
```

(The shape is identical — no functional change; included for clarity in execution.)

- [ ] **Step 3.4: Run cleanup_test.bats**

Run: `bats services/atlas-pr-bootstrap/test/cleanup_test.bats`
Expected: all tests pass (existing 7 + 3 new).

If `cleanup.sh fails without ATLAS_DB_NAMES` or
`cleanup.sh no longer requires ATLAS_ENV in env` fail because the
new cleanup.sh's `require_env` ordering changed, fix by reordering
`require_env`'s arguments to match the test expectations
(`PR_NUMBER DB_HOST DB_PORT DB_USER DB_PASSWORD ATLAS_DB_NAMES BOOTSTRAP_SERVERS REDIS_URL` already matches the test that asserts the next missing var after PR_NUMBER is DB_HOST).

- [ ] **Step 3.5: Run shellcheck on cleanup.sh**

Run: `shellcheck -x services/atlas-pr-bootstrap/scripts/cleanup.sh`
Expected: clean (or only existing pre-task style warnings).

If shellcheck isn't installed locally, skip — CI will catch it.

- [ ] **Step 3.6: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/cleanup.sh \
        services/atlas-pr-bootstrap/test/cleanup_test.bats
git commit -m "task-075: cleanup.sh phases + rpk jq schema fix

Each phase is a do_* function orchestrated through run_phase /
summarize_phases. Header drops -e so a single phase failure no
longer skips subsequent phases — the cause of PR 544's six-phase
leak. Drop-topics / drop-groups jq queries use RPK_TOPICS_JQ /
RPK_GROUPS_JQ (lib.sh) which match rpk 24.3.1's flat-array shape."
```

---

## Task 4 — Port sweep-orphans.sh Kafka phases to rpk + adopt try-all

**Bugs addressed:** 3 (kafka-*.sh → rpk), 5 (try-all on sweep too).

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh`
- Rewrite: `services/atlas-pr-bootstrap/test/sweep_test.bats`

- [ ] **Step 4.1: Replace sweep_kafka in sweep-orphans.sh**

In `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh`, replace
the `sweep_kafka()` function (lines 121-154) with:

```bash
sweep_kafka() {
    local pr_number="$1"
    local env_hash="$2"
    [ -z "${BOOTSTRAP_SERVERS:-}" ] && return 0
    if ! command -v rpk >/dev/null 2>&1; then
        ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log warn "rpk not on PATH; skipping"
        return 0
    fi

    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log info "scanning Kafka topics"
    local topics
    topics=$(rpk topic list -X brokers="$BOOTSTRAP_SERVERS" --format json \
        | jq -r "$RPK_TOPICS_JQ" \
        | { grep -E -- "-${env_hash}\$" || true; })
    while IFS= read -r t; do
        [ -z "$t" ] && continue
        echo "drop-topics ${t}"
        if [ "$APPLY" = "1" ]; then
            rpk topic delete -X brokers="$BOOTSTRAP_SERVERS" "$t" || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log warn "delete topic $t failed"
        fi
    done <<<"$topics"

    ATLAS_ENV="$env_hash" ATLAS_STEP=drop-groups log info "scanning Kafka consumer groups"
    local groups
    groups=$(rpk group list -X brokers="$BOOTSTRAP_SERVERS" --format json \
        | jq -r "$RPK_GROUPS_JQ" \
        | { grep -E -- "\\[${env_hash}\\]\$" || true; })
    while IFS= read -r g; do
        [ -z "$g" ] && continue
        echo "drop-groups ${g}"
        if [ "$APPLY" = "1" ]; then
            rpk group delete -X brokers="$BOOTSTRAP_SERVERS" "$g" || \
                ATLAS_ENV="$env_hash" ATLAS_STEP=drop-groups log warn "delete group $g failed"
        fi
    done <<<"$groups"
}
```

Change the script header (line 22) from `set -euo pipefail` to
`set -uo pipefail` to match cleanup.sh. The sweep script is
already structured around individual `sweep_*` functions that
short-circuit on missing inputs — those continue to work without
`-e` because each phase function explicitly probes its
preconditions.

- [ ] **Step 4.2: Verify no kafka-*.sh references remain in atlas-pr-bootstrap**

Run: `grep -rE 'kafka-(topics|consumer-groups)\.sh' services/atlas-pr-bootstrap/`
Expected: empty output.

- [ ] **Step 4.3: Rewrite sweep_test.bats stubs from kafka-*.sh to rpk**

Replace `services/atlas-pr-bootstrap/test/sweep_test.bats` lines
42-96 (the `phase names appear in --list output` test) with:

```bats
@test "sweep-orphans.sh: phase names appear in --list output" {
    # Mock infra commands to emit one fake resource each, so list mode
    # produces the canonical "phase resource" lines and APPLY=0 means none
    # of them get acted on.
    SHIM_DIR="$(mktemp -d)"
    cat > "$SHIM_DIR/rpk" <<'EOF'
#!/usr/bin/env bash
case "$1 $2" in
    "topic list")
        echo '[{"name":"atlas-faketopic-ed86","partitions":1,"replicas":1}]'
        ;;
    "group list")
        echo '[{"name":"Fake Group [ed86]","members":0}]'
        ;;
    "topic delete"|"group delete")
        echo "FAIL: delete invoked in list mode" >&2; exit 1 ;;
esac
EOF
    cat > "$SHIM_DIR/redis-cli" <<'EOF'
#!/usr/bin/env bash
case "$*" in
    *--scan*) echo "ed86:fake-key" ;;
    *DEL*)    echo "FAIL: DEL invoked in list mode" >&2; exit 1 ;;
esac
EOF
    cat > "$SHIM_DIR/psql" <<'EOF'
#!/usr/bin/env bash
echo "atlas-fake-ed86"
EOF
    cat > "$SHIM_DIR/gh" <<'EOF'
#!/usr/bin/env bash
# Empty results — easier than mocking the rich gh api jq path.
echo ""
EOF
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 1   # "Application not found" — drop-app-finalizers phase no-ops.
EOF
    chmod +x "$SHIM_DIR"/*

    PATH="$SHIM_DIR:$PATH" run env \
        DB_HOST=fake DB_PORT=1 DB_USER=u DB_PASSWORD=p \
        BOOTSTRAP_SERVERS=fake REDIS_URL=fake:6379 \
        GHCR_TOKEN=fake ATLAS_SERVICES=atlas-fake \
        bash "$SCRIPT" 491

    [[ "$output" == *"drop-dbs atlas-fake-ed86"* ]]
    [[ "$output" == *"drop-topics atlas-faketopic-ed86"* ]]
    [[ "$output" == *"drop-groups Fake Group [ed86]"* ]]
    [[ "$output" == *"drop-redis ed86:fake-key"* ]]
    [[ "$output" != *"FAIL:"* ]]

    rm -rf "$SHIM_DIR"
}
```

- [ ] **Step 4.4: Add a sweep-orphans.sh apply-mode spaced-group test**

Append to `services/atlas-pr-bootstrap/test/sweep_test.bats`:

```bats
@test "sweep-orphans.sh: --apply deletes spaced group names intact" {
    SHIM_DIR="$(mktemp -d)"
    CALL_LOG="$BATS_TEST_TMPDIR/rpk-calls.log"
    cat > "$SHIM_DIR/rpk" <<EOF
#!/usr/bin/env bash
printf '%s\n' "rpk \$*" >> "$CALL_LOG"
case "\$1 \$2" in
    "topic list") echo '[]' ;;
    "group list") echo '[{"name":"Party Quest Service [ed86]","members":0}]' ;;
esac
exit 0
EOF
    cat > "$SHIM_DIR/psql" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$SHIM_DIR/redis-cli" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    cat > "$SHIM_DIR/gh" <<'EOF'
#!/usr/bin/env bash
echo ""
EOF
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
    chmod +x "$SHIM_DIR"/*

    PATH="$SHIM_DIR:$PATH" run env CALL_LOG="$CALL_LOG" \
        DB_HOST=fake DB_PORT=1 DB_USER=u DB_PASSWORD=p \
        BOOTSTRAP_SERVERS=fake REDIS_URL=fake:6379 \
        GHCR_TOKEN=fake ATLAS_SERVICES=atlas-fake \
        bash "$SCRIPT" --apply 491

    # rpk group delete was called with the full spaced name as one argv
    grep -F "rpk group delete -X brokers=fake Party Quest Service [ed86]" "$CALL_LOG"

    rm -rf "$SHIM_DIR"
}
```

- [ ] **Step 4.5: Run sweep_test.bats**

Run: `bats services/atlas-pr-bootstrap/test/sweep_test.bats`
Expected: all tests pass (existing 5 + 1 new apply-mode test).

- [ ] **Step 4.6: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/sweep-orphans.sh \
        services/atlas-pr-bootstrap/test/sweep_test.bats
git commit -m "task-075: port sweep-orphans Kafka phases to rpk

The Dockerfile installs rpk, not the Kafka tarball, so the
previous kafka-topics.sh / kafka-consumer-groups.sh gates always
silently no-op'd in production. Sweep now uses the same
rpk-based phases as cleanup.sh (shared RPK_*_JQ constants).
Header drops -e to match cleanup.sh's try-all discipline."
```

---

## Task 5 — Dockerfile copies sweep-orphans.sh + drift guard

**Bugs addressed:** 2 (missing sweep-orphans.sh in image).

**Files:**
- Modify: `services/atlas-pr-bootstrap/Dockerfile`
- Create: `services/atlas-pr-bootstrap/test/dockerfile_test.bats`

- [ ] **Step 5.1: Edit the Dockerfile**

In `services/atlas-pr-bootstrap/Dockerfile`, replace lines 34-39
(the COPY + chmod block) with:

```dockerfile
WORKDIR /atlas
COPY scripts/lib.sh /atlas/lib.sh
COPY scripts/bootstrap.sh /atlas/bootstrap.sh
COPY scripts/cleanup.sh /atlas/cleanup.sh
COPY scripts/sweep-orphans.sh /atlas/sweep-orphans.sh
COPY canonical/ /atlas/canonical/

RUN chmod +x /atlas/bootstrap.sh /atlas/cleanup.sh /atlas/sweep-orphans.sh
```

Also append a fixture-regeneration comment near `ARG RPK_VERSION`
(line 9). Replace lines 7-9 with:

```dockerfile
FROM alpine:3.23

# Bumping RPK_VERSION invalidates test/fixtures/rpk-*.json — rpk's
# --format json schema is a stability boundary, not a CLI flag. After
# bumping, regenerate per test/fixtures/README.md and re-run
# `bats services/atlas-pr-bootstrap/test/`.
ARG RPK_VERSION=24.3.1
```

- [ ] **Step 5.2: Create the drift-guard bats test**

Create `services/atlas-pr-bootstrap/test/dockerfile_test.bats`:

```bats
#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
}

@test "Dockerfile copies every script under scripts/" {
    local missing=()
    for f in "$PROJECT_ROOT"/scripts/*.sh; do
        local base
        base="$(basename "$f")"
        if ! grep -qE "^COPY scripts/${base} /atlas/${base}\$" "$PROJECT_ROOT/Dockerfile"; then
            missing+=("$base")
        fi
    done
    if [ "${#missing[@]}" -ne 0 ]; then
        echo "Dockerfile missing COPY for: ${missing[*]}" >&2
        return 1
    fi
}

@test "Dockerfile chmod +x covers every script under scripts/" {
    local chmod_line
    chmod_line=$(grep -E '^RUN chmod \+x /atlas/' "$PROJECT_ROOT/Dockerfile" | head -1)
    [ -n "$chmod_line" ]
    local missing=()
    for f in "$PROJECT_ROOT"/scripts/*.sh; do
        local base
        base="$(basename "$f")"
        if ! printf '%s\n' "$chmod_line" | grep -qF "/atlas/${base}"; then
            missing+=("$base")
        fi
    done
    if [ "${#missing[@]}" -ne 0 ]; then
        echo "Dockerfile chmod +x line missing entries for: ${missing[*]}" >&2
        return 1
    fi
}
```

- [ ] **Step 5.3: Run the bats suite**

Run: `bats services/atlas-pr-bootstrap/test/`
Expected: every test passes, including the new dockerfile_test.bats.

- [ ] **Step 5.4: Manually verify drift guard fails on a missing COPY**

Create a temporary file `services/atlas-pr-bootstrap/scripts/test-noop.sh` containing:

```bash
#!/usr/bin/env bash
exit 0
```

Run: `bats services/atlas-pr-bootstrap/test/dockerfile_test.bats`
Expected: both tests fail with `Dockerfile missing COPY for: test-noop.sh`.

Remove the temp file:

```bash
rm services/atlas-pr-bootstrap/scripts/test-noop.sh
```

Re-run: `bats services/atlas-pr-bootstrap/test/dockerfile_test.bats`
Expected: tests pass.

- [ ] **Step 5.5: Build the image to confirm it works end-to-end**

Run: `docker buildx bake atlas-pr-bootstrap`
Expected: build succeeds.

Optionally confirm the file lands at the expected path:

```bash
docker run --rm $(docker buildx bake atlas-pr-bootstrap --print 2>/dev/null | jq -r '.target."atlas-pr-bootstrap".tags[0]') ls /atlas
```

Expected: output includes `sweep-orphans.sh`.

- [ ] **Step 5.6: Commit**

```bash
git add services/atlas-pr-bootstrap/Dockerfile \
        services/atlas-pr-bootstrap/test/dockerfile_test.bats
git commit -m "task-075: ship sweep-orphans.sh in the bootstrap image

Dockerfile now COPY+chmod's every script in scripts/. The runbook's
/atlas/sweep-orphans.sh reference resolves to a real path. A new
dockerfile_test.bats locks the COPY-coverage in so adding a new
script without a Dockerfile edit fails CI."
```

---

## Task 6 — `consumergroup.Resolve` variadic args + tests

**Bugs addressed:** 6 (literal `%s` in PR-env consumer-group name).

**Files:**
- Modify: `libs/atlas-kafka/consumergroup/resolver.go`
- Modify: `libs/atlas-kafka/consumergroup/resolver_test.go`

- [ ] **Step 6.1: Add the failing test cases**

Replace `libs/atlas-kafka/consumergroup/resolver_test.go` with:

```go
package consumergroup

import (
	"testing"
)

func TestResolve_envUnset_returnsDefault(t *testing.T) {
	t.Setenv("KAFKA_CONSUMER_GROUP", "")
	if got := Resolve("Character Service"); got != "Character Service" {
		t.Fatalf("Resolve = %q, want %q", got, "Character Service")
	}
}

func TestResolve_envSet_returnsEnvValue(t *testing.T) {
	t.Setenv("KAFKA_CONSUMER_GROUP", "Character Service [a3f7]")
	if got := Resolve("Character Service"); got != "Character Service [a3f7]" {
		t.Fatalf("Resolve = %q, want %q", got, "Character Service [a3f7]")
	}
}

func TestResolve_envWhitespaceOnly_returnsVerbatim(t *testing.T) {
	t.Setenv("KAFKA_CONSUMER_GROUP", "   ")
	// design §5.4 decision: do NOT trim. Whitespace-only is a config bug,
	// but we keep verbatim to avoid silently masking it.
	if got := Resolve("Character Service"); got != "   " {
		t.Fatalf("Resolve = %q, want verbatim whitespace", got)
	}
}

func TestResolve_envWithFormat_substitutes(t *testing.T) {
	// PR-env case: patch generator emits "Channel Service - %s [a1b2]";
	// caller passes the per-channel id as varargs.
	t.Setenv("KAFKA_CONSUMER_GROUP", "Channel Service - %s [a1b2]")
	got := Resolve("Channel Service - %s", "ch-7")
	want := "Channel Service - ch-7 [a1b2]"
	if got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

func TestResolve_defaultWithFormat_substitutes(t *testing.T) {
	// Production case: env unset, default carries the %s.
	t.Setenv("KAFKA_CONSUMER_GROUP", "")
	got := Resolve("Channel Service - %s", "ch-7")
	want := "Channel Service - ch-7"
	if got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

func TestResolve_zeroArgs_doesNotFormat(t *testing.T) {
	// Existing zero-args callers (atlas-account etc.) must keep working
	// even if some future env value happens to contain "%s".
	t.Setenv("KAFKA_CONSUMER_GROUP", "%s literal")
	if got := Resolve("Account Service"); got != "%s literal" {
		t.Fatalf("Resolve = %q, want %q (no formatting when zero args)", got, "%s literal")
	}
}
```

- [ ] **Step 6.2: Run the tests; the new ones must fail**

Run: `go test -race ./libs/atlas-kafka/consumergroup/...`
Expected: 3 existing tests pass; `TestResolve_envWithFormat_substitutes`, `TestResolve_defaultWithFormat_substitutes`, and `TestResolve_zeroArgs_doesNotFormat` fail with compile error (`too many arguments in call to Resolve`).

- [ ] **Step 6.3: Update `resolver.go` to accept variadic args**

Replace `libs/atlas-kafka/consumergroup/resolver.go` with:

```go
// Package consumergroup resolves a service's Kafka consumer group ID.
//
// The default name is the service's historical literal (e.g. "Character Service").
// In environments where consumer-group isolation is required, the deployment
// sets KAFKA_CONSUMER_GROUP to a suffixed value such as
// "Character Service [a3f7]" and the env value is returned verbatim.
//
// Templated callers (atlas-channel, atlas-login) pass the per-channel /
// per-login ID as variadic args and a format string carrying "%s" in either
// the default or the env value; Resolve fmt.Sprintf's at runtime so the
// substitution happens after the PR-overlay patch has been applied.
package consumergroup

import (
	"fmt"
	"os"
)

const envVar = "KAFKA_CONSUMER_GROUP"

// Resolve returns the consumer group ID this service must use.
//
// Behaviour matrix:
//
//	KAFKA_CONSUMER_GROUP    args     result
//	------------------------------------------------------------
//	unset / ""              none     defaultName (verbatim)
//	unset / ""              N>0      fmt.Sprintf(defaultName, args...)
//	non-empty               none     env value (verbatim)
//	non-empty               N>0      fmt.Sprintf(envValue, args...)
//
// Whitespace-only env values are non-empty by this rule and therefore
// returned verbatim; design §5.4 keeps that semantic to surface
// config bugs rather than mask them.
//
// Existing zero-args callers (e.g. atlas-account, atlas-data) are
// source-compatible — they hit the verbatim paths above.
func Resolve(defaultName string, args ...any) string {
	v, ok := os.LookupEnv(envVar)
	if ok && v != "" {
		if len(args) > 0 {
			return fmt.Sprintf(v, args...)
		}
		return v
	}
	if len(args) > 0 {
		return fmt.Sprintf(defaultName, args...)
	}
	return defaultName
}
```

- [ ] **Step 6.4: Run the tests; everything passes**

Run: `go test -race ./libs/atlas-kafka/consumergroup/...`
Expected: 6 tests pass.

- [ ] **Step 6.5: Vet the package**

Run: `go vet ./libs/atlas-kafka/consumergroup/...`
Expected: clean.

- [ ] **Step 6.6: Commit**

```bash
git add libs/atlas-kafka/consumergroup/resolver.go \
        libs/atlas-kafka/consumergroup/resolver_test.go
git commit -m "task-075: consumergroup.Resolve gains variadic args

Templated callers (atlas-channel, atlas-login) need %s substitution
to happen at consumer-registration time so each channel/login id
gets its own group name, even when KAFKA_CONSUMER_GROUP is set by
the PR overlay. Zero-args callers (atlas-account etc.) hit the
existing verbatim paths."
```

---

## Task 7 — Update atlas-channel and atlas-login call sites

**Bugs addressed:** 6 (literal-`%s`).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go:151`
- Modify: `services/atlas-login/atlas.com/login/main.go:66`

- [ ] **Step 7.1: Update atlas-channel call site**

In `services/atlas-channel/atlas.com/channel/main.go`, replace
line 151:

```go
	var consumerGroupId = consumergroup.Resolve(fmt.Sprintf(consumerGroupIdTemplate, config.Id.String()))
```

with:

```go
	var consumerGroupId = consumergroup.Resolve(consumerGroupIdTemplate, config.Id.String())
```

- [ ] **Step 7.2: Update atlas-login call site**

In `services/atlas-login/atlas.com/login/main.go`, replace line 66:

```go
	var consumerGroupId = consumergroup.Resolve(fmt.Sprintf(consumerGroupIdTemplate, config.Id.String()))
```

with:

```go
	var consumerGroupId = consumergroup.Resolve(consumerGroupIdTemplate, config.Id.String())
```

- [ ] **Step 7.3: Verify `fmt` is still used elsewhere in each main.go**

Run: `grep -nE '\bfmt\.' services/atlas-channel/atlas.com/channel/main.go services/atlas-login/atlas.com/login/main.go`

If `fmt` is used elsewhere in the file (likely — both services do plenty of formatting), leave the import alone. If `grep` returns nothing for one file, also remove `"fmt"` from that file's import block.

Expected: both files retain `fmt` references; no import edit needed. (If `goimports`/the toolchain handles it automatically on build, skip.)

- [ ] **Step 7.4: Build both services**

Run:

```bash
go build ./services/atlas-channel/atlas.com/channel/...
go build ./services/atlas-login/atlas.com/login/...
```

Expected: clean build on both.

- [ ] **Step 7.5: Vet both modules**

Run:

```bash
go vet ./services/atlas-channel/...
go vet ./services/atlas-login/...
```

Expected: clean.

- [ ] **Step 7.6: Run race-mode tests for both modules**

Run:

```bash
go test -race ./services/atlas-channel/...
go test -race ./services/atlas-login/...
```

Expected: pre-existing tests still pass; this is a one-line change with no test surface of its own.

- [ ] **Step 7.7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/main.go \
        services/atlas-login/atlas.com/login/main.go
git commit -m "task-075: pass channel/login id to Resolve as varargs

The %s substitution now happens at runtime inside Resolve against
whichever value (default or env) wins, so PR-env consumer-group
names contain the actual channel/login id rather than a literal %s."
```

---

## Task 8 — `gen-cleanup-env.sh` + cluster-infra coordination artifact

**Bugs addressed:** 4 (env-var centralization, part 1 of 2 — generator side).

**Files:**
- Create: `deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh`
- Create: `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`

The sibling-PR ConfigMap shape is documented in `context.md` §4 and
mirrored by the generator's output here.

- [ ] **Step 8.1: Create the generator script**

Create `deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh`:

```bash
#!/usr/bin/env bash
# Generates dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml
# from .github/config/services.json.
#
# Output: a ConfigMap manifest cluster-infra mirrors into the argocd
# namespace. NOT deployed from this repo; the file's purpose is
# review-time visibility plus a stable diff when services.json
# changes.
#
# ATLAS_SERVICES is the cleanup Job's image-cleanup target list,
# derived (sorted, joined by commas) from .github/config/services.json's
# services[*].name. All other ConfigMap fields are static (cluster
# infra hostnames, db-name list). Re-run after adding/removing
# services. CI fails the PR if the file is stale (see
# .github/workflows/pr-validation.yml).

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
OUT="$ROOT/dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml"
SERVICES_JSON="$ROOT/.github/config/services.json"

mkdir -p "$(dirname "$OUT")"

# Sorted, comma-joined list of every service in services.json. We use
# every service entry regardless of type because the cleanup-Job's
# image-cleanup phase iterates ATLAS_SERVICES and 404s on
# non-published targets are tolerated by `|| true` semantics
# (see cleanup.sh do_drop_images).
ATLAS_SERVICES=$(jq -r '.services[].name' "$SERVICES_JSON" | sort | paste -sd, -)

# ATLAS_DB_NAMES is a static literal duplicated from
# deploy/k8s/overlays/pr/kustomization.yaml's atlas-db-names
# configMapGenerator. It is NOT derivable from services.json today —
# DB-name ownership lives on the kustomize side. Keep in sync by
# review.
ATLAS_DB_NAMES="atlas-accounts atlas-bans atlas-buddies atlas-cashshop atlas-characters atlas-configurations atlas-data atlas-drops atlas-fame atlas-gachapons atlas-guilds atlas-inventory atlas-keys atlas-map-actions atlas-maps atlas-merchant atlas-monster-book atlas-notes atlas-npc-conversations atlas-npc-shops atlas-party-quests atlas-pets atlas-portal-actions atlas-quest atlas-reactor-actions atlas-saga-orchestrator atlas-skills atlas-storage atlas-tenants"

cat > "$OUT" <<EOF
# Not deployed from this repo. Mirror into cluster-infra (argocd
# namespace). Generated by deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh
# from .github/config/services.json — do not edit by hand. Re-run the
# generator after adding/removing a service or a DB.
#
# The atlas-pr-cleanup PostDelete Job consumes this ConfigMap via
# envFrom in deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml.
apiVersion: v1
kind: ConfigMap
metadata:
  name: atlas-pr-cleanup-env
  namespace: argocd
  labels:
    app.kubernetes.io/part-of: atlas-pr-cleanup
data:
  DB_HOST: postgres.home
  DB_PORT: "5432"
  BOOTSTRAP_SERVERS: kafka.home:9093
  REDIS_URL: redis.home:6379
  ATLAS_DB_NAMES: "${ATLAS_DB_NAMES}"
  ATLAS_SERVICES: "${ATLAS_SERVICES}"
EOF

echo "Wrote $OUT"
```

- [ ] **Step 8.2: Make it executable and run it**

```bash
chmod +x deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh
deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh
```

Expected: prints `Wrote …/dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`.

- [ ] **Step 8.3: Inspect the generated artifact**

Run: `cat dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`

Expected: a ConfigMap manifest in `argocd` namespace whose
`ATLAS_SERVICES` value is the sorted, comma-joined list of every
service in `.github/config/services.json` (atlas-account first
alphabetically, atlas-wz-extractor near the end). `ATLAS_DB_NAMES`
matches the static string in `postdelete-cleanup.yaml` exactly.

- [ ] **Step 8.4: Idempotency check**

Run the generator a second time:

```bash
deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh
git diff --exit-code dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml
```

Expected: `git diff` exits 0 (no changes between runs).

- [ ] **Step 8.5: Commit**

```bash
git add deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh \
        dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml
git commit -m "task-075: generate cluster-infra ConfigMap coordination artifact

gen-cleanup-env.sh emits the atlas-pr-cleanup-env ConfigMap shape
cluster-infra must own. ATLAS_SERVICES is derived from
.github/config/services.json (sorted, comma-joined) so it stops
drifting from the build-side single source of truth. The generated
file is NOT deployed from this repo — it's a review artifact for
the cluster-infra sibling PR."
```

---

## Task 9 — postdelete-cleanup.yaml `envFrom: configMapRef:` switch

**Bugs addressed:** 4 (env-var centralization, part 2 of 2 — consumer side).

**Files:**
- Modify: `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml`
- Modify: `deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh` (comment-only)

- [ ] **Step 9.1: Edit postdelete-cleanup.yaml**

Replace lines 14-17 of
`deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml` (the
"Note: ATLAS_DB_NAMES and ATLAS_SERVICES are duplicated…" comment
block) with:

```yaml
# Static infra env vars (DB_HOST, DB_PORT, BOOTSTRAP_SERVERS, REDIS_URL,
# ATLAS_DB_NAMES, ATLAS_SERVICES) come from the long-lived
# `atlas-pr-cleanup-env` ConfigMap in the argocd namespace, owned by
# cluster-infra. The shape is mirrored from
# `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`,
# regenerated by deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh
# from .github/config/services.json. See task-075/context.md §4.
```

Replace lines 41-78 (the `envFrom:` and `env:` blocks) with:

```yaml
          envFrom:
            - secretRef:
                name: db-credentials
            - secretRef:
                name: pihole-credentials
            - secretRef:
                # New least-privilege PAT. Fine-grained, scoped to
                # Chronicle20/atlas with Contents: Read-and-write +
                # Metadata: Read + account-level Packages: Read-and-write
                # (or a classic PAT with repo + delete:packages — see
                # docs/runbooks/ephemeral-pr-deployments.md §9.5).
                # Replaces ghcr-pat (which carried only packages scope and
                # 403'd on bot-branch DELETE). Created in cluster-infra
                # sibling PR; same key name (GHCR_TOKEN) so cleanup.sh
                # doesn't change.
                name: atlas-pr-cleanup-gh-token
            - configMapRef:
                # Owned by cluster-infra (argocd namespace). See
                # dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml.
                name: atlas-pr-cleanup-env
          env:
            # PR_NUMBER is the sole per-PR substitution. ATLAS_ENV is
            # derived inside cleanup.sh via lib.sh::compute_atlas_env, so
            # any drift on the Application's atlas.env annotation is
            # harmless (bug #4 defensive fix; see task-070/design.md §3.4).
            - name: PR_NUMBER
              value: "PLACEHOLDER_PR_NUMBER"
```

- [ ] **Step 9.2: Update the comment in gen-consumer-group-patch.sh**

In `deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh`,
replace lines 30-35 (the "Note: pattern 2 services" block) with:

```bash
        # Note: pattern 2 services (atlas-channel, atlas-login) carry
        # the template at runtime. The emitted patch's
        # KAFKA_CONSUMER_GROUP value still contains a literal "%s" —
        # that "%s" is intentional and is now substituted at
        # consumer-registration time by libs/atlas-kafka/consumergroup
        # `Resolve(template, channelId)` (task-075). DO NOT strip the
        # "%s" here; the Go side needs it intact.
```

- [ ] **Step 9.3: Re-render the consumer-group patch and confirm no behavioural change**

Run: `deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh`

Run: `git diff deploy/k8s/overlays/pr/patches/consumer-group-env.yaml`

Expected: no changes to `patches/consumer-group-env.yaml`. The
generator's output shape is unaffected — only the comment moved.

- [ ] **Step 9.4: Validate the rendered postdelete-cleanup with kustomize**

Run:

```bash
kubectl kustomize deploy/k8s/overlays/pr-cleanup > /tmp/pr-cleanup-rendered.yaml
grep -E 'envFrom:|configMapRef:|atlas-pr-cleanup-env|name: PR_NUMBER|name: DB_HOST' /tmp/pr-cleanup-rendered.yaml
```

Expected: render succeeds, and the grep output shows `envFrom:`,
`configMapRef:`, `name: atlas-pr-cleanup-env`, `name: PR_NUMBER`
present, while `name: DB_HOST` (and the other inlined infra vars)
are absent.

(If `kubectl kustomize` isn't installed locally, `kustomize build` is equivalent.)

- [ ] **Step 9.5: Commit**

```bash
git add deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml \
        deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
git commit -m "task-075: source cleanup Job env from atlas-pr-cleanup-env

Inline env: for static infra vars (DB_HOST, ATLAS_DB_NAMES,
ATLAS_SERVICES, ...) is replaced with envFrom: configMapRef:
{ name: atlas-pr-cleanup-env }. The ConfigMap is owned by
cluster-infra (sibling PR) and its shape is mirrored from
dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml.
PR_NUMBER remains inline (per-PR substitution).

gen-consumer-group-patch.sh's comment is rewritten to reflect that
the literal %s in the emitted patch is now substituted at runtime
by Resolve(template, channelId)."
```

---

## Task 10 — Wire `gen-cleanup-env.sh` into pr-validation workflow

**Bugs addressed:** 4 (closes the loop — CI now fails when the coordination artifact drifts).

**Files:**
- Modify: `.github/workflows/pr-validation.yml`

- [ ] **Step 10.1: Locate the update-pr-overlay job's last step before the bot-branch push**

In `.github/workflows/pr-validation.yml`, the `update-pr-overlay`
job's "Bump image tags for built services" step ends around line
342. The next step is "Force-push bot/pr-<N>-resolved" at
line 344. Insert a new step between them.

- [ ] **Step 10.2: Insert the gen-cleanup-env step**

After the "Bump image tags for built services" step (the one whose
shell ends with `yq '.images[0:5]' "$OVERLAY"`), insert:

```yaml
      - name: Regenerate cluster-infra coordination ConfigMap artifact
        run: |
          set -euo pipefail
          deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh
          # The output file is checked in. If a service was added or
          # removed since the last regen, this run's output diverges
          # from the committed copy and CI fails — forcing the PR
          # author to commit the regen alongside their services.json
          # change.
          if ! git diff --exit-code dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml; then
              echo '::error::dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml is stale. Run deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh and commit the result.' >&2
              exit 1
          fi
```

- [ ] **Step 10.3: Lint the workflow with yq (or `yamllint` if available)**

Run: `yq eval '.jobs."update-pr-overlay".steps | length' .github/workflows/pr-validation.yml`
Expected: prints the step count (one more than before).

Run: `yq eval '.jobs."update-pr-overlay".steps[] | .name' .github/workflows/pr-validation.yml | grep "Regenerate cluster-infra"`
Expected: prints `Regenerate cluster-infra coordination ConfigMap artifact`.

- [ ] **Step 10.4: Commit**

```bash
git add .github/workflows/pr-validation.yml
git commit -m "task-075: fail PR CI when cleanup-env artifact is stale

The new update-pr-overlay step runs gen-cleanup-env.sh and
asserts the result equals the committed copy. Adding a service
to .github/config/services.json without regenerating the
coordination artifact now fails CI instead of silently shipping
a drifted ATLAS_SERVICES."
```

---

## Task 11 — Runbook updates

**Bugs addressed:** 2 (sweep-orphans now exists in image), 4 (operator no longer copy-pastes env), 5 (try-all summary changes recovery flow).

**Files:**
- Modify: `docs/runbooks/ephemeral-pr-deployments.md`

The existing section index (from `grep -nE '^##? ' docs/runbooks/ephemeral-pr-deployments.md`) is:

- §9.4 Recovery when teardown wedges — line 190 (subsections Diagnose / Recover / Source-branch-missing scenario)
- §9.11 Orphan sweep — line 329 (subsections One-shot from a workstation / In-cluster / Metric)

- [ ] **Step 11.1: Reword §9.4 Diagnose subsection**

In `docs/runbooks/ephemeral-pr-deployments.md`, find the
`### Diagnose` subsection under `## §9.4 Recovery when teardown wedges`
and add a new paragraph at the top of that subsection (before any
existing content):

```markdown
**Read the summary line first.** As of task-075, `cleanup.sh` runs every
phase regardless of any single phase's outcome. The final log line is
the authoritative status:

```
{"ts":…,"level":"info","atlas.env":"…","atlas.step":"done","msg":"cleanup complete phases_run=7 phases_failed=0"}
```

or, on partial failure:

```
{"ts":…,"level":"error","atlas.env":"…","atlas.step":"done","msg":"cleanup completed with errors phases_run=7 phases_failed=2 failed_phases=[\"drop-topics\",\"drop-redis\"]"}
```

Use the `failed_phases` array to scope your re-run — only the listed
phases need a manual recovery pass.  Every other phase ran to
completion (look for its `phase complete` log line). Pre-task-075
runbooks said "assume every phase after the failed one was skipped"; that
assumption no longer applies.
```

- [ ] **Step 11.2: Reshape §9.11 to use envFrom + Job manifest**

In `docs/runbooks/ephemeral-pr-deployments.md`, find the
`### One-shot from a workstation` subsection under `## §9.11 Orphan sweep`.

Replace the existing content of that subsection with:

```markdown
For one-off recovery you can run the image directly from a workstation
with cluster credentials (kubeconfig pointing at the prod cluster's
`argocd` namespace). The Job manifest form below mirrors the
PostDelete cleanup Job's shape (envFrom the cluster-infra-owned
ConfigMap; PR_NUMBER as the only per-invocation override). It is
preferred over `kubectl run --rm -i` — non-TTY pods don't always stream
logs reliably, and a Job leaves an inspectable record.

Apply this manifest (substitute `PR_NUMBER`):

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  generateName: atlas-pr-cleanup-oneshot-
  namespace: argocd
spec:
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      serviceAccountName: atlas-pr-cleanup
      containers:
        - name: cleanup
          image: ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest
          command: ["/atlas/cleanup.sh"]
          envFrom:
            - secretRef: { name: db-credentials }
            - secretRef: { name: pihole-credentials }
            - secretRef: { name: atlas-pr-cleanup-gh-token }
            - configMapRef: { name: atlas-pr-cleanup-env }
          env:
            - name: PR_NUMBER
              value: "<PR_NUMBER>"
```

Pipe through `kubectl -n argocd create -f -` (no `apply`; oneshot
Jobs use `generateName`). Tail logs with:

```bash
kubectl -n argocd logs -l app.kubernetes.io/part-of=atlas-pr-cleanup --tail=-1 -f
```

The workstation no longer needs to export `DB_HOST`, `BOOTSTRAP_SERVERS`,
`ATLAS_DB_NAMES`, `ATLAS_SERVICES`, etc. — those come from the
cluster-infra-owned `atlas-pr-cleanup-env` ConfigMap. `PR_NUMBER` is
the only value you supply.
```

- [ ] **Step 11.3: Update the §9.11 In-cluster subsection**

In the same file, find `### In-cluster (preferred for production cluster credentials)`. Replace its sweep-orphans.sh invocation block (the section that mounts a ConfigMap with the script) with:

```markdown
`/atlas/sweep-orphans.sh` is part of the published bootstrap image as of
task-075. The legacy `kubectl create configmap` + script-mount workaround
is no longer needed.

```bash
kubectl -n argocd run sweep-orphans \
    --rm -i --restart=Never \
    --serviceaccount=atlas-pr-cleanup \
    --image=ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest \
    --overrides='{
      "spec": {
        "containers": [{
          "name": "sweep-orphans",
          "image": "ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest",
          "command": ["/atlas/sweep-orphans.sh", "--apply", "<PR_NUMBER>"],
          "envFrom": [
            {"secretRef": {"name": "db-credentials"}},
            {"secretRef": {"name": "pihole-credentials"}},
            {"secretRef": {"name": "atlas-pr-cleanup-gh-token"}},
            {"configMapRef": {"name": "atlas-pr-cleanup-env"}}
          ]
        }]
      }
    }'
```

Drop `--apply` (or pass `--list` explicitly) to enumerate without
deleting. The script's Kafka phases use rpk as of task-075; the previous
"kafka-topics.sh not on PATH; skipping" warning is gone.
```

- [ ] **Step 11.4: Add the new §9.12 subsection**

Append a new top-level section to `docs/runbooks/ephemeral-pr-deployments.md` after `## §9.11 Orphan sweep`:

```markdown
## §9.12 Diagnosing partial-cleanup failure

As of task-075 the PostDelete Job runs every phase regardless of any
single phase's outcome. The summary line names which phases failed:

```
cleanup completed with errors phases_run=7 phases_failed=2 failed_phases=["drop-topics","drop-redis"]
```

Re-run only the failed phases via the §9.11 sweep-orphans path with
`--apply`, or manually:

| Phase | Manual re-run |
|---|---|
| `drop-dbs` | `psql -h postgres.home -U <user> -c 'DROP DATABASE IF EXISTS "atlas-<base>-<env>";'` (per leaked DB) |
| `drop-topics` | `rpk topic list -X brokers=kafka.home:9093 --format json \| jq -r '.[].name' \| grep -- '-<env>$' \| xargs -r -n1 rpk topic delete -X brokers=kafka.home:9093` |
| `drop-groups` | `rpk group list -X brokers=kafka.home:9093 --format json \| jq -r '.[].name' \| grep -- '\[<env>\]$' \| xargs -r -d '\n' -n1 rpk group delete -X brokers=kafka.home:9093` |
| `drop-redis` | `redis-cli -u redis://redis.home:6379 --scan --pattern '<env>:*' \| xargs -r -n 1000 redis-cli -u redis://redis.home:6379 DEL` |
| `drop-images` | See §9.5 GHCR token; the image-cleanup phase of `/atlas/sweep-orphans.sh --apply <PR>` is the canonical re-run path |
| `drop-dns` | Pi-hole admin UI on each replica; remove A records ending `… <PR_NUMBER>.atlas.home` |
| `drop-branch` | `gh api --method DELETE /repos/Chronicle20/atlas/git/refs/heads/bot%2Fpr-<PR>-resolved` |

The full re-run path (`/atlas/sweep-orphans.sh --apply <PR>`) is
idempotent and is the recommended recovery — it touches every phase
again with `WHERE NOT EXISTS`-equivalent semantics. The per-phase
recipes above are for cases where the operator wants to address a
single phase in isolation (e.g. the rpk broker is the only thing that
was unavailable during cleanup).

## §9.13 Coordination with cluster-infra

This repo (`Chronicle20/atlas`) deploys per-PR resources into
`atlas-pr-<N>` namespaces. Long-lived `argocd`-namespace dependencies
are owned by the cluster-infra repo. The atlas repo expects these to
already exist in `argocd`:

- `ServiceAccount atlas-pr-cleanup` + `Role` / `RoleBinding` granting
  the PostDelete Job permission to query+patch Applications.
- `Secret atlas-pr-cleanup-gh-token` (fine-grained PAT for GHCR + bot
  branch delete).
- `ConfigMap atlas-pr-cleanup-env` — shape mirrored from
  `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`.

When a new service is added to `.github/config/services.json`,
`gen-cleanup-env.sh` regenerates the example artifact and CI fails
the PR until the artifact is committed. Once that PR merges,
cluster-infra mirrors the new shape into the live ConfigMap. Order
of merges matters: cluster-infra changes land BEFORE the consuming
atlas PR, otherwise the next PostDelete Job wedges with
`CreateContainerConfigError: configmap "atlas-pr-cleanup-env" not found`.
```

- [ ] **Step 11.5: Verify the markdown still has unique headings**

Run: `grep -nE '^## §9\.' docs/runbooks/ephemeral-pr-deployments.md`
Expected output (in order): §9.1, §9.1b, §9.2, §9.3, §9.4, §9.5, §9.6, §9.7, §9.8, §9.9, §9.10, §9.11, §9.12, §9.13.

- [ ] **Step 11.6: Commit**

```bash
git add docs/runbooks/ephemeral-pr-deployments.md
git commit -m "task-075: runbook updates for try-all + envFrom + sweep image

§9.4 Diagnose: reframe around the summary line — failed_phases is
the authoritative status, not 'phases after failure were skipped'.
§9.11: workstation oneshot and in-cluster sweep both source env
from the atlas-pr-cleanup-env ConfigMap; the operator only types
PR_NUMBER. §9.12 new: per-phase manual re-run table for partial
failures. §9.13 new: cluster-infra coordination dependency list."
```

---

## Task 12 — Full local verification

**Bugs addressed:** none directly; this is the gate before requesting code review.

**Files:** none modified; verification only.

- [ ] **Step 12.1: Run the full bats suite**

Run: `bats services/atlas-pr-bootstrap/test/`
Expected: every test passes — at minimum `cleanup_test.bats` (10+), `sweep_test.bats` (6), `lib_test.bats` (9), `dockerfile_test.bats` (2), `bootstrap_test.bats` (2).

- [ ] **Step 12.2: Run Go tests with race for every touched module**

Run:

```bash
go test -race ./libs/atlas-kafka/...
go test -race ./services/atlas-channel/...
go test -race ./services/atlas-login/...
```

Expected: clean across all three modules.

- [ ] **Step 12.3: `go vet` each touched module**

Run:

```bash
go vet ./libs/atlas-kafka/...
go vet ./services/atlas-channel/...
go vet ./services/atlas-login/...
```

Expected: clean.

- [ ] **Step 12.4: `go build` each touched service**

Run:

```bash
go build ./libs/atlas-kafka/...
go build ./services/atlas-channel/...
go build ./services/atlas-login/...
```

Expected: clean.

- [ ] **Step 12.5: Docker buildx bake the touched services**

Per CLAUDE.md "Build & Verification", every service whose `go.mod`
or shared lib it depends on was touched needs a bake. Touched
modules: `libs/atlas-kafka` (consumed by every Go service that
uses Kafka — i.e. nearly all of them), `atlas-channel`,
`atlas-login`, plus `atlas-pr-bootstrap` (Dockerfile changed).

Run the broad bake to catch any missing `COPY libs/...` lines:

```bash
docker buildx bake atlas-channel atlas-login atlas-pr-bootstrap
```

Expected: each image builds. If a bake fails because of a missing
`COPY libs/...` in the shared root Dockerfile, fix per CLAUDE.md
("two `COPY` lines plus one `./libs/<name>` line in `go.work`").

If you want a broader sanity check (every Go service that depends on
atlas-kafka):

```bash
docker buildx bake all-go-services
```

Expected: every service builds. Allow time — this is the slow path.

- [ ] **Step 12.6: Pre-PR code review**

Per CLAUDE.md "Code Review Before PR" plus the task-075 audit
pattern, invoke `superpowers:requesting-code-review` which
dispatches:

- `plan-adherence-reviewer` (verifies every checked task in
  `plan.md` was actually implemented).
- `backend-guidelines-reviewer` (Go DOM-* — relevant for the
  `libs/atlas-kafka/consumergroup` change and the two main.go
  call-site edits).

Address findings written to `docs/tasks/task-075-pr-teardown-regressions/audit.md`.

- [ ] **Step 12.7: Open the PR**

Use `/finishing-a-development-branch` (or `superpowers:finishing-a-development-branch`)
to create the PR. The description MUST include:

- A link to the sibling cluster-infra PR that ships the
  `atlas-pr-cleanup-env` ConfigMap (fill in at PR-open time).
- A statement that cluster-infra's PR must merge first; otherwise
  the next PostDelete Job wedges with
  `CreateContainerConfigError: configmap "atlas-pr-cleanup-env" not found`.
- A pointer to `context.md` §4 for the ConfigMap shape.

---

## Self-Review

### Spec coverage

Mapping each PRD/design requirement to a task:

| Requirement | Task |
|---|---|
| 4.1 fix rpk jq schema mismatch in cleanup.sh | Task 2 (constants + fixtures), Task 3 (cleanup.sh refactor) |
| 4.1 fixture files exist with regeneration comment | Task 2 |
| 4.1 fail-fast bats test | Task 3 (`cleanup.sh fails fast on malformed rpk output`) |
| 4.1 Dockerfile RPK_VERSION comment | Task 5 |
| 4.2 Dockerfile COPY sweep-orphans.sh | Task 5 |
| 4.2 drift-guard bats test | Task 5 |
| 4.2 image build verifies sweep present | Task 5.5 / Task 12.5 |
| 4.3 sweep_kafka uses rpk | Task 4 |
| 4.3 sweep_test.bats stubs rewritten | Task 4 |
| 4.3 no kafka-*.sh references remain | Task 4.2 |
| 4.4 postdelete-cleanup.yaml envFrom | Task 9 |
| 4.4 gen-cleanup-env.sh generator | Task 8 |
| 4.4 coordination artifact under dev/ | Task 8 |
| 4.4 update-pr-overlay wires the generator + diff check | Task 10 |
| 4.4 runbook §9.11 reshape | Task 11 |
| 4.4 context.md sibling-PR documentation | context.md §4 |
| 4.5 cleanup.sh try-all + record_error | Task 1 (helpers), Task 3 (consumer) |
| 4.5 cleanup.sh fail-fast on Postgres unreachable retained | Task 3 (do_drop_dbs probe) |
| 4.5 try-all bats test | Task 3 (`runs every phase even when drop-topics fails`) |
| 4.5 happy-path summary bats test | Task 3 (`exits 0 when all phases succeed`) |
| 4.5 sweep-orphans.sh same try-all treatment | Task 4 (header drops -e; phases already isolated) |
| 4.6 Resolve variadic | Task 6 |
| 4.6 resolver tests | Task 6 |
| 4.6 atlas-channel call-site | Task 7 |
| 4.6 atlas-login call-site | Task 7 |
| 4.6 gen-consumer-group-patch comment | Task 9 |
| 4.6 cleanup-side regex match for substituted name | Covered by Task 3 (regex unchanged) + Task 6 tests |
| 4.7 runbook §9.4 reword | Task 11 |
| 4.7 §9.11 reshape | Task 11 |
| 4.7 §9.12 new | Task 11 |
| 4.7 cluster-infra coordination subsection | Task 11 (§9.13) |
| 4.8 coordination artifact (non-deployed example) | Task 8 |
| 4.8 context.md ConfigMap shape | context.md §4 |
| Design 3.7 runbook subsections | Task 11 |

### Placeholder scan

No `TBD`, `TODO`, `implement later`, `fill in details`, `Add appropriate error handling`, `Similar to Task N`, or unspecified handlers remain in the plan. Every code step shows the actual code. Every command shows the exact invocation and expected output.

### Type / signature consistency

- `Resolve(defaultName string, args ...any) string` — same signature in Task 6 (definition) and Tasks 6.1 (tests) / 7 (callers).
- `record_error <phase> <msg>`, `run_phase <phase_name> <function_name>`, `summarize_phases <total>` — consistent across Task 1 (definition), Task 1 tests, Task 3 (cleanup.sh orchestration), Task 4 (sweep-orphans header changes).
- `RPK_TOPICS_JQ` / `RPK_GROUPS_JQ` — defined in Task 2, used in Tasks 3 (cleanup.sh) and 4 (sweep-orphans.sh).
- Phase names — `drop-dbs`, `drop-topics`, `drop-groups`, `drop-redis`, `drop-images`, `drop-dns`, `drop-branch` — match the PHASES array in Task 3, the sweep-orphans.sh existing function names, the runbook §9.12 table in Task 11, and the bats assertions in Task 3.
- Fixture filenames — `rpk-topic-list.json` / `rpk-group-list.json` — match between Task 2 (creation), Task 3 (cleanup_test.bats reading), and Task 5's `test/fixtures/README.md` references.

No drift detected.
