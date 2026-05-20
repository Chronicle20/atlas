# Task 072 — Implementation Context

Companion to `plan.md`. Captures the concrete files, current behavior, and decisions an implementer needs to keep the plan grounded.

## 1. Current seed surface (verified by grep)

| Service | Seed package(s) | Routes (verbatim) | Bundled data dir (verbatim) | Dockerfile COPY |
|---|---|---|---|---|
| atlas-drop-information | `atlas.com/dis/seed/` | `POST /drops/seed`, `GET /drops/seed/status` | `services/atlas-drop-information/drops/{monsters,continents,reactors}/` | `COPY services/atlas-drop-information/drops /drops` |
| atlas-gachapons | `atlas.com/gachapons/seed/` | `POST /gachapons/seed`, `GET /gachapons/seed/status` | `services/atlas-gachapons/data/{gachapons,gachapon_items,global_gachapon_items}.json` | `COPY services/atlas-gachapons/data /gachapons/data` |
| atlas-map-actions | `atlas.com/map-actions/script/` (seed.go + seed_status.go) | `POST /maps/actions/seed`, `GET /maps/actions/seed/status` | `services/atlas-map-actions/scripts/map/{onUserEnter,onFirstUserEnter}/` | `COPY services/atlas-map-actions/scripts /scripts` |
| atlas-reactor-actions | `atlas.com/reactor/script/` | `POST /reactors/actions/seed`, `GET /reactors/actions/seed/status` | `services/atlas-reactor-actions/scripts/reactors/` | `COPY services/atlas-reactor-actions/scripts /scripts` |
| atlas-portal-actions | `atlas.com/portal/script/` | `POST /portals/scripts/seed`, `GET /portals/scripts/seed/status` | `services/atlas-portal-actions/scripts/portals/` | `COPY services/atlas-portal-actions/scripts /scripts` |
| atlas-npc-conversations | `atlas.com/npc/conversation/{npc,quest}/` | `POST /npcs/conversations/seed`, `GET /npcs/conversations/seed/status`, `POST /quests/conversations/seed`, `GET /quests/conversations/seed/status` | `services/atlas-npc-conversations/conversations/{npc,quests}/` | `COPY services/atlas-npc-conversations/conversations /conversations` |
| atlas-npc-shops | `atlas.com/npc/seed/` (+ loader in `atlas.com/npc/shops/seed.go`) | `POST /shops/seed`, `GET /shops/seed/status` | `services/atlas-npc-shops/shops/*.json` (NOT `atlas.com/npc/data/` — that is Go source) | `COPY services/atlas-npc-shops/shops /shops` |
| atlas-party-quests | `atlas.com/party-quests/definition/` (seed.go + handler in resource.go) | `POST /party-quests/definitions/seed`, `GET /party-quests/definitions/seed/status` | `services/atlas-party-quests/party-quests/*.json` | `COPY services/atlas-party-quests/party-quests /party-quests` |

**PRD Open Questions 1 and 2 resolved against current code:** the URLs above are the verified literals; the migration preserves them verbatim.

**PRD body error corrected:** the PRD §7.8 says atlas-npc-shops bundled data lives at `data/`; in reality the data Dockerfile-copied is at `services/atlas-npc-shops/shops/` (the `atlas.com/npc/data/` tree is Go source for upstream-service clients and is NOT touched by this task).

## 2. Existing seed env vars (to be removed during migration)

These per-service env vars must be deleted from k8s manifests and compose entries on the same commit that drops the bundled data:

- `MONSTER_DROPS_PATH`, `CONTINENT_DROPS_PATH`, `REACTOR_DROPS_PATH` (atlas-drop-information)
- `GACHAPONS_DATA_PATH`, `GACHAPON_ITEMS_DATA_PATH`, `GLOBAL_ITEMS_DATA_PATH` (atlas-gachapons)
- `NPC_CONVERSATIONS_PATH`, `QUEST_CONVERSATIONS_PATH` (atlas-npc-conversations; second one to verify in code)
- `SHOPS_DATA_PATH` (atlas-npc-shops)
- script-services equivalents (`MAP_SCRIPTS_PATH`, `REACTOR_SCRIPTS_PATH`, `PORTAL_SCRIPTS_PATH`; exact names to verify in code)
- party-quests' equivalent (to verify)

Each per-service migration task in `plan.md` includes a grep step to enumerate the actual var names before deletion.

## 3. Existing JSON shapes (need wrapping)

Most catalog files are plain JSON. Only `atlas-drop-information/drops/reactors/reactor-*.json` is already a JSON:API document (verified). Everything else is either:

- A flat array of objects keyed by an id field (e.g., monster_drops.json — array of `{monsterId, itemId, ...}` rows that must be grouped per `monsterId`).
- A single object per file (e.g., npc-shop `21000.json` is `{npcId: 21000, recharger, commodities: [...]}`).

The `wrap-jsonapi` splitter handles all of the per-file cases. The aggregating splitters (`split-monster-drops`, `split-continent-drops`, `split-gachapons`) handle the array-grouping cases.

## 4. Library dependencies that must land in every consumer's Dockerfile

Per CLAUDE.md, adding `libs/atlas-seeder` requires updating each consumer Dockerfile in four locations:

1. The `COPY libs/atlas-<lib>/go.mod ...` lines near the top.
2. The `echo '    ./libs/atlas-<lib>' >> go.work` lines in the synthesized go.work block.
3. The `COPY libs/atlas-<lib> libs/atlas-<lib>` source-copy lines.
4. The `go mod edit -replace=github.com/Chronicle20/atlas/libs/atlas-<lib>=/app/libs/atlas-<lib>` flag in the `go mod edit` call.

Dockerfiles must also drop the corresponding bundled-data `COPY` line (e.g., `COPY services/atlas-drop-information/drops /drops`) on the same commit.

## 5. Plumbing for k8s

`deploy/k8s/base/` is plain YAML manifests assembled by a top-level `kustomization.yaml`. The overlay tree at `deploy/k8s/overlays/{main,pr}/` applies cross-cutting patches. The seed-catalog Kustomize component (per design §6.1) is new infrastructure under `deploy/k8s/base/components/seed-catalog/` and each in-scope `atlas-<svc>.yaml` references it via the component-reference mechanism.

The git-sync image is `registry.k8s.io/git-sync/git-sync:v4.4.0`. Mount path inside service container is `/var/run/seed-catalog`. ConfigMap `seed-catalog-config` carries `GITSYNC_REPO`, `GITSYNC_REF`, `GITSYNC_DIR=deploy/seed`, `GITSYNC_PERIOD=60s`.

## 6. Plumbing for docker-compose

`deploy/compose/docker-compose.core.yml` is where the in-scope services already live (NOT `docker-compose.yml`). The `x-seed-catalog` anchor and the `<<: *seed-catalog` references go in `docker-compose.core.yml`. The bind-mount source is `../seed:/var/run/seed-catalog:ro` (relative to `deploy/compose/`).

## 7. Library design lock-ins (from design.md)

- `Subdomain[J, M]` is generic; `SubdomainAny` is the type-erased adapter from `AdaptSubdomain[J, M](s)`.
- `Group{Name, URLPrefix, Subdomains []SubdomainAny}` is one POST/GET endpoint pair.
- `CatalogSource.Roots(t)` returns an ordered slice; the FilesystemCatalogSource returns one element today but the walker honors order so the `_base/` overlay model is a future config change rather than a refactor.
- Per-subdomain errors aggregate into `SubdomainCounts.Errors` (cap 100); the group does NOT abort on per-file or per-subdomain failure. Only context death / errgroup-level error aborts.
- `seed_state` row is upserted on every `Seed` completion, including partial failure. Outcome label is `success` / `partial` / `failure` for Prometheus counter.
- `Walk(root, relPath)` skips files starting with `_` or `.` and subdirectories starting with `_`.
- Non-entity files (e.g., gachapons' `_global/items.json`) load via a `Subdomain` whose `EntityIDPattern() == nil`, signaling "load exactly one named file" rather than "iterate entity files".

## 8. SEED_CATALOG_ROOT resolution

Containers: env `SEED_CATALOG_ROOT=/var/run/seed-catalog`. Per tenant, the lib resolves the catalog root as:

```
<SEED_CATALOG_ROOT>/<tenant.Region()>/<tenant.MajorVersion()>_<tenant.MinorVersion()>
```

Dev (no env): each service's `seed/groups.go` passes `./deploy/seed` as the fallback root. The lib normalizes via `filepath.Abs` so working-directory drift does not break the fallback.

## 9. Catalog regions/versions to ship on day one

Day-one tree:

```
deploy/seed/
├── _schema/                # JSON Schema files for the linter
├── gms/12_1/               # bootstrapped from 83_1
├── gms/83_1/               # produced by splitters from current bundled data
├── gms/87_1/               # bootstrapped from 83_1
├── gms/92_1/               # bootstrapped from 83_1
├── gms/95_1/               # bootstrapped from 83_1
└── jms/185_1/              # bootstrapped from 83_1
```

Bootstrap copies are byte-identical to `gms/83_1/` except for `CATALOG_REVISION`, which reads `bootstrapped-from-gms-83_1-@<commit-sha>`.

## 10. Migration sequence (locks design §4.1 + risk mitigation)

1. `libs/atlas-seeder` first (no consumers).
2. `atlas-gachapons` second (smallest catalog; stresses the `_global/` non-entity edge case).
3. `atlas-drop-information` third (largest catalog at 3.1 MB; three subdomains validates per-subdomain orchestration).
4. The remaining five (atlas-map-actions, atlas-reactor-actions, atlas-portal-actions, atlas-npc-conversations, atlas-npc-shops, atlas-party-quests) in any order — they are mechanical applications of the same recipe.
5. CI catalog linter ships alongside the first per-service migration (gachapons) so subsequent migrations have a working CI gate.
6. End-to-end verification (compose smoke + k8s `--dry-run=server`) at the end.

## 11. Tests that must survive the migration

These existing test files reference the seed surface; renaming handlers without updating test expectations breaks them:

- `services/atlas-drop-information/atlas.com/dis/seed/{processor_test,status_test}.go`
- `services/atlas-gachapons/atlas.com/gachapons/seed/{status_test,seed_test}.go` (verify)
- `services/atlas-{map,reactor,portal}-actions/atlas.com/.../script/{seed_status_test}.go`
- `services/atlas-npc-conversations/atlas.com/npc/conversation/{npc,quest}/seed_status_test.go`
- `services/atlas-npc-shops/atlas.com/npc/seed/{status_test,seed_test}.go`
- `services/atlas-party-quests/atlas.com/party-quests/definition/seed_test.go` (verify)

Each per-service migration rewrites the test bodies to assert against the new lib-backed handler responses (which must include the new `catalogRevision`, `tenantSeededRevision`, `tenantSeededAt` fields per PRD §5.2). Route URLs and the bulk of the assertions stay identical.

## 12. Existing patterns reused by the lib

- `libs/atlas-tenant`: `tenant.MustFromContext(ctx)`, `tenant.WithContext(ctx, t)`, `tenant.Model.{Id, Region, MajorVersion, MinorVersion}`. Verified at `libs/atlas-tenant/tenant.go:17-29`.
- `libs/atlas-rest/server`: `RouteInitializer`, `New(l).WithContext(...).AddRouteInitializer(...)`.
- `jsonapi.ServerInformation` from `github.com/jtumidanski/api2go/jsonapi` for status response marshaling.
- `golang.org/x/sync/errgroup` for the parallel subdomain fan-out (already a transitive dep in the consumer services).
- `gorm.io/datatypes` for `seed_state.result_summary` JSON column (NEW dep for services that don't have it; verify per service).

## 13. Non-goals (carry from PRD)

- No reconciler Job.
- atlas-configurations and atlas-tenants are NOT touched.
- No `_base/<region>/` overlay used on day one (the seam exists in the loader but no overlay is configured).
- No per-entity bucketing (`monsters/1/100100.json`); flat directories on day one.
- No splitter self-test against the linter; splitters and linter are independently validated.

## 14. Risk hot-spots (from design §9 — implementer must keep an eye on)

- Dockerfile lib-list drift: every per-service migration commit MUST end with `docker build -f services/<svc>/Dockerfile .` from the worktree root before the commit lands.
- Splitter non-determinism: every splitter has a "run twice, byte-diff" test; reruns must be zero-diff.
- URL preservation: every per-service migration sub-step diffs `router.HandleFunc(...)` literals before and after; any URL change blocks the commit.
- `git-sync` sidecar failure surfacing: lib must log `WARN` when `CATALOG_REVISION` is missing/empty; `seed_state.result_summary` records zero counts; this is the operational signal.
