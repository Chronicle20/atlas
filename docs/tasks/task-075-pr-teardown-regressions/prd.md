# PR-Env Teardown Regressions — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-22
---

## 1. Overview

`task-070-pr-env-teardown-fixes` shipped the structural fix for ephemeral PR teardown: the cleanup Job moved to the `argocd` namespace (escaping the finalizer-ordering wedge), the GHCR token was rotated to a least-privilege fine-grained PAT, and an in-cluster recovery sweep (`sweep-orphans.sh`) was added. That work landed in May 2026 and unblocked teardown for the median PR.

On 2026-05-22, **PR 544** merged. The PostDelete cleanup Job ran (proving task-070's architectural fix held), executed `drop-dbs` successfully (29 `DROP DATABASE IF EXISTS …`), then aborted at `drop-topics` with:

```
jq: error (at <stdin>:1): Cannot index array with string "topics"
```

Because `cleanup.sh` uses `set -euo pipefail`, this single jq error killed the script before any of `drop-groups`, `drop-redis`, `drop-images`, `drop-dns`, or `drop-branch` ran. The Job entered `Failed`. ArgoCD's `post-delete-finalizer.argocd.argoproj.io/cleanup` kept the Application in `Terminating` for **~75 minutes** until a controller reconcile drained the finalizer. All per-env state from the failed-mid phases was leaked: Kafka topics, consumer groups, Redis keys, GHCR tags, Pi-hole DNS, and the `bot/pr-544-resolved` branch.

The operator's recovery attempt also failed: the runbook's "in-cluster sweep" pointed at `/atlas/sweep-orphans.sh`, which doesn't exist in the published image (the Dockerfile never `COPY`s it). The workaround required `kubectl create configmap` + script mount + a Job manifest, and even that produced silent-skip warnings because `sweep-orphans.sh` still calls `kafka-topics.sh` (not in the image after task-070's rpk migration).

Post-incident investigation revealed **six independent findings**, all latent in the codebase prior to PR 544:

1. **`cleanup.sh` jq schema mismatch with rpk 24.3.1** — `.topics[]` and `.groups[]` index assumes object-shaped output, but rpk emits a flat array.
2. **`sweep-orphans.sh` is missing from the bootstrap image** — script exists in `scripts/`, Dockerfile never copies it.
3. **`sweep-orphans.sh` Kafka phases use `kafka-topics.sh` / `kafka-consumer-groups.sh`** — task-070 migrated `cleanup.sh` to rpk but not `sweep-orphans.sh`. The phases silently no-op with a warn log.
4. **Env-var defaults duplicated across `kustomization.yaml` and `postdelete-cleanup.yaml`** — `ATLAS_DB_NAMES` and `ATLAS_SERVICES` are inlined in both places. `ATLAS_SERVICES` is also disconnected from `.github/config/services.json` (the build-side single source of truth). The runbook's §9.11 operator one-shot has no shared ConfigMap to source from.
5. **Abort-first failure policy in `cleanup.sh`** — `set -euo pipefail` means any single phase failure leaks every subsequent phase. Bug 1 demonstrated this concretely: one jq error → six phases of leaked state.
6. **atlas-channel / atlas-login emit a literal `%s` in their Kafka consumer-group name in PR envs** — `consumergroup.Resolve` returns the `KAFKA_CONSUMER_GROUP` env var verbatim, and the patch generator emits `"Channel Service - %s [PLACEHOLDER_ATLAS_ENV]"`. The `%s` is never substituted at runtime, so every channel in every PR env joins a group literally named `Channel Service - %s [<env>]` — a real isolation bug across channels within a PR env.

The bats test suites for both `cleanup.sh` and `sweep-orphans.sh` were green throughout the incident because their stubs encode the wrong schemas (Bug 1) and stub the wrong binaries (Bug 3). The tests reinforced the bugs rather than catching them.

This task lands the six fixes plus a regression test approach that pins external-tool schemas to recorded fixtures so the next rpk bump (or kafka tarball migration) can't silently re-break teardown.

## 2. Goals

### Primary goals

- A PR merge triggers a teardown that runs every cleanup phase exactly once, regardless of any single phase's outcome. The Job exits non-zero iff at least one phase failed; ArgoCD sees the Failed state and the operator can investigate.
- `sweep-orphans.sh` is present in the published bootstrap image and uses the same rpk-based Kafka phases as `cleanup.sh`. The runbook's `/atlas/sweep-orphans.sh` reference matches reality.
- External-tool output schemas (rpk topic/group list) are pinned to committed JSON fixtures. The bats stubs replay those fixtures, so the next rpk-version bump that changes the schema fails the test instead of leaking state in production.
- `cleanup.sh` and `sweep-orphans.sh` source their static env defaults from a single ConfigMap (`atlas-pr-cleanup-env` in `argocd` namespace). `ATLAS_SERVICES` is generated from `.github/config/services.json` by `update-pr-overlay` so it stops drifting from the build-side source of truth.
- atlas-channel and atlas-login register Kafka consumer groups with their channel ID interpolated, even when `KAFKA_CONSUMER_GROUP` is set via the PR-overlay patch. No group name in PR env contains a literal `%s`.
- A new bats test exercises the "drop-topics fails, drop-groups still runs" path, locking in the try-all failure policy. Without that test, the policy can regress silently.
- The runbook's §9.4 (recovery) and §9.11 (operator one-shot sweep) match the new code: file paths, kubectl invocation form, ConfigMap source.

### Non-goals

- Revisiting task-070's namespace architecture (PostDelete in `argocd`, hook-delete-policy, finalizers). The wedge happened on top of working architecture; we don't restructure it.
- Migrating away from `set -euo pipefail` for the script header itself — phase-level error handling is added on top, the file-level "fail loud on unset var" stays.
- Bumping rpk to a newer version. The current `RPK_VERSION=24.3.1` pin stands; only the jq query that consumes its output changes.
- Backporting the fix to clean up state already leaked by past failed teardowns. Operators run `sweep-orphans.sh` per PR; the historical sweep is documented in task-070's `recovery-log.md` and is a separate operational concern.
- Restructuring `consumergroup.Resolve` for the non-templated cases. Services using the plain `Resolve("Service Name")` pattern already work correctly; only the `fmt.Sprintf` callers (atlas-channel, atlas-login) need the new variant.
- Creating the `atlas-pr-cleanup-env` ConfigMap itself. That manifest lives in the cluster-infra repo. This task ships the consumer-side changes (`envFrom: configMapRef` in `postdelete-cleanup.yaml`, runbook §9.11 update) and documents the required ConfigMap shape in a sibling-PR coordination note.
- A separate orphan-cleanup CronJob (already exists in cluster-infra). This task fixes the things it would otherwise sweep up.

## 3. User Stories

- **As a developer**, I want my merged PR's environment to fully tear down without leaking Kafka topics or DNS entries, so the cluster doesn't accumulate slow-burn capacity bills I never see.
- **As the cluster operator**, when teardown fails I want every phase to have made its best attempt before the Job exits, so a single bug in one phase doesn't multiply my recovery surface by six.
- **As the operator running `sweep-orphans.sh`**, I want the script to actually exist in the image the runbook tells me to use, and I want its Kafka phases to actually run instead of silently warning.
- **As an operator running a one-shot recovery sweep**, I want one `kubectl get cm` to give me every static env var the script needs, instead of copy-pasting from a YAML manifest.
- **As a CI maintainer**, I want bats tests that fail when rpk's JSON output schema changes, so the next external-tool bump can't silently break production cleanup.
- **As an atlas-channel developer**, I want per-channel consumer-group names in PR env to actually be per-channel (not all sharing one literal-`%s` name), so PR-env multi-channel tests reflect real production behavior.

## 4. Functional Requirements

### 4.1 Fix the rpk jq schema mismatch in cleanup.sh (Bug 1)

**Current behavior:** `services/atlas-pr-bootstrap/scripts/cleanup.sh:56-59` pipes `rpk topic list -X brokers="$BOOTSTRAP_SERVERS" --format json` through `jq -r '.topics[].name'`. rpk 24.3.1 emits a flat JSON array `[{"name":"…","partitions":…}, …]`, not an object with a `.topics` key, so jq aborts with `Cannot index array with string "topics"`. The analogous `drop-groups` block at `:68-71` has the same bug with `.groups[].name`. Both errors propagate as non-zero exit; `set -euo pipefail` kills the script at first failure.

**Required behavior:** The jq query must match rpk 24.3.1's actual output schema. The query must be derived from (and verified against) a committed fixture, not hand-typed.

**Acceptance criteria:**
- `cleanup.sh:drop-topics` uses a jq query that successfully extracts topic names from rpk 24.3.1's `topic list --format json` output (verified against a committed fixture).
- `cleanup.sh:drop-groups` uses a jq query that successfully extracts group names from rpk 24.3.1's `group list --format json` output (verified against a committed fixture).
- Two new fixture files exist under `services/atlas-pr-bootstrap/test/fixtures/`: one for `rpk topic list` (with a mix of `-<env>`-suffixed and unsuffixed topics) and one for `rpk group list` (with a `[<env>]`-suffixed group whose name contains spaces).
- The existing bats tests that currently hand-spell `{"topics":[…]}` are rewritten to `cat` the fixture file. A `make_stubs` helper change is acceptable.
- A new bats test, "cleanup.sh fails fast on malformed rpk output", pipes a recorded broken response (e.g., a non-JSON line) and asserts the script exits non-zero with a recognizable error — not a silent skip.
- Comment at the top of each fixture documents how to regenerate it (`rpk topic list -X brokers=… --format json > test/fixtures/rpk-topic-list.json`).
- A short comment near `ARG RPK_VERSION` in the Dockerfile points at the fixture files: "Bumping this version invalidates `test/fixtures/rpk-*.json`; regenerate against the new rpk and re-run bats."

### 4.2 Add sweep-orphans.sh to the bootstrap image (Bug 2)

**Current behavior:** `services/atlas-pr-bootstrap/Dockerfile:33-37` copies `lib.sh`, `bootstrap.sh`, `cleanup.sh`, and the `canonical/` directory. `scripts/sweep-orphans.sh` exists but is never copied. The runbook's `/atlas/sweep-orphans.sh` reference resolves to "no such file."

**Required behavior:** Every executable shell script in `services/atlas-pr-bootstrap/scripts/` lands at `/atlas/<name>` in the published image. A test ensures no future addition is omitted.

**Acceptance criteria:**
- Dockerfile adds `COPY scripts/sweep-orphans.sh /atlas/sweep-orphans.sh` and includes the path in its `chmod +x` line.
- A bats test ("Dockerfile copies every script") iterates over `scripts/*.sh` and asserts the Dockerfile has a `COPY scripts/<name>` line for each.
- The runbook's §9.11 in-cluster invocation matches the published path (`/atlas/sweep-orphans.sh`); no change to the path itself, but verify the runbook's command renders against the new image.
- An image build via `docker buildx bake atlas-pr-bootstrap` succeeds and the resulting image contains `/atlas/sweep-orphans.sh` (verifiable via `docker run --rm <image> ls /atlas`).

### 4.3 Port sweep-orphans.sh Kafka phases to rpk (Bug 3)

**Current behavior:** `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh:121-153` gates Kafka phases on `command -v kafka-topics.sh`. The Dockerfile installs rpk, not the Kafka tarball, so `kafka-topics.sh` is never on PATH; the phases log a warn and `return 0`. The sweep "succeeds" while reclaiming zero Kafka state. The bats sweep test stubs `kafka-topics.sh` and `kafka-consumer-groups.sh`, hiding the bug from CI.

**Required behavior:** `sweep_kafka()` uses rpk for both topic and group enumeration and deletion, mirroring `cleanup.sh`'s phases (post-Bug-1 fix). The `--apply` gate is honored — list mode enumerates without deleting.

**Acceptance criteria:**
- `sweep_kafka()` in `sweep-orphans.sh` replaces both `kafka-topics.sh --list` and `kafka-consumer-groups.sh --list` invocations with `rpk topic list --format json` and `rpk group list --format json`, parsed with the same jq queries `cleanup.sh` uses (single source for the query).
- Delete operations use `rpk topic delete` and `rpk group delete` with the same `xargs -d '\n' -n 1` pattern `cleanup.sh:71` uses (preserves spaced group names).
- List mode (default, no `--apply`) prints `drop-topics <name>` / `drop-groups <name>` lines for each match without deleting.
- `apply` mode performs deletion and tolerates a not-found result (idempotent re-run).
- The bats test `sweep_test.bats` is updated to stub `rpk` (using the same fixture files from §4.1) instead of `kafka-topics.sh` and `kafka-consumer-groups.sh`.
- The "phase names appear in --list output" bats test still passes against the rpk stubs.
- After the port, `command -v kafka-topics.sh` and `command -v kafka-consumer-groups.sh` no longer appear anywhere in `services/atlas-pr-bootstrap/` (verifiable via grep).

### 4.4 Centralize env-var defaults via shared ConfigMap (Bug 4)

**Current behavior:** `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml:65-78` inlines `DB_HOST`, `DB_PORT`, `BOOTSTRAP_SERVERS`, `REDIS_URL`, `ATLAS_DB_NAMES`, and `ATLAS_SERVICES`. `ATLAS_DB_NAMES` is duplicated from `deploy/k8s/overlays/pr/kustomization.yaml:230-232`. `ATLAS_SERVICES` is duplicated from `.github/config/services.json` (the build-side single source of truth, per CLAUDE.md). The runbook's §9.11 operator one-shot expects the operator to copy-paste values out of the manifest.

**Required behavior:** A long-lived ConfigMap `atlas-pr-cleanup-env` in the `argocd` namespace holds every static env var the cleanup Job needs. `postdelete-cleanup.yaml` switches from inline `env:` to `envFrom: configMapRef:`. The runbook's §9.11 example uses the same ConfigMap. `ATLAS_SERVICES` is generated from `.github/config/services.json` at `update-pr-overlay` time.

The ConfigMap itself is created by a sibling PR in the cluster-infra repo (Argo-managed, long-lived, namespace `argocd`). This task ships the consumer-side changes and the generator; it documents the cluster-infra dependency.

**Acceptance criteria:**
- `postdelete-cleanup.yaml`'s Job container declares `envFrom: - configMapRef: { name: atlas-pr-cleanup-env }` and removes the inline `env:` entries for `DB_HOST`, `DB_PORT`, `BOOTSTRAP_SERVERS`, `REDIS_URL`, `ATLAS_DB_NAMES`, `ATLAS_SERVICES`. `PR_NUMBER` stays inline (it's per-PR, not static).
- The comment at the top of `postdelete-cleanup.yaml` no longer warns about `ATLAS_DB_NAMES` / `ATLAS_SERVICES` duplication; instead it references the `atlas-pr-cleanup-env` ConfigMap and the cluster-infra repo's owning manifest.
- `deploy/k8s/overlays/pr-cleanup/kustomization.yaml` (or a new `gen-cleanup-env.sh` invoked by `update-pr-overlay`) emits a `ConfigMap atlas-pr-cleanup-env` candidate manifest under `dev/cluster-infra-coordination/` (or similar non-deployed staging path) for cluster-infra to mirror. This is a coordination artifact, not a deployed manifest in this repo.
- A new script `deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh` reads `.github/config/services.json`, extracts the service list, and emits the `ATLAS_SERVICES` line. `update-pr-overlay` (`.github/workflows/pr-validation.yml`) calls it on every PR sync, alongside the existing `gen-consumer-group-patch.sh` invocation. The script's output is committed-and-checked-in deterministically (sorted, stable).
- The runbook's §9.11 "One-shot from a workstation" section replaces the per-var export list with a `kubectl -n argocd run cleanup --rm --image=… --overrides='{"spec":...,"envFrom":[{"configMapRef":{"name":"atlas-pr-cleanup-env"}}]}'` invocation (or the equivalent Job manifest form). The list of env vars the operator types by hand is reduced to `PR_NUMBER` only.
- A `context.md` section "Sibling PR (cluster-infra)" documents the exact `ConfigMap atlas-pr-cleanup-env` shape required, mirrored from the coordination artifact this PR emits.

### 4.5 Try-all failure policy in cleanup.sh (Bug 5)

**Current behavior:** `services/atlas-pr-bootstrap/scripts/cleanup.sh:16` declares `set -euo pipefail`. Each phase (`drop-dbs`, `drop-topics`, `drop-groups`, `drop-redis`, `drop-images`, `drop-dns`, `drop-branch`) runs in the script's top-level scope. Any phase that exits non-zero kills the script; every subsequent phase is skipped. PR 544's drop-topics jq failure leaked everything from drop-groups through drop-branch.

**Required behavior:** Each phase runs in its own function. Phase failures are logged, recorded in an `ERRORS` array, and do not kill the script. After all phases run, the script inspects `ERRORS`; if non-empty, it logs a summary and exits 1. ArgoCD still sees `Failed` for the Job, but every phase got its attempt — and the operator's `kubectl logs` shows which phases failed instead of "everything after drop-topics."

The script header keeps `set -uo pipefail` (catch unset vars and pipe failures), but drops `-e`. Each phase function explicitly checks return codes and routes through a shared `record_error <phase> <msg>` helper. The `init`, `drop-dbs` (which still must succeed before phase ordering matters), and final summary phases retain stricter error handling.

**Acceptance criteria:**
- `cleanup.sh` header is `set -uo pipefail` (no `-e`). Each phase is wrapped in a function (`do_drop_topics`, `do_drop_groups`, …) called from a top-level orchestration loop.
- A helper `record_error <phase> <msg>` appends to a global `ERRORS` array and logs at `level=error` with the existing JSON log format.
- The final `cleanup complete` log line is replaced with a summary: on success, log `level=info msg="cleanup complete" phases_run=N phases_failed=0`; on partial failure, log `level=error msg="cleanup completed with errors" phases_failed=N` followed by the per-phase error lines, and exit 1.
- `drop-dbs` retains a hard-fail semantic if Postgres is genuinely unreachable (distinguishing from "a single DB doesn't exist") — that case still aborts before other phases run, because losing Postgres connectivity means cleanup-targeting is broken. Per-DB drop failures (e.g., a DB still has connections) log via `record_error` but don't abort. Treat unreachable host as fatal; treat per-DB failure as recorded-and-continue.
- A new bats test "cleanup.sh runs every phase even when drop-topics fails" stubs rpk to return malformed JSON in `topic list`, asserts the script's exit code is 1, asserts `drop-groups`, `drop-redis`, `drop-images`, `drop-dns`, `drop-branch` all logged their `info` line (proving they ran), and asserts the summary line names `drop-topics` as the failed phase.
- A new bats test "cleanup.sh exits 0 when all phases succeed" verifies the happy path and the `phases_failed=0` summary.
- The `sweep-orphans.sh` script gets the same try-all treatment (consistency; same shared `record_error` helper from `lib.sh`).

### 4.6 Fix the literal-`%s` consumer-group bug (Bug 6)

**Current behavior:** `libs/atlas-kafka/consumergroup/resolver.go:19-24` defines `Resolve(default string) string`: if `KAFKA_CONSUMER_GROUP` env is set, return it verbatim. atlas-channel (`services/atlas-channel/atlas.com/channel/main.go:151`) and atlas-login use the pattern:

```go
const consumerGroupIdTemplate = "Channel Service - %s"
var consumerGroupId = consumergroup.Resolve(fmt.Sprintf(consumerGroupIdTemplate, config.Id.String()))
```

In production (no `KAFKA_CONSUMER_GROUP` env set), the `fmt.Sprintf` runs on the default and produces `"Channel Service - <channel-uuid>"`. In PR env, the patch generator at `deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh:36-38` sets `KAFKA_CONSUMER_GROUP="Channel Service - %s [PLACEHOLDER_ATLAS_ENV]"`. `Resolve` returns the env value verbatim, **including the literal `%s`** — `fmt.Sprintf` already ran on the default string, not on the env value. Every channel in every PR-env atlas-channel pod registers as group `Channel Service - %s [<env>]`, and a single pod hosting multiple channels has them all sharing one group.

**Critical constraint:** atlas-channel and atlas-login may host more than one world/channel/tenant in a single pod. The fix cannot rely on per-pod or per-deployment env var substitution — the channel-id substitution must happen at consumer-registration time, per-channel, inside the Go code.

**Required behavior:** `Resolve` accepts variadic `args` and re-applies `fmt.Sprintf` to the env value when args are passed. atlas-channel and atlas-login update their call sites to pass the channel/login ID through `Resolve`, not pre-formatted into the default.

**Acceptance criteria:**
- `libs/atlas-kafka/consumergroup/resolver.go` adds a new function (or extends `Resolve`) with signature equivalent to `Resolve(defaultName string, args ...any) string`:
  - If `KAFKA_CONSUMER_GROUP` is set and `args` are passed, apply `fmt.Sprintf(envValue, args...)` and return.
  - If `KAFKA_CONSUMER_GROUP` is set and `args` is empty, return env value verbatim (preserves current behavior for plain `Resolve("Account Service")` callers).
  - If `KAFKA_CONSUMER_GROUP` is unset, apply `fmt.Sprintf(defaultName, args...)` and return (so the default also gets formatted when args are present).
- `libs/atlas-kafka/consumergroup/resolver_test.go` adds tests for:
  - Env var with `%s` + args → substituted output.
  - Env var without `%s` + no args → verbatim (existing case).
  - Default with `%s` + args, no env → substituted default.
  - Verify that callers using zero args (e.g., atlas-account) continue to pass.
- `services/atlas-channel/atlas.com/channel/main.go:151` changes from `consumergroup.Resolve(fmt.Sprintf(consumerGroupIdTemplate, config.Id.String()))` to `consumergroup.Resolve(consumerGroupIdTemplate, config.Id.String())`.
- atlas-login's main.go has the analogous change (template + ID-formatted call site).
- The patch generator `gen-consumer-group-patch.sh:31-35` comment is rewritten to reflect that `%s` is now substituted at runtime, not stripped at patch time.
- A new bats / Go test verifies that in PR env (env var set to `"Channel Service - %s [a1b2]"`), the resulting group ID for `config.Id="ch-7"` is `"Channel Service - ch-7 [a1b2]"`, not `"Channel Service - %s [a1b2]"`.
- The consumer-group regex in both `cleanup.sh:70` and `sweep-orphans.sh` continues to match (`\\[${ATLAS_ENV}\\]\$` still matches `"Channel Service - ch-7 [a1b2]"`).

### 4.7 Update the runbook (Bug 2 + Bug 4 side-effects)

**Acceptance criteria:**
- `docs/runbooks/ephemeral-pr-deployments.md` §9.4 (recovery) reflects the try-all behavior: operators no longer need to assume "phases after the failed one were skipped." Each phase's log line is now diagnostic.
- §9.11 (in-cluster one-shot sweep) command form is updated to source env from the `atlas-pr-cleanup-env` ConfigMap. The `kubectl run --rm -i` form (which doesn't reliably stream logs for non-TTY pods) is replaced with a Job manifest form, mirroring the PostDelete Job's shape. The runbook explicitly notes the workstation-side `--env` exports are no longer required for the in-cluster form.
- A new §9.12 "Diagnosing partial-cleanup failure" walks the operator through reading the summary line, identifying which phases failed, and re-running just those phases (e.g., manual `rpk topic delete` for drop-topics, manual `redis-cli DEL` for drop-redis). The try-all summary now makes this tractable.
- A new "Coordination with cluster-infra" subsection in §9.x lists the manifests this repo expects to exist in `argocd` namespace: `atlas-pr-cleanup-env` ConfigMap, `atlas-pr-cleanup` ServiceAccount + Role, the existing token Secret. Points at the cluster-infra sibling-PR convention.

### 4.8 Coordination artifacts

Because Bug 4's fix depends on a ConfigMap that lives in cluster-infra, this task ships:

- A `context.md` (the design-task / plan-task default location) section documenting the required `atlas-pr-cleanup-env` ConfigMap shape (every key, every value, namespace, labels).
- A non-deployed staging manifest under (e.g.) `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` showing the exact shape, with a header comment that says "Not deployed from this repo. Mirror into cluster-infra. This file's purpose is review-time visibility for the cluster-infra reviewer."

The `postdelete-cleanup.yaml` change to `envFrom: configMapRef:` only takes effect after the cluster-infra ConfigMap exists. The PR landing order is enforced by the runbook + the `context.md` notes; this task does not introduce a runtime check.

## 5. API Surface

No HTTP / JSON:API surface changes. The Go-side change is `libs/atlas-kafka/consumergroup`: the public `Resolve` function gains a variadic `args ...any` parameter. Existing callers that pass zero args remain source-compatible; this is an additive change.

## 6. Data Model

No persistent data model changes. No migration required.

## 7. Service Impact

### atlas-pr-bootstrap (image)
- Dockerfile: COPY for `sweep-orphans.sh`, fixture-related comment near `RPK_VERSION`.
- `scripts/cleanup.sh`: jq queries fixed, try-all failure policy, `record_error` helper, summary line.
- `scripts/sweep-orphans.sh`: same try-all policy, rpk-based Kafka phases, same shared jq queries.
- `scripts/lib.sh`: new `record_error` helper, possibly a shared `rpk_topics_query` / `rpk_groups_query` constant.
- `test/`: rewritten stubs using fixture files; new tests for fail-fast jq, try-all behavior, Dockerfile drift guard.
- `test/fixtures/`: new directory with `rpk-topic-list.json`, `rpk-group-list.json`.

### deploy/k8s/overlays/pr-cleanup
- `postdelete-cleanup.yaml`: inline `env:` → `envFrom: configMapRef:`, header comment rewritten.

### deploy/k8s/overlays/pr
- `scripts/gen-cleanup-env.sh`: new generator for the cluster-infra coordination artifact.
- `scripts/gen-consumer-group-patch.sh`: comment update reflecting that `%s` now substitutes at runtime.
- (no change to `patches/consumer-group-env.yaml`'s generated form — atlas-channel's literal `"Channel Service - %s [PLACEHOLDER_ATLAS_ENV]"` stays, because the Go code now handles the `%s`).

### .github/workflows
- `pr-validation.yml`: `update-pr-overlay` step invokes `gen-cleanup-env.sh` alongside `gen-consumer-group-patch.sh`.

### .github/config
- `services.json` is now read by `gen-cleanup-env.sh`; no schema change.

### libs/atlas-kafka/consumergroup
- `resolver.go`: `Resolve` signature gains variadic args.
- `resolver_test.go`: new test cases.

### services/atlas-channel/atlas.com/channel
- `main.go:151`: one line change to pass `config.Id.String()` as varargs to `Resolve` instead of pre-formatting.

### services/atlas-login/atlas.com/login
- `main.go`: analogous change to atlas-channel.

### docs/runbooks
- `ephemeral-pr-deployments.md`: §9.4, §9.11, new §9.12.

### dev/cluster-infra-coordination (new)
- `atlas-pr-cleanup-env.example.yaml`: non-deployed reference manifest for the sibling cluster-infra PR.

## 8. Non-Functional Requirements

### Observability
- Every cleanup phase logs `info` on start and either `info msg="phase complete"` or `error msg="<phase failure detail>"` on finish.
- The final summary line is grep-friendly: a single JSON record with `phases_run`, `phases_failed`, and a `failed_phases` array.
- No new metrics introduced; the existing JSON-structured logs are sufficient for Loki queries.

### Backwards compatibility
- `consumergroup.Resolve(string)` continues to work — existing callers (atlas-account, atlas-buddies, etc.) pass zero args and get the existing behavior.
- The new fixture-based bats tests must still run on a host without rpk installed (stubs replay fixtures). No new system-level dependencies.

### Idempotency
- `cleanup.sh` and `sweep-orphans.sh` remain re-runnable on the same PR number with no side effect beyond logs.
- The try-all policy preserves this: a failed-then-rerun cleanup converges to the same final state as a single successful cleanup.

### Security
- No new secrets, no new tokens, no new RBAC. The `atlas-pr-cleanup-env` ConfigMap holds no sensitive values (cluster-static hostnames and DB-name lists). Token secrets stay in their existing Secret resources.

### Multi-tenancy
- No tenant-scoped data touched. PR-env state is single-tenant by construction.

## 9. Open Questions

- **Q1.** Should `gen-cleanup-env.sh` emit the cluster-infra coordination artifact as a deployable manifest (Argo CD applies it from this repo) or as a documentation-only example? The current direction is documentation-only because `argocd` namespace is cluster-infra's territory, but if cluster-infra would prefer a single-source-of-truth in this repo, that's a different shape. Defer to design phase.
- **Q2.** Should the `record_error` helper accumulate errors as JSON for the summary line, or as plain text? JSON is grep-friendlier but adds escaping complexity in bash. Defer to design phase.
- **Q3.** atlas-channel's `consumerGroupIdTemplate = "Channel Service - %s"` works for the runtime-format approach. Does atlas-login follow the exact same pattern, or does it have a different template? Verify in design phase before assuming the call-site fix is one-line in both services.
- **Q4.** Should the bats fixtures for rpk be generated by a one-shot helper script (e.g., `test/fixtures/regenerate.sh` that requires a live broker), or committed as static JSON the dev hand-edits? Static is simpler; helper script is more accurate when rpk versions change. Recommend static + a `# regenerate by:` comment.
- **Q5.** The Dockerfile drift guard for §4.2 — is bats the right place for it, or should it be a separate `tools/check-dockerfile-script-coverage.sh` script invoked by CI? Bats keeps it co-located with the other image tests; a separate script is reachable without bats installed. Defer to design phase.

## 10. Acceptance Criteria

A PR opened against `main` with a synthetic minor change (no behavioral impact) goes through the full lifecycle:

1. PR open → ApplicationSet creates `atlas-pr-N`, namespace boots, pods come up.
2. atlas-channel pod's Kafka consumer groups appear in `rpk group list` with names like `Channel Service - ch-1 [<env_hash>]` (not `Channel Service - %s [<env_hash>]`).
3. PR labeled-removed → ArgoCD Application transitions to Terminating → PostDelete Job runs.
4. The Job's container reads its env from `atlas-pr-cleanup-env` ConfigMap (verified via `kubectl describe pod`).
5. cleanup.sh runs every phase. Each phase logs its `info` start line. drop-topics and drop-groups succeed against rpk 24.3.1 (no jq error). Summary line reports `phases_failed=0`.
6. Within 5 minutes of PR-close: no per-env Postgres DBs, no per-env Kafka topics, no per-env consumer groups, no per-env Redis keys, no per-PR GHCR tags, no `<N>.atlas.home` DNS entry, no `bot/pr-N-resolved` branch.
7. The Application is fully gone (no Terminating state, no orphaned finalizers).

A separate failure-injection test confirms the try-all policy:

8. With rpk's broker temporarily blackholed (or a forced bad-JSON return), cleanup.sh's drop-topics fails. drop-groups, drop-redis, drop-images, drop-dns, drop-branch all still execute. The Job exits non-zero. The summary line names drop-topics as the failed phase. The Application enters cleanup-failed state but doesn't wedge in Terminating beyond ArgoCD's normal reconcile cadence.

A CI / bats run:

9. `bats services/atlas-pr-bootstrap/test/` is green.
10. `docker buildx bake atlas-pr-bootstrap` succeeds.
11. `go test ./libs/atlas-kafka/consumergroup/...` is green.
12. `go test ./services/atlas-channel/... ./services/atlas-login/...` is green (no regressions from the call-site change).
13. The Dockerfile-drift guard test fails when a script is added to `scripts/` without a corresponding COPY (verified by manually adding a no-op script and running tests).

Runbook:

14. `docs/runbooks/ephemeral-pr-deployments.md` §9.4, §9.11, §9.12 are coherent with the new behavior. An operator following §9.11 with a fresh shell does not need to copy-paste env values out of a YAML manifest.
