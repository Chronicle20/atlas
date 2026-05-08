# Ephemeral Per-PR Deployments — Design

Version: v1.1
Status: Draft
Created: 2026-05-08
Revised: 2026-05-08

PRD: [`prd.md`](./prd.md)

---

## Revision notes

**v1.1** — Two design decisions reversed after planning recon:

1. **`main` is symmetric.** Earlier draft kept `main` unsuffixed to avoid migration. Reversed: `ATLAS_ENV=main`, namespace `atlas-main`, every DB/topic/group/Redis-key suffixed `-main`. The cutover is a one-time scheduled-downtime migration (~30 min) — see §7.
2. **Game-socket is in v1 scope.** Earlier draft deferred per-PR game-socket exposure. Reversed: each PR gets one MetalLB-allocated LoadBalancer IP that backs both atlas-login and atlas-channel; the bootstrap Job seeds the IP into atlas-tenants `services` config so the login server hands clients the right host. See §4.6 and §6.2. Soft cap on concurrent PR envs = MetalLB pool free count.

The plan's Phase 0 (preflight.md) verifies the live cluster's Longhorn capacity, MetalLB pool, and atlas-tenants config schema before any code lands.

---

## 1. Overview

This design realises the PRD by composing four layers, each with an explicit owner repo:

| Layer | Owner repo | Purpose |
|---|---|---|
| **Argo CD control plane** | `tumidanski/k3s` (bee infra) | Argo install, `Application(main)`, `ApplicationSet(pr)`, cleanup CronJob, Pi-hole secrets |
| **Atlas manifest layout** | `Chronicle20/atlas` (this repo) | `deploy/k8s/` restructured into Kustomize `base` + `overlays/main` + `overlays/pr` |
| **Atlas service code** | `Chronicle20/atlas` | `libs/atlas-redis` env-aware keyPrefix, new `libs/atlas-kafka/consumergroup` resolver, ~49 service `main.go` sweep, audit of raw-client redis users |
| **Atlas CI / image plumbing** | `Chronicle20/atlas` `.github/` | per-PR image builds tagged `pr-<N>-<sha7>`, PR-close cleanup workflow |

The `bee` cluster's data-plane pods (Postgres, Kafka, Redis, Traefik) are unchanged. Isolation is **logical only**: every per-environment Postgres database, Kafka topic, Redis key, and Kafka consumer-group ID is name-suffixed (or prefixed) with a 4-hex-char `ATLAS_ENV` token derived deterministically from the PR number. Determinism lets Argo CD restarts and Application recreations re-attach to existing data instead of leaking it.

The Atlas repo carries **all PR-environment-specific config** as Kustomize patches. The bee infra repo carries **only the Argo wiring** (which paths to sync, how to template per PR, and where to find Pi-hole credentials). This split lets the Atlas team ship a manifest change in a normal Atlas PR without touching the bee infra repo.

---

## 2. Open Questions Resolved

The PRD listed ten open questions. Each is resolved below. The reasoning is in the relevant section that follows.

| # | Question | Resolution | Section |
|---|---|---|---|
| 1 | Bootstrap WZ-data source | **Longhorn `ReadOnlyMany` PVC** mounted into the bootstrap Job | §6.1 |
| 2 | Bootstrap caching for v1 | **No caching.** Always run full WZ ingest. Defer pre-extracted PVC optimization. | §6.1 |
| 3 | Kafka auto-create | **Assume `auto.create.topics.enable=true`** (verified at plan-phase start). PreSync hook only as fallback if disabled. | §5.2 |
| 4 | Kustomize plugin vs. `replacements:` | **Kustomize-native `replacements:` + `configMapGenerator`.** No Argo CD plugin. | §4.4 |
| 5 | `main` cutover sequencing | **Argo adopts existing resources** via `ServerSideApply=true` and `prune=false`. Enable prune after a stability window. | §7 |
| 6 | Argo SSO at v1 | **No SSO at v1.** Default admin password; SSO via Cattle/Rancher is follow-up. | §3.1 |
| 7 | Game-socket port for PR envs | **In scope at v1.** One MetalLB-allocated LoadBalancer IP per PR, backing both atlas-login and atlas-channel (their wire ports don't collide). Bootstrap Job seeds the IP into atlas-tenants `services` config. Soft cap = MetalLB pool free count. | §4.6 |
| 8 | Bootstrap auth on SetupPage | **No auth required.** SetupPage endpoints are open in dev mode; bootstrap Job calls them via in-cluster service URLs. | §6.2 |
| 9 | `main` env hash naming | **`ATLAS_ENV=main` everywhere.** Symmetric with PR envs: namespace `atlas-main`, DB/topic/group/Redis-key all suffixed `-main`. One-time cutover migration documented in §7. | §5 |
| 10 | Concurrent-PR resource budget | **No `LimitRange` at v1.** Measure during the first month; add caps if pressure observed. | §8.5 |

---

## 3. Argo CD Control Plane

### 3.1 Argo CD install

- Manifest committed to `tumidanski/k3s` at `bee/argocd.yml`. Rendered from the upstream Argo CD vanilla install (`https://raw.githubusercontent.com/argoproj/argo-cd/v2.13.x/manifests/install.yaml`) with two patches:
  1. `argocd-server` `command:` includes `--insecure` (Traefik terminates HTTP at the edge).
  2. `argocd-cm` ConfigMap sets `application.instanceLabelKey: argocd.argoproj.io/instance` (default; explicit for clarity).
- Namespace `argocd`. RBAC unchanged from upstream.
- Traefik `IngressRoute` at `argocd.bee.tumidanski` → service `argocd-server:80`. HTTP only at v1.
- **Auth at v1: default `admin` password** (read from `argocd-initial-admin-secret`). SSO via Cattle is deferred.
- GitHub credentials: a fine-scoped GitHub PAT with read-only access to `Chronicle20/atlas` is stored as Argo CD repo credentials (`argocd-repo-creds-chronicle20-atlas` Secret). PAT lives outside both repos (1Password). The `bee/argocd.yml` manifest references the secret name; the secret is created out-of-band the first time and is not committed.

### 3.2 `Application(atlas-main)`

- File: `tumidanski/k3s` `bee/argocd-atlas-main.yml`.
- Spec:
  ```yaml
  source:
    repoURL: https://github.com/Chronicle20/atlas.git
    targetRevision: main
    path: deploy/k8s/overlays/main
  destination:
    server: https://kubernetes.default.svc
    namespace: atlas
  syncPolicy:
    automated:
      selfHeal: true
      prune: false  # enable after stability window
    syncOptions:
      - ServerSideApply=true
      - CreateNamespace=false  # atlas namespace already exists
  ```
- `prune: false` initially so the cutover from manual `kubectl apply` cannot accidentally delete a resource Argo doesn't yet recognise. Re-enabled after ~1 week of clean syncs.
- `ServerSideApply=true` lets Argo adopt existing resources without recreating them.

### 3.3 `ApplicationSet(atlas-pr)`

- File: `tumidanski/k3s` `bee/argocd-atlas-pr.yml`.
- Generator: GitHub PR generator polling `Chronicle20/atlas`, every 30s.
- Template substitutes `{{number}}`, `{{branch}}`, `{{head_sha}}`, `{{head_short_sha}}` into a per-PR `Application`:
  ```yaml
  metadata:
    name: atlas-pr-{{number}}
    annotations:
      atlas.env: "{{atlasEnv}}"          # computed via Go template helper
      atlas.pr-number: "{{number}}"
      atlas.cleanup-grace: "24h"
      atlas.head-sha: "{{head_sha}}"
  spec:
    source:
      repoURL: https://github.com/Chronicle20/atlas.git
      targetRevision: "{{head_sha}}"
      path: deploy/k8s/overlays/pr
      kustomize:
        commonAnnotations:
          atlas.env: "{{atlasEnv}}"
        replacements:
          - source: { value: "{{atlasEnv}}" }
            targets: [{ select: { kind: ConfigMap, name: atlas-env-tokens } }]
        # plus per-PR labels via commonLabels
        commonLabels:
          atlas.env: "{{atlasEnv}}"
          atlas.pr-number: "{{number}}"
        images:
          - name: ghcr.io/chronicle20/atlas-account/atlas-account
            newTag: pr-{{number}}-{{head_short_sha}}
          # ... one entry per service; auto-generated by tooling, see §5.5
    destination:
      namespace: atlas-pr-{{number}}
    syncPolicy:
      automated: { selfHeal: true, prune: true }
      syncOptions: [ServerSideApply=true, CreateNamespace=true]
  ```
- The `atlasEnv` template variable is computed by a small Go template helper registered via Argo's `goTemplate: true` mode: `{{- printf "%.4s" (sha256 (printf "pr-%d" .number)) -}}`.

### 3.4 Cleanup CronJob

- File: `tumidanski/k3s` `bee/argocd-cleanup-cronjob.yml`.
- Runs hourly in the `argocd` namespace as a service account with permissions:
  - `applications.argoproj.io`: `get`, `list`, `delete`, `patch`
  - GitHub PR status reader (uses Argo's existing PAT secret, mounted read-only)
- Logic (inline shell, ~30 lines):
  1. List all `applications.argoproj.io` matching label `atlas.pr-number`.
  2. For each, hit `GET /repos/Chronicle20/atlas/pulls/<N>` with the Argo CD PAT and read `state`.
  3. If `state == closed` and the Application's `atlas.cleanup-deadline` annotation is unset, set it to `now + atlas.cleanup-grace` (default 24h).
  4. If `state == closed` and `now > cleanup-deadline`, `kubectl delete application atlas-pr-<N> -n argocd`. Argo's PostDelete hooks then run.
  5. If `state == open`, clear `cleanup-deadline` (handles re-opened PRs).

---

## 4. Atlas Manifest Layout

### 4.1 Directory restructure

```
deploy/k8s/
├── base/                       # all current per-service Deployments + Services + ConfigMaps
│   ├── atlas-account.yaml
│   ├── atlas-asset-expiration.yaml
│   ├── ... (one file per service, copied as-is from current deploy/k8s/*.yaml)
│   ├── atlas-ingress.yaml      # was ingress.yaml — split namespace.yaml out
│   ├── env-configmap.yaml
│   ├── secrets.example.yaml    # not applied; documentation only
│   └── kustomization.yaml      # listing all resources
├── overlays/
│   ├── main/
│   │   └── kustomization.yaml
│   └── pr/
│       ├── kustomization.yaml
│       ├── ingress-route.yaml          # Traefik IngressRoute for <N>.atlas.home
│       ├── presync-create-dbs.yaml     # Argo PreSync hook Job
│       ├── postsync-bootstrap.yaml     # Argo PostSync hook Job (WZ + seed)
│       ├── postsync-pihole-add.yaml    # Argo PostSync hook Job (DNS register)
│       ├── postdelete-cleanup.yaml     # Argo PostDelete hook Job (DBs/topics/groups/keys/DNS)
│       └── patches/
│           ├── db-name-suffix.yaml         # patches DB_NAME on every Deployment
│           ├── consumer-group-env.yaml     # adds KAFKA_CONSUMER_GROUP env to every Deployment
│           ├── atlas-env-env.yaml          # adds ATLAS_ENV env to every Deployment
│           └── topic-suffix-configmap.yaml # generates the per-env atlas-env ConfigMap
└── README.md
```

The current flat manifests in `deploy/k8s/*.yaml` move into `deploy/k8s/base/` verbatim with one mechanical edit: every Deployment's `metadata.namespace: atlas` is **removed** so overlays can inject the namespace.

### 4.2 `overlays/main`

`main` is symmetric to PR envs. Same `replacements:` machinery, same patches, only the literal `ATLAS_ENV` value differs:

```yaml
# kustomization.yaml
namespace: atlas-main
resources:
  - ../../base
  - atlas-env-tokens.yaml             # ATLAS_ENV: "main" (literal)
patches:
  - path: ../pr/patches/db-name-suffix.yaml          # shared with PR overlay
  - path: ../pr/patches/consumer-group-env.yaml      # shared with PR overlay
  - path: lb-pin.yaml                                # pins .231/.232 (existing reservations)
configMapGenerator:
  - name: atlas-env
    behavior: replace
    literals:
      # Same shape as PR overlay; PLACEHOLDER_ATLAS_ENV → "main" via replacements
replacements:
  # Same rule shape as PR overlay
images:
  - name: ghcr.io/chronicle20/atlas-account/atlas-account
    newTag: latest
  # ... one per service
commonLabels:
  atlas.env: main
```

The differences vs `overlays/pr`:

- `lb-pin.yaml` keeps `atlas-login-lb` at `192.168.23.231` and `atlas-channel-lb` at `192.168.23.232` (existing home-network reservations the Pi-holes already point at). PR overlay clears these so MetalLB allocates fresh IPs (§4.6).
- No PostDelete cleanup hook: Application(atlas-main) is never deleted.
- No PreSync DB-create hook: cutover migration (§7) renames existing DBs in place; subsequent fresh `main` re-installs from scratch are theoretical and use a distinct runbook.
- No Pi-hole register hook: `main.atlas.home` and the bare `dev.atlas.home` are pinned in Pi-hole out-of-band.
- `images:` pinned to `:latest`, not `pr-<N>-<sha>`.

Symmetry means service code and `replacements:` rules don't branch on env type. Adding a future `staging` env is the same overlay shape with a different literal.

### 4.3 `overlays/pr`

```yaml
# kustomization.yaml
namespace: atlas-pr-PLACEHOLDER  # rewritten by ApplicationSet's commonAnnotations / namespace
resources:
  - ../../base
  - ingress-route.yaml
  - presync-create-dbs.yaml
  - postsync-bootstrap.yaml
  - postsync-pihole-add.yaml
  - postdelete-cleanup.yaml
patchesStrategicMerge:
  - patches/db-name-suffix.yaml
  - patches/consumer-group-env.yaml
  - patches/atlas-env-env.yaml
configMapGenerator:
  - name: atlas-env
    behavior: replace
    literals:
      # full list of EVENT_TOPIC_*, COMMAND_TOPIC_*, every value suffixed -<ATLAS_ENV>
      - COMMAND_TOPIC_ACCOUNT=COMMAND_TOPIC_ACCOUNT-PLACEHOLDER_ATLAS_ENV
      - ...
replacements:
  - source: { kind: ConfigMap, name: atlas-env-tokens, fieldPath: data.ATLAS_ENV }
    targets:
      - select: { kind: Deployment }
        fieldPaths:
          - spec.template.spec.containers.[name=*].env.[name=ATLAS_ENV].value
        options: { create: true }
      # plus DB_NAME suffix replacements, topic suffix replacements, consumer-group suffix replacements
images:
  - name: ghcr.io/chronicle20/atlas-account/atlas-account
    newTag: pr-PLACEHOLDER-PLACEHOLDER  # rewritten by ApplicationSet
commonLabels:
  atlas.env: PLACEHOLDER
  atlas.pr-number: PLACEHOLDER
```

The literal `PLACEHOLDER` strings are rewritten by Argo's ApplicationSet via the per-PR `kustomize.replacements`/`commonLabels`/`images` fields shown in §3.3. This keeps the overlay valid Kustomize that can be rendered standalone for testing (`kustomize build deploy/k8s/overlays/pr` against a PLACEHOLDER will produce manifests with PLACEHOLDER literals — useful for diff inspection only).

### 4.4 Why Kustomize-native `replacements:` not an Argo CD plugin

Argo CD supports two ways to inject per-environment values into Kustomize:

1. **Kustomize-native `replacements:` / `configMapGenerator`** (since Kustomize 4.5). Pure data manipulation, no plugin install.
2. **Argo CD Kustomize plugin** (KSOPS, replicate, custom). Requires installing a binary into the `argocd-repo-server` pod.

`replacements:` covers the full need (string substitution into manifest fields, with `options: { create: true }` to add fields that don't exist). No reason to take on a plugin's install cost and supply-chain risk. The only thing `replacements:` cannot do is template logic (e.g. computing the hash from PR number) — that is computed once by the ApplicationSet's Go template helper and pushed into the per-PR Application as a literal value, so by the time `kustomize build` runs, the value is just a string.

### 4.5 Per-PR ingress

`overlays/pr/ingress-route.yaml`:

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: atlas-pr-ingress
spec:
  entryPoints: [web]
  routes:
    - match: Host(`PLACEHOLDER.atlas.home`)
      kind: Rule
      services:
        - name: atlas-ingress
          port: 80
```

Routes only the existing nginx `atlas-ingress` (which already proxies to atlas-ui at `/`). `PLACEHOLDER` is rewritten to the PR number. The per-PR namespace contains its own `atlas-ingress` Deployment via the base, so no cross-namespace traffic.

### 4.6 Per-PR game-socket exposure

Each PR env exposes the Maple game socket via **one MetalLB-allocated LoadBalancer IP** that backs both atlas-login (wire ports 1200/8300/8700/9200/9500/18500) and atlas-channel (wire ports 1201/8301/8701/18501). The wire ports don't collide, so a single IP per PR suffices.

**Mechanism:**

1. The base `atlas-login.yaml` and `atlas-channel.yaml` declare `Service` of type `LoadBalancer` with `loadBalancerIP: 192.168.23.231` / `.232` pinned for the main env.
2. The PR overlay's `patches/lb-allocate.yaml` clears these to `loadBalancerIP: ""` so MetalLB assigns from the pool.
3. After Argo's PostSync hooks run, the bootstrap Job (§6.2) does `kubectl get svc atlas-channel-lb -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`, then PATCHes atlas-tenants' `services` configuration with that IP.
4. atlas-channel reads the `ipAddress` field from atlas-tenants config (`services/atlas-channel/atlas.com/channel/configuration/rest.go:35`); atlas-login uses the same configuration to broadcast `(host, port)` per channel to the connecting client.

**Concurrency cap:** MetalLB pool free count after subtracting the 5 existing fixed allocations (.230 traefik, .231 main login, .232 main channel, .235 nginx-proxy-manager, .237 tempo-collector). Verified at plan-phase Task 0.5; documented in `preflight.md` and the runbook.

**Failure modes:**

- *MetalLB pool exhausted.* Argo PostSync stalls waiting for `atlas-channel-lb` to acquire an IP; bootstrap step `channel-config` fails because `LB_IP` is empty. Mitigation: expand pool, or fall back to port-shifting on a shared IP (§10 future work).
- *atlas-tenants PATCH fails.* Bootstrap Job exits non-zero; Application stays `Degraded`. Maintainer investigates per runbook.
- *MetalLB IP not released on PR close.* Namespace deletion releases the LB Service; MetalLB observes and frees the IP. No explicit cleanup step needed.

> **Note** — `main` env keeps its existing IP reservations (.231/.232) because the home network's Pi-holes already point at them; no migration of those bindings during cutover.

---

## 5. Atlas Service Code Sweeps

### 5.1 `ATLAS_ENV` token

- A short lowercase string set on every pod and every Argo hook Job in the env.
- For `main`: literal `"main"`. (PRD §4.1's `m4in` literal is honoured as `main` for readability — switch is a one-line change in `deploy/k8s/overlays/main/atlas-env-tokens.yaml`.)
- For PR envs: `sha256("pr-<PR_NUMBER>")[:4]`. Computed by the ApplicationSet Go template, materialised as a literal in the per-PR Application's `replacements:`, propagated into:
  - The `atlas-env-tokens` ConfigMap created per env (single-key: `ATLAS_ENV: a3f7`).
  - The pod-level `env: ATLAS_ENV: a3f7` on every Deployment via Kustomize patch.
- The dual destination (ConfigMap + pod env) is intentional. The pod env is what `libs/atlas-redis` reads at startup; the ConfigMap is what the Argo hook Jobs (PreSync DB create, PostDelete cleanup) read because they are not patched by the same overlay.

### 5.7 PVC isolation

Three services own PVCs (`atlas-data-pvc`, `atlas-wz-input-pvc`, `atlas-assets-pvc`). PVCs are namespace-scoped: when the PR overlay deploys to `atlas-pr-<N>`, Kubernetes creates fresh same-named PVCs in that namespace and Longhorn provisions a separate PV for each. **Isolation is automatic; no explicit suffixing needed.**

Cleanup follows from namespace deletion. With the longhorn StorageClass at `reclaimPolicy: Delete` (verified Phase 0 Task 0.4), the backing PVs are deleted along with the namespace. The PostDelete cleanup hook does not need to handle PVCs explicitly.

Storage budget is the per-env footprint times the soft cap from preflight.md. If usage approaches Longhorn pool capacity, add per-namespace `LimitRange` (§10 future work).

### 5.2 Postgres DB-name suffixing

`libs/atlas-database/connection.go` is unchanged. It already reads `DB_NAME` from env. The Kustomize PR overlay patches every service Deployment's `DB_NAME` env value, e.g.

```yaml
# patches/db-name-suffix.yaml — strategic merge patch
- name: atlas-character
  spec:
    template:
      spec:
        containers:
          - name: character
            env:
              - name: DB_NAME
                value: atlas-characters-PLACEHOLDER  # PLACEHOLDER → ATLAS_ENV via replacements
```

A `replacements:` rule in `kustomization.yaml` substitutes `PLACEHOLDER` with the live `ATLAS_ENV` value.

**PreSync hook** (`presync-create-dbs.yaml`): a single Job that connects to Postgres as `db-credentials.DB_USER` and runs `CREATE DATABASE IF NOT EXISTS "<orig-name>-<ATLAS_ENV>"` for every service-owned database. Database list is materialised as a ConfigMap `atlas-db-names` produced by `configMapGenerator` so adding a service doesn't require touching the hook script.

**Postgres role check** (one-time, in plan phase): verify the existing `db-credentials.DB_USER` role has `CREATEDB`. If not, apply `ALTER ROLE <user> CREATEDB` once during the cutover.

GORM `AutoMigrate` runs at every service cold start, populating empty schema. No migration coordination required.

**PostDelete hook** drops every per-env DB. Idempotent (`DROP DATABASE IF EXISTS`).

### 5.3 Kafka topic suffixing

The PR overlay's `configMapGenerator` regenerates the `atlas-env` ConfigMap with every `EVENT_TOPIC_*` / `COMMAND_TOPIC_*` value suffixed `-<ATLAS_ENV>`. Services pick the suffixed values up via existing `envFrom: configMapRef: name: atlas-env`.

Topic auto-creation is assumed (`auto.create.topics.enable=true`). Plan phase verifies via `kubectl exec -n atlas <kafka-pod> -- kafka-configs.sh --describe --entity-type brokers --entity-name 0 | grep auto.create`. If disabled, a PreSync hook Job using a kafka-tools image creates topics from the topic-name list materialised by `configMapGenerator`.

**PostDelete hook** runs `kafka-topics.sh --bootstrap-server kafka.home:9093 --delete --topic '.*-<ATLAS_ENV>$'`. The pattern is regex; broker-side validates safety (`-<ATLAS_ENV>$` cannot match a name without the suffix).

### 5.4 Kafka consumer-group ID resolution

**New library**: `libs/atlas-kafka/consumergroup/resolver.go` (~25 lines including doc comment):

```go
package consumergroup

import "os"

const envVar = "KAFKA_CONSUMER_GROUP"

// Resolve returns the consumer group ID for this service.
// If KAFKA_CONSUMER_GROUP is set, it is used verbatim (deployment is
// expected to set this to "<defaultName> [<ATLAS_ENV>]" for PR envs).
// Otherwise the defaultName is returned for backwards compatibility
// with the main env, which never sets the variable.
func Resolve(defaultName string) string {
    if v, ok := os.LookupEnv(envVar); ok && v != "" {
        return v
    }
    return defaultName
}
```

With unit tests covering: env unset → default; env set non-empty → env value; env set to whitespace-only → trimmed handling (decision: do NOT trim — empty after trim is invalid config, the Kustomize patch always produces a non-blank value).

**Service sweep** across all 49 services:

```diff
- const consumerGroupId = "Character Service"
+ var consumerGroupId = consumergroup.Resolve("Character Service")
```

Each service additionally adds `import "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"`. The sweep is mechanical and can be driven by a one-shot `gofmt -r` / `goimports` script, but is committed as ordinary code review for safety. Exact services to touch are enumerated by `grep -l "const consumerGroupId" services/atlas-*/atlas.com/*/main.go`.

**Patch in PR overlay** (`patches/consumer-group-env.yaml`):

```yaml
# Strategic merge patch applied to every Deployment via kustomize "patches" list.
# Each entry pairs (deployment name, original consumer-group string).
- target:
    kind: Deployment
    name: atlas-character
  patch: |
    - op: add
      path: /spec/template/spec/containers/0/env/-
      value:
        name: KAFKA_CONSUMER_GROUP
        value: "Character Service [PLACEHOLDER]"
```

The patch is generated, not hand-written. The plan phase produces a one-shot script that walks `services/atlas-*/atlas.com/*/main.go`, extracts the `const consumerGroupId = "..."` literal, and emits the matching patch entry. The script is committed alongside the patch so future service additions can re-run it.

**PostDelete hook** runs `kafka-consumer-groups.sh --bootstrap-server kafka.home:9093 --delete --group '.*\[<ATLAS_ENV>\]'`. As the PRD notes, this is best-effort: empty groups expire after the broker's `offsets.retention.minutes` (default 7 days) regardless.

### 5.5 Redis key prefixing

**Decision: modify `libs/atlas-redis/keys.go` to make `keyPrefix` env-aware.** Do not wrap the `*goredis.Client` directly.

Reasoning: every helper in the lib (`Registry`, `TenantRegistry`, `Index`, `Lock`, `coalesced`, `id`, `ttl`) routes key construction through `namespacedKey(namespace, parts...)`, which prepends `keyPrefix`. Changing `keyPrefix` once propagates to every caller for free. Wrapping `*goredis.Client` requires re-implementing dozens of methods (`Get`, `Set`, `Del`, `SCAN`, `Keys`, `MGet`, `MSet`, `Watch`, `Pipeline`, `TxPipelined`, `Exists`, `Expire`, `SetNX`, `SAdd`, `SRem`, `SMembers`, etc.); even a thin facade is a 200+ line change with risk of missing methods. The constant tweak is ~10 lines.

The change:

```go
// libs/atlas-redis/keys.go
package redis

import (
    "os"
    "strings"

    "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const keyPrefixBase = "atlas"
const keySeparator = ":"

// keyPrefix is computed once at package init from ATLAS_ENV.
// Empty env (the main env) yields the legacy "atlas" prefix.
var keyPrefix = computeKeyPrefix(os.Getenv("ATLAS_ENV"))

func computeKeyPrefix(atlasEnv string) string {
    if atlasEnv == "" {
        return keyPrefixBase
    }
    return atlasEnv + keySeparator + keyPrefixBase
}

// KeyPrefix returns the env-aware key prefix used by every helper.
// Exported so services that must compose keys outside the helpers
// (audited in §5.6) can avoid hardcoding "atlas:".
func KeyPrefix() string {
    return keyPrefix
}

// namespacedKey, tenantEntityKey, tenantScanPattern, CompositeKey, TenantKey: unchanged.
```

Tests in `keys_test.go`:
- `computeKeyPrefix("")` → `"atlas"`
- `computeKeyPrefix("a3f7")` → `"a3f7:atlas"`
- `namespacedKey("buffs", "_tenants")` with env unset → `"atlas:buffs:_tenants"`
- `namespacedKey("buffs", "_tenants")` with env `"a3f7"` (using `t.Setenv` in subtest, plus a parallel `computeKeyPrefix` invocation) → `"a3f7:atlas:buffs:_tenants"`

The package-level `var keyPrefix = ...` reads env at first import. Tests that need to override use `computeKeyPrefix` directly rather than relying on `t.Setenv` after package init.

### 5.6 Audit of raw-client redis users

The PRD called out that some services compose Redis keys outside the helper. Confirmed: `services/atlas-buffs/atlas.com/buffs/character/registry.go:45` has

```go
func (r *Registry) tenantSetKey() string {
    return "atlas:" + r.characters.Namespace() + ":_tenants"
}
```

This bypasses `keyPrefix`. Fix:

```go
import atlas "github.com/Chronicle20/atlas/libs/atlas-redis"

func (r *Registry) tenantSetKey() string {
    return atlas.KeyPrefix() + ":" + r.characters.Namespace() + ":_tenants"
}
```

Plan-phase audit step:

```sh
grep -rn '"atlas:' services/ | grep -v _test.go | grep -v '.md:'
grep -rn 'keyPrefix' services/ | grep -v _test.go
```

Every hit must either (a) route through the helpers, or (b) call `atlas.KeyPrefix()`. Estimated 1-3 services affected (atlas-buffs confirmed; atlas-npc-shops, atlas-pets, atlas-messengers, atlas-guilds, atlas-reactors, atlas-channel use raw `client.*` calls but appear to flow through registries — confirmed at plan phase).

**PostDelete hook** runs:

```sh
redis-cli -h redis.home --scan --pattern "<ATLAS_ENV>:*" | xargs -r -n 1000 redis-cli -h redis.home DEL
```

`--scan` with chunking avoids `KEYS *` blocking the server.

---

## 6. Bootstrap and DNS Hooks

### 6.1 Bootstrap WZ-data source — Longhorn PVC

**Decision: a single cluster-side Longhorn `ReadOnlyMany` PVC** named `atlas-wz-canonical` mounted into the bootstrap Job.

Alternatives considered:

| Option | Pro | Con | Verdict |
|---|---|---|---|
| **Longhorn PVC (chosen)** | already on bee, no external deps, single source of truth, easy update by mounting into a one-shot writer pod | requires creating a dedicated PVC | Selected |
| ghcr image with WZ baked in | immutable, versioned | image is ~GB; bake-and-push pipeline; per-env pull cost | Rejected — pull cost compounds |
| HTTP endpoint on `main` env | reuses existing atlas-data | runtime dependency on `main`; `main` outage blocks bootstrap | Rejected — coupling |
| S3-compatible store | strong concurrency story | external dep; setup overhead | Rejected — overkill |

PVC is created out-of-band once (one-time procedure documented in the runbook, §9.1). Subsequent canonical-WZ updates: a maintainer mounts the PVC into a temporary pod and replaces the file. Update cadence is low (monthly at most).

### 6.2 PostSync bootstrap Job

`overlays/pr/postsync-bootstrap.yaml`:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: atlas-pr-bootstrap
  annotations:
    argocd.argoproj.io/hook: PostSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
spec:
  backoffLimit: 3
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: bootstrap
          image: ghcr.io/chronicle20/atlas-pr-bootstrap:latest
          envFrom:
            - configMapRef: { name: atlas-env-tokens }  # for ATLAS_ENV
          env:
            - name: ATLAS_UI_BASE
              value: http://atlas-ingress.atlas-pr-PLACEHOLDER.svc.cluster.local
          volumeMounts:
            - name: wz-canonical
              mountPath: /opt/wz
              readOnly: true
      volumes:
        - name: wz-canonical
          persistentVolumeClaim:
            claimName: atlas-wz-canonical-readonly
```

The bootstrap container is a small image (~50MB on alpine) that runs a sequence of HTTP calls against the in-namespace `atlas-ingress`. Endpoints — verified in plan phase by reading `services/atlas-ui/src/services/api/seed.service.ts` and `services/atlas-ui/src/lib/hooks/api/useSeed.ts`, which are the exact same calls SetupPage makes:

1. **Wait-ready**: `GET /api/data/health`, `GET /api/wz/health`, retry until 200 or 60s.
2. **Upload WZ**: `POST /api/wz/inputs` with the canonical zip (multipart, ~1GB).
3. **Run extraction**: `POST /api/wz/extractions/runs`. Poll `GET /api/wz/extractions/status` until complete.
4. **Run data processing**: `POST /api/data/runs`. Poll `GET /api/data/status` until complete.
5. **Seed**: parallel POSTs to seed endpoints (drops, gachapons, NPC conversations, quest conversations, NPC shops, portal scripts, reactor scripts, map action scripts), each with poll-to-ready.

The Job is **idempotent**: each step's status endpoint reports a non-empty count when complete. The Job short-circuits on already-populated stages.

**Auth**: SetupPage in dev mode does not require auth on these endpoints. The bootstrap Job runs in-cluster on the per-PR namespace's default ServiceAccount; calls go to `http://atlas-ingress.atlas-pr-<N>.svc.cluster.local` (cluster-internal DNS). NetworkPolicy is not v1 scope.

**Failure**: `backoffLimit: 3`. On exhaustion the Job is failed; Argo marks the Application `Degraded` and the env stays up so a maintainer can `kubectl logs job/atlas-pr-bootstrap` and rerun manually (`kubectl create -f` after deleting the failed job, or scale a new copy via `kubectl create job --from=cronjob/...` pattern). The runbook (§9) covers this.

### 6.3 PostSync DNS-register Job

`overlays/pr/postsync-pihole-add.yaml`:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: atlas-pr-pihole-register
  annotations:
    argocd.argoproj.io/hook: PostSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
    argocd.argoproj.io/sync-wave: "10"  # after bootstrap (default wave 0)
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: pihole-register
          image: curlimages/curl:8.10.1
          command: ["/bin/sh", "/scripts/register.sh"]
          envFrom:
            - configMapRef: { name: atlas-env-tokens }  # ATLAS_ENV
          env:
            - name: PR_NUMBER
              value: "PLACEHOLDER"
            - name: TRAEFIK_LB_IP
              value: "192.168.23.230"
          envFrom:
            - secretRef: { name: pihole-credentials }   # PIHOLE_API_BASE_1, PIHOLE_TOKEN_1, ...
          volumeMounts:
            - name: scripts
              mountPath: /scripts
      volumes:
        - name: scripts
          configMap: { name: atlas-pr-pihole-script }
```

The script performs `POST /api/config/dns/hosts` against both Pi-hole servers. **Tolerates per-server failure**: logs and exits 0 even if one server returns non-2xx, but exits non-zero only if **both** fail. This matches the PRD's "DNS will resolve via the other" guarantee.

`pihole-credentials` Secret lives in **the per-PR namespace**, projected from a sealed secret in the `argocd` namespace via Argo CD's resource templating. The plan phase chooses between (a) a Secret per PR namespace generated from a master Secret, or (b) a ClusterSecret-style sharing approach via `kubernetes-reflector`. **Recommendation: (a)**, generated as part of the Argo hook Job's setup, since the credentials are static and small.

### 6.4 PostDelete cleanup Job

`overlays/pr/postdelete-cleanup.yaml`:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: atlas-pr-cleanup
  annotations:
    argocd.argoproj.io/hook: PostDelete
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
spec:
  backoffLimit: 0  # do not retry — leave for manual investigation if it fails
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: cleanup
          image: ghcr.io/chronicle20/atlas-pr-bootstrap:latest  # same image, different entrypoint
          command: ["/bin/cleanup.sh"]
          envFrom:
            - configMapRef: { name: atlas-env-tokens }
            - configMapRef: { name: atlas-db-names }
            - secretRef: { name: db-credentials }
            - secretRef: { name: pihole-credentials }
```

Sequence (in this order; one transaction per resource type, each emits structured logs to Loki):

1. Drop Postgres DBs: for each name in `atlas-db-names`, `psql -c 'DROP DATABASE IF EXISTS "<name>-<env>"'`.
2. Delete Kafka topics: `kafka-topics.sh --delete --topic '.*-<env>$'`.
3. Delete Kafka consumer groups: `kafka-consumer-groups.sh --delete --group '.*\[<env>\]'`.
4. Redis SCAN+DEL: `redis-cli --scan --pattern '<env>:*' | xargs -n 1000 redis-cli DEL`.
5. ghcr image-tag delete: `gh api --method DELETE /user/packages/container/atlas-<svc>/versions/<id>` for every tag matching `pr-<N>-*`. PAT injected from secret. (gh CLI is bundled in the bootstrap image.)
6. Pi-hole DELETE: revoke A record on both servers.

**Failure semantics**: Each step is wrapped in a `set -e` shell function. If any step fails, the Job exits non-zero, and Argo records the Application as `cleanup-failed`. The Application is **not deleted from Argo** in that case — the PostDelete annotation is kept and the runbook (§9.4) covers manual investigation. Partial cleanups are visible in Loki via `atlas.cleanup-step` log fields.

---

## 7. Migration Strategy for `main`

The current `main` is deployed by manual `kubectl apply -f deploy/k8s/` into namespace `atlas` with unsuffixed DBs/topics/keys. Symmetric `main` (`ATLAS_ENV=main`, namespace `atlas-main`, every name suffixed `-main`) requires a **one-time scheduled-downtime migration** of ~30 minutes:

1. **Drain `atlas` namespace.** Scale every Atlas Deployment to 0 — clients disconnect. This is the start of the maintenance window.
2. **Rename Postgres DBs.** `ALTER DATABASE "atlas-<svc>" RENAME TO "atlas-<svc>-main"` per service. Atomic, fast, no data move.
3. **Drop legacy Kafka topics** (best-effort). Topics with no consumers carry no in-flight load. New `-main`-suffixed topics auto-create on first publish.
4. **Flush legacy Redis keys.** `SCAN+DEL` under the `atlas:` prefix. Atlas Redis content is mostly cache; warm-up is automatic on next request.
5. **Delete the `atlas` namespace.** Reclaims orphaned PVCs (`atlas-data-pvc`, `atlas-wz-input-pvc`, `atlas-assets-pvc`); Longhorn deletes the PVs (verified `reclaimPolicy: Delete`).
6. **Apply `Application(atlas-main)` on bee.** Argo creates the `atlas-main` namespace, applies the rendered overlay, fires PostSync hooks. Bootstrap re-uploads the canonical WZ zip and seeds every domain into the freshly-renamed `*-main` DBs.
7. **End of maintenance window.** Verify `ATLAS_ENV=main` everywhere; smoke-test a Maple client connecting via `192.168.23.231` (login) → `192.168.23.232` (channel).
8. **Stability window.** 7 days of green Argo syncs against `Application(atlas-main)`, then flip `prune: false` → `prune: true` and reapply.

The plan's Phase 11 (Tasks 11.0–11.10) is the runnable checklist for this procedure.

PR-environment infrastructure (`ApplicationSet(atlas-pr)`, cleanup CronJob) can land on bee **before** the cutover — they target the `atlas-pr-*` namespace pattern and don't touch `atlas`. Test the PR pipeline against an early canary PR before draining `main`.

**Rollback:** if cutover fails after Task 11.5 (namespace deleted), restore from the pg_dumpall snapshot taken in Task 11.0 plus the PVC contents (which are gone, so a re-bootstrap of the WZ zip is required). Time to recover ≈ time of original cutover. PVs from before the namespace deletion are not recoverable past the point Longhorn reclaims them — capture the snapshot before Task 11.5.

---

## 8. Failure Modes and Recovery

| Failure | Detection | Recovery |
|---|---|---|
| Hash collision (two PRs hash to same suffix) | Argo `Application` sync fails with namespace/resource conflict | Manual: close-and-reopen one of the PRs (changes the head SHA, recomputes hash); long-term: bump suffix length to 6 hex chars |
| Bootstrap Job fails (e.g. WZ corrupt) | Argo PostSync hook reports `Failed`; Application status `Degraded` | Inspect `kubectl logs job/atlas-pr-bootstrap`. If transient, delete the Job and `argocd app sync atlas-pr-<N> --force`. If WZ canonical itself is bad, replace the Longhorn-PVC contents and resync. |
| Pi-hole API down (one server) | Job logs warning, exits 0 if other succeeded | DNS resolves via the surviving server. No action needed. Long-tail: replicate Pi-hole secrets to a third server. |
| Pi-hole API down (both) | Job exits non-zero, Application `Degraded` | Manual: register A records via Pi-hole admin UI; mark Application as `synced` via Argo CLI. |
| PostDelete cleanup fails partway | Argo records `cleanup-failed`; Application not deleted | Runbook §9.4: re-run the Job. Each step is idempotent; rerunning is safe. |
| Kafka topic auto-create disabled | First publish in PR env throws; Application `Degraded` | PreSync hook is the fallback path. Plan phase verifies broker config; if auto-create is off, the PreSync hook ships in the initial overlay. |
| GitHub PAT for Argo CD expires | All ApplicationSet syncs fail; Argo logs token-auth error | Rotate PAT; replace `argocd-repo-creds-...` Secret. Documented in §9.5. |
| ghcr image-tag delete rate-limited | Cleanup Job logs 429 retries | Step is best-effort; tags expire eventually; runbook covers manual cleanup if accumulating. |
| Re-opened closed PR | CronJob clears `cleanup-deadline`; Argo re-syncs Application from `head_sha` | No action needed; designed flow. |

Single greatest risk: bootstrap latency. If the WZ ingest takes longer than reviewers will tolerate, the value of the entire feature is reduced. v1 measurement plan: emit Loki logs with `atlas.bootstrap-step-duration-ms` per step; runbook §9.6 has the queries.

---

## 9. Documentation Deliverables

Committed to this repo at PR time:

- **`deploy/k8s/README.md`** — describes the base/overlay structure, how `ATLAS_ENV` flows, how to add a new service.
- **`docs/runbooks/ephemeral-pr-deployments.md`** — covers, in order:
  9.1 First-time setup: creating the Longhorn `atlas-wz-canonical` PVC and seeding the canonical WZ zip.
  9.2 Opening / closing a PR env manually (force-create `Application` for branches not from a PR; force-delete bypassing grace period).
  9.3 Inspecting a stuck env (`argocd app get`, `kubectl describe job`, Loki query examples).
  9.4 Re-running a failed PostDelete cleanup.
  9.5 Rotating credentials: GitHub PAT for Argo, Pi-hole API tokens.
  9.6 Loki/Grafana queries for bootstrap-duration and cleanup-success metrics.
  9.7 Hash-collision resolution.
- **`docs/observability.md`** — append a section: "Filtering by environment uses the label `atlas.env=<token>`. Main env is `atlas.env=main`. PR envs are `atlas.env=<4hex>`." Include sample Loki / Prometheus queries.

Lives in the bee infra repo (`tumidanski/k3s`):

- A short `bee/argocd-README.md` describing where the Argo manifests came from and how to upgrade Argo (re-render the upstream manifest, re-apply the patches).

---

## 10. Out-of-Scope / Future Work

Tracked here so the plan phase doesn't accidentally pull these in:

- **TLS** for PR envs. v1 is HTTP only.
- **Auth** on PR envs. Open to anyone on the home network.
- **Per-env ephemeral Postgres/Kafka/Redis pods.** Shared infra is a deliberate constraint.
- **Cross-environment data migration** (cloning `main` into a PR env).
- **Tenant data preservation across envs.** Each PR starts empty.
- **Slack/GitHub notifications** when an env becomes ready or is reclaimed.
- **Performance/load-test grade envs.** v1 is functional review only.
- ~~Game-socket exposure per PR env.~~ — **In v1 scope as of design v1.1.** See §4.6.
- **Argo CD SSO.** Default admin password at v1.
- **Pre-extracted WZ PVC** for bootstrap-cache (open question 2 future work).
- **`external-dns` operator** with Pi-hole webhook (PRD §4.12 future work).
- **Per-env `LimitRange`.** Add when measured pressure justifies (PRD open question 10).
- **6-hex `ATLAS_ENV`** if the 4-hex collision rate exceeds tolerance.
- **Hardening of bootstrap NetworkPolicy** to scope the bootstrap ServiceAccount.

---

## 11. Acceptance Mapping

Each PRD §10 acceptance criterion maps to a section of this design:

| PRD AC | Design section |
|---|---|
| 10.1 Argo CD operational | §3, §7 |
| 10.2 Per-PR env lifecycle | §3.3, §6, §3.4 |
| 10.3 Isolation verification | §5 |
| 10.4 Code sweeps complete | §5.4, §5.5, §5.6 |
| 10.5 CI / image pipeline | §3.3, §4.3, plus §12 below |
| 10.6 Documentation | §9 |
| 10.7 Failure modes documented | §8, §9 |
| 10.8 No regressions in main env | §7 |

---

## 12. CI Pipeline Changes

`.github/workflows/pr-validation.yml`:

- Add a new `build-docker-pr` job (after the existing `build-docker` validation-only job). Mirrors the build steps of `main-publish.yml`'s `build-amd64` job (single-arch is enough for v1; bee nodes are mixed amd64/arm64 but Argo schedules to amd64 nodes — verified at plan phase). Tag: `pr-<PR_NUMBER>-<HEAD_SHORT_SHA>`. Pushes to ghcr (PR validation already authenticates via `secrets.GHCR_TOKEN`).
- Trigger condition: `needs.detect-changes.outputs.docker-services-matrix != '[]'`. Unchanged services keep the `:latest` tag from `main-publish` — the per-PR `images:` mapping in the Kustomize PR overlay covers only services that were rebuilt for this PR. Generation: at plan-phase a small action computes the per-PR `images:` mapping from the changed-service matrix and emits it as a workflow artifact consumed by the per-PR Argo `Application` (or, simpler: the ApplicationSet template defaults all services to `:latest` and only overrides those with a per-PR build via a `kustomize edit set image` step in a follow-up Argo hook). **Recommendation: defaults to `:latest`, ApplicationSet's `images:` list overrides for changed services only**, computed by Argo's Go template from a `.github/changed-services.json` artifact published as a release asset on the PR's head SHA tag.

`.github/workflows/pr-cleanup.yml` (new):

- Trigger: `pull_request: { types: [closed] }`.
- Steps:
  1. Checkout (just for action availability).
  2. Compute the list of services that ever had a per-PR image built for this PR (query ghcr API: list package versions with tag prefix `pr-<N>-`).
  3. Delete those tags via `gh api --method DELETE`.
  4. (Optional) Notify Argo via webhook to immediately mark the Application's deletion deadline. The cleanup CronJob (§3.4) is the canonical actor; the webhook is a latency optimisation only.

The CI pipeline is intentionally minimal at v1 — Argo CD owns deployment; CI owns image production.

---

## 13. Estimated Plan Decomposition (preview only)

Surface the plan phase will likely produce, listed for sequencing intuition:

1. Verify Postgres `db-credentials` user has `CREATEDB`; verify `auto.create.topics.enable=true`. Patch if not.
2. `libs/atlas-redis` env-aware `keyPrefix` + tests + audit fix in atlas-buffs (and any other found by grep).
3. `libs/atlas-kafka/consumergroup` resolver + tests.
4. Service sweep: 49 `main.go` edits (one PR or split for review ergonomics — likely split by service area).
5. Generate `patches/consumer-group-env.yaml` from main.go literals (committed alongside the script).
6. Restructure `deploy/k8s/` into base + overlays.
7. Build the bootstrap container image and publish to ghcr.
8. Create the Longhorn `atlas-wz-canonical` PVC, seed it with canonical WZ.
9. Argo CD install on bee + GitHub PAT secret.
10. `Application(atlas-main)` with `prune: false`; verify zero-diff sync.
11. ApplicationSet for PR generator + cleanup CronJob + Pi-hole credentials Secret (sealed).
12. Extend `pr-validation.yml`; create `pr-cleanup.yml`.
13. Test end-to-end with one canary PR.
14. After 1 week stability, flip `Application(atlas-main)` to `prune: true`.
15. Documentation (READMEs, runbooks).

This is sequencing intuition only; the plan phase produces the authoritative task graph.
