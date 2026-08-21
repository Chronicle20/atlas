# Backend Audit — task-256-zombify-healing-consequences

- **Service Path(s):** services/atlas-channel/atlas.com/channel, services/atlas-consumables/atlas.com/consumables
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Commit range:** 1461bfc96..HEAD (HEAD = 072017280)
- **Build:** PASS (both modules)
- **Tests:** 2548 passed (2313 atlas-channel, 235 atlas-consumables), 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd services/atlas-channel/atlas.com/channel && go build ./...      -> exit 0, no output
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1  -> ok, all packages, exit 0
cd services/atlas-consumables/atlas.com/consumables && go build ./... -> exit 0, no output
cd services/atlas-consumables/atlas.com/consumables && go test ./... -count=1 -> ok, all packages, exit 0
```

## Scope

Changed packages (per `git diff --stat 1461bfc96..HEAD`):

- `atlas-channel/character/buff` (model.go, model_test.go)
- `atlas-channel/skill/handler/heal` (formula.go, formula_test.go, heal.go, heal_apply_test.go)
- `atlas-consumables/character/buff` (model.go [new], model_test.go [new], processor.go, processor_notfound_test.go [new], requests.go [new], rest.go [new])
- `atlas-consumables/character/buff/mock` (processor.go)
- `atlas-consumables/character/buff/stat` (rest.go [new])
- `atlas-consumables/consumable` (morph_coupon.go, morph_coupon_test.go, processor.go, processor_test.go)

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | `atlas-consumables/character/buff` and `.../stat` gained `model.go`/`rest.go`; `atlas-channel/character/buff/model.go` changed and package has `rest.go`; `atlas-consumables/consumable` has `model.go` (pre-existing) |
| FILE placement (FILE-01..06) | Yes | every changed Go package, no exemption |
| SUB sub-domain (SUB-01..04) | No | no changed package has `resource.go` without `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Yes | changed packages have `rest.go`/`processor.go`; no `resource.go` in any changed package |
| Constants reuse (DOM-21) | No | diff only *uses* `charconst.TemporaryStatTypeUndead` (`libs/atlas-constants/character/temporary_stat.go:122`), declares no new type/const/classification |
| Testing (DOM-10,20,24,33) | Yes | diff touches many `_test.go` files; `buff.Processor` interface (atlas-consumables) gained `GetByCharacterId` |
| Cache (DOM-29) | No | no `cache.go`, no cached state held by any changed processor/struct |
| Messaging (DOM-30) | Yes (family) / N/A (rule) | `atlas-consumables/character/buff` has `producer.go` (unchanged) — but the package performs no DB write on any path (documented exception 2, no-DB-state) |
| Multi-tenancy (DOM-31) | Yes | changed packages have `rest.go` and pass `ctx` through client calls |
| Migration hygiene (DOM-34/35) | No | no symbol moved between a service and `libs/atlas-*` |
| Deploy & topics (DOM-22/23) | No | no new `libs/atlas-*` module; no new/renamed Kafka topic env var (`buff2.EnvCommandTopic` pre-existing) |
| Runtime safety (DOM-26) | Yes (family) / N/A (rule) | non-test Go files changed; no bare `go` statement added |
| Channel wire values (DOM-25) | Yes (family) / N/A (rule) | diff touches `atlas-channel`; no client-interpreted byte/opcode literal added or changed |
| Resilience (DOM-27/28) | No | no DB-backed handler and no `model.Decorator`/enrichment path changed |
| External clients (EXT-01..04) | Yes | `atlas-consumables/character/buff` (new) and `atlas-channel/character/buff` (in scope via `model.go`) call `requests.DrainProvider`/`requests.RootUrlFor` against atlas-buffs |
| Scaffolding (SCAFFOLD-01..09) | No | no new `services/atlas-<svc>/`, no new channel Writer/Handler, no `routes.conf` change |
| Security (SEC-01..04) | No | neither service handles auth/tokens/redirects/secrets |
| patterns-provider.md (foundational) | No | no new `provider.go`; `requests.DrainProvider` is a pre-existing library call, not a newly composed GORM-backed provider |
| patterns-functional.md (foundational) | No | `heal.Apply`'s curried shape is pre-existing and untouched by this diff; the new seam `var`s are simple function-typed indirection, not new decorators/combinators |

## Checklist Results

### atlas-channel/character/buff (domain package — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`/setters/`Build()` | FAIL | No `builder.go` in the package (`find` confirms absence); construction is via `NewBuff(...)` in `model.go:81`. Package is in scope because `model.go` changed (added `IsZombified`, lines 26-49) |
| DOM-02/03 | `ToEntity()`/`Make()` in `entity.go` | N/A | No `entity.go` in package |
| DOM-04 | `func Transform(` in `rest.go` | FAIL | `rest.go` defines `Extract` only (`rest.go:34`); no `Transform` function exists |
| DOM-05 | `TransformSlice` used by list handlers | N/A | No `resource.go`/list handler in package (client-only) |
| DOM-11 | providers lazy via `database.Query` | N/A | No `provider.go` |
| DOM-16 | `administrator.go` for writes | N/A | Package performs no local DB writes — it is a Kafka/REST client to atlas-buffs, no `gorm.DB` anywhere in the package |
| FILE-01..06 | file placement | PASS | Processor in `processor.go`, RestModel/Extract in `rest.go`, requests in `requests.go`; no package-named catch-all file (no `buff.go`) |
| DOM-18 | RestModel implements `GetName/GetID/SetID` | PASS | `rest.go:21,25,29` |
| DOM-19 | request models flat | PASS | `RestModel` in `rest.go` has no nested Data/Type/Attributes |
| EXT-01 | target RestModel has `SetToOneReferenceID`/`SetToManyReferenceIDs` | FAIL | `rest.go` has neither method (grep confirms absence); package calls `requests.DrainProvider[RestModel, Model]` (`processor.go:64`) against atlas-buffs |
| EXT-02 | httptest fixture asserts populated struct | PASS | `processor_drain_test.go:45` `TestByCharacterIdProviderDrainsBeyondOnePage` serves multi-page JSON fixtures via `httptest.NewServer` and exercises the real `Extract`/drain path (pre-existing, unchanged by this diff, but in-scope package) |
| EXT-03 | only genuine 404 → not-found | PASS | `processor.go:67` `errors.Is(err, requests.ErrNotFound)` |
| EXT-04 | URL via root-url helper, not hardcoded DNS | PASS | `requests.go:15` `requests.RootUrlFor(ctx, "BUFFS")` |
| DOM-20 | table-driven tests | PASS | `model_test.go:11` `TestIsZombified` uses `tests := []struct{...}` + `t.Run` |
| DOM-33 | mocks updated for interface change | N/A | `buff.Processor` interface in this package is unchanged by the diff (`GetByCharacterId` pre-existed at `processor.go:74`); `IsZombified` is a package function, not an interface method |

### atlas-channel/skill/handler/heal (support package — no `model.go`/`resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | file placement | PASS | No Processor/RestModel/entity/builder content in this package; `heal.go` (package-named) holds only handler wiring/seam vars, not ≥2 of the FILE-01..05 responsibilities — not a FILE-06 catch-all |
| DOM-20 | table-driven tests | WARN | `formula_test.go:97` `TestHealDelta` is table-driven (PASS), but `heal_apply_test.go:138,184,219` (`TestApply_NotZombified_HealsEveryRecipient`, `TestApply_ZombifiedCaster_DamagesEveryRecipient`, `TestApply_ZombifyReadIsCasterOnlyAndIssuedOnce`) are three separate scenario functions, not a `tests := []struct{...}` + `t.Run` table |
| DOM-24 | producertest stub for reachable emit path | N/A | `Apply`'s seams (`changeHpFunc`, `awardExperienceFunc`, `announceCastFunc`) are all replaced with in-test fakes in `installHealSeams` (`heal_apply_test.go:62-136`); no real `producer.ProviderImpl`/`AndEmit` is reached |
| DOM-25 | client-interpreted bytes resolved from writer-options table | N/A | No dispatcher mode/sub-op/message-code literal added or changed; `announceCastFunc` only re-wraps pre-existing `AnnounceSkillUse`/`AnnounceForeignSkillUse` calls |
| DOM-26 | goroutines via `routine.Go` | N/A | No `go` statement added |

### atlas-consumables/character/buff (domain package — `model.go`, `rest.go` new this diff)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` | FAIL | No `builder.go`; constructed via `NewBuff(...)` (`model.go:51`) |
| DOM-02/03 | entity conversion | N/A | No `entity.go` |
| DOM-04 | `func Transform(` in `rest.go` | FAIL | `rest.go` (49 lines) defines `Extract` only (`rest.go:34`); no `Transform` |
| DOM-05 | `TransformSlice` | N/A | No `resource.go`/list handler |
| DOM-11 | lazy providers | N/A | No `provider.go` |
| DOM-16 | `administrator.go` for writes | N/A | No local DB writes — package is a Kafka/REST client only (`producer.go` sends commands, no `gorm.DB`) |
| FILE-01..06 | file placement | PASS | Processor in `processor.go`; RestModel/Extract in `rest.go`; requests in `requests.go`; Model/NewBuff in `model.go`; no package-named catch-all (`buff.go` does not exist) |
| DOM-18 | RestModel implements JSON:API interface | PASS | `rest.go:21,25,29` |
| DOM-19 | flat request models | PASS | No nested Data/Type/Attributes |
| DOM-31 | tenant/trace only via context | PASS | `requests.go` takes `ctx` and resolves via `requests.RootUrlFor(ctx, "BUFFS")`; no tenant field on `RestModel` |
| EXT-01 | target RestModel has `SetToOneReferenceID`/`SetToManyReferenceIDs` | FAIL | `rest.go:10-32` — neither method defined. Package calls `requests.DrainProvider[RestModel, Model]` (`processor.go:67`) against atlas-buffs |
| EXT-02 | httptest fixture asserts populated struct | FAIL | Only httptest test is `processor_notfound_test.go:22` `TestGetByCharacterIdTreatsNotFoundAsNoBuffs`, which serves a bare 404 and asserts an *empty* slice. No test serves a representative JSON:API buff fixture (sourceId/level/changes/createdAt/expiresAt) and asserts a populated `[]Model`. `model_test.go`'s `TestIsZombified`/`TestExpiredHonoursNoExpiry` exercise `Model`/`IsZombified` directly, bypassing `Extract`/unmarshal entirely, so they do not satisfy this check |
| EXT-03 | only genuine 404 → not-found | PASS | `processor.go:68` `errors.Is(err, requests.ErrNotFound)` |
| EXT-04 | URL via root-url helper | PASS | `requests.go:15` `requests.RootUrlFor(ctx, "BUFFS")`, consistent with `RootUrlFor`'s established use across 14 other `requests.go` files in this service |
| DOM-20 | table-driven tests | PASS | `model_test.go:11` `TestIsZombified` |
| DOM-33 | mocks updated for interface change | PASS | `buff.Processor` gained `GetByCharacterId(characterId uint32) ([]Model, error)` (`processor.go:22`); `mock/processor.go:15,43-48` (`ProcessorMock.GetByCharacterIdFunc`) implements it in the same diff |

### atlas-consumables/character/buff/stat (domain package — `rest.go` new this diff)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` | FAIL | No `builder.go`; `Model{Type, Amount}` is a plain struct literal (no constructor at all) |
| DOM-04 | `func Transform(` in `rest.go` | FAIL | `stat/rest.go:10` defines `Extract` only |
| DOM-18 | RestModel implements JSON:API interface | N/A | `stat.RestModel` is a nested attribute type embedded in `buff.RestModel.Changes []stat.RestModel` (`rest.go:15`), never unmarshaled as a standalone JSON:API primary resource; the pre-existing, unmodified `atlas-channel/character/buff/stat/rest.go` follows the identical shape (no `GetName`/`GetID`/`SetID`), evidencing this is the established design for a nested attribute DTO in this codebase, not an omission of a top-level resource |
| FILE-01..06 | file placement | PASS | `Model` in `model.go` (pre-existing), `RestModel`/`Extract` in `rest.go`; no catch-all |

### atlas-consumables/character/buff/mock (support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-33 | mock matches interface | PASS | `mock/processor.go:11-16` declares `GetByCharacterIdFunc`; `mock/processor.go:43-48` implements `GetByCharacterId` with the nil-check-default pattern |
| FILE-01..06 | file placement | PASS | Single-purpose mock file, no catch-all |

### atlas-consumables/consumable (domain package — `model.go` pre-existing)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `processor.go:95` `NewProcessor(l logrus.FieldLogger, ctx context.Context)` |
| DOM-33 | mocks updated for interface change | N/A | `consumable.Processor` interface itself is unchanged; only unexported helper functions (`computeEffectPlan`, `computeMorphCouponPlan`, `halveIfZombified`, `resolveZombified`) gained a `zombified` parameter |
| DOM-30 | DB write + Kafka emit via `AndEmit`/`message.Buffer` | N/A | `ApplyItemEffects`/`consumeMorphCoupon` call `bp.Apply(...)`/`d.buff.Apply(...)` which reach `producer.ProviderImpl` directly (`character/buff/processor.go:42`), but neither function performs a local DB write on any path — documented exception (no-DB-state), unchanged by this diff |
| DOM-20 | table-driven tests | PASS | `processor_test.go` `TestComputeEffectPlan_Zombify` (table with 8 cases); `morph_coupon_test.go` `TestComputeMorphCouponPlan` (extended table with 3 new zombify rows) |
| DOM-24 | producertest stub for reachable emit path | PASS | New zombify tests inject `buffmock.ProcessorMock` (`morph_coupon_test.go` `h.deps.buff = &buffmock.ProcessorMock{...}`); no real producer reached |
| FILE-01..06 | file placement | PASS | `morph_coupon.go`/`processor.go` are not package-named (package is `consumable`, files are not `consumable.go`), so FILE-06 does not fire on them |

## Security Review

Not applicable — SEC-01..04 trigger requires a service handling authentication, authorization, tokens, redirects, or secrets. Neither `atlas-channel`'s heal handler nor `atlas-consumables`' buff client/consumable processor does.

## Not evaluable from the diff

- None. All applicable checks were settled from the changed files plus targeted same-package greps (processor.go/rest.go/requests.go/model.go in the buff packages, and the sibling atlas-buffs server-side `buff/rest.go` used only to confirm the upstream JSON:API shape for EXT-01/EXT-02).

## Summary

### Blocking (must fix)

- DOM-01: `atlas-channel/character/buff` — no `builder.go` (package in scope via `model.go` change)
- DOM-01: `atlas-consumables/character/buff` — no `builder.go` (new package)
- DOM-01: `atlas-consumables/character/buff/stat` — no `builder.go` (new `rest.go` puts package in scope)
- DOM-04: `atlas-channel/character/buff/rest.go` — no `func Transform(`
- DOM-04: `atlas-consumables/character/buff/rest.go` — no `func Transform(`
- DOM-04: `atlas-consumables/character/buff/stat/rest.go` — no `func Transform(`
- EXT-01: `atlas-channel/character/buff/rest.go` — `RestModel` missing `SetToOneReferenceID`/`SetToManyReferenceIDs`
- EXT-01: `atlas-consumables/character/buff/rest.go` — `RestModel` missing `SetToOneReferenceID`/`SetToManyReferenceIDs`
- EXT-02: `atlas-consumables/character/buff` — no httptest test serves a populated JSON:API buff fixture and asserts a non-empty decoded `[]Model`; only the 404-path test exists

### Non-Blocking (should fix)

- DOM-20: `atlas-channel/skill/handler/heal/heal_apply_test.go` — the three `TestApply_*` scenario tests are not table-driven (`tests := []struct{...}` + `t.Run`)
