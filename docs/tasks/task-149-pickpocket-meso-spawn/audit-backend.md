# Backend Audit — task-149-pickpocket-meso-spawn (atlas-channel)

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Review Range:** `3c499bad4..92f4e5d3c` (7 commits, HEAD `92f4e5d3c`)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-26
- **Build:** PASS
- **Tests:** all packages `ok` (no `FAIL` lines), `-count=1`, full `atlas-channel` module
- **Overall:** PASS

## Build & Test Results

```
cd services/atlas-channel/atlas.com/channel && go build ./...      # clean, no output
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1
# every listed package reports "ok" or "[no test files]"; zero FAIL lines
# specifically: ok atlas-channel/socket/handler 0.403s (includes the new
# character_attack_pick_pocket_test.go)
```

Also ran, from repo root (mandatory per `CLAUDE.md` build/verify list, scoped checks relevant to a Go-only channel diff):
- `go vet ./...` (atlas-channel module) — clean, no output.
- `gofmt -l` over `drop/`, `socket/handler/`, `kafka/message/drop/` — no files listed (formatted).
- `tools/goroutine-guard.sh` — exit 0, no violations.
- `tools/redis-key-guard.sh` — exit 0, no violations.

Diff stat (`git diff 3c499bad4..92f4e5d3c --stat`): 9 files changed, 712 insertions(+), 25 deletions(-), entirely inside `services/atlas-channel/atlas.com/channel`.

## Domain Discovery

| Package | Classification | Files touched by this diff |
|---|---|---|
| `socket/handler` | Support package (packet-processing glue; no `model.go`) | `character_attack_common.go`, `character_attack_pick_pocket_test.go` (new), `character_attack_mp_eater_test.go`, `character_attack_drain_test.go` |
| `drop` | Domain-shaped REST-client/command package (has `model.go`, but pre-existing shape has no `entity.go`/`administrator.go`/`provider.go` — atlas-channel mirrors the owning `atlas-drops` service rather than persisting drop state itself; this is the same shape used by every other atlas-channel remote-view package, e.g. `monster`, `character`, `buff`) | `processor.go`, `producer.go`, `mock/processor.go`, `producer_test.go` (new) |
| `kafka/message/drop` | Support package (Kafka DTO/contract definitions only) | `kafka.go` |

No new packages were created; no `resource.go` was touched (SUB-* checklist not triggered); no new HTTP client call sites were added (External HTTP Client Checklist not triggered — `SpawnMeso` emits via `producer.ProviderImpl`/Kafka, not `requests.GetRequest[T]`/`PostRequest[T]`).

## File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | `Processor` interface + method in `processor.go` | PASS | `drop/processor.go:19` adds `SpawnMeso(...)` to the `Processor` interface; `drop/processor.go:56-58` implements it on `*ProcessorImpl`. No processor logic leaked into `producer.go` or `model.go`. |
| FILE-02 | RestModel/Transform/JSON:API in `rest.go` | PASS (unaffected) | Diff does not touch `drop/rest.go`; no new REST models were introduced by this feature. |
| FILE-03 | Cross-service request funcs in `requests.go` | PASS (N/A for new code) | The new capability is a Kafka producer, not a cross-service HTTP request; `drop/requests.go` is untouched. New code correctly lives in `producer.go` (`drop/producer.go:32-63`, `SpawnMesoCommandProvider`), matching the `producer.go` file-responsibility ("Kafka message creation ... `producer.ProviderImpl`"). |
| FILE-04 | Entity + Migration + TableName in `entity.go` | N/A | `drop` package has no persistent entity (mirrors `atlas-drops`); no `entity.go` exists, consistent with every other atlas-channel remote-view package. Pre-existing shape, not introduced or altered by this diff. |
| FILE-05 | Builder/Model/administrator/provider placement | N/A for new code | No new fields were added to `drop.Model`; `builder.go` untouched by this diff. |
| FILE-06 | No package-named catch-all file | PASS | `drop/producer.go` holds only producer/provider functions (`RequestReservationCommandProvider`, `SpawnMesoCommandProvider`); `processor.go` holds only the interface + impl methods; no single file bundles ≥2 responsibilities (Processor+RestModel+requests). `socket/handler/character_attack_common.go` holds only packet-processing helper functions (no Processor/RestModel/requests symbols at all — this file predates the diff and is the established location for all per-monster attack-passive helpers: MP Eater, drain-family heal, and now Pick Pocket), so FILE-06's "≥2 of the responsibilities above" bar does not apply to it. |

**Mock synchronization (testing-guide.md Interface Change Workflow):** `Processor.SpawnMeso` was added at `drop/processor.go:19`; the mock was updated in the same commit set at `drop/mock/processor.go:14` (`SpawnMesoFunc` field) and `drop/mock/processor.go:39-44` (method with nil-check + default `nil` return), following the documented mock pattern exactly. PASS.

## Domain Checklist Results (`drop` package, scoped to touched symbols)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor accepts `logrus.FieldLogger` | PASS (pre-existing, unaffected) | `drop/processor.go:24` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` — untouched by this diff, still correct. |
| DOM-11 | Providers use lazy evaluation | N/A | No new provider (read) functions were added; `SpawnMeso` is a write/emit operation, `producer.ProviderImpl(...)(...)(...)` is the correct curried emit call, matching the pattern used by pre-existing `RequestReservation` (`drop/processor.go:53`). |
| DOM-21 | No duplication of atlas-constants types | PASS | New skill-id references (`skill3.RogueDoubleStabId`, `BanditSavageBlowId`, `ChiefBanditAssaulterId`, `ChiefBanditBandOfThievesId`, `ShadowerAssassinateId`, `ShadowerTauntId`, `ShadowerBoomerangStepId`, `ChiefBanditPickpocketId`) all resolve to existing constants in `libs/atlas-constants/skill/constants.go:3143-3187` — none redeclared locally. The new `TemporaryStatTypePickPocket` reference (`character_attack_common.go:245`) resolves to `libs/atlas-constants/character/temporary_stat.go:33` (`"PICK_POCKET"`), not a locally reinvented string. Verified via grep against `libs/atlas-constants/`; no new type/enum/numeric-classification was declared in the diff. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS (N/A — no live producer touched) | `drop/producer_test.go` calls `drop.SpawnMesoCommandProvider(f, ...)()` directly — this builds an in-memory `[]kafka.Message` via `producer.SingleMessageProvider` and never reaches `producer.ProviderImpl`/the real writer factory, so no Kafka stub is required. `character_attack_pick_pocket_test.go` never calls `Processor.SpawnMeso`/`drop.NewProcessor(...).SpawnMeso` either — `pickPocketTryProc` is tested with an injected `spawnMeso func(...) error` closure (e.g. `character_attack_pick_pocket_test.go:300-303`), so the emit path is never exercised against the real producer. No transitive emit path in the changed test files. |

## Anti-Pattern / Cross-Cutting Checks

| Check | Status | Evidence |
|---|---|---|
| Bare `go` statements | PASS | `git diff 3c499bad4..92f4e5d3c` contains zero added lines matching a bare `go func`/`go <ident>` statement; `tools/goroutine-guard.sh` exits 0 tree-wide. |
| Tenant fields not leaked into public API | PASS | `Processor.SpawnMeso(f field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) error` (`drop/processor.go:19`) carries only character/monster ids and coordinates — no `tenantId` parameter; tenant flows via `p.ctx` per the existing constructor pattern. |
| Curried/context-aware producer construction | PASS | `producer.ProviderImpl(p.l)(p.ctx)(drop2.EnvCommandTopic)(SpawnMesoCommandProvider(...))` (`drop/processor.go:57`) matches the documented `producer.ProviderImpl(log)(ctx)` invocation exactly (patterns-multitenancy-context.md). |
| Cross-service Kafka contract match | PASS | `kafka/message/drop/kafka.go:37-53` (`SpawnCommandBody`) is field-for-field identical (name, JSON tag, type, order) to `atlas-drops`' `CommandSpawnBody` (`services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go:123-138`), minus the embedded `EquipmentData`, which the code comment (`kafka.go:36`) correctly notes zero-fills on decode — confirmed against `atlas-drops`' `handleSpawn` (`services/atlas-drops/atlas.com/drops/kafka/consumer/drop/consumer.go:52-81`), which only reads `EquipmentData` fields for equip-item drops. `Command[E]` field types (`world.Id`, `channel.Id`, `_map.Id`) match on both sides (both import `libs/atlas-constants`). `DropType=2`/`PlayerDrop=true` claim ("universal pickup via `CanBeReservedBy`") verified against `services/atlas-drops/atlas.com/drops/drop/model.go:98-113`: `CanBeReservedBy` returns `true` unconditionally when `playerDrop == true`, regardless of `ownerId`. `Mod` claim ("atlas-drops discards the field today") verified: `handleSpawn` (consumer.go:52-81) never reads `c.Body.Mod`. |

## Considered, Not Flagged

**DOM-25 (client-interpreted wire values) — `DropType: 2` literal in `drop/producer.go:50`.** `DropType` is written to the client wire byte-for-byte in `libs/atlas-packet/drop/clientbound/spawn.go:66` (`w.WriteByte(m.dropType)`), and its wire position/semantics are IDA-verified stable across all 4 audited client versions (gms_v83, gms_v87, gms_v95, jms_v185 — see `docs/packets/audits/gms_v83/DropSpawn.md:18` etc., all `✅`, same row). Checked whether this should be config-resolved per DOM-25: unlike the GuildBBS/NoticeFailReason precedents DOM-25 exists to catch (dispatcher mode bytes / notice-arm codes that vary in meaning between client versions and therefore need a tenant writer-options table), `DropType` here is a structural gameplay-ownership flag (FFA vs. owned drop), not a per-version-varying dispatcher/notice code, and no tenant seed template anywhere in the repo carries a `dropType`/`DropType` config table (checked `services/atlas-configurations/seed-data/templates/`). Every sibling producer across the fleet treats it the same way — `services/atlas-monsters/atlas.com/monsters/monster/drop/producer.go:17` hardcodes the identical `DropType: 2` for FFA monster-death drops, and `atlas-character`/`atlas-inventory`/`atlas-monster-death`/`atlas-saga-orchestrator` all pass it through as an ordinary domain parameter. No config-resolution mechanism for this field exists anywhere to be reused or omitted from. Not raised as a finding — this is an established, version-stable domain field, not a client-dispatcher code within DOM-25's documented scope.

**DOM-28 (no silent degradation) — `pickPocketResolveState`/`pickPocketTryProc` swallow buff/effect/monster-fetch errors.** `pickPocketResolveState` (`character_attack_common.go:222-283`) and `pickPocketTryProc` (`character_attack_common.go:290-322`) fetch remote data (buff REST call, skill-effect lookup, monster snapshot) and, on failure, disable the proc and return, logging via `l.WithError(err).Errorf(...)`/`Debugf(...)` rather than via `model.ErrDecorator` + `degrade.Observe(...)`. Checked `patterns-resilience.md`: the mandated `degrade.Observe`/`atlas_enrichment_degraded_total` mechanism is scoped to `model.Decorator[Model]` implementations that enrich a `Model` handed back to a caller (reference impls: `atlas-login/character.InventoryDecorator`, `atlas-skills`, `atlas-consumables` — confirmed by grep, these are the only 3 call sites of `degrade.Observe`/`model.ErrDecorator` in the whole repo). Pick Pocket's gate functions are not decorators returning an enriched `Model` to a consumer; they are internal side-effect gates within a socket packet handler (same shape as the pre-existing, unmodified `mpEaterTryProc`, `character_attack_common.go:395-402`, which also logs-and-swallows a skill-effect-lookup error with no `degrade.Observe`). Not raised as a finding — this code does not match the decorator/enrichment shape DOM-28 governs, and it mirrors the established sibling pattern in the same function for the same reason (never abort the attack pipeline on a passive-proc dependency failure).

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None.

**Overall: PASS.** Build and tests are clean; all touched files place their symbols in the guideline-designated files (Processor method in `processor.go`, producer/provider function in `producer.go`, mock kept in sync); no reinvented atlas-constants types; no bare goroutines; the new cross-service Kafka contract (`SpawnCommandBody` → atlas-drops' `CommandSpawnBody`) was verified field-for-field against the consumer, not merely assumed.
