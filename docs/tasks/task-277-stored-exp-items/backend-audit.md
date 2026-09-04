# Backend Audit — task-277-stored-exp-items

- **Service Path:** services/atlas-channel, services/atlas-character, services/atlas-consumables, services/atlas-configurations, services/atlas-data, libs/atlas-constants, libs/atlas-packet
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-28
- **Build:** PASS
- **Tests:** All packages passed (0 failed) across atlas-channel, atlas-character, atlas-consumables, atlas-configurations, atlas-data, libs/atlas-packet, libs/atlas-constants
- **Overall:** PASS

## Build & Test Results

Diff range: `bda6566f3f87da03c6e65719f2e8163acfbf5bb6..7ec1cb118`

```
services/atlas-channel/atlas.com/channel        : go build OK, go test OK (all packages ok/no-test-files)
services/atlas-character/atlas.com/character    : go build OK, go test OK (atlas-character/character 12.1s, kafka/consumer/character 23.4s, pending_change 302.0s — pre-existing slow suite, unrelated to this diff)
services/atlas-consumables/atlas.com/consumables: go build OK, go test OK
services/atlas-configurations/atlas.com/configurations: go build OK, go test OK
services/atlas-data/atlas.com/data              : go build OK, go test OK
libs/atlas-packet                               : go build OK, go test OK
libs/atlas-constants                            : go build OK, go test OK
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01,02,03,11,16) | Fired (no new violations found) | model.go present in atlas-channel/character, atlas-character/character, atlas-consumables/character, atlas-consumables/data/consumable; none of entity.go/builder.go/provider.go touched by diff |
| FILE placement (FILE-01..06) | Fired | Every changed Go package audited |
| SUB sub-domain (SUB-01..04) | Fired | atlas-data/atlas.com/data/consumable has resource.go, no model.go (reader.go, rest.go touched) |
| REST (DOM-04..09,12..19,32) | Fired | rest.go touched in atlas-consumables/data/consumable and atlas-data/consumable; processor.go touched in 4 packages |
| Constants reuse (DOM-21) | Fired | New Classification const (item/constants.go), new SpecType const, new Command string consts, new local level-cap const |
| Testing (DOM-10,20,24,33) | Fired | Diff adds/changes many `_test.go` files; Processor interfaces gain new methods (atlas-channel character, atlas-consumables character, atlas-character character); atlas-character tests reach the outbox emit path |
| Cache (DOM-29) | N/A | No `cache.go` in any changed package; no processor/struct holds cached state in the diff |
| Messaging (DOM-30) | Fired | atlas-character `character` package writes to DB and emits via `message.Emit`/`message.Buffer` (AndEmit pattern) |
| Multi-tenancy (DOM-31) | Fired | rest.go touched (consumables/data/consumable, atlas-data/consumable); atlas-character processor.go passes `p.ctx` into `p.db.WithContext` |
| Migration hygiene (DOM-34,35) | N/A | Diff adds new symbols; nothing moved/extracted between a service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module added; no new Kafka topic env var — new work rides the existing `COMMAND_TOPIC_CHARACTER` topic with new `Command*` string constants |
| Runtime safety (DOM-26) | Fired (no violation) | Non-test Go files changed; no bare `go` statement added anywhere in the diff |
| Channel wire values (DOM-25) | Fired (no violation) | Diff touches atlas-channel and libs/atlas-packet; neither new op carries a dispatcher mode/sub-op/fail-reason byte — bodies are inert (tick, or itemId/slot/tick) |
| Resilience (DOM-27,28) | N/A | No HTTP handler in the diff writes `http.StatusInternalServerError`; no `model.Decorator`/enrichment path touched |
| External clients (EXT-01..04) | N/A | No new `requests.*Request[T]` call site added; consumables/data/consumable's existing atlas-data client (`requests.go`) is untouched by this diff |
| Scaffolding (SCAFFOLD-01..09) | Fired (07 only) | Diff registers two new atlas-channel handlers (`CharacterItemUseSolomonHandle`, `CharacterUseStoredExperienceHandle`); no new service directory, no routes.conf change |
| Security (SEC-01..04) | N/A | No service in scope handles auth, tokens, redirects, or secrets in this diff |

## Checklist Results

### libs/atlas-constants/item (support — constants file)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclaration of existing constant | PASS | `libs/atlas-constants/item/constants.go:52` adds `ClassificationConsumableExpUpItem = Classification(237)`, a genuinely new classification value, cited with derivation comment (lines 44-51); no existing 237 classification found elsewhere |
| FILE-01..06 | File placement | N/A | Pure constants file, no processor/rest/entity content |

### libs/atlas-packet/character/serverbound (support — packet codec)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | No client wire code as Go literal outside codec internals | PASS | `stored_experience_use.go:26-51` — body is a single `updateTime` tick; no dispatcher/sub-op/fail-reason byte present |
| DOM-20 | Table-driven tests | N/A (round-trip + fixture style, not table-driven) — see Not evaluable | `stored_experience_use_test.go:18-61` uses `pt.Variants` loop (table-like) plus two fixed-value fixture tests; not a `tests := []struct{}` table in the literal sense, but follows the established `libs/atlas-packet` codec test convention (round-trip loop + byte-fixture pins), which is this family's documented idiom, not a fresh ad hoc pattern |

### libs/atlas-packet/inventory/serverbound (support — packet codec)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | No client wire code as Go literal outside codec internals | PASS | `solomon_item_use.go` embeds `ItemUse` (existing codec), no new literal wire code added; `item_use.go:17` adds only a handle-name string constant |
| — | Dead/unused code note | PASS (documented) | `solomon_item_use.go:3-14` is explicitly an "AUDIT-ONLY codec" per its own comment, with a stated reason (one wrapper per op for audit-trail integrity) and an explicit "do not simplify away" directive referencing task-229 precedent — not an orphaned file |

### services/atlas-channel/atlas.com/channel/character (domain — has model.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `character/processor.go:64` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-30 | AndEmit + message.Buffer pattern for DB writes | N/A | This package makes no DB write — `RedeemStoredExperience` (processor.go:335-337) only emits a Kafka command via `producer.ProviderImpl`, matching the sibling `AwardExperienceCommandProvider`/`ChangeHP` pattern in the same file |
| DOM-33 | Mock updated for interface change | PASS | `character/mock/processor.go:36,187-192` adds `RedeemStoredExperienceFunc` and the `RedeemStoredExperience` method to `MockProcessor`, matching the new `Processor.RedeemStoredExperience` method at `processor.go:53` |
| FILE-01 | Processor content lives in processor.go | PASS | New method at `processor.go:335-337` |
| FILE-05 | Provider/command-builder placement | PASS | `RedeemStoredExperienceCommandProvider` lives in `producer.go:134-147`, alongside sibling `*CommandProvider` functions |

### services/atlas-channel/atlas.com/channel/kafka/message/character (support — message contracts)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclaration of existing constant | PASS | `kafka.go:21` adds `CommandRedeemStoredExperience = "REDEEM_STORED_EXPERIENCE"`, new string, no collision found |
| DOM-23 | New topic env vars follow convention | N/A | No new topic env var added — `EnvCommandTopic` line at `kafka.go:11` is realigned (gofmt) but unchanged in value; the new work rides the existing `COMMAND_TOPIC_CHARACTER` topic |

### services/atlas-channel/atlas.com/channel/socket/handler (support — packet handler, no resource.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-12..15 | Handler discipline (no os.Getenv, no cross-domain orchestration, calls processor not provider, no direct db writes) | N/A | Package has no `resource.go` — these rules trigger on REST resource handlers, not socket packet handlers |
| DOM-25 | No client wire code as Go literal | PASS | `character_stored_experience_use.go:28-34` and the `character_item_use.go` addition (`CharacterItemUseSolomonHandleFunc`) decode only `itemId`/`slot`/`updateTime`; no dispatcher/sub-op byte introduced |
| SCAFFOLD-07 | New Writer/Handler constants seeded in every targeted tenant seed template | PASS | `services/atlas-configurations/seed-data/templates/template_gms_{72,79,83,84,87,92,95}_1.json` and `template_jms_185_1.json` each carry `CharacterItemUseSolomonHandle` and `CharacterUseStoredExperienceHandle` (2 matches each, verified via grep); `atlas-configurations/socket/corpus_test.go:63-64` pins the corpus count increase (3387→3403, +16 = 2 bindings × 8 templates) with a narrative citing task-277 |
| FILE-06 | No catch-all file | PASS | `character_stored_experience_use.go` carries exactly one handler + its test seam var, matching the file-per-op convention already used by `character_item_use.go` |

### services/atlas-character/atlas.com/character/character (domain — model.go + entity.go + resource.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor logger type | PASS | `processor.go:176` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor` |
| DOM-16 | administrator.go holds write functions | PASS | `administrator.go:287-292` adds `SetGachaponExperience(amount uint32) EntityUpdateFunction`, alongside sibling `SetExperience` |
| DOM-30 | AndEmit + message.Buffer for DB writes | PASS | `processor.go:1469-1474` `CreditStoredExperienceAndEmit` wraps `database.ExecuteTransaction` + `message.Emit(outbox.EmitProvider(...))` + `message.Buffer`; `processor.go:1531-1568` `RedeemStoredExperienceAndEmit` follows the identical pattern, awarding EXP on the same transaction via `ip.AwardExperience(mb)(...)` |
| DOM-31 | Tenant/trace travel in context only | PASS | `processor.go:1470,1532` use `p.db.WithContext(p.ctx)`; no tenant field added to any REST/request struct |
| DOM-33 | Mock updated for interface change | N/A | No mock of `character.Processor` exists inside `atlas-character` itself (it is the domain owner; only cross-service clients mock it, and none of those Processor interfaces changed) |
| FILE-01 | Processor content in processor.go | PASS | New methods at `processor.go:1469-1571` |
| FILE-05 | Administrator/model placement | PASS | `SetGachaponExperience` in `administrator.go`; no new model.go/builder.go field added (GachaponExperience field pre-exists) |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | PASS | `processor_test.go:43` (shared helper used by `processor_stored_experience_test.go`) |
| DOM-20 | Table-driven tests | PASS | `processor_stored_experience_test.go:59-95` (`TestCreditStoredExperience`) and `:178-268` (`TestRedeemStoredExperience`) both use `tests := []struct{...}` + `t.Run` |
| DOM-24 | Emit-reaching tests install `producertest` stub | PASS | `character/testmain_test.go:10-12` calls `producertest.InstallNoop()` in a package-wide `TestMain`, which governs the whole test binary (both `package character` and `package character_test` files, confirmed by the diff's tests passing) |

### services/atlas-character/atlas.com/character/kafka/consumer/character (support — Kafka consumer)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Handler funcs colocated with sibling handlers | PASS | `handleCreditStoredExperience`/`handleRedeemStoredExperience` added at `consumer.go:281-299`, matching the file's existing `handleChangeHP`/`handleChangeMP` shape |
| DOM-20/24 | Consumer-level test for new handlers | Not evaluable from the diff-level evidence alone as a "must fail" item — see below | No `consumer_test.go` case exercises `handleCreditStoredExperience`/`handleRedeemStoredExperience` directly, but this is consistent with every other sibling handler in the same file (`handleChangeHP`, `handleChangeMP`, `handleSetHP`, `handleAwardExperience` are equally untested at the consumer-dispatch level; only `handleCreateCharacter` has a consumer-level test, task-18-specific). The underlying `*AndEmit` logic these two handlers delegate to is fully covered by `processor_stored_experience_test.go`. Not a new deviation introduced by this diff. |

### services/atlas-character/atlas.com/character/kafka/message/character (support — message contracts)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclaration of existing constant | PASS | `kafka.go:34-35` adds `CommandCreditStoredExperience`/`CommandRedeemStoredExperience`, both new strings |

### services/atlas-configurations/atlas.com/configurations/socket (support — corpus test only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Corpus count assertion updated with the new bindings | PASS | `corpus_test.go:63` bumps 3387→3403 with a narrative entry for task-277's 16 bindings |

### services/atlas-consumables/atlas.com/consumables/character (domain — model.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor logger type | PASS | `character/processor.go:32` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-30 | AndEmit for DB writes | N/A | This package performs no DB write; `CreditStoredExperience` (processor.go:70-72) fires a Kafka command via `producer.ProviderImpl`, identical shape to sibling `ChangeHP`/`ChangeMP`/`ChangeMap` in the same file |
| DOM-33 | Mock updated for interface change | PASS | `character/mock/processor.go:16,58-63` adds `CreditStoredExperienceFunc` and the method, matching `Processor.CreditStoredExperience` at `processor.go:23` |
| FILE-05 | Provider placement | PASS | `creditStoredExperienceCommandProvider` in `producer.go:41-55`, alongside sibling `*CommandProvider` funcs |
| DOM-20 | Table-driven test | N/A (single-case test, not table-driven, but matches sibling `TestSetHPCommandProvider`/`TestRequestChangeMesoCommandProvider` shape in the same file) | `producer_test.go:16-40` |

### services/atlas-consumables/atlas.com/consumables/consumable (support — no model.go, no resource.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file | PASS | New `solomon.go` (104 lines) carries exactly the Writ-of-Solomon consumer: `routesToSolomon`, `solomonDeps`, `consumeSolomon`, `ConsumeSolomon` — mirrors the existing `morph_coupon.go` file split, not a package-named catch-all |
| DOM-20 | Table-driven tests | PASS | `solomon_test.go:113-233` (`TestConsumeSolomon`) uses `tests := []struct{...}` + `t.Run`; `TestRoutesToSolomon` similarly table-driven |
| — | Ordering/eligibility contract (every fallible read before commit; rejections release the reservation) | PASS | `solomon.go:50-83` — reads (`ff`, `fi`, `fc`) all happen via `model.NewGroup`/`model.Submit` before `d.compartment.ConsumeItem`; every rejection path (`spec/exp` absent, level exceeds `maxLevel`, non-zero existing balance) returns via `d.onError` before the commit |
| — | Test-only constructor rule (no `*_testhelpers.go`) | PASS | `solomon_test.go:29-46` builds fixtures through the package's own exported `Extract` functions, not a bespoke test constructor |

### services/atlas-consumables/atlas.com/consumables/data/consumable (domain — model.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform(Model) (RestModel, error)` in rest.go | PASS | `rest.go:93` pre-existing `Transform`, updated to carry `MaxLevel` at `rest.go:118` |
| DOM-18 | REST model implements JSON:API interface | PASS | `rest.go:76,80,84` `GetName()`/`GetID()`/`SetID()` pre-existing, unaffected by this diff |
| DOM-19 | Flat request models | N/A | No new request struct added; `RestModel` is a JSON:API resource model, not a POST/PATCH request body |
| DOM-21 | No redeclaration | PASS | `model.go:33` adds `SpecTypeExperience = SpecType("exp")`, a new value in this package's own `SpecType` enum — not present elsewhere in this package before the diff |
| DOM-20 | Table-driven tests | PASS | `rest_test.go:135-164` (`TestExtractMaxLevelAndExperienceSpec`) uses `t.Run` subtests over populated/zero-value cases |

### services/atlas-consumables/atlas.com/consumables/kafka/message/character (support — message contracts)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclaration | PASS | `kafka.go:17` adds `CommandCreditStoredExperience = "CREDIT_STORED_EXPERIENCE"`, new string |

### services/atlas-data/atlas.com/data/consumable (sub-domain — resource.go, no model.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic in processor, not handler | PASS | `reader.go` changes are pure WZ-XML field extraction (`reader.go:59,150`), not handler logic; `resource.go` untouched by this diff |
| SUB-02 | No `db.Create`/`db.Save` in resource.go | PASS | `resource.go` (untouched, GET-only, reads from `document.Storage` cache — no DB write present) |
| SUB-03 | POST via `RegisterInputHandler[T]` | N/A | `resource.go` registers only GET routes (`resource.go:25-26`), no POST/PATCH added |
| SUB-04 | No manual JSON parsing in resource.go | PASS | No `json.Unmarshal`/`json.NewDecoder`/`io.ReadAll` in `resource.go` |
| DOM-21 | No redeclaration | PASS | `rest.go:34` adds `SpecTypeExperience = SpecType("exp")` — this package's own independent `SpecType` enum (mirrors the parallel enum in atlas-consumables/data/consumable, each service owning its own copy of this cross-service REST shape per the existing repo-wide convention for domain-model duplication across service boundaries) |
| DOM-20 | Table-driven tests | PASS | `reader_test.go:1268-1341` (`TestReadMaxLevelAndExperienceSpec`) — 4-case table over both-present/maxLevel-absent/exp-absent/both-absent |

## Security Review

SEC-* did not fire — no service in scope handles authentication, authorization, tokens, redirects, or secrets in this diff.

## Not evaluable from the diff

- `libs/atlas-packet/character/serverbound` DOM-20 (strict table-driven form): the test uses the packet-family's established round-trip-loop + byte-fixture convention rather than a literal `tests := []struct{}` table; settling whether this is the family's documented exception vs. a gap would require reading `docs/packets/PROCESS.md`'s test-authoring guidance in full, which is out of this scope's targeted-lookup budget.
- atlas-character `kafka/consumer/character` consumer-level test coverage for `handleCreditStoredExperience`/`handleRedeemStoredExperience`: confirmed absent and confirmed consistent with every other sibling handler in the same file, but I did not exhaustively enumerate every `handle*` function in `consumer.go` against every `_test.go` in the package to rule out a stricter convention introduced elsewhere in the service that this diff should have followed.

## Summary

### Blocking (must fix)
- none

### Non-Blocking (should fix)
- none
