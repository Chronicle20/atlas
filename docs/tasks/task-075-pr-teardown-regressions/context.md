# task-075 — Context

Companion to `prd.md`, `design.md`, and `plan.md`. Captures the
file-level entry points, decisions, and cross-repo coordination an
implementer or reviewer needs before opening the plan.

---

## 1. Key files (where the work lands)

### Bash + image surface — `services/atlas-pr-bootstrap/`

- `scripts/lib.sh` — shared helpers. Gains `record_error`,
  `run_phase`, `summarize_phases`, and `RPK_TOPICS_JQ` /
  `RPK_GROUPS_JQ` constants.
- `scripts/cleanup.sh` — PostDelete entrypoint. Header drops `-e`;
  every phase becomes a `do_*` function; orchestrated by
  `run_phase` + `summarize_phases`. The jq queries against rpk
  output (`:57`, `:69`) are rewritten to `$RPK_TOPICS_JQ` /
  `$RPK_GROUPS_JQ`.
- `scripts/sweep-orphans.sh` — operator one-shot. `sweep_kafka`
  (`:121-154`) is ported from `kafka-topics.sh` /
  `kafka-consumer-groups.sh` to `rpk`. Phases route through the
  same `run_phase` + `summarize_phases` framework as cleanup.sh
  (per-PR aggregation).
- `Dockerfile` — adds `COPY scripts/sweep-orphans.sh /atlas/…` plus a
  comment near `ARG RPK_VERSION` pointing at the fixtures.
- `test/cleanup_test.bats` — `make_stubs` defaults to fixture
  files. New tests for try-all behaviour and fail-fast jq.
- `test/sweep_test.bats` — stubs change from `kafka-*.sh` to `rpk`.
- `test/lib_test.bats` — extended with unit tests for the three
  new helpers.
- `test/dockerfile_test.bats` (new) — Dockerfile-drift guard.
- `test/fixtures/rpk-topic-list.json` (new) — rpk 24.3.1 schema.
- `test/fixtures/rpk-group-list.json` (new) — rpk 24.3.1 schema.
- `test/fixtures/README.md` (new) — regeneration instructions.

### Kustomize overlays — `deploy/k8s/overlays/`

- `pr-cleanup/postdelete-cleanup.yaml` — inline `env:` for
  static infra vars (`DB_HOST`, `DB_PORT`, `BOOTSTRAP_SERVERS`,
  `REDIS_URL`, `ATLAS_DB_NAMES`, `ATLAS_SERVICES`) replaced with
  `envFrom: - configMapRef: { name: atlas-pr-cleanup-env }`.
  `PR_NUMBER` stays inline.
- `pr/scripts/gen-cleanup-env.sh` (new) — reads
  `.github/config/services.json`, writes the cluster-infra
  coordination artifact.
- `pr/scripts/gen-consumer-group-patch.sh` — comment-only update
  to reflect runtime `%s` substitution.

### Workflow — `.github/workflows/pr-validation.yml`

- `update-pr-overlay` job grows a step that runs
  `gen-cleanup-env.sh` and a `git diff --exit-code` check on its
  output so a stale coordination file fails PR CI.

### Cluster-infra coordination — `dev/cluster-infra-coordination/`

- `atlas-pr-cleanup-env.example.yaml` (new) — non-deployed
  reference manifest mirroring the ConfigMap shape cluster-infra
  must own.

### Go consumer-group resolver — `libs/atlas-kafka/consumergroup/`

- `resolver.go` — `Resolve` gains variadic `args ...any`. Behaviour
  matrix locked in by `resolver_test.go`.
- `resolver_test.go` — four new test cases per design §4.4.

### Service call sites

- `services/atlas-channel/atlas.com/channel/main.go:151` — one-line.
- `services/atlas-login/atlas.com/login/main.go:66` — one-line.

### Runbook — `docs/runbooks/ephemeral-pr-deployments.md`

- §9.4 reworded for try-all summary.
- §9.11 reshaped: ConfigMap-sourced env, Job-manifest form.
- §9.12 (new) — diagnosing partial-cleanup failure.
- New "Coordination with cluster-infra" subsection lists
  `argocd`-namespace dependencies.

---

## 2. Decisions already made (from design §2)

- **Coordination artifact is documentation-only.** Lives at
  `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`.
  Not part of any kustomize root.
- **`record_error` payload is plain text per phase; summary is JSON.**
  Phase-level details use `log warn` / `log error` (already JSON via
  `lib.sh::log`). Summary collects phase names via
  `jq -Rsc 'split("\n") | map(select(length>0))'`.
- **atlas-login template** confirmed as
  `"ChannelConnect Service - %s"` at `main.go:43`, consumed at
  `:66`. One-line fix in both services.
- **Fixtures are static JSON**, hand-checked-in, with a
  `# regenerate by:` header. The Dockerfile gets a matching
  comment near `ARG RPK_VERSION`.
- **Dockerfile drift guard lives in bats** (`test/dockerfile_test.bats`).
- **Only `run_phase` calls `record_error`.** Phase functions return
  non-zero on failure with detail logs via `log warn` / `log error`;
  `run_phase` records the phase name exactly once. This resolves the
  design-doc ambiguity (§3.1 sketched both patterns) and prevents
  double-counting a phase in `phases_failed`.

---

## 3. External-tool schemas (the fixtures pin these)

`rpk 24.3.1 topic list -X brokers=… --format json` emits:

```json
[
  {"name": "<topic>", "partitions": <int>, "replicas": <int>},
  …
]
```

`rpk 24.3.1 group list -X brokers=… --format json` emits:

```json
[
  {"name": "<group>", "members": <int>},
  …
]
```

Both are flat arrays. The pre-fix queries (`.topics[].name`,
`.groups[].name`) assume objects with those keys — bug #1. The new
queries are `.[].name` for both, captured as `RPK_TOPICS_JQ` /
`RPK_GROUPS_JQ` in `lib.sh`. Bumping `ARG RPK_VERSION` in the
Dockerfile invalidates the fixtures; regeneration must run against
the new rpk and re-run bats.

---

## 4. Sibling PR (cluster-infra) — required ConfigMap

`atlas-pr-cleanup-env` in `argocd` namespace, owned by
cluster-infra. Exact shape this repo expects (mirrored from
`dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`):

```yaml
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
  ATLAS_DB_NAMES: "atlas-accounts atlas-bans atlas-buddies atlas-cashshop atlas-characters atlas-configurations atlas-data atlas-drops atlas-fame atlas-gachapons atlas-guilds atlas-inventory atlas-keys atlas-map-actions atlas-maps atlas-merchant atlas-monster-book atlas-notes atlas-npc-conversations atlas-npc-shops atlas-party-quests atlas-pets atlas-portal-actions atlas-quest atlas-reactor-actions atlas-saga-orchestrator atlas-skills atlas-storage atlas-tenants"
  ATLAS_SERVICES: "atlas-account,atlas-asset-expiration,…"   # sorted, derived from .github/config/services.json
```

**Landing order risk:** if this repo's PR merges first, the next
PostDelete Job fails with `CreateContainerConfigError: configmap
"atlas-pr-cleanup-env" not found`. Mitigations:

1. The PR description must link the cluster-infra sibling PR by URL
   and state the ordering ("merge cluster-infra first, then this").
2. Runbook §9.11's new "Coordination with cluster-infra" subsection
   documents the dependency.
3. No runtime fallback is added in `cleanup.sh` — design §6 rejects
   that as bug-bait.

---

## 5. Call-site inventory (zero-arg `Resolve` callers)

`grep -rE 'consumergroup\.Resolve\(' --include='*.go' services/`
confirms every existing call passes exactly one string argument
(zero-args path), except the two templated services:

- `services/atlas-channel/atlas.com/channel/main.go:151` — switch from
  `consumergroup.Resolve(fmt.Sprintf(consumerGroupIdTemplate, …))` to
  `consumergroup.Resolve(consumerGroupIdTemplate, config.Id.String())`.
- `services/atlas-login/atlas.com/login/main.go:66` — same shape.

Every other call (atlas-account, atlas-buddies, atlas-data, …)
remains source-compatible — `Resolve("…")` with zero args goes
through the unchanged behaviour path.

---

## 6. Verification surface

A "ready for PR" claim for this task requires, per CLAUDE.md
"Build & Verification":

1. `bats services/atlas-pr-bootstrap/test/` is green.
2. `go test -race ./libs/atlas-kafka/consumergroup/...` is green.
3. `go test -race ./...` is green across every module whose
   sources were touched (libs/atlas-kafka, atlas-channel,
   atlas-login).
4. `go vet ./...` clean in each touched module.
5. `go build ./...` clean in each touched service.
6. `docker buildx bake atlas-channel atlas-login atlas-pr-bootstrap`
   succeeds. atlas-channel and atlas-login because their `go.mod`
   modules transitively depend on the modified `libs/atlas-kafka`;
   atlas-pr-bootstrap because the Dockerfile changed.
7. The Dockerfile-drift guard (`test/dockerfile_test.bats`) fails
   when a script is added to `scripts/` without a matching `COPY`
   — verifiable by manually adding a no-op script.

---

## 7. Out-of-scope follow-ups

Captured for future tasks, not addressed here:

- Move `ATLAS_DB_NAMES` into `.github/config/services.json` so
  cluster-infra's ConfigMap has a single source of truth for it
  too.
- Bump `RPK_VERSION` past 24.3.1; will require regenerating the
  fixtures.
- Promote `dev/cluster-infra-coordination/` to a real
  generated-artifact pipeline.
- Surface `summarize_phases` JSON to a Prometheus counter via a
  sidecar (Loki is sufficient today).
