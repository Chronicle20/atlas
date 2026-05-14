# Ephemeral Per-PR Deployments — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-03
---

## 1. Overview

Atlas ships from a monorepo of 57 Go services plus a Next.js UI, runs as ~80 pods in the shared k3s cluster, and currently has no automated review-environment story. Every code change is reviewed against the running `main` deployment, which means reviewers can't exercise a feature in a real cluster before merge, and a regression in `main` can cascade across all in-flight work.

This task introduces **two coordinated deployment surfaces** managed by Argo CD on the existing the cluster:

1. A **persistent `main` environment** that auto-syncs from the `main` branch on every push.
2. A pool of **ephemeral per-PR environments**, each spun up on PR open and torn down (after a grace period) on PR close/merge.

The the cluster's shared infrastructure pods — Postgres, Kafka, Redis, Traefik, observability — are reused by every environment. Isolation is achieved at the **name/key level**, not the pod level: each environment carries a 4-character hex hash (`ATLAS_ENV`) that suffixes Postgres database names, prefixes Redis keys, suffixes Kafka topic names, and namespaces consumer-group IDs. The `main` environment's hash is the literal string `m4in`; PR environments derive their hash deterministically from the PR number (`hash("pr-<N>")[:4]`). Determinism is chosen over randomness so Argo CD restarts and Application CRD recreations re-attach to existing data instead of leaking it.

The work spans three planes:

- **Cluster infra:** install Argo CD on the cluster, add an Argo `ApplicationSet` driven by the GitHub PR generator, install the DNS-update side-channel for the two out-of-cluster Pi-hole servers.
- **Repository plumbing:** refactor `deploy/k8s/*.yaml` (currently flat manifests) into a Kustomize base + `main` overlay + parameterized `pr` overlay; extend `.github/workflows/pr-validation.yml` to build and push images per PR.
- **Service code sweeps:** make Kafka consumer-group IDs env-driven (currently hardcoded in ~50 service `main.go` files) and add transparent key-prefixing to `libs/atlas-redis` so all Redis-using services pick up isolation without per-service code changes.

The success metric is end-to-end: open a PR → ~minutes later → reviewer hits `{N}.atlas.home` and can exercise the feature against an isolated stack with its own data plane → close the PR → after the configured grace period, all per-env state (databases, topics, consumer groups, Redis keys, container image tags, namespace, DNS) is reclaimed.

## 2. Goals

### Primary goals

- Argo CD installed on the cluster and accessible via Traefik ingress.
- A `main` Argo Application that auto-syncs from `main` branch and replaces the current ad-hoc apply-by-hand flow.
- An `ApplicationSet` with a GitHub PR generator that creates one Argo Application per open PR against `Chronicle20/atlas`.
- Each PR environment is fully isolated from `main` and from every other PR environment along these dimensions: Postgres data, Kafka messages, Kafka consumer-group offsets, Redis state, container images.
- Each PR environment is reachable at `{PR_NUMBER}.atlas.home` over HTTP via Traefik.
- Each PR environment runs the existing `atlas-ui` SetupPage bootstrap (WZ upload → extract → ingest → seed) automatically as part of becoming ready, with no manual steps.
- PR-close cleanup, gated by a grace period, removes every per-env artifact: namespace, Postgres DBs, Kafka topics, Kafka consumer-group offsets, Redis keys, ghcr.io image tags, Pi-hole DNS entries.
- Determinism: re-deriving the hash from the PR number yields the same value across Argo restarts, dev workstations, and cleanup hooks.

### Non-goals

- Production-grade hardening (multi-AZ, autoscaling, HA Postgres) — `main` is dev-grade infrastructure.
- TLS for PR environments (HTTP only at v1; see open question).
- Authentication/authorization on PR environments — accessible to anyone on the home network.
- Per-PR ephemeral copies of Postgres / Kafka / Redis pods. Shared infrastructure is a deliberate constraint.
- Cross-environment data migration (e.g. cloning `main` data into a PR env). Bootstrap from canonical seed only.
- Tenant or character data preservation across PR environments. Each PR starts from an empty data plane.
- Slack/GitHub-comment notifications when an environment becomes ready or is reclaimed.
- Performance or load-test-grade environments. PR envs are functional-review only.
- Image-pull credentials. The `chronicle20/atlas-*` ghcr repos are public; verified by all 50+ atlas pods running on the cluster with zero image-pull secrets configured on the `atlas` namespace's default ServiceAccount.

## 3. User Stories

- **As a code reviewer**, I want to open a feature PR's environment in my browser so I can exercise the change against a real stack without merging or pulling the branch locally.
- **As a code reviewer**, I want each PR's environment to start from a clean, seeded data plane so my testing isn't contaminated by previous reviewers' test characters.
- **As a feature author**, I want the PR environment URL to appear within minutes of opening the PR so the review loop isn't blocked.
- **As a feature author**, I want the same environment URL to remain stable across pushes to my PR branch (no environment churn on every commit) so reviewers don't lose links.
- **As a maintainer**, I want PR environments to be torn down automatically (with a grace period) so I never have to chase leaked databases or topics.
- **As a maintainer**, I want a single source of truth (Kustomize base) for every environment's manifests so a config drift in `main` doesn't surprise me when a PR env reproduces it.
- **As a maintainer**, I want `main` to be a regular Argo Application that auto-syncs on push so deployment becomes "merge to main" instead of "manually run kubectl apply."
- **As an oncall person**, I want PR environments to be obviously labeled in observability dashboards and never page me when they break — they are explicitly tagged ephemeral.

## 4. Functional Requirements

Requirements are grouped by the system surface they touch.

### 4.1 ATLAS_ENV — the environment hash

- Every Atlas environment carries a value `ATLAS_ENV`, a 4-character lowercase hex string. This is the load-bearing isolation token across the data plane.
- `main` environment: `ATLAS_ENV=m4in` (literal, committed).
- PR environment: `ATLAS_ENV=hash("pr-<PR_NUMBER>")[:4]` where `hash` is SHA-256 hex output. Concretely, `ApplicationSet` template parameter computed from `{{number}}`. Deterministic; recomputable from the PR number alone.
- `ATLAS_ENV` MUST be readable by every service at startup either via `envFrom: configMapRef: name: atlas-env` (existing convention) or a per-service `env:` entry. Implementation chooses one consistently.
- A **collision policy** is documented but not implemented at v1: birthday-paradox math gives <2% chance of two open PRs hashing to the same suffix at our PR concurrency. If a collision occurs, Argo refuses to apply the second Application (namespace conflict), and a human resolves manually. Future work could escalate to 6 hex chars.

### 4.2 Postgres isolation (per-environment databases)

- `libs/atlas-database/connection.go:86` already reads `DB_NAME` from `os.LookupEnv`. **No code change in `libs/atlas-database`.**
- The Kustomize PR overlay patches every service manifest's `DB_NAME` env value with suffix `-<ATLAS_ENV>`. Example: `atlas-characters` → `atlas-characters-a3f7`.
- Postgres role used in `db-credentials` Secret MUST have the `CREATEDB` role attribute. This is verified during Phase 1 implementation; if missing, a one-line GRANT is applied to the existing role.
- An Argo `PreSync` hook Job, scoped to each Application, runs `CREATE DATABASE "<name>-<ATLAS_ENV>"` for every service-owned database. The job is idempotent (`IF NOT EXISTS`).
- Schema population happens organically: GORM `AutoMigrate` runs at every service's cold start against the empty database, materializing the branch's current schema. **No migration-script coordination is required.**
- An Argo `PostDelete` hook Job runs `DROP DATABASE` for every per-environment database when the Application is deleted, gated by the grace-period mechanism.

### 4.3 Kafka topic isolation

- All Atlas services resolve topic names from environment variables of the form `EVENT_TOPIC_*` or `COMMAND_TOPIC_*` via `topic.EnvProvider(l)(envVarName)()`. The values for these variables are sourced from a single shared ConfigMap, `atlas-env`, applied via `envFrom` in every service's Deployment.
- The Kustomize PR overlay produces a per-PR `atlas-env-<ATLAS_ENV>` ConfigMap that is byte-identical to `main`'s except every topic value is suffixed `-<ATLAS_ENV>`. Example: `account-status` → `account-status-a3f7`.
- Topic auto-creation is assumed (the cluster's Kafka broker config `auto.create.topics.enable=true`). This is verified during Phase 1; if disabled, an Argo `PreSync` hook explicitly creates topics with `kafka-topics.sh --create`.
- An Argo `PostDelete` hook deletes every topic matching the pattern `*-<ATLAS_ENV>` via `kafka-topics.sh --delete --topic '.*-<ATLAS_ENV>$'`.

### 4.4 Kafka consumer-group isolation

- Each service's `main.go` currently declares a literal: `const consumerGroupId = "Character Service"` (or equivalent). Two PR environments running the same service would join the same consumer group on the shared Kafka cluster, splitting partitions and silently losing events.
- **Code sweep across ~50 services**: replace each `const consumerGroupId = "..."` with a function that reads `os.Getenv("KAFKA_CONSUMER_GROUP")` with the literal as fallback. Mechanical refactor.
- The Kustomize PR overlay sets `KAFKA_CONSUMER_GROUP=<original-literal> [<ATLAS_ENV>]` per service. Brackets are chosen to make the group ID human-readable in `kafka-consumer-groups.sh --list` output: `Character Service [a3f7]`.
- An Argo `PostDelete` hook deletes every consumer group matching the pattern `* \[<ATLAS_ENV>\]` via `kafka-consumer-groups.sh --delete --group '.*\[<ATLAS_ENV>\]'`. Note: empty consumer groups expire after Kafka's `offsets.retention.minutes` (default 7 days), so this hook is best-effort cleanup, not strict-correctness.

### 4.5 Redis key isolation

- Atlas services use Redis key compositions ranging from tenant-keyed (`atlas:npc-shop-chars:{tenantId}:{shopId}`) to fully literal (`coordinator:agreement:{id}`). Tenant-keying is insufficient for environment isolation because (a) tenants may be re-used inside a PR environment for ergonomic reasons, and (b) tenant-unaware services exist (`atlas-assets`, `atlas-monster-book`, `atlas-query-aggregator`).
- **Modify `libs/atlas-redis/connection.go`** to wrap the `redis.Client` with a key-prefix decorator that transparently prepends `<ATLAS_ENV>:` to every key passed to `Get`, `Set`, `Del`, `SCAN`, `Keys`, `MGet`, `MSet`, etc. The prefix is read from `os.Getenv("ATLAS_ENV")` once at client construction.
- Services that compose Redis keys without going through `libs/atlas-redis` are bugs to be fixed in this task. A grep audit during Phase 1 enumerates them.
- An Argo `PostDelete` hook runs `redis-cli --scan --pattern '<ATLAS_ENV>:*' | xargs -r redis-cli DEL` (chunked) to reclaim keyspace on environment teardown.

### 4.6 Argo CD

- Argo CD installed via a manifest at `<infra-repo>/argocd.yml` (committed to the cluster infra repo, not the Atlas repo).
- Argo CD's UI exposed via Traefik IngressRoute at `argocd.home` (matching existing cluster naming convention used by `dns.home` for Pi-hole).
- Argo CD's GitHub credentials configured via a Kubernetes Secret containing a fine-scoped GitHub PAT with read access to `Chronicle20/atlas`. PAT lives in 1Password / outside the repo.

### 4.7 Argo CD ApplicationSet for PR environments

- An `ApplicationSet` resource in `<infra-repo>/argocd-atlas.yml` (or under a dedicated `<infra-repo>/argocd/` subdirectory) configured with the **GitHub Pull Request generator** pointing at `Chronicle20/atlas`, polling at 30s.
- For each open PR, the generator emits parameters: `{{number}}`, `{{branch}}`, `{{head_sha}}`, `{{labels}}`. The template substitutes these into a per-PR Application:
  - `metadata.name: atlas-pr-{{number}}`
  - `spec.source.path: deploy/k8s/overlays/pr`
  - `spec.source.targetRevision: {{head_sha}}` (so the env always tracks the latest commit on the PR branch)
  - `spec.destination.namespace: atlas-pr-{{number}}`
  - Kustomize plugin parameters: `ATLAS_ENV` (computed from `{{number}}`), `IMAGE_TAG: pr-{{number}}-{{head_sha_short}}`
- ApplicationSet `syncPolicy` enables `automated` with `prune: true` and `selfHeal: true`.
- Application-level `syncPolicy.preserveResourcesOnDeletion: false` (the default), so deleting an Application deletes the namespace and all per-env resources.

### 4.8 Argo CD Application for the main environment

- A single `Application` resource at `<infra-repo>/argocd-atlas-main.yml` pointing at `Chronicle20/atlas` `main` branch, path `deploy/k8s/overlays/main`, destination namespace `atlas`.
- `automated` sync with `selfHeal: true`. No `prune` initially (to avoid surprise deletions during the migration); revisited after the migration is stable.
- Replaces the current ad-hoc `kubectl apply -f deploy/k8s/` flow.

### 4.9 Kustomize refactor

- `deploy/k8s/base/` contains all current per-service manifests, with the following templating:
  - `image:` references use `${IMAGE_TAG}` placeholder (default `latest`)
  - `DB_NAME` env values use literal current values; overlays patch them
  - Namespace omitted from base; overlays inject
  - The shared `atlas-env` ConfigMap reference is name-fixed; overlays generate replacement ConfigMaps with hash-suffixed values
- `deploy/k8s/overlays/main/` contains:
  - `kustomization.yaml` with `namespace: atlas`
  - `images:` mapping pinning each service to `:latest`
  - A `configMapGenerator` that produces `atlas-env` (no suffix on topic names since `ATLAS_ENV=m4in` literal — but for consistency the same template is used)
  - `commonLabels: { atlas.env: m4in }`
- `deploy/k8s/overlays/pr/` contains:
  - `kustomization.yaml` parameterized via Argo CD Kustomize plugin parameters (or `replacements:` if plugin-free)
  - `namespace: atlas-pr-${PR_NUMBER}`
  - `images:` mapping pinning each service to `:pr-${PR_NUMBER}-${SHA}`
  - A `configMapGenerator` producing `atlas-env` with every topic value suffixed `-${ATLAS_ENV}`
  - JSON patch suffixing every `DB_NAME` env value with `-${ATLAS_ENV}`
  - JSON patch adding a `KAFKA_CONSUMER_GROUP` env var to every service Deployment, value `<service-default-group> [${ATLAS_ENV}]`
  - JSON patch adding `ATLAS_ENV` env var to every service Deployment, value `${ATLAS_ENV}`
  - Traefik IngressRoute for `${PR_NUMBER}.atlas.home` routing to atlas-ui and the game socket service
  - `commonLabels: { atlas.env: ${ATLAS_ENV}, atlas.pr-number: ${PR_NUMBER} }`

### 4.10 Image plumbing

- Extend `.github/workflows/pr-validation.yml` to add a `docker-build-pr` job that mirrors the build steps of `.github/workflows/main-publish.yml`, but tags images `ghcr.io/chronicle20/atlas-<svc>/atlas-<svc>:pr-<PR_NUMBER>-<SHA_SHORT>`. Re-uses the existing `.github/actions/detect-changes` action so unchanged services don't rebuild.
- For services unchanged by the PR, the PR overlay falls back to the `:latest` tag (i.e. uses the current `main` image). This allows a UI-only PR to run with current-`main` backend services.
- Add `.github/workflows/pr-cleanup.yml` triggered on `pull_request: types: [closed]`. The job (a) deletes ghcr.io image tags matching `pr-<PR_NUMBER>-*` for every Atlas image, (b) optionally calls Argo CD's API to immediately mark the Application for deletion (or leaves Argo CD's PR generator polling to handle it).

### 4.11 Per-PR bootstrap (PostSync hook)

- Each service's Postgres DB is empty after `PreSync`. After all service pods are ready, a `PostSync` hook Job invokes the existing `atlas-ui` SetupPage flow programmatically.
- The bootstrap Job: (1) waits for `atlas-data` and `atlas-wz-extractor` to report ready, (2) downloads a canonical WZ zip from a shared cluster-side location (see open question #4), (3) POSTs the zip to `atlas-data` upload endpoint inside the per-PR namespace, (4) triggers extraction via `atlas-wz-extractor`, (5) triggers ingest, (6) waits for ingest completion, (7) triggers per-service seeding.
- The Job is idempotent: if invoked against an already-bootstrapped environment, it short-circuits.
- Bootstrap latency target: under 5 minutes from `PostSync` start to "ready for review." Subject to revision once measured.

### 4.12 DNS automation for Pi-hole

- The cluster relies on **two external (out-of-cluster) Pi-hole servers** for home-network DNS. The in-cluster `pihole` namespace exists but has no running pods (verified via `kubectl get pods -n pihole`).
- An Argo `PostSync` hook Job, after the bootstrap Job, calls each Pi-hole's REST API (Pi-hole v6+) to register an A record `<PR_NUMBER>.atlas.home → <Traefik LB IP>`. The job needs:
  - Both Pi-hole API base URLs (configured via Secret in the cluster)
  - Both Pi-hole API tokens / passwords
  - The Traefik LoadBalancer IP from `traefik-helmchart.yaml`'s `loadBalancerIP: 192.168.23.230`
- An Argo `PostDelete` hook removes the same A record from both Pi-holes.
- Failure to update one Pi-hole MUST NOT block the environment from being marked ready (DNS will resolve via the other), but MUST be alerted (logged with severity to observability).
- An evaluated alternative — installing `external-dns` with a Pi-hole webhook provider — is documented as future work but not v1, because it adds an operator + provider + Pi-hole API plumbing for marginal gain at our scale.

### 4.13 Cleanup with grace period

- "Cleanup" comprises: namespace deletion, Postgres DB drops, Kafka topic deletes, Kafka consumer-group deletes, Redis SCAN+DEL, ghcr image-tag deletes, Pi-hole A-record removal.
- The grace period is **24 hours by default**, configurable via an annotation on the Argo Application (`atlas.cleanup-grace: 24h`). After PR close, the Application is annotated with a deletion timestamp; a CronJob in the `argocd` namespace scans for past-grace Applications and triggers their deletion.
- The grace period exists so a maintainer can re-investigate a closed PR's logs / state in case of post-close issues. Within grace, the env is preserved as-is.
- Cleanup MUST be transactional in failure semantics: if any step fails, the Application stays annotated as "cleanup-failed" rather than partially destroyed. A maintainer is alerted via observability.
- Force-cleanup escape hatch: deleting the Application directly (Argo UI or `kubectl delete`) bypasses the grace period.

## 5. API Surface

This task is primarily infrastructure; few new HTTP APIs are introduced. The relevant interfaces:

### 5.1 Per-PR ingress (HTTP)

- Hostname: `<PR_NUMBER>.atlas.home`
- Routing: Traefik IngressRoute in the per-PR namespace
- Backend services exposed:
  - `atlas-ui` (port 80 / 3000) — SetupPage and the rest of the UI
  - `atlas-ingress` (game socket) — server entrypoint for Maple clients pointed at the PR env
- All other Atlas service HTTP APIs remain cluster-internal; reviewers reach them transitively through atlas-ui.
- TLS: not in v1 (HTTP only).

### 5.2 Argo CD UI

- Hostname: `argocd.home`
- Authentication: Argo CD admin password (initial) → SSO via Cattle / Rancher if/when relevant. SSO not in v1 scope.

### 5.3 Pi-hole automation API

- Each Pi-hole server's REST API (per Pi-hole v6+ docs):
  - `POST /api/config/dns/hosts` — add A record
  - `DELETE /api/config/dns/hosts/<id>` — remove A record
  - `GET /api/config/dns/hosts` — list (used by cleanup verification)
- Authentication via API token stored in a Kubernetes Secret in the `argocd` namespace.

### 5.4 Internal bootstrap API (existing, used by hook)

- The PostSync bootstrap Job invokes the same endpoints `atlas-ui`'s SetupPage hits today. Concrete endpoint paths to be enumerated during Phase 1 by reading `services/atlas-ui/src/pages/SetupPage.tsx` and the React Query hooks it uses.

## 6. Data Model

This task introduces no new domain entities. It introduces deployment-level state captured outside Postgres:

### 6.1 Per-environment Postgres databases

- Database name pattern: `<service-default-name>-<ATLAS_ENV>` (e.g. `atlas-characters-a3f7`)
- Each service's GORM `AutoMigrate` populates schema at first cold start
- Postgres role: existing `db-credentials` user, augmented with `CREATEDB` if not already present

### 6.2 Per-environment Kafka topics

- Topic name pattern: `<service-default-topic>-<ATLAS_ENV>`
- Auto-created on first publish (if Kafka auto-create is enabled) or pre-created by `PreSync` hook
- Retention: inherits Kafka cluster default

### 6.3 Per-environment Redis keyspace

- Key pattern: `<ATLAS_ENV>:<service-key-composition>`
- Implemented in `libs/atlas-redis` so all services pick up automatically
- Keyspace shared across services within an environment (matches current production behavior)

### 6.4 Argo CD Application annotations

- `atlas.env: <ATLAS_ENV>` — set on every per-env Application
- `atlas.pr-number: <N>` — set on per-PR Applications only
- `atlas.cleanup-grace: <duration>` — overrides the default 24h grace period
- `atlas.cleanup-deadline: <RFC3339 timestamp>` — set when PR closes; the cleanup CronJob compares this to `now()`

### 6.5 Pi-hole A records

- Naming: `<PR_NUMBER>.atlas.home` for PR envs, `<TBD>.atlas.home` for main (open question #2 includes whether main also takes a hostname)
- Target: Traefik LoadBalancer IP (`192.168.23.230` per current `<infra-repo>/traefik-helmchart.yaml`)

## 7. Service Impact

### 7.1 Code changes

- **`libs/atlas-redis/connection.go`** — add transparent key-prefix wrapper reading `ATLAS_ENV`. Single library change, all consumers benefit. Estimated diff: ~50-100 lines including tests.
- **~50 service `main.go` files** — replace `const consumerGroupId = "..."` with `os.Getenv("KAFKA_CONSUMER_GROUP")` with literal fallback. Mechanical sweep, scriptable via `gofmt`-style rewrite.
- **`libs/atlas-database/connection.go`** — verify (no changes expected; already env-driven).
- **Audit grep**: identify any service composing Redis keys outside `libs/atlas-redis` and route them through the wrapper. Estimated 0-3 services.

### 7.2 Manifests & CI

- **`deploy/k8s/`** — restructured from flat into Kustomize base + main overlay + PR overlay. Retains all current functionality of `main` env.
- **`.github/workflows/pr-validation.yml`** — add `docker-build-pr` job and matrix.
- **`.github/workflows/pr-cleanup.yml`** — new file, deletes PR-tagged ghcr images on PR close.

### 7.3 New infrastructure (in the cluster infra repo, not Atlas repo)

- `<infra-repo>/argocd.yml` — Argo CD install (Helm-rendered or vanilla manifest set).
- `<infra-repo>/argocd-atlas.yml` — ApplicationSet for PR generator.
- `<infra-repo>/argocd-atlas-main.yml` — main Application.
- `<infra-repo>/argocd-cleanup-cronjob.yml` — past-grace Application cleanup.
- `<infra-repo>/argocd-pihole-secret.yml` (sealed) — Pi-hole API tokens.

### 7.4 No service is removed or refactored beyond consumer-group sweep

- All 57 services keep their current responsibilities, public interfaces, and database schemas.
- The only behavioral change at runtime is: services now read consumer group ID from env, and Redis keys carry a prefix.

## 8. Non-Functional Requirements

### 8.1 Performance

- **PR env time-to-ready** target: under 5 minutes from PR open to reviewer-loadable URL. Phases:
  - PR open → ApplicationSet detects (≤30s, polling cadence)
  - Application sync → DB creation + pod boot (≤2 minutes)
  - PostSync bootstrap → WZ ingest + service seeding (≤2 minutes for shared-zip path; longer for fresh extract)
- **PostDelete cleanup** target: under 2 minutes for full reclamation.
- Bootstrap latency is the dominant cost; if WZ extraction is in the critical path, the env is unusable for several minutes longer. The shared-zip optimization (open question #4) is the variable that controls this.

### 8.2 Observability

- All per-env pods carry labels `atlas.env=<ATLAS_ENV>` and (for PR envs) `atlas.pr-number=<N>`. Loki / Grafana dashboards filter by these.
- A new dashboard `atlas-pr-environments` summarizes: number of open envs, time-to-ready per env, cleanup status, bootstrap step durations.
- Argo CD's own dashboard suffices for sync-status visibility.
- Cleanup-failure alerts fire to the existing observability stack (no new pager integration in v1).

### 8.3 Security

- HTTP only at v1; no PII or production data passes through PR envs. Reviewers exercise the system locally on the home network.
- ghcr.io repos are public (verified). No image-pull secret needed.
- Pi-hole API tokens stored as Kubernetes Secrets, never committed.
- Postgres `db-credentials` reused across envs (each env has its own DB names but the same user). A future hardening could provision per-env users; out of scope.

### 8.4 Multi-tenancy

- The Atlas multi-tenant model (`tenant.MustFromContext`, `tenant_id` scoping) is **orthogonal** to environment isolation. A tenant ID `default` exists separately within each environment; tenant data is isolated at the `(ATLAS_ENV, tenant_id)` pair, not the `tenant_id` alone.
- The 7 tenant-unaware services (`atlas-assets`, `atlas-monster-book`, `atlas-query-aggregator`, `atlas-tenants`, `atlas-ui`, `atlas-messages`, `atlas-monster-death`) are still isolated by environment via the same DB / Kafka / Redis / consumer-group mechanisms.

### 8.5 Reliability

- Argo CD self-heal repairs drift between cluster state and Git (e.g. someone `kubectl edit`-ing a deployment).
- Bootstrap Job retries 3 times on transient failure before marking the env as `bootstrap-failed`.
- Cleanup hooks log every step; cleanup failures leave the env intact and annotated `cleanup-failed`, never partially destroyed.

### 8.6 Cost / capacity

- Each PR env is approximately 50 pods. Concurrent open PRs is low (typical: 1-3, occasional bursts to 5-10). Cluster has 4 nodes (`theia`, `helios`, `gaia`, `eos`); steady-state utilization remains comfortable.
- Postgres: ~50 logical databases per env. InnoDB-equivalent buffer pool is shared; expected to stay well below saturation at our scale. Monitor `pg_database` count.
- No explicit per-env resource caps in v1. Add `LimitRange` per namespace if a single PR env saturates the node.

## 9. Open Questions

These are issues whose decisions belong in the **design** phase (`/design-task`), not in this PRD. Listed here so they're explicit, not lost.

1. **Bootstrap WZ-data source.** Where does the canonical WZ zip live cluster-side? Options: a Longhorn PVC mounted into the bootstrap Job; a ghcr container image with the zip baked in; an internal HTTP endpoint on a permanent service; an S3-compatible object store. Each has different update workflows.
2. **Is bootstrap caching worth it for v1?** Skipping WZ extraction by sharing a pre-extracted XML PVC is an optimization that cuts bootstrap from minutes to seconds, but only correct for PRs that don't touch atlas-data extraction logic. Risk is missing the case where a PR changes extraction; benefit is much faster review loops. Defer or include?
3. **Kafka auto-create cluster setting.** Verify `auto.create.topics.enable=true` on the cluster's Kafka broker, or implement explicit topic creation in the `PreSync` hook. (10 minutes of investigation in design phase.)
4. **Argo CD Kustomize plugin vs. `replacements:`.** The PR overlay needs to substitute the per-PR `ATLAS_ENV` value into multiple places. Argo CD Kustomize plugin parameters and Kustomize `replacements:` are both options. Kustomize-native is simpler; plugin gives more flexibility. Pick during design.
5. **Initial migration sequencing.** Today, `main` is deployed by manual `kubectl apply`. Cutting over to Argo `Application(main)` requires either (a) flushing the cluster and letting Argo recreate or (b) having Argo adopt existing resources. Option (b) is safer; document the procedure.
6. **Argo CD auth / SSO.** v1 uses the default admin password. Is SSO via Cattle (Rancher is installed) worth setting up at v1, or follow-up?
7. **Game-socket port for PR envs.** Reviewers exercising a PR env via the actual Maple client must point at the PR's game-socket port. Traefik exposes ports per PR env via different LoadBalancer IPs, or a shared LB with port mapping, or a single LB with SNI-based routing (TCP, so SNI doesn't trivially apply). Routing model TBD.
8. **Bootstrap Job credentials for atlas-ui SetupPage.** Does the SetupPage flow require an authenticated session, or is it open in dev mode? Investigation needed before writing the bootstrap Job.
9. **`main` env hash naming.** Is `m4in` the literal value used everywhere (DB names, topics, Redis prefix), or does main get a non-hashed pass-through (status quo) and only PR envs have suffixes? Consistency argument says `m4in` everywhere; pragmatism argument says preserving current `main` DB names avoids a one-time data migration.
10. **Simultaneous-PR resource budget.** What's the soft cap on concurrent open envs before we add `LimitRange`? Pulling a number out of the air feels wrong; better to measure during the migration.

## 10. Acceptance Criteria

The task is complete when **every** item below is demonstrably true.

### 10.1 Argo CD operational

- [ ] Argo CD is installed in the cluster, accessible at `argocd.home` over HTTP.
- [ ] An Argo `Application` named `atlas-main` exists, points at `Chronicle20/atlas` `main` branch, syncs the manifests under `deploy/k8s/overlays/main/`, and `automated.selfHeal=true`.
- [ ] Pushing a commit to `main` automatically updates the cluster within 60 seconds (Argo's poll interval).
- [ ] An Argo `ApplicationSet` named `atlas-pr` exists, configured with the GitHub PR generator pointing at `Chronicle20/atlas`.

### 10.2 Per-PR env lifecycle

- [ ] Opening a PR creates an Argo Application `atlas-pr-<N>` within 60 seconds.
- [ ] The Application's PreSync, Sync, PostSync hooks all complete green.
- [ ] After PostSync, the env is reachable at `<N>.atlas.home` over HTTP and serves the atlas-ui SetupPage URL.
- [ ] Pushing a new commit to the PR branch updates the env's image tags and re-bootstraps if needed.
- [ ] Closing or merging the PR annotates the Application with a 24h cleanup deadline; the env remains live during grace.
- [ ] After the grace period, the cleanup CronJob deletes the Application; PostDelete hooks reclaim DBs, topics, consumer groups, Redis keys, ghcr image tags, Pi-hole DNS entries.

### 10.3 Isolation verification

- [ ] Postgres: opening two PRs simultaneously yields two distinct sets of databases (e.g. `atlas-characters-a3f7` and `atlas-characters-b2c1`); no cross-contamination of GORM-migrated columns.
- [ ] Kafka: messages published in one PR env are not consumed by services in another PR env (verified by inspecting `kafka-console-consumer` against the topic list).
- [ ] Kafka consumer groups: `kafka-consumer-groups.sh --list` shows distinct group IDs for each env.
- [ ] Redis: keys produced in one PR env do not appear in another's `SCAN` output (verified by prefix grep).
- [ ] DNS: `<N>.atlas.home` resolves only after env is ready; both Pi-holes return the same A record.

### 10.4 Code sweeps complete

- [ ] All ~50 atlas service `main.go` files read `KAFKA_CONSUMER_GROUP` from env with literal fallback. Verified by grep audit.
- [ ] `libs/atlas-redis/connection.go` wraps `redis.Client` with transparent prefix; tested.
- [ ] No service composes Redis keys outside the wrapper. Verified by grep audit.
- [ ] `libs/atlas-database/connection.go` is unchanged (verification only).

### 10.5 CI / image pipeline

- [ ] `.github/workflows/pr-validation.yml` builds and pushes per-PR-tagged images to ghcr.io for every changed service.
- [ ] `.github/workflows/pr-cleanup.yml` deletes those tags on PR close.
- [ ] Unchanged services in a PR env reuse `:latest` image tags (no unnecessary rebuilds).

### 10.6 Documentation

- [ ] `deploy/k8s/README.md` documents the Kustomize base + overlay structure and how `ATLAS_ENV` flows.
- [ ] A runbook at `docs/runbooks/ephemeral-pr-deployments.md` covers: opening / closing a PR env manually, force-cleanup, debugging stuck cleanups, rotating Pi-hole API tokens.
- [ ] `docs/observability.md` adds a section on filtering by `atlas.env` label.

### 10.7 Failure modes documented

- [ ] Hash collision (two PR numbers producing the same 4-char hash) handling documented in the runbook.
- [ ] Bootstrap failure handling documented (env stays up, marked `bootstrap-failed`, manual rerun procedure).
- [ ] Cleanup partial-failure handling documented.

### 10.8 No regressions in `main` env

- [ ] After cutting over `main` to Argo CD, all 57 services remain running and pass their existing health checks.
- [ ] No data loss in `main` Postgres DBs during the migration (the DB names don't change for `main`).
- [ ] No Kafka topic or consumer-group disruption during cutover.
