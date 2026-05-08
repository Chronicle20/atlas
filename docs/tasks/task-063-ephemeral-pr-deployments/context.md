# Ephemeral Per-PR Deployments — Implementation Context

Companion to `plan.md`. Lists the load-bearing files, the irreversible decisions made in `design.md`, and the order in which dependencies bind. Read this first if you've never opened the Atlas repo before — it points at the code the plan touches without re-explaining patterns the design already covered.

---

## 1. Repository layout that matters here

```
atlas/                                        # this worktree's root
├── deploy/k8s/                               # flat manifests today; restructured by Phase 7
│   ├── atlas-*.yaml                          # one Deployment+Service per microservice (54 files)
│   ├── env-configmap.yaml                    # shared atlas-env ConfigMap (topic+infra env vars)
│   ├── ingress.yaml                          # nginx atlas-ingress + Traefik Ingress for dev.atlas.home
│   ├── namespace.yaml
│   └── secrets.example.yaml
├── libs/
│   ├── atlas-database/connection.go          # already env-driven (DB_NAME via os.LookupEnv); no change
│   ├── atlas-kafka/                          # Phase 2 adds consumergroup/ subpackage
│   │   ├── consumer/                         # InitConsumers(...)(cmf)(groupId) registration helper
│   │   ├── topic/provider.go                 # EnvProvider(l)(envVarName)() — already env-driven
│   │   └── producer/manager.go
│   ├── atlas-object-id/allocator.go          # Phase 3 fix: 2 hardcoded "atlas:" prefixes
│   ├── atlas-redis/
│   │   ├── connection.go                     # Connect(l) reads REDIS_URL/PASSWORD; Phase 1 leaves untouched
│   │   ├── keys.go                           # Phase 1 modifies: const keyPrefix → var + computeKeyPrefix() + KeyPrefix()
│   │   ├── registry.go / tenant_registry.go  # use namespacedKey via the const — pick up env-aware prefix automatically
│   │   ├── coalesced.go / id.go / index.go / lock.go / ttl.go
│   └── atlas-tenant/                         # Model.Region/MajorVersion/MinorVersion used by TenantKey
├── services/
│   ├── atlas-<name>/atlas.com/<module>/main.go  # 49 services, Phase 4 sweep
│   ├── atlas-buffs, atlas-npc-shops, atlas-portals, atlas-pets, atlas-skills,
│   │   atlas-expressions, atlas-maps, atlas-chairs, atlas-storage, atlas-character,
│   │   atlas-chalkboards, atlas-monsters    # Phase 5 audit fixes (12 services)
│   └── atlas-pr-bootstrap/                   # Phase 6 NEW: Dockerfile + scripts/ + test/
├── .github/
│   ├── config/services.json                  # service registry (Phase 6 adds atlas-pr-bootstrap; Phase 9 reads for cleanup)
│   ├── actions/detect-changes/               # used by both PR validation and main publish
│   └── workflows/
│       ├── main-publish.yml                  # builds :latest amd64+arm64, pushes manifest list
│       ├── pr-validation.yml                 # Phase 9 adds build-docker-pr job
│       └── pr-cleanup.yml                    # Phase 9 NEW
├── docs/
│   ├── runbooks/ephemeral-pr-deployments.md  # Phase 10.2 NEW
│   ├── observability.md                      # Phase 10.3 appends env-label section
│   └── tasks/task-063-ephemeral-pr-deployments/   # this PRD/design/plan
└── deploy/argocd-bee/                        # Phase 8 NEW (delivered to tumidanski/k3s manually)
    ├── argocd.yml
    ├── argocd-atlas-main.yml
    ├── argocd-atlas-pr.yml
    ├── argocd-cleanup-cronjob.yml
    ├── argocd-pihole-secret.yml.example
    ├── argocd-ghcr-secret.yml.example
    └── README.md
```

## 2. Key code surfaces

### 2.1 `libs/atlas-redis/keys.go`

Current (verbatim):
```go
const keyPrefix = "atlas"
const keySeparator = ":"

func TenantKey(t tenant.Model) string { … fmt.Sprintf … }
func namespacedKey(namespace string, parts ...string) string { … }
func tenantEntityKey(namespace string, t tenant.Model, entityKey string) string { … }
func tenantScanPattern(namespace string, t tenant.Model) string { … }
func CompositeKey(parts ...string) string { … }
```

Phase 1 changes ONLY `const keyPrefix = "atlas"` to `var keyPrefix = computeKeyPrefix(os.Getenv("ATLAS_ENV"))`, plus adds `computeKeyPrefix(string) string` and `KeyPrefix() string`. Do not touch the helpers — they all route through the package-level `keyPrefix` and pick up the change for free. The plan offers a "minimal diff" path that preserves `fmt.Sprintf` in `TenantKey` to avoid churn; take it.

### 2.2 Service `main.go` consumer-group literal pattern

47 services have:
```go
const serviceName = "atlas-<x>"
const consumerGroupId = "<Capitalised> Service"
// ...
func main() {
    ...
    cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
    accountConsumer.InitConsumers(l)(cmf)(consumerGroupId)
    ...
}
```

The edit is mechanical: keyword change `const` → `var`, RHS becomes `consumergroup.Resolve("<literal>")`. Add the import. Build the service module. Done.

`atlas-channel` and `atlas-login` are special — they compute the group at runtime from a UUID-bearing template after loading `configuration.GetServiceConfig()`. The wrap is the same shape but applied at the runtime line, not the const declaration.

### 2.3 Audit-fix pattern (Phase 5)

Every service that hardcodes `"atlas:"` does so inside a key-construction function. Rewrite the literal as `fmt.Sprintf("%s:...", atlasredis.KeyPrefix(), ...)` (preserving the rest of the format string). Several services already alias `libs/atlas-redis` as `atlasredis`; verify with `grep -n 'atlasredis "' <file>` before editing.

`audit-redis-prefix.txt` (committed by Phase 0 Task 0.3) is the source of truth for what to fix and where. If new hits appear after Phase 5 closes, add a Task 5.13 follow-up.

### 2.4 `deploy/k8s/base/<svc>.yaml` shape

Each looks like:
```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-<svc>
  namespace: atlas         # ← stripped by Phase 7.1 sed
spec:
  replicas: 2
  template:
    spec:
      containers:
        - name: <container>          # often shorter than <svc>, e.g. "account" not "atlas-account"
          image: ghcr.io/.../atlas-<svc>:latest
          envFrom:
            - configMapRef:
                name: atlas-env
          env:
            - name: LOG_LEVEL
              value: "debug"
            - name: DB_USER
              valueFrom: { secretKeyRef: { name: db-credentials, key: DB_USER } }
            - name: DB_PASSWORD
              valueFrom: { secretKeyRef: { name: db-credentials, key: DB_PASSWORD } }
            - name: DB_NAME
              value: "atlas-<svc>s"  # this literal is what the PR overlay's db-name-suffix patch suffixes
```

Container name varies — the consumer-group patch generator must read `containers[0].name` from the Deployment manifest, not guess it. See plan §7.3 Step 1's note on `yq eval`.

### 2.5 `.github/config/services.json`

Source of truth for what's an Atlas image. Used by:
- detect-changes action → matrix outputs
- main-publish.yml → which images to build :latest
- pr-validation.yml → which images to validate-build
- (Phase 9) pr-cleanup.yml → which packages to scan for `pr-<N>-*` tags
- (Phase 7) `deploy/k8s/overlays/main/kustomization.yaml`'s `images:` list (generated by jq script)
- (Phase 7) PostDelete cleanup hook's `ATLAS_SERVICES` env (generated by jq script)

When adding `atlas-pr-bootstrap` to it, follow the existing schema. Use `type: support-image` since it's not a long-running service. The schema at `.github/config/services.schema.json` may not have this type; if it complains, add the type to the schema.

## 3. Decisions locked by design.md (v1.1)

These are not up for revisitation in the plan phase — they were resolved during brainstorming:

1. **`replacements:` over Argo Kustomize plugin.** Plain Kustomize 4.5+ string substitution covers everything; no plugin install needed.
2. **No caching in v1 bootstrap.** Always run full WZ ingest. Pre-extracted-PVC optimization is future work.
3. **Longhorn `ReadOnlyMany` PVC for canonical WZ zip.** Not ghcr image, not S3.
4. **`ATLAS_ENV=main` on the main env, `ATLAS_ENV=<4hex>` on PR envs.** Symmetric — same overlay shape, same code path; only the literal differs. Cutover migration in Phase 11 renames existing DBs in place and recreates the namespace as `atlas-main`. (Reversed from earlier draft that left main unsuffixed; the asymmetry was paying special-case complexity in code, manifests, and hooks for one-time migration savings — net loss.)
5. **HTTP only at v1.** No TLS, no auth on PR envs. Reviewers exercise on home network.
6. **Per-PR game-socket exposure via one MetalLB-allocated LB IP.** Both atlas-login and atlas-channel back the same per-PR IP (their wire ports don't collide). Bootstrap Job seeds the IP into atlas-tenants `services` config. Soft cap on concurrent PRs = MetalLB pool free count (`preflight.md`).
7. **Argo SSO out of v1.** Default admin password.
8. **24h cleanup grace by default.** Annotation-overridable per Application.
9. **Bootstrap image is one image with two entrypoints.** Same Dockerfile, `bootstrap.sh` and `cleanup.sh` paths. Image carries `kubectl` so bootstrap can read its own LB Service status for the channel-host seed.
10. **`prune: false` on `Application(atlas-main)` initially.** Flip to `prune: true` after 1 week of clean syncs.
11. **PVC isolation by namespace.** `atlas-data-pvc`, `atlas-wz-input-pvc`, `atlas-assets-pvc` are namespace-scoped; the PR overlay creates fresh same-named PVCs in `atlas-pr-<N>`. Longhorn `reclaimPolicy: Delete` (verified preflight) reclaims PVs on namespace deletion.
12. **Main env keeps existing LoadBalancer IP reservations** (`192.168.23.231` login, `.232` channel). PR envs draw from the remaining MetalLB pool. The `lb-pin.yaml` patch in `overlays/main` enforces this.

## 4. Dependency order

Phases must execute mostly in order, but several are independent:

```
Phase 0 (preflight) ─→ Phase 1 (atlas-redis) ─┐
                                              ├─→ Phase 3 (atlas-object-id) ─→ Phase 5 (service audits)
Phase 0           ─→ Phase 2 (consumergroup) ─┴─→ Phase 4 (service sweep)

Phase 4  ─→ Phase 7.3 (consumer-group patch generator reads main.go literals)
Phase 5  ─→ (no downstream code, but Phase 7 manifests should be applied AFTER Phase 5 lands)

Phase 6 (bootstrap image) ─→ Phase 7.7 (PostSync references the image)
Phase 7 (manifest restructure) ─→ Phase 8 (Argo Application points at overlay paths)
Phase 8 (bee artifacts) ─→ Phase 11 (deployment cutover)

Phase 9 (CI) is independent of 1–8; can land in parallel.
Phase 10 (docs) follows 8–9 so URLs and file paths reference real artifacts.
```

Within phases, tasks are TDD where applicable (write test → fail → implement → pass → commit). Phase 4 and Phase 5 are mechanical but each commit must build cleanly — no batched broken-state commits.

## 5. Verification gates

| Gate | Command | When |
|---|---|---|
| atlas-redis test | `cd libs/atlas-redis && go test -race ./...` | After Phase 1 |
| consumergroup test | `cd libs/atlas-kafka && go test ./consumergroup/...` | After Phase 2 |
| atlas-object-id test | `cd libs/atlas-object-id && go test ./...` | After Phase 3 |
| Whole workspace build | `go build ./...` from repo root | After Phase 4 |
| Whole workspace test | `go test ./...` from repo root | After Phase 5 |
| Preflight Longhorn capacity | Phase 0 Task 0.4 commands | Before Phase 7 |
| Preflight MetalLB pool | Phase 0 Task 0.5 commands | Before Phase 7.10 |
| Preflight Longhorn RecurringJob exclusion label | Phase 0 Task 0.6 | Before Phase 7 PR overlay PVC patches |
| Preflight atlas-tenants channel-host config | Phase 0 Task 0.7 | Before Phase 6 bootstrap.sh implementation |
| Audit clean | `grep -rn '"atlas:' services/ libs/ --include='*.go' \| grep -v _test.go` | End of Phase 5 |
| Bootstrap image lint | `shellcheck services/atlas-pr-bootstrap/scripts/*.sh && bats services/atlas-pr-bootstrap/test/` | End of Phase 6 |
| Kustomize main render | `kustomize build deploy/k8s/overlays/main` | After Phase 7.2 |
| Kustomize PR render | `kustomize build deploy/k8s/overlays/pr` | End of Phase 7 |
| Workflows lint | `actionlint .github/workflows/*.yml` | End of Phase 9 |

## 6. External cluster facts referenced by the plan

These come from inspecting the live `bee` cluster (must be true at plan-execution time; Phase 0 verifies):

- Postgres host: `postgres.home:5432`, role from `db-credentials` secret, must have `CREATEDB`.
- Kafka brokers: `kafka.home:9093` (TLS port per current ConfigMap value).
- Redis: `redis.home:6379`, no password by default in dev.
- Traefik LoadBalancer IP: `192.168.23.230` (per `bee/traefik-helmchart.yaml`).
- Pi-holes: two external servers, base URLs and tokens via Secret to be created in `argocd` namespace.
- Longhorn storage class: `longhorn` (default).
- ghcr.io registry public; no image-pull secret needed by `atlas` namespace.

## 7. Out-of-scope bookkeeping

The plan deliberately does NOT cover:

- Per-env TLS termination
- Authentication on PR envs
- Grafana dashboard authoring (only filtering syntax in `observability.md`)
- atlas-tenants seed data per PR env (assumed to exist; if absent at first canary, add a presync-create-tenant Job in a follow-up)
- 6-hex `ATLAS_ENV` collision-resistant variant
- `external-dns` Pi-hole webhook provider
- Per-namespace `LimitRange`
- Bootstrap-cache (pre-extracted WZ PVC)
- Port-shifting fallback if MetalLB pool exhausts

If any of these surface during execution, file follow-up tasks rather than expanding the plan.

## 8. PR-time review focus

When the implementer dispatches `superpowers:requesting-code-review` after the plan completes, they should expect:

- `plan-adherence-reviewer` to verify every `[ ]` task is checked and committed.
- `backend-guidelines-reviewer` to scan the Go changes — primary risk areas are package-level `var keyPrefix` in `libs/atlas-redis` (init-order subtlety: anything reading `keyPrefix` at init time before this package init runs would break; in practice only the package's own helpers read it, and they're called lazily) and the Phase 4 mechanical sweep (forgotten imports).
- `frontend-guidelines-reviewer` is N/A for this task — no atlas-ui changes.

The audit reports land at `docs/tasks/task-063-ephemeral-pr-deployments/audit.md`.
