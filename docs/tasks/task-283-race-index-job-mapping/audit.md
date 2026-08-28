# Backend Audit — task-283-race-index-job-mapping

- **Service Path:** services/atlas-character-factory, services/atlas-configurations, libs/atlas-constants
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-28
- **Build:** PASS
- **Tests:** 681 passed (libs/atlas-constants) + all `atlas-character-factory` packages `ok` (no failures); 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ cd services/atlas-character-factory/atlas.com/character-factory && go build ./...
(exit 0, no output)

$ go test ./... -count=1
ok  	atlas-character-factory	0.012s
ok  	atlas-character-factory/configuration	0.104s
ok  	atlas-character-factory/configuration/projection	0.010s
ok  	atlas-character-factory/data	0.024s
ok  	atlas-character-factory/factory	0.023s
ok  	atlas-character-factory/job	0.020s
ok  	atlas-character-factory/kafka/consumer/saga	0.012s
(all other packages: [no test files])

$ cd libs/atlas-constants && go build ./... && go test ./... -count=1
(exit 0)
Go test: 681 passed in 22 packages
```

`services/atlas-configurations` has no changed `.go` files in this diff (only seed-data
JSON), so no Go build/test was required there. JSON validity of the four changed seed
templates was confirmed with `python3 -c "json.load(...)"` — all four parse cleanly.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE-01..06 | Yes | `factory` (processor.go/resource.go changed) and `job` (carousel.go new) packages changed |
| RUNTIME (DOM-26) | Yes | Non-test `.go` files changed in `factory`, `job`, `libs/atlas-constants/job` |
| DOM structure (DOM-01..05,11,16) | Yes | `factory` package has `rest.go` (checklist: DOM-04/05 apply to any package with `rest.go`, model.go or not) |
| SUB-01..04 | Yes | `factory` package has `resource.go`, no `model.go` — sub-domain/action-event shape |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `factory` package has `resource.go`, `rest.go`, `processor.go` |
| Constants reuse (DOM-21) | Yes | New `job.CitizenId` const added to `libs/atlas-constants/job/constants.go`; new `job.Slot`/`job.Carousel` types added in `character-factory/job/carousel.go` |
| Testing (DOM-10,20,24,33) | Yes | `_test.go` files touched in `factory`, `job`, `libs/atlas-constants/job` |
| Cache (DOM-29) | N/A | No `cache.go`, no cached processor/struct state in changed packages |
| Messaging (DOM-30) | N/A | No `producer.go`; no new `AndEmit`/`message.Emit`/`producer.ProviderImpl` call sites added |
| Multi-tenancy (DOM-31) | Yes | `factory` package has `rest.go`; `processor.go` reads `tenant.MustFromContext(ctx)`; `job/carousel.go` takes a `tenant.Model` parameter |
| Migration hygiene (DOM-34,35) | Yes | `libs/atlas-constants/job/model.go`'s `FromIndex` was deleted (a "twin" mapper), and `character-factory/job/model.go`'s `JobFromIndex` was deleted, replaced by a single new `character-factory/job/carousel.go` mapper |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module added; no Kafka topic env var added/renamed |
| Channel wire values (DOM-25) | N/A | Diff does not touch `atlas-channel` or `atlas-packet`; `job.Id` is an internal domain value emitted on the saga's `CharacterCreatePayload`, not a client-interpreted wire byte |
| Resilience (DOM-27,28) | N/A | `atlas-character-factory` has no `database.Connect` anywhere in the service (not DB-backed); no `model.Decorator`/enrichment path touched |
| External clients (EXT-01..04) | N/A | No new `requests.RootUrl`/`requests.GetRequest[T]`/`requests.PostRequest[T]` call sites added by this diff |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/` directory, channel writer/handler, or `routes.conf` change |
| Security (SEC-01..04) | N/A | Neither service handles authentication/authorization/tokens/redirects/secrets |
| **Foundational: patterns-provider.md** | N/A | No provider defined or composed in changed code |
| **Foundational: patterns-functional.md** | Yes (brief) | `carouselFor`/`FromIndex` are curried-ish pure functions over `tenant.Model`; reviewed inline under DOM-11 note below, no violation found |

## Checklist Results

### `factory` (sub-domain: has `resource.go`, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/constructor/impl in `processor.go` | PASS | `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:46` (`type Processor interface`), `:61` (`func NewProcessor`) |
| FILE-02 | RestModel/Transform/JSON:API methods in `rest.go` | PASS | `.../factory/rest.go:10` (`RestModel`), `:36-51` (`GetName/GetID/SetID`) — all JSON:API methods correctly placed |
| FILE-03 | Cross-service request functions in `requests.go` | N/A | No `requests.RootUrl`/`GetRequest`/`PostRequest` symbols exist anywhere in this package |
| FILE-04 | Entity struct/Migration/TableName in `entity.go` | N/A | Package has no `entity.go` and no such symbols |
| FILE-05 | Builder/Model/writes/readers placement | N/A | Package has no `model.go`, `builder.go`, `administrator.go`, or `provider.go` — this is a write/orchestration-only package with no domain-model persistence |
| FILE-06 | No catch-all file carrying ≥2 responsibilities | PASS | `processor.go` (Processor only), `resource.go` (routes/handlers only), `rest.go` (RestModel definitions only) — no collapse across `processor.go`/`resource.go`/`rest.go` |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | WARN | `.../factory/rest.go` (whole file) defines no `Transform` function and no domain `Model` type at all — this rule's own trigger (`package has rest.go`) fires per the checklist's explicit "model.go or not" callout, so it is technically unmet. Pre-existing: `rest.go` is unchanged by this diff (git diff shows zero hunks to `rest.go`), and the package is architecturally create-only (POST → saga, no domain read-back), so there is nothing to transform. Not introduced or worsened by task-283; flagged for visibility, not blocking this change. |
| DOM-05 | `TransformSlice` used by list handlers | N/A | `resource.go` registers only `Create*` POST handlers (`.../factory/resource.go:26,32,33`) — no list/GET handler exists to require `TransformSlice` |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `.../factory/processor.go:61` `func NewProcessor(l logrus.FieldLogger) Processor` |
| DOM-07 | Handlers pass `d.Logger()` to `NewProcessor` | PASS | `.../factory/resource.go:113`, `:170` `NewProcessor(d.Logger())` |
| DOM-08 | POST/PATCH via `RegisterInputHandler[T]` | PASS | `.../factory/resource.go:26,32,33` — all three POST routes use `rest.RegisterInputHandler[...]` |
| DOM-09 | Every `Transform(` call checks its error | N/A | `resource.go` contains zero `Transform(` call sites (grep returned no matches) |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `resource.go` — zero matches for `os.Getenv` |
| DOM-13 | No cross-domain orchestration in handlers | PASS | `handleCreateCharacter` (`.../factory/resource.go:168`) delegates all logic to `NewProcessor(...).Create(...)` |
| DOM-14 | Handlers call processor methods only | PASS | `resource.go` — zero provider-function call sites; only `NewProcessor(...)` calls |
| DOM-15 | No `db.Create`/`db.Save`/`db.Delete` in handlers | PASS | `resource.go` — zero matches |
| DOM-17 | Domain errors map to correct HTTP status | PASS | `.../factory/resource.go:129-141` `categorizeError`: `ErrInvalidRaceIndex`→400, `ErrTemplateNotFound`→400, `ErrNameDuplicate`→409 (new `errors.Is` switch added by this diff) |
| DOM-18 | RestModels implement `GetName/GetID/SetID` | PASS | `.../factory/rest.go:36-51` (`RestModel`), `:58-69` (`CreateCharacterResponse`), `:78-80` (`PresetCreateRestModel`) |
| DOM-19 | Request models are flat | PASS | `.../factory/rest.go:10-34` `RestModel` — flat struct, no nested `Data`/`Type`/`Attributes` |
| DOM-21 | No redeclared shared type/const | PASS | `job.CitizenId = Id(3000)` (`libs/atlas-constants/job/constants.go:183`) is a genuinely new job id, registered once in `Jobs` map (`:92`) and in `IsBeginner` (`libs/atlas-constants/job/model.go:57`); `job.Slot`/`job.Carousel` (`character-factory/job/carousel.go:9-18`) do not collide with `libs/atlas-constants/inventory/slot.Slot` (different domain, no shared-lib equivalent for a race/subjob ordinal pair) |
| DOM-26 | No bare `go` statements | PASS | `grep -nE '^\s*go (func\|[A-Za-z_])'` on `processor.go`, `resource.go`, `carousel.go`, `libs/atlas-constants/job/{constants,model}.go` — zero matches; `tools/goroutine-guard.sh` exits 0 for the whole repo |
| DOM-31 | Tenant/trace only via context | PASS | `.../factory/processor.go:101` `t := tenant.MustFromContext(ctx)`; no `TenantId` field on `RestModel` (`rest.go:10-34`); `job.FromIndex(t tenant.Model, ...)` (`job/carousel.go:113`) takes the tenant as an internal function parameter, not a public request field |
| DOM-32 | Routes register via `server.RegisterHandler`/`RegisterInputHandler[T]` | PASS | `.../factory/resource.go:26,32,33` use `rest.RegisterInputHandler[T](l)(si)(...)` (existing, unchanged wrapper); no bare `http.HandlerFunc` route bodies added |
| SUB-01 | Business logic in processor, not handler | PASS | All race-index resolution and validation logic added in `.../factory/processor.go:98-111` (`Create`), not in `resource.go` |
| SUB-02 | Writes via administrator; no `db.Create`/`Save` in `resource.go` | PASS | Service has no database at all (no `database.Connect` anywhere in `atlas-character-factory`); `resource.go` has zero `db.*` matches |
| SUB-03 | POST via `RegisterInputHandler[T]` | PASS | Same evidence as DOM-08 |
| SUB-04 | No manual JSON parsing in `resource.go` | PASS | Zero matches for `json.NewDecoder`/`json.Unmarshal`/`io.ReadAll` in `resource.go` |
| DOM-20 | Tests table-driven | PASS | `.../factory/processor_test.go:1686` `TestCreate_RejectsOffCarouselRaceIndex` and `.../factory/resource_test.go:149` `TestCategorizeError_InvalidRaceIndexIsBadRequest` both use `tests := []struct{...}` + `t.Run` |
| DOM-24 | Emit-reaching tests stub the producer | PASS | `.../factory/testmain_test.go:10-11` `TestMain` calls `producertest.InstallNoop()` — covers `TestSagaEmissionToKafka` (`processor_test.go:783`) and every other test in the package, including the diff's new `TestCreate_RejectsOffCarouselRaceIndex`/`TestCreate_ValidatorOrderIsPreserved` which call `p.Create(ctx, input)` |
| DOM-33 | Interface changes update all mocks | N/A | `Processor` interface method signatures unchanged by this diff (`Create(ctx, RestModel) (string, error)` at `processor.go:47` is identical before/after); only the unexported `buildCharacterCreationSaga` helper gained a parameter, and its two call sites (production + all test call sites) were updated in the same diff |
| DOM-10 | Test DB registers tenant callbacks | N/A | No test in this package opens a GORM DB directly (no `setupTestDB`/`gorm.Open` found) |

### `job` (character-factory) (support: no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file | PASS | `carousel.go` defines only `Slot`, `Carousel`, the per-version carousel vars, `carouselFor`, and `FromIndex` — single responsibility (race-ordinal → job-id mapping) |
| DOM-21 | No redeclared shared type/const | PASS | See factory-package DOM-21 evidence above (`job.Slot`/`job.Carousel` — no collision) |
| DOM-26 | No bare `go` statements | PASS | `grep` on `carousel.go` — zero matches |
| DOM-31 | Tenant only via context/param, no public leak | PASS | `job/carousel.go:113` `FromIndex(t tenant.Model, ...)` — internal function parameter, not a REST-exposed field; the tenant is sourced from context by the one caller (`factory/processor.go:101`) |
| DOM-20 | Tests table-driven | PASS | `job/carousel_test.go:22` `TestFromIndex_PerVersionCarousel` — `tests := []struct{...}` + `t.Run`; `job/carousel_test.go:109` `TestFromIndex_RejectsOffCarouselSlots` same shape |
| DOM-24 | Emit-reaching tests stub producer | N/A | No `AndEmit`/`message.Emit`/`producer.Produce` reachable from any `job` package test — pure mapping logic only |
| Version predicate convention | Version checks use `IsRegion`/`MajorAtLeast`/`MajorAtMost`/`MajorInRange`, never raw `MajorVersion() >`/`<`/`==` | PASS | `job/carousel.go:98-105` `carouselFor` — every branch uses `t.IsRegion(...)` combined with `t.MajorAtLeast(...)` or `t.MajorInRange(...)`; `grep` for `MajorVersion() *[<>=]` in `carousel.go` and `processor.go` returns no matches (the one `MajorVersion()` use, `processor.go:107`, is inside an `Errorf` log message, not a predicate) |
| Migration hygiene (DOM-34) | No alias/re-export left after the mapper move | PASS | `character-factory/job/model.go` (old `JobFromIndex`) fully deleted (`git show 9cd1ec5af:.../job/model.go` vs. absence on HEAD); no delegating wrapper introduced in `carousel.go` |
| Migration hygiene (DOM-35) | No dead references left | PASS | `grep -rn "FromIndex" --include="*.go" .` shows only the new `job.FromIndex` (carousel.go) and its call sites/tests — zero references to the deleted `libs/atlas-constants/job.FromIndex` or `character-factory/job.JobFromIndex` remain anywhere in the repo |

### `libs/atlas-constants/job`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclared shared type/const | PASS | `CitizenId = Id(3000)` (`constants.go:183`) is the sole declaration of this constant; registered once in `Jobs` (`constants.go:92`) |
| DOM-26 | No bare `go` statements | PASS | `grep` on `constants.go`/`model.go` — zero matches |
| DOM-20 | Tests table-driven | PASS | `advancement_test.go:19-30` (added rows) and new `TestIsBeginner_CoversEveryBeginnerId` (`advancement_test.go`) both use `tests := []struct{...}` + `t.Run` |
| Migration hygiene (DOM-34/35) | Twin mapper removed cleanly, no dead code | PASS | `libs/atlas-constants/job/model.go` — old `func FromIndex(jobIndex uint32, subJobIndex uint32) Id` fully deleted; `grep -rn "FromIndex" --include="*.go" .` confirms zero remaining references to it anywhere in the repo |
| Constants count invariant updated | (no rule ID — test hygiene) | PASS | `constants_test.go:9` updated `len(Jobs)` expectation from 82 → 83 to match the new `CitizenId` registration |

### `services/atlas-configurations` (seed-data)

No Go packages changed in this diff — only `services/atlas-configurations/seed-data/templates/template_gms_{48,61,72,95}_1.json`. No FILE/DOM/SUB/REST rule applies to a JSON data file; the checklist's Go-package triggers do not fire here.

| Check | Status | Evidence |
|----|-------|--------|
| Seed template JSON validity | PASS | `python3 -c "json.load(...)"` succeeded for all four changed template files |
| Alignment with carousel (cross-checked, not a numbered rule) | PASS | `character-factory/job/correspondence_test.go:109` `TestCarouselMatchesSeedTemplates` asserts bidirectional agreement between `job/carousel.go` and every seed template under `services/atlas-configurations/seed-data/templates/`, including the four files this diff touches; test passes (`go test ./job/... ` → `ok`) |
| Removed unreachable rows (gms_48/61/72 jobIndex 0/2) | PASS | `git diff 9cd1ec5af..HEAD -- .../template_gms_48_1.json` removes the `jobIndex:0` and `jobIndex:2` template entries, matching `job/carousel.go`'s `noRaceCarousel` comment ("the (0,0) and (2,0) rows... are deliberately NOT carried into this carousel") |

## Security Review

Not applicable — `atlas-character-factory` and `atlas-configurations` do not handle
authentication, authorization, tokens, redirects, or secrets in the changed code. SEC-*
family trigger did not fire.

## Not evaluable from the diff

- None. Every applicable checklist item was settled from the diff plus targeted reads of
  the surrounding package files (`factory/rest.go`, `job/model.go` at the pre-change
  commit, `libs/atlas-constants/job/model.go`) and the `go build`/`go test`/
  `tools/goroutine-guard.sh` runs.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- DOM-04: `services/atlas-character-factory/atlas.com/character-factory/factory/rest.go` defines no `Transform(Model) (RestModel, error)` and the package has no domain `Model` type at all. This predates task-283 (the file has zero hunks in this diff) and the package is architecturally create-only, but the checklist's rule fires unconditionally on any package with `rest.go`. Worth a follow-up decision on whether this package's shape should be reconciled with the documented pattern, separate from this task.
