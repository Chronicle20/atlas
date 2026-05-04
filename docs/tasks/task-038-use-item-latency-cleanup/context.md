# Task-038 — Context for Implementing Agents

Quick reference for executing the plan. The PRD (`prd.md`) has the *what/why*; the design (`design.md`) has the architectural choices; this file is the on-ramp.

## What's Being Built

Three independent latency / log-noise cleanups, plus one supporting library primitive:

1. **`libs/atlas-model`** — new `Group` / `Submit` / `Future` primitives for typed concurrent fan-in. (Leaf dependency for #4.)
2. **`atlas-channel`** — add `PartyDecorator` on the character model; HP-announce paths short-circuit when the character isn't in a party; remove misleading "Unable to announce…" debug logs.
3. **`atlas-saga-orchestrator`** — `AcceptEvent` rejects `uuid.Nil` transaction IDs before any saga storage lookup; new `nil_transaction_id` skip reason.
4. **`atlas-consumables`** — `ConsumeStandard`, `ConsumeTownScroll`, `ConsumeSummoningSack` issue their independent reads concurrently via `model.Group`. Other `Consume*` variants get a one-line code comment explaining why they don't.

## Sequencing

- **#1 must land first** — atlas-consumables depends on it.
- #2, #3, #4 are independent of each other (and of #1 once #1 has shipped to the lib). They can land in any order.
- No coordinated rollout. No event-payload changes. No DB migrations.

## Critical Files

### atlas-model
- **Create:** `libs/atlas-model/model/parallel_group.go` — `Group`, `Submit[T]`, `Future[T]`, `NewGroup`, `Wait`.
- **Create:** `libs/atlas-model/model/parallel_group_test.go` — TDD tests for the primitive.
- **Modify:** `libs/atlas-model/go.mod` — add `golang.org/x/sync` direct require.

### atlas-channel
- **Modify:** `services/atlas-channel/atlas.com/channel/character/model.go` — add `party party.Model` field, `Party()` getter, `InParty()` getter, `SetParty(party.Model) Model` top-level helper.
- **Modify:** `services/atlas-channel/atlas.com/channel/character/builder.go` — add `party` field + `SetParty(party.Model)` builder method + propagation in `CloneModel`/`Build`.
- **Modify:** `services/atlas-channel/atlas.com/channel/character/processor.go` — add `PartyDecorator(m Model) Model` to the `Processor` interface and `ProcessorImpl`.
- **Modify:** `services/atlas-channel/atlas.com/channel/character/mock/processor.go` — add `PartyDecorator` to the mock.
- **Modify:** `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go` (lines 98–107) — fetch with `PartyDecorator`, short-circuit on `!InParty()`, drop misleading log.
- **Modify:** `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` (lines 225–236) — same shape.

### atlas-saga-orchestrator
- **Modify:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` (constants block at lines 215–221) — add `SkipReasonNilTransactionId`.
- **Modify:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` (`AcceptEvent` at line 362) — guard `uuid.Nil` first.
- **Modify:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/accept_event_test.go` — new test `TestAcceptEvent_NilTransactionId`.

### atlas-consumables
- **Modify:** `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (`ConsumeStandard` lines 212–242, `ConsumeTownScroll` lines 244–284, `ConsumeSummoningSack` lines 358–397, `ConsumePetFood` lines 286–319, `ConsumeCashPetFood` lines 321–356, `ConsumeScroll` family in `RequestScroll` lines 399+).

## Key Type / API Notes

- `character.Processor` interface lives in `services/atlas-channel/atlas.com/channel/character/processor.go:24-39`. Adding a method **breaks** the existing mock (`character/mock/processor.go`), so the mock update is part of every plan task that touches the interface.
- `party.Model` already exists in `services/atlas-channel/atlas.com/channel/party/model.go`. The empty zero value (`party.Model{}`) has `id == 0` — that is the "solo" sentinel.
- `party.Processor.GetByMemberId(memberId uint32) (Model, error)` — already returns `model.ErrNoResultFound` (via `FirstProvider`) when the character is not in a party. Treat as "not in party," not as an error.
- `model.Decorator[Model]` is `func(M) M` (no error return). Existing decorators (`InventoryDecorator`, `SkillModelDecorator`, `QuestModelDecorator`) swallow REST errors silently and return `m` unchanged. `PartyDecorator` follows the same convention.
- `character2.NewProcessor(l, ctx).GetMap(characterId)` in atlas-consumables returns `field.Model` (NOT `_map.Model`). The design has a typo; the plan uses the correct type.
- `_map.Id` is `uint32`; `field.Model` has `MapId() _map.Id`.
- The atlas-saga `EventKind` is the enum-like type used by `AcceptEvent`. Tests construct it as a string-typed constant — see `accept_event_test.go` (`EventKindAssetCreated`).
- `LogSkip(l, fields, reason)` in `event_acceptance.go:226` writes a structured debug log. Reuse it.
- `golang.org/x/sync` is already in the workspace `go.work.sum` at `v0.20.0` (see `services/atlas-fame/.../go.mod`). atlas-model can pin to the same version.

## Decorators Pattern (for §2)

```go
// existing convention in processor.go
func (p *ProcessorImpl) InventoryDecorator(m Model) Model {
    i, err := p.ip.GetByCharacterId(m.Id())
    if err != nil {
        return m
    }
    return m.SetInventory(i)
}
```

`PartyDecorator` mirrors this exactly. The `Processor` interface gains `PartyDecorator(Model) Model`.

## "Solo" Sentinel Convention (for §2)

`InParty()` is implemented as `m.party.Id() != 0`. The empty `party.Model{}` produced when the decorator finds no party has `id == 0`, which the convention reads as "not in a party."

`Party()` returns the field by value. Callers in scope (the two announce paths) check `InParty()` first.

## Concurrency Primitive (for §1, used by §4)

```go
// libs/atlas-model/model/parallel_group.go
package model

import (
    "context"
    "golang.org/x/sync/errgroup"
)

type Group struct{ g *errgroup.Group }
type Future[T any] struct{ value T }

func (f *Future[T]) Get() T { return f.value }

func NewGroup(ctx context.Context) (*Group, context.Context) {
    g, gctx := errgroup.WithContext(ctx)
    return &Group{g: g}, gctx
}

func Submit[T any](g *Group, p Provider[T]) *Future[T] {
    f := &Future[T]{}
    g.g.Go(func() error {
        v, err := p()
        if err != nil { return err }
        f.value = v
        return nil
    })
    return f
}

func (g *Group) Wait() error { return g.g.Wait() }
```

`Submit` is a free function because Go does not allow type parameters on methods.

`Provider[T]` already exists in `libs/atlas-model/model/processor.go:78`.

## Saga Guard (for §3)

```go
// AcceptEvent in saga/processor.go:362 — first thing inside the function:
if transactionId == uuid.Nil {
    LogSkip(p.l, logrus.Fields{"event_kind": kind}, SkipReasonNilTransactionId)
    return AcceptDecision{}, false
}
```

`transaction_id` is intentionally **not** put on the log — the value is meaningless.

## Consumable Refactor Pattern (for §4)

Each variant produces this shape (heterogeneous-typed reads → `model.Group`):

```go
pg, _ := model.NewGroup(ctx)
fa := model.Submit(pg, func() (TypeA, error) { return readA() })
fb := model.Submit(pg, func() (TypeB, error) { return readB() })
fc := model.Submit(pg, func() (TypeC, error) { return readC() })
if err := pg.Wait(); err != nil {
    return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
}
a, b, c := fa.Get(), fb.Get(), fc.Get()
// dependent reads continue sequentially below…
```

`ConsumeError` is invoked with the error from the first read that fails. Concurrent execution does not change the observable error semantics.

## Testing Notes

- atlas-channel tests use `logtest "github.com/sirupsen/logrus/hooks/test"` for log assertions and `tenant.WithContext` for tenant-bearing contexts. See `character/processor_test.go` for the existing pattern.
- atlas-saga-orchestrator tests use the same `logtest` hook. See `accept_event_test.go:35` (`TestAcceptEvent_SagaNotFound`) as the closest sibling.
- atlas-channel character mock: `MockProcessor.GetById` already calls each decorator on the returned model (line 47–49), so the mock decorator implementation is just `return c`.
- atlas-channel mock `PartyDecorator` should also `return c` — its job in tests is to be a no-op pass-through; tests that want a decorated character pre-populate via `mock.AddCharacter(c)` where `c` already has the desired `party` field via the builder.

## Out of Scope (do not do)

- Header- or topic-level filtering of non-saga events.
- Adding `partyId` to atlas-character's REST/storage.
- Caching party state in atlas-channel.
- Removing the GET at the start of `handleStatusEventStatChanged`.
- Parallelizing `ConsumeScroll` (its reads are not independent).
- Adding business-attribute span enrichment.
- Changing saga subscription topology.
- Frontend / atlas-ui.

## Verification

After all tasks land:

- `go build ./...` clean in all four affected modules: `libs/atlas-model`, `services/atlas-channel/atlas.com/channel`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `services/atlas-consumables/atlas.com/consumables`.
- `go test ./...` passes in each.
- Manual verification (deferred to acceptance): fresh Tempo trace of a solo HP-potion use shows the three reads in `ConsumeStandard` overlap; Loki shows zero "Unable to announce…" lines and zero `saga_not_found` for nil-UUID events.
