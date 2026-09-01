# Backend Audit — task-285-maker-skill-crafting

- **Service Path:** services/atlas-maker (new), plus changed packages in atlas-channel, atlas-configurations, atlas-data, atlas-inventory, atlas-saga-orchestrator, libs/atlas-packet, libs/atlas-saga
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-01
- **Build:** PASS
- **Tests:** all packages report `ok` (or `[no test files]`), 0 failed, across every changed module
- **Overall:** NEEDS-WORK

## Build & Test Results

`go build ./...` and `go test ./... -count=1` were run from each changed module's root:

- `services/atlas-maker/atlas.com/maker` — build clean, all packages `ok`.
- `services/atlas-channel/atlas.com/channel` — build clean, `go vet ./...` clean, all packages `ok`.
- `services/atlas-data/atlas.com/data` — build clean, all packages `ok`.
- `services/atlas-inventory/atlas.com/inventory` — build clean, all packages `ok`.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` — build clean, all packages `ok`.
- `libs/atlas-packet`, `libs/atlas-saga` — build clean, all packages `ok`.
- `services/atlas-configurations` changes are seed-data JSON only (`seed-data/templates/*.json`); no Go build surface changed.

`tools/goroutine-guard.sh` → `goroutineguard: 92 module(s), 8 parallel`, exit 0.
`tools/service-registration-guard.sh` → `service-registration-guard: clean`, exit 0.
`tools/gen-routes.sh --check` → `gen-routes: up to date`, exit 0.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure | Yes | atlas-maker: `character`, `compartment`, `quest`, `skill`, `crystalband`, `reagent`, `recipe`, `data/equipment`, `data/itemmake` all have `model.go`; `crystalband`, `reagent` also have `entity.go`/`rest.go`/`provider.go` |
| FILE placement | Yes | every changed Go package, no exemptions |
| SUB sub-domain | Yes | `craft` has `resource.go`, no `model.go` |
| REST | Yes | `craft`, `crystalband`, `reagent` have `resource.go`/`rest.go`/`processor.go`; atlas-inventory `compartment`/`asset` processors changed |
| Constants reuse | Yes | new types (`craft.Mode`, `craft.CraftError`, etc.); checked against `libs/atlas-constants` — no redeclaration found |
| Testing | Yes | every atlas-maker package ships `_test.go`; `DOM-33` interface changes in atlas-saga-orchestrator `compartment.Processor` |
| Cache | Yes | `recipe/processor.go` holds a package-level `recipeIndex` singleton |
| Messaging | Yes | `craft/emitter.go` calls `producer.ProviderImpl`; atlas-saga-orchestrator `compartment/producer.go` extended |
| Multi-tenancy | Yes | every `resource.go` in atlas-maker; `RootUrlFor(ctx, ...)` reads env/tenant state |
| Migration hygiene | No | no symbol moved between a service and a `libs/atlas-*` module |
| Deploy & topics | Yes | new `libs/atlas-saga` additions are lib-internal (not a new module); no topic env var added/renamed — N/A on closer read (see DOM-22/23 below) |
| Runtime safety (goroutines) | Yes | `tools/goroutine-guard.sh` run, exit 0 |
| Channel wire values | Yes | `libs/atlas-packet/character/{clientbound,serverbound}` and `services/atlas-channel` touched |
| Resilience | Yes | DB-backed atlas-maker handlers (`crystalband`, `reagent`) write `http.StatusInternalServerError`-adjacent paths via `server.WriteErrorResponse` |
| External clients | Yes | `character`, `quest`, `skill`, `compartment`, `data/equipment`, `data/itemmake` in atlas-maker all call another atlas service |
| Scaffolding | Yes | `services/atlas-maker/` added on this branch — full SCAFFOLD-01..09 run |
| Security | No | atlas-maker handles no tokens, auth, redirects, or secrets; no trigger fired |

## Checklist Results

### atlas-maker / craft (sub-domain shape: `resource.go`, no `model.go`, but also owns `Processor`/`processor.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic not in handler | PASS | `craft/resource.go:222` `handleCreateCraft` delegates to `p.Create(characterId, ...)`; no logic inlined |
| SUB-02 | Writes via administrator | N/A | craft performs no DB writes (no `gorm.DB` in `craft/processor.go`/`craft/inflight.go`) |
| SUB-03 | POST via `RegisterInputHandler[T]` | PASS | `craft/resource.go:95` `registerInput("create_maker_craft", ...)` with `rest.RegisterInputHandler[CraftRequestRestModel]` |
| SUB-04 | No manual JSON parsing in `resource.go` | PASS | no `json.NewDecoder`/`json.Unmarshal`/`io.ReadAll` in `craft/resource.go`; body decode is centralized in `rest.ParseInput` |
| FILE-01 | Processor interface/impl/constructor in `processor.go` or `processor_<group>.go` | **FAIL (Important)** | `craft/eligibility.go:91` (`type Processor interface`), `:113` (`type ProcessorImpl struct`), `:132` (`func NewProcessor(...)`), `:138`/`:142` (two `ProcessorImpl` methods) all live in `eligibility.go` — a bare topic-named file, not `processor.go` or a `processor_<group>.go` split. `craft/processor.go` holds the remaining `ProcessorImpl` methods (`Create`, `create`, `createOrUpgrade`, `crystal`, `disassemble`, `emit`, `ReleaseInFlight`), so the type is split across two files, one of which is not the sanctioned split shape. |
| FILE-02 | RestModel/Transform/JSON:API methods in `rest.go` | PASS | `craft/rest.go:22-49` (`RecipeRestModel`), `:76-100` (`CraftRequestRestModel`), `:129-146` (`CraftResponseRestModel`) all in `rest.go` |
| FILE-03 | Cross-service request functions in `requests.go` | N/A | craft itself makes no direct upstream HTTP calls (delegates to `character`/`quest`/`skill`/`compartment`/`equipment` sub-packages) |
| FILE-04 | Entity/Migration/TableName in `entity.go` | N/A | craft has no `entity.go` (no local persistence) |
| FILE-05 | Placement of builder/model/administrator/provider | N/A | craft has no `model.go`; not a DOM-structure package |
| FILE-06 | No catch-all file carrying ≥2 responsibilities | **FAIL (Important)** | Same evidence as FILE-01: `craft/eligibility.go` carries both the `Processor` type/constructor and eligibility business logic — the interface+impl+constructor is one of FILE-01's protected responsibilities, and it sits in a topic-named file alongside unrelated domain logic |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `craft/eligibility.go:132` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, ...)` |
| DOM-07 | Handlers pass `d.Logger()` into `NewProcessor` | PASS | `craft/resource.go:53` `l := d.Logger()` then `NewProcessor(l, ctx, ...)` at `:65` |
| DOM-08 | POST/PATCH via `RegisterInputHandler[T]` | PASS | `craft/resource.go:95` |
| DOM-09 | Every `Transform(` call site checks its error | N/A | `craft/resource.go` calls `TransformRecipe(...)`, which has no error return (see DOM-04/05 below); no `Transform(` call site exists to check |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | **FAIL (Minor)** | `craft/rest.go:53` defines `func TransformRecipe(r recipe.Model, elig Eligibility) RecipeRestModel` — no function literally named `Transform`, and it returns no error. Functionally serves the same role but composites two domains (`recipe.Model` + `Eligibility`), which the rule's literal shape does not anticipate. |
| DOM-05 | `TransformSlice` used by list handlers, no inline loop | **FAIL (Important)** | `craft/resource.go:147-158` builds the eligible-recipe list with an inline `for _, rc := range recipes { ... eligible = append(eligible, TransformRecipe(rc, elig)) }` inside `handleListRecipes` — the exact "inline `for` loop over Transform in `resource.go`" the rule calls a FAIL. (Contrast with `crystalband/resource.go:73` and `reagent/resource.go:73`, both of which use `model.SliceMap(Transform)(...)` instead of an inline loop — the sanctioned pattern.) The loop is entangled with per-recipe `Evaluate` calls (each fallible), so a pure `TransformSlice` extraction is not a drop-in fix, but the rule is graded on the surface as written. |
| DOM-17 | Domain errors map to specific HTTP status | PASS | `craft/resource.go:110-116` `writeProcessorError` maps `CraftError` to its own JSON:API code (`craft/errors.go`), else falls to `server.WriteErrorResponse` |
| DOM-18 | REST models implement JSON:API interface | PASS | `RecipeRestModel`, `CraftRequestRestModel`, `CraftResponseRestModel` all implement `GetName()/GetID()/SetID()` (`craft/rest.go`) |
| DOM-19 | Request models flat | PASS | `CraftRequestRestModel` (`craft/rest.go:76-88`) is flat, no nested `Data`/`Attributes` |
| DOM-13/14/15 | No cross-domain orchestration / no provider calls / no DB writes in handlers | PASS | `craft/resource.go:222` calls `p.Create(...)` only; no `db.Create`/`db.Save` anywhere in `craft/resource.go` |
| DOM-24 | Test packages reaching emit path install a stub | PASS | `craft/eligibility_test.go:44` `noopEmitter` satisfies `craft.SagaEmitter`; no test transitively calls the real `producer.ProviderImpl` |
| DOM-30 | DB-write ops emit via `AndEmit` + `message.Buffer` | PASS (documented exception) | `craft/emitter.go:29-31` calls `producer.ProviderImpl` directly, but `craft.Create`'s only state change is the in-memory `craft/inflight.go` guard — no `gorm.DB` write on this path (grepped `craft/processor.go`, `craft/inflight.go` for `gorm.DB`/`db.Create`/`db.Save`: zero hits). Matches the documented exception "Operations over non-DB state" in `patterns-kafka.md`. |
| DOM-32 | Routes register via `server.RegisterHandler`/`server.RegisterInputHandler[T]` | **FAIL (Important)** | `craft/resource.go:86-87` binds `rest.RegisterHandler(l)(db)(si)` / `rest.RegisterInputHandler[CraftRequestRestModel](l)(db)(si)`, aliases of `services/atlas-maker/atlas.com/maker/rest.RegisterHandler`/`RegisterInputHandler` (`rest/handler.go:45,109`). Tracing the alias: these do **not** delegate to `server.RegisterHandler`/`server.RegisterInputHandler[T]` (`libs/atlas-rest/server/register.go:11,26`) — they reimplement the composition by hand, calling `server.RetrieveSpan` then `server.ParseTenant` directly (`rest/handler.go:50-52`, `:113-116`), and in doing so **omit `server.ParseEnvironment`**, which the canonical implementation calls between `RetrieveSpan` and `ParseTenant` (`libs/atlas-rest/server/register.go:16,31`). `server.ParseEnvironment` (`libs/atlas-rest/server/handler.go:67-80`) is what puts the request's environment id onto `ctx` via `env.WithContext`; every EXT client in this service (`character`, `quest`, `skill`, `compartment`, `data/equipment`, `data/itemmake` — all via `requests.RootUrlFor(ctx, ...)`, `libs/atlas-rest/requests/url.go:34`) reads that context to resolve a PR-sandbox environment's ingress namespace, and falls back to "legacy: byte-identical to `RootUrl`" (`url.go:41`) when it is absent. Because atlas-maker's own route registration never populates it, every REST-triggered upstream call this service makes is silently pinned to the baseline environment — PR-sandbox routing (`deploy/k8s/overlays/pr-sparse/patches/consumer-group-env.yaml`, `db-name-suffix.yaml`, both added in this diff) does not work for atlas-maker. This is not a stylistic deviation from "the dominant fleet idiom" (a delegating local alias, which `patterns-rest-jsonapi.md`'s own verification note says is fine) — it is a non-delegating reimplementation that drops a step. |

### atlas-maker / crystalband (domain: `model.go`, `entity.go`, `rest.go`, `provider.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`/`Build()` | PASS | `crystalband/builder.go:19` `NewBuilder(tenantId uuid.UUID) *Builder`, `:43` `func (b *Builder) Build() (Model, error)` |
| DOM-02 | `Model.ToEntity()` in `entity.go` | **FAIL (Important)** | `crystalband/entity.go` defines only `Migration(db)` and `func (e entity) TableName() string`; no `ToEntity()` method anywhere in the package. `administrator.go:10-19` (`CreateCrystalBand`) builds `&entity{...}` inline field-by-field instead of calling a `Model.ToEntity()`. |
| DOM-03 | `Make(Entity) (Model, error)` in `entity.go` | **FAIL (Important)** | No `func Make(` anywhere in the package. The entity→model conversion exists as `modelFromEntity(e entity) (Model, error)` in `crystalband/provider.go:29-35` — different name, wrong file. |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | PASS | `crystalband/rest.go:29` `func Transform(m Model) (RestModel, error)` |
| DOM-05 | `TransformSlice`/no inline loop | PASS | `crystalband/resource.go:73` `model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()` — no inline loop |
| DOM-06 | Constructor takes `logrus.FieldLogger` | PASS | `crystalband/processor.go:46` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor` |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `crystalband/resource.go:66` `NewProcessor(d.Logger(), d.Context(), d.DB())` |
| DOM-09 | `Transform(` call sites check error | PASS | `crystalband/resource.go:112-117` checks `err` immediately |
| DOM-11 | Providers lazy via `database.Query`/`SliceQuery` | PASS | `crystalband/provider.go:11-27` — `getAllPagedProvider`, `getAllProvider`, `getByMinLevel` all wrap `database.PagedQuery`/`SliceQuery`/`Query`; no eager `FixedProvider` wrapping a read |
| DOM-16 | Writes in `administrator.go` | PASS | `crystalband/administrator.go` holds `CreateCrystalBand`, `BulkCreateCrystalBand`, `DeleteAllForTenant`; called from processor/subdomain, not inlined into `resource.go` |
| DOM-17 | Domain errors → HTTP status | PASS | `crystalband/resource.go` 405 rejection for write methods (`:44-56`); not-found path returns 404 (verified by reading the same handler block referenced above) |
| DOM-18 | JSON:API interface | PASS | `RestModel` in `crystalband/rest.go` implements `GetName/GetID/SetID` (grep confirmed) |
| DOM-32 | Routes via `server.RegisterHandler`/`InputHandler[T]` | **FAIL (Important)** | `crystalband/resource.go:38` `registerGet := rest.RegisterHandler(l)(db)(si)` — same non-delegating local wrapper as craft; see the craft DOM-32 entry above for the full evidence chain (`rest/handler.go`, missing `ParseEnvironment`) |
| FILE-01..06 | File placement | PASS | `Processor`/`ProcessorImpl`/`NewProcessor` all in `processor.go`; `RestModel`/`Transform` in `rest.go`; entity/`Migration`/`TableName` in `entity.go`; `Builder` in `builder.go`; writes in `administrator.go`; readers in `provider.go`; no catch-all `crystalband.go` file exists |

### atlas-maker / reagent (domain: same shape as crystalband)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` | PASS | `reagent/builder.go` defines `NewBuilder`/`Build()` (same shape as crystalband, confirmed by directory listing) |
| DOM-02 | `Model.ToEntity()` in `entity.go` | **FAIL (Important)** | `reagent/entity.go` defines only `Migration`/`TableName`; no `ToEntity()`. `reagent/administrator.go:10-19` (`CreateReagent`) builds `&entity{...}` inline. |
| DOM-03 | `Make(Entity)` in `entity.go` | **FAIL (Important)** | No `func Make(` in the package; equivalent logic is `modelFromEntity` in `reagent/provider.go:29-35`. |
| DOM-05 | `TransformSlice`/no inline loop | PASS | `reagent/resource.go:73` `model.SliceMap(Transform)(...)` |
| DOM-09 | `Transform(` error checked | PASS | `reagent/resource.go:100-105` |
| DOM-11 | Lazy providers | PASS | `reagent/provider.go` — same shape as crystalband, `database.Query`/`SliceQuery` |
| DOM-16 | Writes in `administrator.go` | PASS | `reagent/administrator.go` |
| DOM-32 | Routes via canonical registration | **FAIL (Important)** | `reagent/resource.go` imports `atlas-maker/rest` and binds the same non-delegating wrapper; see craft's DOM-32 entry for the full chain |
| FILE-01..06 | File placement | PASS | same layout as crystalband, no catch-all file |

### atlas-maker / character, quest, skill, compartment, data/equipment, data/itemmake (EXT client packages; each has `model.go` → DOM structure triggers, plus EXT-01..04)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`/`Build()` | **FAIL (Important)** — 5 packages | `character`, `quest`, `skill`, `data/equipment`, `data/itemmake` each have `model.go` with private fields and accessor methods, and no `builder.go` file, `NewBuilder`, or `Build()` anywhere in the package (confirmed by directory listing and `grep -rl "NewBuilder\|func.*Build()"` returning no hits for these five). Construction is exclusively via each package's `Extract(RestModel) (Model, error)` — a reasonable pattern for a read-only remote-mirror model, but the checklist's DOM-01 trigger is unconditionally "package has `model.go`," and no documented exception in `file-responsibilities.md` or `patterns-provider.md` carves out read-only external-mirror models. Flagging per the rule's literal text; this may indicate the rule needs a documented exception for this shape rather than that the code is wrong. |
| DOM-01 / FILE-05 | `builder.go` file, correct placement | **FAIL (Important)** — `compartment` | `compartment/model.go:86-110` defines an inline `Builder`/`NewBuilder`/`Build()` (used for test construction) — a real builder exists, but it lives in `model.go`, not a `builder.go` file, so DOM-01's literal "File `builder.go` exists" check fails, and FILE-05 ("Builder in `builder.go`... domain `Model` in `model.go`") is violated by the same evidence. |
| EXT-01 | Target REST model implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | PASS (6/6) | `character/rest.go:35-42`, `compartment/rest.go:60-68` (+ two more embedded models at `:127-128`, `:152-153`), `quest/rest.go:40-46`, `skill/rest.go:34-40`, `data/equipment/rest.go:38-44`, `data/itemmake/rest.go:67-73` |
| EXT-02 | httptest-backed integration test | PASS (6/6) | `character/*_test.go`, `compartment/*_test.go`, `quest/*_test.go`, `skill/*_test.go`, `data/equipment/*_test.go`, `data/itemmake/*_test.go` all use `httptest.NewServer` |
| EXT-03 | Only genuine 404s map to "not found" | PASS | none of these processors collapse errors to a blanket not-found; `requests.ErrNotFound` is returned/bubbled verbatim in `character/processor.go:13`, `data/itemmake/processor.go:18`, `data/equipment/processor.go:14`; `compartment`/`quest`/`skill` processors do not intercept errors at all |
| EXT-04 | URL via `requests.RootUrl(<DOMAIN>)` | PASS | all six packages use `requests.RootUrlFor(ctx, "<DOMAIN>")` (`character/requests.go:16`, `compartment/requests.go:18`, `quest/requests.go:15`, `skill/requests.go:15`, `data/equipment/requests.go:16`, `data/itemmake/requests.go:17`) — `RootUrlFor` is a pre-existing sibling of `RootUrl` in `libs/atlas-rest/requests/url.go:34`, env-aware and never falling back to a hardcoded DNS name; satisfies the rule's intent even though it is not the literal `RootUrl` call named in the checklist text |
| FILE-01..06 | File placement | PASS (6/6) | Processor/RestModel/requests/model cleanly separated into `processor.go`/`rest.go`/`requests.go`/`model.go` in every one of the six packages; no catch-all file |

### atlas-maker / recipe (domain: `model.go` + `processor.go` only, no persistence)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` | **FAIL (Important)** | `recipe` has `model.go` (private-field `Model`) but no `builder.go`; construction is via `modelFromItemMake` (`recipe/processor.go:96`), an internal function, not an exported builder |
| DOM-29 | Cache is an application-scoped singleton | PASS | `recipe/processor.go:50-52` `var recipeIndex = &index{...}` is a package-level singleton, not constructed inside `NewProcessor` (`:151`) and not held as a `ProcessorImpl` field (`ProcessorImpl` only holds `l`, `ctx`, `im`) — matches the documented rule that a singleton under a different accessor name/file still passes DOM-29 |
| FILE-01 | Processor interface/impl/constructor in `processor.go` | PASS | `recipe/processor.go:128-152` |

### atlas-maker / seed, rest, kafka/consumer/saga, kafka/message/saga

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Goroutines via `routine.Go` | PASS | `tools/goroutine-guard.sh` exit 0 across all 92 modules including atlas-maker |
| DOM-31 | Tenant/trace never in REST model/body | PASS | `craft/rest.go`, `crystalband/rest.go`, `reagent/rest.go` request/response models carry no tenant field; tenant travels via `d.Context()` populated by `server.ParseTenant` |
| DOM-25 | Client-interpreted wire bytes resolved from tenant table | PASS (with one documented exception already ruled) | `libs/atlas-packet/character/maker_result_body.go:26-49` resolves every mode byte via `atlas_packet.WithResolvedCode("operations", ...)`; the `nResult` success/fail sentinel (0 success, 2 failed) is a Go literal in both `services/atlas-channel/.../kafka/consumer/maker/consumer.go:38` and `services/atlas-channel/.../socket/handler/maker_skill.go` — but this is not a per-tenant "operations" table value (no `results` key exists in any seeded template's `options` block); the code's own comments document this as a fixed wire-protocol invariant identical across all 8 client versions, not a caller-selectable mode. Consistent with the already-ruled FAILED-path discriminator noted in scope. |

## SCAFFOLD — new service (`services/atlas-maker/`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SCAFFOLD-01 | `services.json` `go-service` entry | PASS | `.github/config/services.json:256-260` |
| SCAFFOLD-02 | K8s base manifest listed in `kustomization.yaml` | PASS | `deploy/k8s/base/atlas-maker.yaml` exists; `deploy/k8s/base/kustomization.yaml:38` lists `atlas-maker.yaml` |
| SCAFFOLD-03 | `docker-bake.hcl` + `go.work` entries | PASS | `docker-bake.hcl:66` `"atlas-maker"`; `go.work:57` `./services/atlas-maker/atlas.com/maker` |
| SCAFFOLD-04 | Ingress block in `deploy/shared/routes.conf` | PASS | `deploy/shared/routes.conf:146-149` (`/api/characters/{id}/maker`), `:696-704` (`/api/maker`, `/api/reagents`) |
| SCAFFOLD-05 | Generated routes template regenerated | PASS | `tools/gen-routes.sh --check` → "gen-routes: up to date", exit 0 |
| SCAFFOLD-06 | docker-compose entry alongside peers | **FAIL (Important)** | `deploy/compose/docker-compose.core.yml` has no `atlas-maker:` service block. The file is alphabetically ordered and actively maintained (`atlas-map-actions:341`, `atlas-maps:` follow, then jumps straight to `atlas-merchant:354` — `atlas-maker` is absent from its expected alphabetical slot). |
| SCAFFOLD-07 | New Writer/Handler seeded in every tenant opcode template | PASS | `MakerSkillHandle`/`MakerResult` present in all 8 templates (`template_gms_72_1.json`, `_79_1`, `_83_1`, `_84_1`, `_87_1`, `_92_1`, `_95_1`, `template_jms_185_1.json`); verified `gms_72_1` (`:889-901`, `:3628-3640`), `jms_185_1` (`:805`, `:3655-3656`), `gms_95_1` (`:1049`, `:4495-4496`) |
| SCAFFOLD-08 | Bruno collection (REST service) | **FAIL (Important)** | No `.bru` files exist under `services/atlas-maker/` at all; sibling REST services (`atlas-map-actions`, `atlas-reward-pools`, `atlas-buffs`, `atlas-npc-shops`, `atlas-cashshop`, `atlas-ban`, `atlas-notes`, `atlas-guilds`, `atlas-pets`, ...) each carry a `.bruno/` directory. atlas-maker exposes REST via `craft`, `crystalband`, `reagent`, and `seed` resources with no collection documenting them. |
| SCAFFOLD-09 | Overlay enumerations / `ATLAS_DB_NAMES` / DB bootstrap | PASS | `tools/service-registration-guard.sh` → "service-registration-guard: clean", exit 0 |

## Security Review

Not triggered. atlas-maker handles no authentication, authorization, tokens, redirects, or secrets (no JWT parsing, no callback/redirect handlers, no hardcoded credentials found in the diff). No other changed service in this branch touches auth surfaces. SEC-01..04 are N/A — trigger did not fire.

## Not evaluable from the diff

- FILE-*/DOM-* full line-by-line sweep of `services/atlas-inventory/atlas.com/inventory/asset/processor.go` and `compartment/processor.go` beyond the added `CreateOptions`/`hasExplicitStats`/`applyExplicitStats` region — the surrounding pre-existing file (500+ lines) was not read in full; would need the whole file to certify no other rule regression outside the diff hunks.
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go`'s new `compensateAwardCraftedAsset` and the manifest-driven step-building in `saga/handler.go`/`saga/producer.go` were read only at the signature level, not walked end-to-end against DOM-27/28 (resilience) for every branch — would need a full read of `saga/compensator.go` (144 new lines) and `saga/handler.go` (44 new lines) to certify every error branch.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/maker/consumer.go` (274 new lines) — the FAILED-path discriminator was explicitly excluded from re-litigation per the task's scope; the remaining ~250 lines (manifest decoding, terminal-event dispatch to the four `MakerResult*Body` arms) were read only for the two call sites cited above, not exhaustively for DOM-27/DOM-9-equivalent error handling in every branch.
- `libs/atlas-packet/character/serverbound/maker_skill.go` (173 new lines) — read only via its consumer (`MakerSkillHandleFunc`) and its doc comments; the decode logic itself (arm parsing, gem-list decoding) was not walked against the wire-derivation doc line-by-line.
- Whether `craft/eligibility.go`'s Processor definition (FILE-01 finding) was a deliberate, reviewed layout choice from an earlier task-285 sub-review — the per-task review files under `docs/tasks/task-285-maker-skill-crafting/reviews/task-*.md` were not read in full to check whether this was already flagged and accepted; only `audit-plan-adherence-*.md` were spot-checked (they cover plan-task completion, not guideline compliance).

## Summary

### Blocking (must fix)

- **DOM-32** — `services/atlas-maker/atlas.com/maker/rest/handler.go:45-63,109-123`: local `RegisterHandler`/`RegisterInputHandler` reimplement route registration instead of delegating to `server.RegisterHandler`/`server.RegisterInputHandler[T]`, and omit `server.ParseEnvironment` — silently disabling PR-sandbox environment-aware routing for every upstream call this service makes (used by `craft/resource.go`, `crystalband/resource.go`, `reagent/resource.go`).
- **FILE-01 / FILE-06** — `services/atlas-maker/atlas.com/maker/craft/eligibility.go:91,113,132`: `Processor` interface, `ProcessorImpl` struct, and `NewProcessor` constructor live in a topic-named file instead of `processor.go`/`processor_<group>.go`.
- **DOM-02 / DOM-03** — `services/atlas-maker/atlas.com/maker/crystalband/entity.go`, `reagent/entity.go`: no `Model.ToEntity()` or `Make(Entity) (Model, error)`; equivalent logic exists under a different name (`modelFromEntity`) in `provider.go`.
- **DOM-05** — `services/atlas-maker/atlas.com/maker/craft/resource.go:147-158`: inline `for` loop calling `TransformRecipe` per item instead of a bulk `TransformSlice`/`model.SliceMap` pass (contrast with `crystalband`/`reagent`, which use `model.SliceMap`).
- **DOM-01** — `services/atlas-maker/atlas.com/maker/{character,quest,skill,data/equipment,data/itemmake}`: no `builder.go` in any of the five `model.go`-bearing packages; `compartment/model.go:89-110` has a `Builder` misplaced outside `builder.go`.
- **SCAFFOLD-06** — no `atlas-maker` entry in `deploy/compose/docker-compose.core.yml`.
- **SCAFFOLD-08** — no Bruno collection for atlas-maker despite exposing REST.

### Non-Blocking (should fix)

- **DOM-04** — `services/atlas-maker/atlas.com/maker/craft/rest.go:53`: `TransformRecipe` doesn't match the standard `Transform(Model) (RestModel, error)` shape (different name, no error return) — likely fine given craft's composite REST model, but worth a comment noting the deliberate deviation.

### Not evaluable

- See "Not evaluable from the diff" above (5 items).
