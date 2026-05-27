# task-075 — PR-Env Teardown Regressions — Design

Version: v1
Status: Draft
Created: 2026-05-22
Inputs: `prd.md` (this folder)
---

## 1. Scope recap

Six independent regressions made PR-544's teardown leak per-env state and wedge the Argo CD Application in `Terminating` for ~75 minutes. Each regression is small; their interaction is the failure mode. This design addresses all six in one PR because they share three integration surfaces — the `atlas-pr-bootstrap` image, the bats test harness, and the cleanup contract documented in the runbook — and splitting them into six PRs would force three rounds of coordinated rework against the same files.

The design is organised around four work zones:

1. **Bash phase-runner refactor** (Bugs 1, 3, 5) — `cleanup.sh` + `sweep-orphans.sh` share a single phase orchestration helper that records errors instead of aborting, and both use the same rpk-based Kafka phases.
2. **Image / fixture surface** (Bugs 1, 2, 3) — the Dockerfile copies every script, rpk's JSON shape is pinned to committed fixtures, and the bats stubs replay those fixtures.
3. **Env-var centralization** (Bug 4) — `postdelete-cleanup.yaml` switches to `envFrom: configMapRef:`, and `update-pr-overlay` emits a coordination artifact that mirrors the ConfigMap shape cluster-infra must own.
4. **Consumer-group `%s` substitution** (Bug 6) — `consumergroup.Resolve` gains a variadic argument that `fmt.Sprintf`s the env-var value when args are passed; atlas-channel + atlas-login move their per-channel ID into that varargs slot.

The Go change in zone 4 is the only behavioural change outside the bootstrap image and overlay; everything else is bash, kustomize, and docs.

## 2. Open-question resolutions (from PRD §9)

These are the design-phase decisions referenced by the PRD. Each is committed here so plan-task and execute-task don't reopen them.

- **Q1. Coordination artifact shape.** Emit the `atlas-pr-cleanup-env` ConfigMap as a **documentation-only example** under `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`. Reasons: (a) the `argocd` namespace is cluster-infra's territory and adding a deployable manifest from this repo violates the existing "atlas repo deploys atlas-pr-<N> namespace; cluster-infra deploys argocd-namespace infra" boundary task-070 established; (b) the file is review-time visibility only — cluster-infra's reviewer reads this file and mirrors it into their repo. The header comment names it `Not deployed from this repo` so a future maintainer can't mistake the path for a kustomize root.
- **Q2. `record_error` payload format.** Plain text per phase, JSON only in the final summary. Reasons: per-phase `record_error <phase> <msg>` already routes through `log error`, which is JSON-encoded by `lib.sh::log`. The summary collects `phase` names (no caller-supplied text) into a JSON array via `printf '%s\n' "${ERRORS[@]}" | jq -Rsc 'split("\n") | map(select(length>0))'`. This avoids per-phase bash escaping while keeping the summary grep-friendly and machine-parseable.
- **Q3. atlas-login template.** Confirmed by reading `services/atlas-login/atlas.com/login/main.go:43` — the template is `"ChannelConnect Service - %s"` and is consumed at `:66` with the same `consumergroup.Resolve(fmt.Sprintf(template, config.Id.String()))` shape as atlas-channel. The fix is one-line in both services.
- **Q4. Fixture regeneration.** Static JSON, hand-checked-in, with a `# regenerate by:` header line. Reasons: (a) bumping rpk is the *event* that needs to force review of the fixture, so requiring a hand-edit (or running `rpk topic list --format json > test/fixtures/rpk-topic-list.json` and re-running bats) is the correct friction; (b) a helper script that calls a live broker breaks bats hermeticity. The Dockerfile gets a one-line comment near `ARG RPK_VERSION` pointing at the fixtures.
- **Q5. Dockerfile drift guard.** Co-locate in bats (`test/dockerfile_test.bats` — new file). Reasons: (a) bats is already the test entry point for this image; (b) the assertion is a simple "for each `scripts/*.sh`, the Dockerfile contains a `COPY scripts/<name>` line" loop that fits in 15 lines of bats; (c) a separate `tools/` script would need its own CI invocation. Bats is the cheapest delivery.

## 3. Architecture

### 3.1 The phase-runner pattern (Bugs 1, 3, 5)

Both `cleanup.sh` and `sweep-orphans.sh` are refactored around a shared mini-framework added to `lib.sh`:

```bash
# lib.sh additions (sketch):
declare -ga ATLAS_PHASE_ERRORS=()

# record_error <phase> <msg>
#   Appends "<phase>" to ATLAS_PHASE_ERRORS and logs at level=error.
record_error() {
    local phase="$1"; shift
    ATLAS_PHASE_ERRORS+=("$phase")
    ATLAS_STEP="$phase" log error "$@"
}

# run_phase <phase_name> <function_name>
#   Logs a start line, runs the function, and records errors on non-zero return.
#   Returns 0 always (so set -e wouldn't fire even if it were on).
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
#   Emits one JSON summary line and exits 0 (success) or 1 (errors recorded).
summarize_phases() {
    local total="$1"
    local failed="${#ATLAS_PHASE_ERRORS[@]}"
    local failed_json
    failed_json=$(printf '%s\n' "${ATLAS_PHASE_ERRORS[@]+"${ATLAS_PHASE_ERRORS[@]}"}" \
        | jq -Rsc 'split("\n") | map(select(length>0))')
    if [ "$failed" -eq 0 ]; then
        ATLAS_STEP=done log info "cleanup complete phases_run=$total phases_failed=0"
        return 0
    fi
    ATLAS_STEP=done log error "cleanup completed with errors phases_run=$total phases_failed=$failed failed_phases=$failed_json"
    return 1
}
```

Script header changes from `set -euo pipefail` to `set -uo pipefail` (dropping `-e`). The `init` block before the first phase keeps strict semantics by calling `require_env` (which `exit 1`s on its own when a var is missing) and by hard-exiting if `compute_atlas_env` fails. `drop-dbs`'s "Postgres unreachable" branch is treated as fatal via an explicit `exit 1` inside `do_drop_dbs` *before* the per-DB loop: if the very first `psql -d postgres -c "SELECT 1"` probe fails, cleanup-targeting is broken and no other phase can be trusted. Per-DB drop failures inside the loop are routed through `record_error` and continue.

Each phase function (`do_drop_dbs`, `do_drop_topics`, `do_drop_groups`, `do_drop_redis`, `do_drop_images`, `do_drop_dns`, `do_drop_branch`) takes no arguments and reads globals (`ATLAS_ENV`, `PR_NUMBER`, …). Functions return non-zero on failure; `run_phase` catches that and records the phase name. Internal `set -e`-style fatality is achieved per phase by using `|| record_error <phase> <msg> ; return 1`.

The top-level orchestration loop in `cleanup.sh` looks like:

```bash
PHASES=(drop-dbs do_drop_dbs drop-topics do_drop_topics drop-groups do_drop_groups \
        drop-redis do_drop_redis drop-images do_drop_images drop-dns do_drop_dns \
        drop-branch do_drop_branch)
TOTAL=$(( ${#PHASES[@]} / 2 ))
for ((i=0; i<${#PHASES[@]}; i+=2)); do
    run_phase "${PHASES[i]}" "${PHASES[i+1]}"
done
summarize_phases "$TOTAL"
exit $?
```

`sweep-orphans.sh` reuses `run_phase`/`record_error`/`summarize_phases` but loops over PR numbers as the outer loop and over phases as the inner loop. Each phase's enumerator is updated to use rpk where applicable (see §3.2).

Trade-off considered and rejected: a `trap ERR` global handler that records the failing phase via `ATLAS_STEP`. Rejected because `trap ERR` interacts poorly with subshells in pipelines (the very thing rpk → jq → xargs uses), and because explicit `run_phase` makes the orchestration visible at the top of the script — a future reader can grep `run_phase` and see the full phase list in one place.

### 3.2 Shared rpk jq queries (Bug 1, Bug 3)

Two query constants are added to `lib.sh`:

```bash
# rpk 24.3.1 emits a flat array for topic/group list. See test/fixtures/rpk-*.json.
# Bumping ARG RPK_VERSION in the Dockerfile invalidates these fixtures —
# regenerate against the new rpk and re-run bats.
readonly RPK_TOPICS_JQ='.[].name'
readonly RPK_GROUPS_JQ='.[].name'
```

Both `cleanup.sh:drop-topics` and `sweep-orphans.sh:sweep_kafka` reference `$RPK_TOPICS_JQ` and `$RPK_GROUPS_JQ`. A single edit moves the schema (or any future schema change) to one place. The constants live in `lib.sh` rather than the scripts because both scripts source `lib.sh`.

The fail-fast subcase from PRD §4.1 — "malformed rpk output should make the script exit non-zero with a recognizable error, not silently skip" — is handled by *not* wrapping the rpk-pipe with `|| true`. The current `cleanup.sh:58` `|| true` is on the grep step (no-match is fine and idempotent); the upstream rpk-and-jq pipe is allowed to fail. A jq schema error returns non-zero from jq, the pipeline exits non-zero, and the phase function returns non-zero — which `run_phase` records and reports in the summary. The "fail-fast" semantic from §4.1 thus folds into the try-all framework: malformed JSON makes `drop-topics` the failed phase, not a silent skip; the other six phases still run.

### 3.3 Kafka phases use rpk on both scripts (Bug 3)

`sweep_kafka` in `sweep-orphans.sh` is rewritten to mirror `cleanup.sh`'s post-fix shape:

```bash
sweep_kafka() {
    local pr_number="$1" env_hash="$2"
    [ -z "${BOOTSTRAP_SERVERS:-}" ] && return 0
    if ! command -v rpk >/dev/null 2>&1; then
        ATLAS_ENV="$env_hash" ATLAS_STEP=drop-topics log warn "rpk not on PATH; skipping"
        return 0
    fi

    ATLAS_STEP=drop-topics log info "scanning Kafka topics"
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

    ATLAS_STEP=drop-groups log info "scanning Kafka consumer groups"
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

The two `command -v kafka-topics.sh` / `command -v kafka-consumer-groups.sh` gates are deleted. Grep guard test: `grep -rE 'kafka-(topics|consumer-groups)\.sh' services/atlas-pr-bootstrap/` returns empty after this change; a bats assertion locks that in.

Argument-form note: `cleanup.sh` uses `xargs -d '\n' -n 1` to preserve spaced group names. `sweep_kafka` reads the group names through `while IFS= read -r g` (one full line per iteration) and passes each one to `rpk group delete` as a separate `"$g"` argument; spaces are preserved by quoting. The two scripts arrive at the same "delete spaced groups correctly" semantics through different syntax — both are correct because both pass the full name as one argv element. The bats test in §4 exercises a spaced group name end-to-end on each script.

### 3.4 Dockerfile + image surface (Bug 2)

The Dockerfile gains:

```dockerfile
COPY scripts/sweep-orphans.sh /atlas/sweep-orphans.sh
RUN chmod +x /atlas/bootstrap.sh /atlas/cleanup.sh /atlas/sweep-orphans.sh
```

The drift guard (new bats file `test/dockerfile_test.bats`) is one test:

```bats
@test "Dockerfile copies every script under scripts/" {
    for f in "$PROJECT_ROOT"/scripts/*.sh; do
        local base="$(basename "$f")"
        grep -qE "^COPY scripts/${base} /atlas/${base}\$" "$PROJECT_ROOT/Dockerfile" \
            || { echo "Dockerfile missing COPY for $base" >&2; return 1; }
    done
}
```

Comment near `ARG RPK_VERSION=24.3.1`:

```
# Bumping this invalidates test/fixtures/rpk-*.json. Regenerate with:
#   rpk topic list -X brokers=<broker> --format json > test/fixtures/rpk-topic-list.json
#   rpk group list -X brokers=<broker> --format json > test/fixtures/rpk-group-list.json
# and re-run `bats services/atlas-pr-bootstrap/test/`.
```

### 3.5 ConfigMap centralization + generator (Bug 4)

#### `postdelete-cleanup.yaml`

The inline `env:` for `DB_HOST`, `DB_PORT`, `BOOTSTRAP_SERVERS`, `REDIS_URL`, `ATLAS_DB_NAMES`, `ATLAS_SERVICES` is removed. `PR_NUMBER` stays inline (per-PR). The container declares:

```yaml
envFrom:
  - secretRef: { name: db-credentials }
  - secretRef: { name: pihole-credentials }
  - secretRef: { name: atlas-pr-cleanup-gh-token }
  - configMapRef: { name: atlas-pr-cleanup-env }
env:
  - name: PR_NUMBER
    value: "PLACEHOLDER_PR_NUMBER"
```

Header comment is rewritten to reference the cluster-infra-owned ConfigMap and to point at `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`.

#### `gen-cleanup-env.sh` (new)

Lives at `deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh`. Reads `.github/config/services.json`, extracts the `services[*].name` array, sorts ascendingly, joins with commas. Writes `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` deterministically. The output file is checked in (so the diff is reviewable). `pr-validation.yml`'s `update-pr-overlay` job invokes this script alongside `gen-consumer-group-patch.sh`, but the file is **not** consumed by kustomize on either overlay — cluster-infra reads it manually.

Schema of the generated YAML (this is what cluster-infra mirrors):

```yaml
# Not deployed from this repo. Mirror into cluster-infra. Generated by
# deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh from .github/config/services.json.
# Do not edit by hand — re-run the script after adding/removing a service.
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
  ATLAS_DB_NAMES: "atlas-accounts atlas-bans …"   # static, also in this script
  ATLAS_SERVICES: "atlas-account,atlas-asset-expiration,…"  # generated
```

`ATLAS_DB_NAMES` stays a static literal inside `gen-cleanup-env.sh` (it's not derivable from `services.json` — DB-name list is a kustomize-side concern). The script's header notes that `ATLAS_DB_NAMES` must stay in sync with `deploy/k8s/overlays/pr/kustomization.yaml`'s `atlas-db-names` configMapGenerator; that synchronization stays a manual review item (PRD non-goal: "Creating the `atlas-pr-cleanup-env` ConfigMap itself"). Future work — outside this task — could move the DB-name list into `.github/config/services.json` as well.

#### `gen-consumer-group-patch.sh` comment update

The current `:31-35` comment explaining that `%s` is intentionally preserved is rewritten to reflect that the Go side now formats at runtime. The generated patch shape doesn't change.

### 3.6 Consumer-group `%s` substitution (Bug 6)

#### `consumergroup.Resolve` signature

Current:
```go
func Resolve(defaultName string) string
```

New:
```go
func Resolve(defaultName string, args ...any) string
```

Behaviour matrix:

| `KAFKA_CONSUMER_GROUP` | `args` | Result |
|---|---|---|
| unset | none | `defaultName` |
| unset | non-empty | `fmt.Sprintf(defaultName, args...)` |
| set, non-empty | none | env value verbatim (preserves current behaviour) |
| set, non-empty | non-empty | `fmt.Sprintf(envValue, args...)` |
| set, whitespace-only | any | verbatim (preserves PRD §5.4 decision) |

This is additive: every existing caller passes zero args today, so `Resolve("Account Service")` → unchanged behaviour. The whitespace-only verbatim case from `resolver_test.go:TestResolve_envWhitespaceOnly_returnsVerbatim` is preserved by ordering the check (`v != ""` keeps the existing test green; whitespace-only is non-empty and skips `Sprintf` only because args are zero in the existing tests — the new test cases all use non-whitespace values).

#### Call-site changes

`services/atlas-channel/atlas.com/channel/main.go:151`:
```go
// before
var consumerGroupId = consumergroup.Resolve(fmt.Sprintf(consumerGroupIdTemplate, config.Id.String()))
// after
var consumerGroupId = consumergroup.Resolve(consumerGroupIdTemplate, config.Id.String())
```

`services/atlas-login/atlas.com/login/main.go:66`: analogous.

The `fmt` import in those files is still needed if used elsewhere; otherwise it's dropped. (atlas-channel uses `fmt` in many other places — keep. atlas-login: spot-check during execution.)

#### Compatibility considerations

- atlas-account, atlas-buddies, etc. that call `consumergroup.Resolve("Account Service")` keep working — zero args path returns `defaultName` unchanged.
- The literal `%s` in `gen-consumer-group-patch.sh`'s output (`KAFKA_CONSUMER_GROUP="Channel Service - %s [PLACEHOLDER_ATLAS_ENV]"`) is **intentional now** because the Go code formats at runtime.
- The cleanup-side grep `\\[${ATLAS_ENV}\\]\$` continues to match `"Channel Service - ch-7 [a1b2]"` exactly the same as `"Channel Service - %s [a1b2]"`.

#### Edge case: what if a future env value contains `%s` but the caller passes zero args?

`fmt.Sprintf("foo %s", nil)` returns `"foo %!s(<nil>)"` — undesirable. The chosen behaviour explicitly checks `len(args) > 0` before calling `fmt.Sprintf`; zero-args + env-value-with-`%s` returns the env value verbatim. This protects existing callers from accidental format-string surprises.

### 3.7 Runbook updates (PRD §4.7)

- §9.4 (recovery) — replaces "assume every phase after the failed one was skipped" with "read the `phases_failed=N failed_phases=[…]` summary line; each phase has its own success/error log line. Re-run only the failed phases."
- §9.11 (in-cluster one-shot) — replaces `kubectl run --rm -i` with a Job manifest example mirroring `postdelete-cleanup.yaml`'s shape (envFrom: configMapRef + per-PR `env: PR_NUMBER`). Argument: the runbook should show the operator the exact Job YAML they can `kubectl apply -f -` against, not a one-liner that has reliability gotchas.
- §9.12 (new) — "Diagnosing partial-cleanup failure": example summary line; how to identify the failing phase; per-phase manual rerun commands.
- New subsection "Coordination with cluster-infra" lists the manifests this repo expects in `argocd` namespace: `atlas-pr-cleanup-env` ConfigMap, `atlas-pr-cleanup` ServiceAccount + Role, the existing token Secret.

## 4. Test strategy

### 4.1 Bats — new fixture files

`services/atlas-pr-bootstrap/test/fixtures/rpk-topic-list.json`:

```json
[
  {"name": "boss-spawn-events", "partitions": 3, "replicas": 1},
  {"name": "boss-spawn-events-a1b2", "partitions": 3, "replicas": 1},
  {"name": "character-events-a1b2", "partitions": 6, "replicas": 1},
  {"name": "configurations-events", "partitions": 1, "replicas": 1}
]
```

`services/atlas-pr-bootstrap/test/fixtures/rpk-group-list.json`:

```json
[
  {"name": "Account Service", "members": 0},
  {"name": "Channel Service - 7e3a-… [a1b2]", "members": 1},
  {"name": "Party Quest Service [a1b2]", "members": 1},
  {"name": "Party Quest Service [other]", "members": 1}
]
```

Both files carry a leading JSON-comment-less convention plus a sibling `fixtures/README.md` documenting regeneration. The fixtures are scoped to `ATLAS_ENV=a1b2` and exercise: a non-suffixed topic (must NOT delete), an env-suffixed topic (must delete), an env-suffixed group with spaces (must delete intact), and a non-target-env-suffixed group (must NOT delete).

### 4.2 Bats — rewritten stubs

`make_stubs` in `cleanup_test.bats` is refactored to default to the fixture files instead of `{"topics":[…]}` literals:

```bash
make_stubs() {
    local topic_fixture="${1:-$PROJECT_ROOT/test/fixtures/rpk-topic-list.json}"
    local group_fixture="${2:-$PROJECT_ROOT/test/fixtures/rpk-group-list.json}"
    cp "$topic_fixture" "$BATS_TEST_TMPDIR/topic_list.json"
    cp "$group_fixture" "$BATS_TEST_TMPDIR/group_list.json"
    # rpk stub unchanged: it cats $BATS_TEST_TMPDIR/{topic,group}_list.json
    …
}
```

Existing tests that hand-spelled inline JSON (e.g., the "deletes only suffixed topics" test at `cleanup_test.bats:142`) are updated to copy a fixture-derived file and reference it. The `env_hash` helper (`fixture_env`) still computes against PR_NUMBER=99, but the fixture's `a1b2` literal is for the new tests below; existing tests inject the computed `env_hash` into their fixture variants via a sed pipeline if the existing assertion depends on a dynamically derived hash.

### 4.3 Bats — new tests

In `cleanup_test.bats`:

- `cleanup.sh runs every phase even when drop-topics fails` — rpk stub returns a non-JSON line on `topic list`, runs cleanup, asserts exit=1, asserts `drop-groups` / `drop-redis` / `drop-images` / `drop-dns` / `drop-branch` each emitted their "phase complete" log line, asserts summary line names `drop-topics` as failed.
- `cleanup.sh exits 0 when all phases succeed` — happy path with fixture data; asserts `phases_failed=0` summary line.
- `cleanup.sh fails fast on malformed rpk output` — rpk stub emits `<not-json>` on `topic list`; jq fails inside the pipe; assertion: exit code is 1, log contains a recognizable jq error, drop-groups still ran.

In `sweep_test.bats` (rewritten): every existing `kafka-topics.sh` / `kafka-consumer-groups.sh` stub becomes an `rpk` stub backed by the same fixture files as cleanup_test.bats. The "phase names appear in --list output" tests stay structurally identical with `rpk topic list` / `rpk group list` as the matched invocations.

In `test/dockerfile_test.bats` (new): the drift guard test from §3.4.

In `test/lib_test.bats` (extended): unit tests for `record_error`, `run_phase` (success + failure paths), `summarize_phases` (both branches).

### 4.4 Go tests — `consumergroup.Resolve`

`libs/atlas-kafka/consumergroup/resolver_test.go` adds:

- `TestResolve_envWithFormat_substitutes` — `KAFKA_CONSUMER_GROUP="Channel Service - %s [a1b2]"`, args `"ch-7"`, expected `"Channel Service - ch-7 [a1b2]"`.
- `TestResolve_envWithoutFormat_noArgs_verbatim` — already covered, stays.
- `TestResolve_defaultWithFormat_substitutes` — env unset, defaultName `"Channel Service - %s"`, args `"ch-7"`, expected `"Channel Service - ch-7"`.
- `TestResolve_zeroArgs_doesNotFormat` — env set to `"%s literal"`, args `()`, expected `"%s literal"` (no `%!s(MISSING)`).
- `TestResolve_envWhitespaceOnly_returnsVerbatim` — stays.

No additional Go-side integration test; the `gen-consumer-group-patch.sh` end-to-end is covered by the eventual PR-env smoke step in §5 of the PRD.

### 4.5 Integration / smoke

Out of scope for this PR's automated suite (`docker buildx bake atlas-pr-bootstrap` + bats + Go tests + `go vet ./...` cover the unit surface). The end-to-end "PR open → teardown → no orphans" loop happens when this PR itself merges; it is the deployment's acceptance test.

## 5. Migration / rollout

This task ships in one PR. Coordination with cluster-infra is required:

1. **This repo's PR** lands `postdelete-cleanup.yaml`'s `envFrom: configMapRef:` change.
2. **Cluster-infra's PR** lands the `atlas-pr-cleanup-env` ConfigMap in `argocd` namespace.

If this repo's PR merges before the cluster-infra ConfigMap exists, the next PostDelete Job fails with `CreateContainerConfigError: configmap "atlas-pr-cleanup-env" not found`. To prevent that ordering risk:

- The PR description (and `context.md` Coordination section that the plan emits) prominently states the dependency.
- This repo's PR title prefix is `task-075:` and the description links to the cluster-infra sibling PR by URL (filled in at PR-open time).
- The runbook §9.11 "Coordination with cluster-infra" subsection documents the order.

The Bug 6 Go change is independent of cluster-infra: it ships in the same PR and starts working immediately for any new PR-env deployment. Existing in-flight PR envs that already have atlas-channel pods running with the broken consumer-group keep their existing group names until the next deploy; this is acceptable because (a) those envs are ephemeral by definition, (b) the broken group name causes a cross-channel-within-pod isolation bug, not a data-loss bug, and (c) operators can cycle the affected pods to pick up the fix.

## 6. Risk register

| Risk | Mitigation |
|---|---|
| rpk 24.3.1's `--format json` schema differs between `topic list` and `group list` in a way the single shared `$RPK_TOPICS_JQ` / `$RPK_GROUPS_JQ` can't capture | Two separate constants exist precisely for this. The fixtures are derived from real rpk output, not hand-typed; a schema mismatch surfaces during fixture generation. |
| Phase-runner refactor loses an existing edge-case (e.g., the `psql … || { log error … ; exit 1; }` semantics for `drop-dbs`) | `do_drop_dbs` retains the hard-fail-on-unreachable-Postgres probe before the per-DB loop. The new bats "every phase runs" test only covers per-DB failures, not host-unreachable failures, so the unit test for unreachable Postgres is added explicitly. |
| Generated `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` drifts from `gen-cleanup-env.sh`'s actual output | `update-pr-overlay` job adds a `git diff --exit-code` check after running `gen-cleanup-env.sh`; a drifted file fails PR CI. Same pattern `gen-consumer-group-patch.sh` already uses (verify in execution). |
| `Resolve(defaultName, args ...any)` is a Go API change in a shared lib; any out-of-tree caller breaks | This is an additive variadic — zero-args callers are source-compatible. Confirmed by grep across all `consumergroup.Resolve(` call sites; every existing one passes exactly one string. |
| Cluster-infra sibling PR not landed before this repo's PR merges → next PR's PostDelete hook wedges with `configmap "atlas-pr-cleanup-env" not found` | Documented in §5 and in the PR description. Operationally enforceable via the runbook coordination subsection; a runtime fallback (try inline values, fall back to ConfigMap) is explicitly rejected as bug-bait. |

## 7. Non-goals (explicit)

- No new metrics (PRD §8 Observability).
- No new RBAC, secrets, or PATs.
- No structural change to task-070's namespace architecture.
- No rpk version bump.
- No restructure of `consumergroup.Resolve` for callers that don't pass args.
- No backport-style cleanup of state already leaked by past failed teardowns.
- No CronJob for ongoing sweep; cluster-infra already runs one.

## 8. File-touch matrix

| Path | Bug | Change type |
|---|---|---|
| `services/atlas-pr-bootstrap/Dockerfile` | 2 | +1 `COPY`, +1 `chmod` target, +1 fixture-regeneration comment |
| `services/atlas-pr-bootstrap/scripts/lib.sh` | 1,3,5 | + `record_error`, `run_phase`, `summarize_phases`, `RPK_TOPICS_JQ`, `RPK_GROUPS_JQ` |
| `services/atlas-pr-bootstrap/scripts/cleanup.sh` | 1,5 | Header `-e` drop; phases become functions; orchestration loop; summary call |
| `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` | 3,5 | Kafka phases ported to rpk; phases routed through `run_phase`; same summary call |
| `services/atlas-pr-bootstrap/test/cleanup_test.bats` | 1,5 | Stubs use fixtures; new try-all + fail-fast tests |
| `services/atlas-pr-bootstrap/test/sweep_test.bats` | 3 | Stubs rewritten from kafka-* to rpk |
| `services/atlas-pr-bootstrap/test/dockerfile_test.bats` (new) | 2 | Drift guard |
| `services/atlas-pr-bootstrap/test/lib_test.bats` | 5 | Unit tests for record_error / run_phase / summarize_phases |
| `services/atlas-pr-bootstrap/test/fixtures/rpk-topic-list.json` (new) | 1 | rpk schema fixture |
| `services/atlas-pr-bootstrap/test/fixtures/rpk-group-list.json` (new) | 1 | rpk schema fixture |
| `services/atlas-pr-bootstrap/test/fixtures/README.md` (new) | 1 | regen instructions |
| `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml` | 4 | `env:` → `envFrom: configMapRef:` |
| `deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh` (new) | 4 | Generator |
| `deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh` | 6 | Comment update only (no behavioural change) |
| `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` (new) | 4 | Coordination artifact (output of generator) |
| `.github/workflows/pr-validation.yml` | 4 | `update-pr-overlay` invokes `gen-cleanup-env.sh` + diff-exit-code check |
| `libs/atlas-kafka/consumergroup/resolver.go` | 6 | Variadic args |
| `libs/atlas-kafka/consumergroup/resolver_test.go` | 6 | 4 new test cases |
| `services/atlas-channel/atlas.com/channel/main.go` | 6 | One-line call-site change |
| `services/atlas-login/atlas.com/login/main.go` | 6 | One-line call-site change |
| `docs/runbooks/ephemeral-pr-deployments.md` | 2,4,5 | §9.4 reword, §9.11 reshape, §9.12 new, Coordination subsection |

## 9. Out-of-scope follow-ups (capture, don't act)

- Migrating `ATLAS_DB_NAMES` into `.github/config/services.json` so cluster-infra's ConfigMap has a single source of truth for it too.
- Bumping rpk past 24.3.1 (deferred; will require regenerating the fixtures).
- Promoting `dev/cluster-infra-coordination/` to a real generated-artifact pipeline (today it's a static-on-disk file).
- Surfacing the `summarize_phases` JSON to a Prometheus counter via a sidecar — out of scope, the JSON log is sufficient for Loki today.

---

End of design.
