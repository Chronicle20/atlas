# Backend Audit — task-130 Vega's Spell

- **Scope:** Go changes on branch `task-130-vegas-spell` (merge-base `38d4d0ba2` → `ba7b2799a`)
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/SEC-*)
- **Date:** 2026-07-03
- **Build:** PASS (stated gates + changed packages compile & test)
- **Tests:** PASS — changed packages complete in <1s (item 0.003s, cash codecs 0.008/0.012s, consumables/consumable 0.012s, channel/socket/handler 0.009s)
- **Overall:** PASS

## Phase 2 — Domain Discovery

The change set contains **no REST domain package** (no `model.go`/`entity.go`/`rest.go`/`resource.go` were added or touched). The touched Go packages are:

| Package | Type | Notes |
|---------|------|-------|
| `libs/atlas-constants/item` (vegas_spell.go) | Shared constants lib | New item ids + classification + classifier |
| `libs/atlas-packet/cash/{clientbound,serverbound}` | Packet codecs | VegaScroll writer arms + ItemUseVegaScroll sub-body |
| `services/atlas-consumables/.../consumable` | Kafka-driven processor pkg | RequestVegaScroll chain, applyScrollCore refactor |
| `services/atlas-consumables/.../kafka/{message,consumer,producer}` | Kafka contract | REQUEST_VEGA_SCROLL command, VEGA_SCROLL event |
| `services/atlas-channel/.../consumable` + `kafka` + `socket/handler` | Mirror + dispatch + writer registration | |

Consequently the CRUD-shaped DOM checks (DOM-01..05, DOM-11, DOM-16..19) are **N/A** — there is no domain model/entity/REST resource. The cross-cutting checks below apply.

## DOM Checklist Results (applicable)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `logrus.FieldLogger` | PASS | consumables `NewProcessor` processor.go:60; channel `NewProcessor` consumable/processor.go:20; both `logrus.FieldLogger`, no `*logrus.Logger` |
| DOM-07 | Handlers pass `l` through, no StandardLogger | PASS | Kafka handlers thread `l` (consumer.go:71-79); socket handler threads `l` (character_cash_item_use.go:26,125); no `logrus.StandardLogger()` introduced |
| DOM-12 | No `os.Getenv()` in handlers | PASS | Topics resolved via `topic.EnvProvider`/env constants; no `os.Getenv` in changed files |
| DOM-13 | No cross-domain logic in handlers | PASS | Socket handler validates packet + delegates to `consumable.NewProcessor(...).RequestVegaScrollUse` (character_cash_item_use.go:108-126); business logic lives in consumables processor |
| DOM-14/15 | Handlers don't call providers / DB directly | PASS | No `db.Create/Save/Delete`, no provider calls in handlers; writes go through compartment processor (reserve/consume/destroy) |
| DOM-20 | Table-driven tests | PASS | `TestVegaRates` vega_test.go:18; `TestUsesStandardConsumer` processor_test.go:321; `TestGetCashSlotItemTypeVegasSpell` character_cash_item_use_test.go:20 all use `cases := []struct{}` + `t.Run` |
| DOM-21 | No duplication of atlas-constants types | PASS | Ids/classification/classifier live only in `libs/atlas-constants/item/vegas_spell.go`; consumables (vega.go:31-33,100) and channel (character_cash_item_use.go:115,499) consume `item.VegasSpell10/60`, `item.IsVegasSpell`, `item.ClassificationVegasSpell`. `CashSlotItemType(68/71)` are client-protocol UI-slot enum values (channel wire concern), not item classifications — correctly local |
| DOM-23 | Kafka topic naming | PASS (N/A new topics) | No new topic env keys introduced. Uses existing `COMMAND_TOPIC_CONSUMABLE` / `EVENT_TOPIC_CONSUMABLE_STATUS` (kafka.go:14,71); task adds only new command/event *types* (`REQUEST_VEGA_SCROLL`, `VEGA_SCROLL`) within them |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS (N/A) | Emit paths (`RequestVegaScroll`, `ConsumeVegaScroll`, `VegaScrollError`, producer providers) are not exercised by any test; tests cover only pure funcs (vegaRates, resolveVegaEquip, buildScrollChanges, GetCashSlotItemType). Packages complete in ~0.01s — no 42s unstubbed-emit hang. No stub required |

## Multi-Tenancy

| Check | Status | Evidence |
|-------|--------|----------|
| Tenant gating in channel event handlers | PASS | `t := tenant.MustFromContext(ctx); if !t.Is(sc.Tenant()) { return }` in `handleErrorConsumableEvent` (consumer.go:69-72), `handleScrollConsumableEvent` (112-115), `handleVegaScrollConsumableEvent` (132-135) |
| Socket handler tenant | PASS | `t := tenant.MustFromContext(ctx)` (character_cash_item_use.go:27); version branching via `t.Region()`/`t.MajorVersion()` |
| Consumer header parsers | PASS | `consumer.TenantHeaderParser` set on both consumables (consumer.go:22) and channel (consumer.go:30) registrations |

## Cross-Service Kafka Compatibility

| Check | Status | Evidence |
|-------|--------|----------|
| Command mirror field parity | PASS | channel `RequestVegaScrollBody` (kafka.go:44-49) matches consumables `RequestVegaScrollBody` (kafka.go:51-56) tag-for-tag: `vegaSlot/vegaItemId/scrollSlot/equipSlot` |
| Event mirror field parity | PASS | Both `VegaScrollBody` = `{success, cursed}` (channel kafka.go:78-81 / consumables kafka.go:102-105); consumer emits `VegaScrollEventProvider` (producer.go:45) |
| TransactionId asymmetry | PASS (benign) | consumables `Command` carries `TransactionId` (kafka.go:24), channel mirror omits it — pre-existing (used only by APPLY/CANCEL effect commands not mirrored by channel). Vega path generates its own transactionId server-side (vega.go:97), never reads `c.TransactionId` (consumer.go:71-79) |

## Error-Handling / Reservation Rollback

| Check | Status | Evidence |
|-------|--------|----------|
| Every up-front rejection emits VEGA_INVALID (no silent wedge) | PASS | All `RequestVegaScroll` rejections funnel through `p.VegaScrollError(...)` which cancels reservations and emits `ErrorTypeVegaInvalid` (vega.go:51-62, 100-158) |
| Chained-reservation rollback | PASS | Sync producer failure in `ReserveVegaScrollStage` cancels the vega CASH reservation (vega.go:179); async inventory rejection intentionally TTL-expires (documented vega.go:162-166) |
| No swallowed errors in refactor | PASS | `applyScrollCore` propagates `buildScrollChanges` and `ChangeStat` errors (processor.go:679-694); commit-side `ConsumeItem/DestroyItem` are log-only, matching the pre-existing `ConsumeScroll` contract (processor.go:738-754 vs vega.go:229-239) |
| `applyScrollCore` behavior-preserving | PASS | rand order preserved: success roll → change assembly (chaos rand inside) → curse roll → ChangeStat (processor.go:670-695); locked by `TestBuildScrollChanges_*` table (processor_test.go:362-451) |

## Packet Codecs

| Check | Status | Evidence |
|-------|--------|----------|
| Encode/Decode symmetric, FieldLogger sigs | PASS | `ItemUseVegaScroll` reads/writes the same six int32s (item_use_vega_scroll.go:56-78); `VegaScroll*` arms single-byte (vega_scroll.go) |
| Mode byte config-resolved (owner-mandated) | PASS | `WithResolvedCode("operations", <named key>, ...)` for START/RESULT/INVALID (vega_scroll.go:170-196); unconfigured key → 99 → safe notice arm (documented) |

## Security Review

Not an auth/token service — SEC-* N/A.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (observations)
- `services/atlas-channel/.../socket/handler/character_cash_item_use.go:129` carries a pre-existing `// TODO for v83 there is a trailing updateTime.` in the fallthrough default arm. **Pre-existing** (commit b8475413f, "Issue error when attempting to use pet cash food…"), not introduced by task-130 — out of scope for this task but noted against the repo's no-TODO policy.
