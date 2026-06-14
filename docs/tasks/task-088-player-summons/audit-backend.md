# Backend Audit — atlas-summons (task-088-player-summons)

- **Service Path:** services/atlas-summons/atlas.com/summons
- **Also audited:** libs/atlas-constants/summon, libs/atlas-packet/summon, services/atlas-channel (summon additions), services/atlas-monsters (ADD_PUPPET/REMOVE_PUPPET + puppet_registry + controller bias)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-12
- **Build:** PASS
- **Tests:** PASS (summon pkg `ok`, all other pkgs no test files)
- **Overall:** PASS

This is a registry-based (Redis) in-memory service modeled on atlas-monsters — it has NO GORM persistence layer. The DOM checklist items that presuppose `entity.go` / `administrator.go` / `provider.go` GORM access (DOM-02, DOM-03, DOM-04/05, DOM-08/09/16/17/18/19, DOM-10/11) map to the registry/REST equivalents and are evaluated against that reality, exactly as they would be for the atlas-monsters blueprint. No item that genuinely applies fails.

## Build & Test Results (verbatim)

- `cd services/atlas-summons/atlas.com/summons && go vet ./...` → clean (exit 0)
- `... && go build ./...` → clean (exit 0)
- `... && go test ./... -count=1` → `ok atlas-summons/summon 0.062s`; all other packages `[no test files]`
- `cd services/atlas-monsters/atlas.com/monsters && go vet ./...` → clean (exit 0)
- `rediskeyguard ./...` (built GOWORK=off, run via go.work) in summons → exit 0; in monsters → exit 0
- `docker buildx bake atlas-summons` → image built and exported (`naming to docker.io/library/atlas-summons:local done`), confirming the shared root Dockerfile + services.json + docker-bake.hcl wiring (DOM-22 truth-test PASS)

## Domain Checklist Results — summon package (primary domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go + NewBuilder/fluent setters/Build | PASS | summon/builder.go:36 `NewBuilder()`, :53-76 setters, :78 `Build()`. NOTE: `Build()` returns `Model` (no error). Matches the atlas-monsters blueprint convention for registry models; not a violation. |
| DOM-02 | Model→storage transform | PASS (registry analog) | summon/registry.go:74 `toStored(t,m)`; no GORM entity, so `ToEntity()` is N/A. |
| DOM-03 | Make/decode | PASS (registry analog) | summon/registry.go:113 `fromStored(s) (tenant.Model, Model, error)` returns `(…, error)`. |
| DOM-04 | Transform function | PASS | summon/rest.go:43 `func Transform(m Model) (RestModel, error)`. |
| DOM-05 | TransformSlice / no inline loops in handler | PASS | List handler uses `model.SliceMap(summon.Transform)(…)(model.ParallelMap())` — world/resource.go:46. No inline transform loop. |
| DOM-06 | Processor accepts FieldLogger | PASS | summon/processor.go:83 `NewProcessor(l logrus.FieldLogger, ctx context.Context)`. inventory/processor.go:29, data/skill/processor.go:22, effectivestats/processor.go all `logrus.FieldLogger`. |
| DOM-07 | Handlers pass d.Logger() | PASS | summon/resource.go:28 `NewProcessor(d.Logger(), d.Context())`; world/resource.go:38 same. No `logrus.StandardLogger()` anywhere. |
| DOM-08 | POST/PATCH use RegisterInputHandler | N/A | Service exposes only GET (`get_summon`, `get_summons_in_map`). No POST/PATCH REST endpoints; mutations arrive over Kafka commands. |
| DOM-09 | Transform errors handled | PASS | summon/resource.go:34-39 checks the err from `model.Map(Transform)`; world/resource.go:47-51 checks the SliceMap err. No `_, _ :=`. |
| DOM-10 | Test DB tenant callbacks | N/A | No GORM/SQLite; tests use miniredis (registry_test.go:19, processor_spawn_test.go:47). |
| DOM-11 | Providers lazy | PASS (registry analog) | Registry reads are method-based; REST clients return curried `requests.Request[T]` (data/skill/requests.go:17, inventory/requests.go:17). |
| DOM-12 | No os.Getenv in handlers | PASS | grep: only `kafka/consumer/consumer.go:23` (BOOTSTRAP_SERVERS, standard), main.go, leaderconfig.go. None in resource.go handlers. |
| DOM-13 | No cross-domain logic in handlers | PASS | Both handlers call only `summon.NewProcessor(...)` methods. |
| DOM-14 | Handlers don't call providers directly | PASS | summon/resource.go + world/resource.go call processor methods only. |
| DOM-15 | No direct entity creation in handlers | PASS | No `db.Create/Save/Delete` anywhere (grep clean); writes go through registry via processor. |
| DOM-16 | Write ops centralized | PASS (registry analog) | All state mutation goes through `Registry.Put/Update/Remove` (registry.go:187,238,256), invoked only from processor.go. |
| DOM-17 | Domain error → HTTP status | PASS | summon/resource.go:31 not-found→404, :37 transform fail→500; world/resource.go:42 →500, :49 →500. |
| DOM-18 | JSON:API interface on REST models | PASS | summon/rest.go:30-41 `GetID/SetID/GetName`. External client models add relationship methods (see EXT-01). |
| DOM-19 | Flat request models | N/A | No request (POST/PATCH) models. Response `RestModel` is flat (rest.go:12-28). |
| DOM-20 | Table-driven tests | PASS | e.g. ceiling_test.go, model_test.go, registry_test.go use `t.Run` subtests; 11 test files in summon pkg. |
| DOM-21 | Reuse atlas-constants types | PASS | world.Id/channel.Id/_map.Id used throughout (rest.go:24-26, kafka.go:24-27). Weapon ceiling uses `item.WeaponType`/`item.GetWeaponType`/`item.Id` (ceiling.go:80-111, inventory/processor.go:43). Skill ids referenced via `libs/atlas-constants/skill` named constants (roster.go:34-54). Summon roster lives in the shared `libs/atlas-constants/summon` package, not redeclared per-service. See Notes for two faithfully-copied literals. |
| DOM-22 | Docker build viable | PASS | Project uses a single root `Dockerfile` (ARG SERVICE) + services.json (line 432) + docker-bake.hcl (line 88) + go.work (line 76); the per-service 4-block template does not apply here. `docker buildx bake atlas-summons` built successfully. New libs already present in root Dockerfile (atlas-constants line 32/61, atlas-packet line 40/69). |
| DOM-23 | Kafka topic naming/configmap | PASS | `COMMAND_TOPIC_SUMMON` and `EVENT_TOPIC_SUMMON_STATUS` present in deploy/k8s/base/env-configmap.yaml:70,141 with `KEY: "KEY"` shape; NOT redeclared as literal env in deploy/k8s/base/atlas-summons.yaml (which uses `envFrom: configMapRef: atlas-env`, :21-23). ADD_PUPPET/REMOVE_PUPPET reuse the existing `COMMAND_TOPIC_MONSTER` (env-configmap.yaml:46) — no new topic needed. |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS | The processor exposes an `emit` field seam (processor.go:73-81); tests inject a no-op emitter (processor_spawn_test.go:69-70) so no live producer is hit. The 42s-hang failure mode is structurally impossible in these tests. BeholderTask uses the same seam (beholder_task.go:27,35). No `t.Cleanup(producer.ResetInstance)`. |

## Sub-Domain / Support Package Results

| Package | Type | Verdict |
|---------|------|---------|
| data/skill, data/skill/effect | external HTTP client + value objects | See EXT checklist |
| effectivestats | external HTTP client | See EXT checklist |
| inventory | external HTTP client (weapon type) | See EXT checklist |
| monster | local re-decl of atlas-monsters command contract (producer) | PASS — envelope tags verified byte-identical to consumer (see Cross-Service) |
| buff, character | local re-decl of buff/character command contracts (producer) | PASS — used by BeholderTask sweep |
| kafka/consumer/{summon,character} | command/event consumers | PASS — type-guarded handlers (consumer.go:44-87), header parsers set (:20) |
| world | GET list resource | PASS — see DOM rows |
| rest, logger, tasks, kafka/producer | support | PASS — standard wiring |

## External HTTP Client Checklist (data/skill, effectivestats, inventory)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | Relationship interface methods on JSON:API target structs | PASS | data/skill/rest.go:38-39 `SetToOneReferenceID`/`SetToManyReferenceIDs`; effectivestats/rest.go:31-32 same; inventory/rest.go:33,75 plus full GetReferences/GetReferencedIDs/SetReferencedStructs for the `assets` relationship (:46-99). `effect.RestModel` is a nested array attribute (`json:"effects"`), not a top-level api2go resource, so it correctly omits the methods. |
| EXT-02 | httptest-backed integration test | **FAIL (Important)** | No `httptest` server anywhere in atlas-summons (grep: zero matches). None of inventory/effectivestats/data-skill clients (incl. the api2go relationship-unmarshal path for the `assets` include in inventory/rest.go:83 `SetReferencedStructs`) is exercised against a real JSON:API body. The relationship-include decode is exactly the kind of logic that silently breaks and that EXT-02 exists to catch. Fix: add an httptest server per client that serves a representative atlas-data / atlas-inventory / atlas-effective-stats JSON:API fixture (inventory's fixture MUST carry an `assets` relationships block + `included` so SetReferencedStructs runs) and assert the domain method returns a populated struct. |
| EXT-03 | Distinguish 404 from other failures | PASS | Clients propagate the original error unchanged (data/skill/processor.go:30, inventory/processor.go:37-39, effectivestats). A genuine 404 surfaces as `requests.ErrNotFound`; transport/decode/5xx bubble up with their own error. Nothing collapses every failure into a misleading "not found", which is what EXT-03 guards against. |
| EXT-04 | URL via RootUrl(domain), not hardcoded | PASS | data/skill/requests.go:14 `requests.RootUrl("DATA")`; inventory/requests.go:14 `RootUrl("INVENTORY")`; effectivestats uses `RootUrl("EFFECTIVE_STATS")`. No hardcoded DNS. |

## Cross-Service Envelope Discipline (re-declared, not imported)

Verified byte-identical JSON tags between each atlas-summons local re-declaration and the owning consumer/producer (full diff in subagent verification):

- **COMMAND_TOPIC_MONSTER** (summons producer monster/producer.go vs monsters consumer kafka.go): PASS. DAMAGE / APPLY_STATUS / ADD_PUPPET / REMOVE_PUPPET bodies all match; monsters consumer has handlers wired for the two new puppet commands (consumer.go:68,71 → handleAddPuppet/handleRemovePuppet → PuppetRegistry).
- **EVENT_TOPIC_SUMMON_STATUS** (summons producer summon/kafka.go vs atlas-channel consumer): PASS. StatusEvent + CREATED/MOVED/ATTACKED/DAMAGED/DESTROYED/SKILL bodies match; all six handlers wired channel-side.
- **COMMAND_TOPIC_SUMMON SPAWN/MOVE/ATTACK/DAMAGE** (atlas-channel producer vs summons consumer): PASS. Envelope + SpawnCommandBody (incl. auraLevel/hexLevel) match.
- **EVENT_TOPIC_CHARACTER_STATUS** (atlas-character producer vs summons consumer kafka.go): PASS. LOGOUT / CHANNEL_CHANGED / MAP_CHANGED bodies match (wire string values identical; local Go constant names differ, which is irrelevant).

## Multi-Tenancy / Redis / Leader Election / Registry Singleton

- Tenant from context only: processor.go:85 `tenant.MustFromContext(ctx)`; sweeps rebuild tenant-scoped ctx via `tenant.WithContext` (beholder_task.go:57). No tenant in any REST payload (rest.go).
- Redis access ONLY via libs/atlas-redis types: registry.go uses `atlasredis.Registry`/`atlasredis.KeyedSet` (:158-159); puppet_registry.go same (:38-42); id_allocator wraps `objectid` lib. rediskeyguard clean in both modules.
- Registry singletons via sync.Once: registry.go:164,174 (`once.Do`); id_allocator.go:19,22; monsters puppet_registry.go:46,49.
- Leader election: main.go:98-117 gates sweep-task registration on `lock.New(...)` leader election; configurable + range-validated (leaderconfig.go). Non-leader path warns and runs unconditionally (explicit, documented).

## Summary

### Blocking (must fix)
- None.

### Non-Blocking — Important (should fix)
- **EXT-02**: No httptest-backed integration test for the three external HTTP clients (data/skill, effectivestats, inventory). The inventory client's api2go relationship-include unmarshal (`SetReferencedStructs`, inventory/rest.go:83) is untested against a real JSON:API body — exactly the decode path EXT-02 exists to protect. Add one httptest fixture test per client.

### Non-Blocking — Minor (notes)
- `data/skill/processor.go:41` bounds check `len(s.Effects()) < int(level-1)` is an off-by-one (accesses `Effects()[level-1]` when `len == level-1` would panic; correct guard is `< int(level)`). This is copied verbatim from the atlas-channel blueprint (`channel/data/skill/processor.go:42`); flagged as a latent panic risk, not a new-code guideline violation.
- `inventory/requests.go:18` passes the literal inventory type `1` rather than `inventory.TypeValueEquip` from `libs/atlas-constants/inventory`. This is copied verbatim from the effective-stats blueprint (`external/inventory/requests.go:10`, identical `type=%d` literal). Not flagged as a DOM-21 defect because it faithfully mirrors the blueprint; a future cleanup could use the named constant.
- `builder.Build()` returns `Model` with no validation/error. Consistent with the registry-model blueprint; acceptable, but the builder performs no invariant enforcement (e.g. it does not reject an empty `summonType`). Spawn-side logic gates this instead (processor.go:112 roster lookup).
