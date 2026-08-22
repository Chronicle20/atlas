# Backend Audit — task-246 (factory / saga / character / configurations)

- **Scope:** `services/atlas-character-factory/`, `services/atlas-configurations/`, `services/atlas-saga-orchestrator/`, `services/atlas-character/` — task-246 Maple Life (Cash/0543 in-game character creation)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Range:** `24a33a2e6..77b302206`
- **Date:** 2026-08-21
- **Build:** PASS (all four modules)
- **Tests:** PASS (all four modules; no failures)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
atlas-character-factory: go build ./... => exit 0; go test ./... -count=1 => ok (all packages, or no test files)
atlas-configurations:    go build ./... => exit 0; go test ./... -count=1 => ok (all packages, or no test files)
atlas-saga-orchestrator: go build ./... => exit 0; go test ./... -count=1 => ok (all packages, or no test files)
atlas-character:         go build ./... => exit 0; go test ./... -count=1 => ok (all packages, or no test files)
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (file-responsibilities.md) | Yes | `factory/maple_life.go`, `factory/rest.go`, `factory/preset_rest.go`, `configuration/tenant/maplelife/rest.go`, `configuration/tenant/rest.go`, `configuration/tenant/characters/preset/rest.go` all have `rest.go`-family files in scope |
| FILE placement | Yes | Runs unconditionally on every changed Go package |
| SUB sub-domain | Yes | `factory` package has `resource.go`, no `model.go` |
| REST (patterns-rest-jsonapi.md) | Yes | `factory/resource.go`, `factory/processor.go`, `factory/rest.go` changed |
| Constants reuse (DOM-21) | Yes | `saga/model.go:60-63` introduces `MapleLifeUse` — sourced from `sharedsaga.MapleLifeUse` (lib alias), not redeclared |
| Testing (testing-guide.md) | Yes | Diff touches numerous `_test.go`; `character/processor.go` interface signature changed (DOM-33) |
| Cache (DOM-29) | No | No `cache.go`, no cached processor state introduced |
| Messaging (DOM-30) | Yes | `kafka/producer/seed/producer.go`, `character/producer.go` emit Kafka messages |
| Multi-tenancy (DOM-31) | Yes | `factory/maple_life.go:80` reads `tenant.MustFromContext(ctx)`; multiple `rest.go` files changed |
| Migration hygiene (DOM-34/35) | No | No symbol moved between a service and `libs/atlas-*` in this diff |
| Deploy & topics (DOM-22/23) | No | No new `libs/atlas-*` module; no new/renamed Kafka topic env var |
| Runtime safety (DOM-26) | Yes (family) / N/A (rule) | Non-test Go files changed, but no `go` statement added in the diff — grep confirms zero `go func(` / `routine.Go` additions |
| Channel wire values (DOM-25) | No | Diff does not touch `services/atlas-channel` or `libs/atlas-packet`; no client-interpreted byte is emitted by this diff (Maple Life's identifiers are semantic ordinals/job ids consumed server-side) |
| Resilience (DOM-27/28) | No | No `model.Decorator` / enrichment-fallback path changed |
| External clients (EXT-01..04) | No new call | `data.Processor.GetSkillsByIds`/`GetItemById` already existed pre-diff; only the returned struct gained a field. No new `requests.*Request[T]` call site added |
| Scaffolding (SCAFFOLD-01..09) | No | No new `services/atlas-<svc>/` directory, no new channel writer/handler, no `routes.conf` change |
| Security (SEC-01..04) | No | None of the four services in scope handle auth tokens/redirects/secrets |

## Checklist Results

### `factory` (atlas-character-factory) — sub-domain (`resource.go`, no `model.go`) + REST surface

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/constructor/`ProcessorImpl` methods in `processor.go` or `processor_<group>.go` | FAIL (Important) | `factory/maple_life.go:53` (`CreateMapleLife`) and `:79` (`resolveMapleLifePreset`) are `ProcessorImpl` methods living in a bare topic-named file (`maple_life.go`), not `processor.go` and not a `processor_maple_life.go`-named split. Contrast: the sibling preset flow's `CreateFromPreset` and `buildPresetCharacterCreationSaga` live in `factory/processor.go:272,360`. |
| FILE-02 | `RestModel`, `Transform`/`Extract`, JSON:API methods live in `rest.go` | FAIL (Important) | `MapleLifeCreateRestModel` type is defined at `factory/maple_life.go:26`, not in any `rest.go`-family file. Its own `GetName`/`GetID`/`SetID` live separately in `factory/maple_life_rest.go:8-10`. Contrast: `PresetCreateRestModel` and its JSON:API methods are co-located in `factory/preset_rest.go:3-12`. |
| FILE-06 | No single file carries ≥2 of the responsibilities above | FAIL (Important) | `factory/maple_life.go` carries both the RestModel type (`:26`) and Processor business logic (`:53`, `:79`, plus `toPreset` at `:230` which is itself a Transform-shaped projection). Same shape as the `wallet.go` precedent named in file-responsibilities.md:202-203. |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `factory/processor.go:60` `func NewProcessor(l logrus.FieldLogger) Processor`; unchanged by this diff and still honored. |
| DOM-07 | Handlers pass `d.Logger()` into `NewProcessor` | PASS | `factory/resource.go:94` `processor := newProcessor(d.Logger())` inside `newMapleLifeHandler`. |
| DOM-08 | POST/PATCH routes use `RegisterInputHandler[T]` | PASS | `factory/resource.go:33` `rest.RegisterInputHandler[MapleLifeCreateRestModel](l)(si)(CreateMapleLife, handleCreateMapleLife)`. |
| DOM-13 | No cross-domain orchestration in handlers | PASS | `handleCreateMapleLife` (`resource.go:83`) and `newMapleLifeHandler` (`:91`) delegate immediately to `processor.CreateMapleLife`; all validation/orchestration lives in `maple_life.go`. |
| DOM-14 | Handlers call processor methods only | PASS | `resource.go:95` calls `processor.CreateMapleLife`, never a provider function directly. |
| DOM-15 | No `db.Create`/`db.Save`/`db.Delete` in handlers | PASS | `resource.go` has no direct DB calls (grep negative); factory has no DB at all — it emits a saga. |
| DOM-17 | Domain errors map to correct HTTP status | PASS | `factory/resource.go:57-79` `categorizeMapleLifeError` maps `ErrClassOrdinalUnknown`/`ErrLookInvalid`/`ErrSPInvalid`/`NameInvalidError` → 400, `ErrNameDuplicate` → 409, `ErrAtlasDataUnreachable` → 502, `ErrMapleLifeNotConfigured` → 500 (defensible: an unconfigured tenant is a server-side content gap, not a client input error). |
| DOM-18 | JSON:API interface implemented | PASS | `factory/maple_life_rest.go:8-10` implements `GetName`/`GetID`/`SetID` on `MapleLifeCreateRestModel`. |
| DOM-19 | Request models flat, no nested Data/Type/Attributes | PASS | `MapleLifeCreateRestModel` (`maple_life.go:26-37`) is a flat struct — JSON:API envelope unwrapping happens in the shared `rest.RegisterInputHandler`/`ParseInput`, not in the model. |
| DOM-31 | Tenant/trace travel in context only | PASS | `maple_life.go:80` `t := tenant.MustFromContext(ctx)`; `MapleLifeCreateRestModel` carries no tenant/trace field. |
| SUB-01 | Business logic not in the handler | PASS | All validation/projection logic lives in `maple_life.go`'s `ProcessorImpl` methods, not in `resource.go`. |
| SUB-02 | No `db.Create`/`db.Save` in `resource.go` | PASS | Confirmed by grep — no such call sites. |
| SUB-03 | POST endpoints use `RegisterInputHandler[T]` | PASS | Same evidence as DOM-08. |
| SUB-04 | No manual JSON parsing in `resource.go` | PASS | No `json.NewDecoder`/`json.Unmarshal`/`io.ReadAll` in `resource.go` (grep negative). |
| DOM-20 | Tests table-driven (`tests := []struct` + `t.Run`) | FAIL | `factory/maple_life_test.go:334,417,452,481` — `TestCreateMapleLifeSagaPayload`, `_SPZero`, `_MagicianMP`, `_NoSkillClass` are four near-duplicate single-scenario functions (same setup, different `ClassOrdinal`/`SP` input, different derived-field assertions) that belong in one table with `t.Run` cases, not four separate `func Test...`. `factory/resource_test.go:46,56,68` — `TestHandleCreateFromPreset_BadJSON`/`_MissingPresetId`/`_InvalidPresetIdFormat` are three single-assertion (`rr.Code != http.StatusBadRequest`) functions with the same shape, equally consolidatable. (`TestCreateMapleLife` at `maple_life_test.go:141` and `TestCategorizeMapleLifeError`/`TestCategorizePresetError` at `resource_test.go:76,105` ARE correctly table-driven — PASS on those.) |
| DOM-24 | Test packages reaching an emit path stub the producer | N/A | `factory`/`resource_test.go` and `maple_life_test.go` never reach `AndEmit`/`message.Emit`/`producer.Produce` — `CreateMapleLife` only calls `saga.NewProcessor(...).Create(sg)`, and the tests exercise `resolveMapleLifePreset` (pure) or the HTTP handler with a fake `Processor` (`resource_test.go:157`), never the real saga-emitting path. |
| — (test-seam shape, no rule ID) | `newMapleLifeHandler` injection replacing a package-level mutable var | No finding | `factory/resource.go:87-107`: a one-line delegating indirection (`newMapleLifeHandler(NewProcessor)`) used identically in production (`resource.go:84`) and tests (`resource_test.go:157`). It is a legitimate constructor-injection seam — no numbered rule governs test-seam shape, and it does not violate DOM-13/14/07. The asymmetry with the neighboring `handleCreateFromPreset`, which still calls `NewProcessor` directly with no injection seam (`resource.go:111-125`), is a minor inconsistency but not itself a guideline violation. |

### `configuration/tenant/maplelife`, `configuration/tenant/characters/preset` (atlas-character-factory) — nested config-projection packages (no `model.go`, no processor)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | N/A | `maplelife/rest.go` and `characters/preset/rest.go` are nested attribute structs decoded as part of the parent `tenant.RestModel` (`configuration/tenant/rest.go:20` `MapleLife maplelife.RestModel`), which is the actual JSON:API resource implementing `GetName`/`GetID`/`SetID` (`tenant/rest.go:24-33`). Every sibling in the same subtree (`worlds/rest.go`, `npcs/rest.go`, `characters/rest.go`) follows the identical zero-Transform, zero-JSON:API-method shape; this is the established pattern for tenant-config value objects, not a domain RestModel round-tripping a local Model. |
| DOM-18/19 | JSON:API interface / flat request model | N/A | Same reasoning — these are not independently-addressed JSON:API resources. |
| — (design constraint) | `mapleLife` block is ordinal-addressed, no admin preset UUIDs | PASS | `configuration/tenant/maplelife/rest.go:59-68` (`ClassEntry`) carries no `presetId`/UUID field; addressed purely by `Ordinal`+`Gender`. Confirmed empirically against `seed-data/templates/template_gms_83_1.json` — no UUID-shaped field anywhere in the `mapleLife` block. |
| — (design constraint) | AP/SP additive, `omitempty` mirrored identically | **FAIL** (design-brief claim not fully true) | `preset/rest.go:46-47` (`AP uint16 \`json:"ap"\`` / `SP string \`json:"sp"\``) and `maplelife/rest.go:59-60` (same) have **no** `omitempty`, while `CreateCharacterCommandBody.AP`/`.SP` in both `atlas-character/kafka/message/character/kafka.go:83-84` and `atlas-saga-orchestrator/kafka/message/character/kafka.go:206-207` **do** carry `omitempty`. The additivity itself holds (zero-value defaults on unmarshal are unaffected by the tag), so no existing caller breaks, but the brief's "mirrored identically... both omitempty" claim does not hold for the two config-side types. Non-blocking (cosmetic JSON-marshal-output difference only), reported per the task's explicit instruction to verify this constraint. |

### `atlas-configurations/templates/maplelife`, `atlas-configurations/tenants/maplelife`, `.../templates/rest.go`, `.../tenants/rest.go`, `.../tenants/characters/preset/rest.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Byte-identical mirror of the factory's config-projection types | PASS | `diff` between `atlas-character-factory/.../configuration/tenant/maplelife/rest.go` and both `atlas-configurations/.../templates/maplelife/rest.go` and `.../tenants/maplelife/rest.go` returns no differences. |
| DOM-04/18/19 | Transform / JSON:API / flat model | N/A | Same reasoning as the factory-side package above; `templates/rest.go:20-23` / `tenants/rest.go:20-23` embed `MapleLife maplelife.RestModel` as a nested attribute of the top-level resource. |
| — (design constraint) | Same AP/SP `omitempty` gap | FAIL (same finding, mirrored) | `atlas-configurations/.../tenants/characters/preset/rest.go` is byte-identical to the factory copy (same `AP`/`SP` fields, no `omitempty`) — same finding as above, not duplicated in Summary. |
| DOM-20 | Tests table-driven | PASS | `tenants/maplelife/rest_test.go:82` `TestMapleLifeBlockRoundTrips` uses `t.Run` sub-cases (full document / absent block / json tags) against one function; `tenants/rest_test.go:14` `TestTenantRestModelCarriesMapleLife` likewise uses `t.Run` sub-cases. |

### `saga` (atlas-saga-orchestrator)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclaration of a type/const already in `libs/atlas-constants` | PASS | `saga/model.go:60-63` `MapleLifeUse = sharedsaga.MapleLifeUse` — an alias to the shared `atlas-saga` lib constant, not a local redeclaration. |
| DOM-33 | Interface change updates every mock in the same diff | PASS | `character.Processor.RequestCreateCharacter` gains `ap uint16, sp string` params. Updated together in the same diff: interface (`character/processor.go:46`), impl (`character/processor.go:214`), mock (`character/mock/processor.go:41,251`), and every call site (`saga/handler.go:1929`, `character/producer.go:250,276`). `go build ./...` for `atlas-saga-orchestrator` passes, and `character/producer_test.go`, `saga/handler_test.go`, `saga/processor_test.go` were all updated in the same diff to the new signature. |
| DOM-30 | DB writes emit via `AndEmit` + `message.Buffer`, not a bare `producer.ProviderImpl` from the success path | N/A (no DB write in this package) | `saga` package doesn't own a DB write here; `character.ProcessorImpl.RequestCreateCharacter` (`character/processor.go:214`) uses `message.Emit(p.p)(...)` to enqueue the command — pre-existing pattern, unchanged shape. |

### `character` (atlas-character)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-16 | Writes go through `administrator.go` | PASS | `character/administrator.go:17` `func create(...)` remains the sole write path, called from `processor.go:383`. |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | PASS | `kafka/consumer/character/consumer_test.go:30` `database.RegisterTenantCallbacks(l, db)`. |
| DOM-24 | Test packages reaching an emit path stub the producer | N/A (outbox sink, not live Kafka) | `TestHandleCreateCharacterForwardsApAndSp` (`consumer_test.go:59`) drives `handleCreateCharacter` → `character.ProcessorImpl.CreateAndEmit` (`character/processor.go:352`), which calls `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(...)` — the DB-backed transactional outbox, not `producer.ProviderImpl`/a live Kafka call. Confirmed empirically: `go test -run TestHandleCreateCharacterForwardsApAndSp -v` completes in 0.025s with no `producertest` install in this package — consistent with no real broker round-trip, inconsistent with the documented ~42s-per-emit cost an unstubbed live producer would incur. |
| — (bugfix, no rule ID) | SP default-pool separator fix | No finding (fix, not a violation) | `character/administrator.go:24-27`: the pre-existing `"0, 0, 0, ..."` (space-separated) default is replaced with a bare-comma `"0,0,0,0,0,0,0,0,0,0"`, matching `Model.SPs()`'s comma-only split — a real bugfix (the old default silently truncated to one usable SP-book entry), not itself a guideline violation. |

## Security Review

Not applicable — none of the four services in this scope handle authentication, authorization, tokens, redirects, or secrets (SEC-01..04 trigger did not fire).

## Not evaluable from the diff

- Full content-accuracy sweep of every `mapleLife.classes[]` entry (5 classes × 2 genders × 4 tenants = 40 rows) against `maple-life-content.md`'s derivation tables — I spot-checked the Warrior (ordinal 0, gender 0) row in `template_gms_83_1.json` against the doc's AP (`145-31=114`), HP/MP midpoint (804/150), and `spSkillId` (1000001) derivations and they matched exactly, and confirmed structurally that no tenant/class row carries a preset UUID. A full row-by-row reconciliation across all 4 tenants and 10 rows each would require re-deriving each class's stat floor/AP/HP/MP from the doc's NPC/skill tables, which is game-content verification rather than a backend-guidelines check; out of this reviewer's evidence bar without re-doing that derivation work.
- Whether `atlas-login`'s consumer of `EVENT_TOPIC_SEED_STATUS`'s new `TransactionId` field (kafka/message/seed/kafka.go:12) is actually read by the login-side consumer — that consumer lives in `services/atlas-login`, outside this review's assigned scope (owned by the sibling `atlas-channel` reviewer or a separate audit pass).

## Summary

### Blocking (must fix)
- FILE-01 (Important): `factory/maple_life.go:53,79` — `ProcessorImpl.CreateMapleLife`/`resolveMapleLifePreset` belong in `processor.go` or a `processor_maple_life.go` split, not a bare topic-named file.
- FILE-02 (Important): `factory/maple_life.go:26` — `MapleLifeCreateRestModel` type belongs co-located with its own JSON:API methods (currently split across `maple_life.go` and `maple_life_rest.go`); the established sibling pattern (`preset_rest.go:3-12`) keeps type + methods together.
- FILE-06 (Important): `factory/maple_life.go` collapses RestModel definition + Processor business logic + a Transform-shaped projection (`toPreset`, `:230`) into one file — the same shape as the `wallet.go` precedent the guideline explicitly calls out.

### Non-Blocking (should fix)
- DOM-20: `factory/maple_life_test.go:334,417,452,481` and `factory/resource_test.go:46,56,68` — collapse the near-duplicate single-scenario test functions into table-driven `t.Run` cases.
- Design-constraint gap (no rule ID, flagged per task brief): `preset/rest.go:46-47` and `maplelife/rest.go:59-60` (both in `atlas-character-factory` and mirrored in `atlas-configurations`) lack `omitempty` on `AP`/`SP`, unlike the mirrored `CreateCharacterCommandBody.AP`/`.SP` in both kafka message packages, which do carry it. Functionally harmless; breaks the stated "mirrored identically" design intent.
