# Use-Item Server Latency Cleanup — Design

Version: v1
Status: Draft
Created: 2026-04-30
Companion to: `prd.md`

---

## 1. Scope

This document covers architecture and implementation choices for the three workstreams in the PRD. The PRD already establishes the *what* and *why*; this document records *how*, including the alternatives considered and the trade-offs accepted.

Workstreams:

1. **atlas-channel** — `PartyDecorator` on the character model; HP-announce paths short-circuit on solo characters; misleading log lines deleted.
2. **atlas-saga-orchestrator** — `AcceptEvent` rejects `uuid.Nil` transaction IDs before any saga lookup; new structured-log reason `nil_transaction_id`.
3. **atlas-consumables** — `ConsumeStandard`, `ConsumeTownScroll`, `ConsumeSummoningSack` issue their independent reads concurrently via a new `model.Group` helper.

A fourth, supporting change:

4. **libs/atlas-model** — new `Group` / `Submit` / `Future` primitives that enable typed concurrent fan-in for heterogeneous reads. This is the reusable core that the atlas-consumables work consumes.

---

## 2. Overall Architecture

```
libs/atlas-model/                 → Group / Submit / Future (new)
   │
   ├── services/atlas-channel/    → PartyDecorator on character model
   │                                announce paths short-circuit on !InParty()
   │                                misleading log lines deleted
   │
   ├── services/atlas-saga-orchestrator/
   │                                AcceptEvent guards uuid.Nil up front
   │                                new SkipReasonNilTransactionId constant
   │
   └── services/atlas-consumables/
                                    Consume* paths use model.Group for
                                    independent reads
```

**Sequencing.** atlas-model lands first (leaf dependency for atlas-consumables). atlas-channel and atlas-saga-orchestrator are independent of each other and of atlas-model and may land in any order.

**No coordinated rollout.** No event-payload changes, no protocol changes, no DB migrations. Each service ships on its own merge cycle.

**Trace-propagation invariant.** Concurrency added uses `errgroup.WithContext(ctx)` (via `model.Group`). Child goroutines call providers that close over the same `ctx` containing the active span. The Tempo trace shows the parallelized reads as overlapping siblings of the consume span rather than a sequential chain.

---

## 3. `libs/atlas-model` — `Group` / `Submit` / `Future`

### 3.1 Purpose

Provide a reusable primitive for "run N independent providers concurrently and join their results with full type safety." The existing `model.ParallelMap` parallelizes a transform over a *homogeneous* slice; it does not fit cases where N differently-typed reads must be joined into a single struct or local variables.

### 3.2 Alternatives Considered

| Alternative | Outcome |
|---|---|
| **A. Inline `errgroup.WithContext` at every call site.** | Works; ~20 lines per site. Reviewed for the three Consume\* sites: code is repetitive, and a future fourth site is one copy-paste away from drift. Rejected on long-term maintainability. |
| **B. Arity-typed helpers `Parallel2[A,B]`, `Parallel3[A,B,C]`, …** | Type-safe; minimal call-site code (one line). But forces a new function for every new arity needed. Rejected because the futures-handle pattern (D) achieves the same call-site density without arity proliferation. |
| **C. Variadic `ParallelAny(ctx, …Provider[any]) ([]any, error)`.** | Composes to any arity but loses type safety; every call site needs a runtime type assertion per result. Rejected. |
| **D. Futures-handle pattern: `Group` + free generic `Submit[T]` + typed `Future[T]`.** | Selected. Composes to any N, type-safe per handle, only one generic instantiation per provider, no nested-tuple gymnastics. |

### 3.3 Surface

New file: `libs/atlas-model/model/parallel_group.go`.

```go
package model

import (
    "context"
    "golang.org/x/sync/errgroup"
)

// Group runs heterogeneously-typed providers concurrently. It is a thin
// wrapper around errgroup.Group that pairs each registered provider with a
// typed Future handle so call sites can reclaim results without runtime
// type assertions.
type Group struct {
    g *errgroup.Group
}

// Future holds the result of a provider submitted to a Group. After Wait
// returns nil, Get returns the provider's successful value. Get's behaviour
// is undefined when Wait returned an error.
type Future[T any] struct {
    value T
}

func (f *Future[T]) Get() T { return f.value }

// NewGroup returns a Group bound to a child of ctx. The child context is
// cancelled when any submitted provider returns a non-nil error or when
// Wait completes.
func NewGroup(ctx context.Context) (*Group, context.Context) {
    g, gctx := errgroup.WithContext(ctx)
    return &Group{g: g}, gctx
}

// Submit registers a provider with the group, returning a typed Future.
// Submit is a free function rather than a method because Go does not allow
// type parameters on methods.
func Submit[T any](g *Group, p Provider[T]) *Future[T] {
    f := &Future[T]{}
    g.g.Go(func() error {
        v, err := p()
        if err != nil {
            return err
        }
        f.value = v
        return nil
    })
    return f
}

// Wait blocks until all submitted providers complete and returns the first
// non-nil error, if any.
func (g *Group) Wait() error { return g.g.Wait() }
```

### 3.4 Dependency

`golang.org/x/sync/errgroup` becomes a new direct dependency of `libs/atlas-model`. It is already a transitive dep used by other services (atlas-cashshop, atlas-fame, atlas-npc-shops, atlas-map-actions, atlas-guilds, atlas-drop-information, atlas-gachapons). Adding it to the lib's `go.mod` is a one-line `require`.

### 3.5 Tests

New file: `libs/atlas-model/model/parallel_group_test.go`.

| Case | Expectation |
|---|---|
| Two successful providers | `Wait()` returns nil; both `Future.Get()` return their values. |
| One provider errors | `Wait()` returns that error. |
| Both providers error | `Wait()` returns the first error registered to the errgroup. |
| Three providers, all succeed | Verifies the compose-to-N invariant. |
| Concurrency proof | Two providers each `time.Sleep(50ms)`; `Wait()` returns in <100ms (parallel) rather than ~100ms (sequential). |

### 3.6 Non-Features

Deliberate omissions:

- **No in-flight cancellation.** `Provider[T] = func() (T, error)` does not take a `ctx`. A provider that started before a sibling errored runs to completion. Threading `ctx` through every provider in the codebase is a separate refactor.
- **No retry / fan-in / fan-out / worker pool / concurrency limit.** Group spawns one goroutine per `Submit`. Adequate for 2–3 reads; not a primitive for "parallelize 1000 reads."

---

## 4. atlas-channel — `PartyDecorator` and Announce Cleanup

### 4.1 `Party()` Accessor Convention

**Decision: non-pointer `party.Model` with `IsZero()`-style semantics**, mirroring `Inventory()` / `Skills()` / `Quests()` / `Pets()`.

Rationale:

- Matches every other decorator-loaded field on `character.Model`. Consistency wins; the model has no precedent for pointer-typed decorator fields.
- The "tri-state" (not decorated / in party / solo) is technically conflated, but the call sites in scope explicitly precede their `InParty()` check with `cp.GetById(cp.PartyDecorator)(...)`. By construction, `!c.InParty()` after that call means "not in a party," not "forgot to decorate."
- Smallest surface change to the model + builder.

`InParty()` is implemented as `m.party.Id() != 0`. The empty `party.Model{}` produced when the character is solo has `id == 0`, which is the same convention used elsewhere (`MemberModel{}` returned from `Leader()` when no leader matches).

### 4.2 `character.Model` Changes

- New private field `party party.Model`.
- New accessors: `Party() party.Model`, `InParty() bool`.
- Builder gains `SetParty(party.Model)`; `CloneModel` copies the field; `Build`/`MustBuild` propagate it.
- Top-level `model.go` gets a top-level `SetParty(p party.Model) Model` helper mirroring `SetInventory` / `SetSkills` / `SetPets` / `SetQuests` for use from the decorator.

### 4.3 `PartyDecorator`

In `character/processor.go`:

```go
func (p *ProcessorImpl) PartyDecorator(m Model) Model {
    pp := party.NewProcessor(p.l, p.ctx)
    pm, err := pp.GetByMemberId(m.Id())
    if err != nil {
        // Not in a party (FirstProvider returns an error on empty slice) is
        // not an error from the model's POV; return the character undecorated
        // so InParty() reports false. Genuine REST failures are also swallowed
        // here, matching the InventoryDecorator / SkillModelDecorator pattern.
        return m
    }
    return m.SetParty(pm)
}
```

Why swallow errors silently rather than distinguishing "not in party" from "REST failure":

- Consistency with every other decorator on `ProcessorImpl` (`InventoryDecorator`, `SkillModelDecorator`, `QuestModelDecorator`, `PetAssetEnrichmentDecorator`). They all swallow errors and return the model undecorated.
- The party REST endpoint already returns 200 OK with an empty list for solo characters; the only path that produces a non-`FirstProvider` error is a transport-level failure, which is rare and surfaces upstream via the GET that called the decorator.
- Distinguishing the two cases would require either a sentinel value (`party.Model` with a special id) or making `Party()` tri-state (pointer), which §4.1 rejects.

The interface gains:

```go
type Processor interface {
    // …existing methods…
    PartyDecorator(m Model) Model
}
```

### 4.4 Announce Path — `kafka/consumer/character/consumer.go`

Current (lines 98–107):

```go
if hpChange {
    f := field.NewBuilder(e.WorldId, e.Body.ChannelId, c.MapId()).Build()
    imf := party.OtherMemberInMap(f, c.Id())
    oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(party.NewProcessor(l, ctx).ByMemberIdProvider(e.CharacterId)))
    err = session.NewProcessor(l, ctx).ForEachByCharacterId(sc.Channel())(oip, session.Announce(l)(ctx)(wp)(partycb.PartyMemberHPWriter)(partycb.NewPartyMemberHP(c.Id(), c.Hp(), c.MaxHp()).Encode))
    if err != nil {
        l.WithError(err).Debugf("Unable to announce character [%d] health to party members.", c.Id())
    }
}
```

The character `c` was fetched two lines earlier with bare `cp.GetById()(e.CharacterId)`. The fix:

1. Re-fetch the character with `cp.PartyDecorator` applied (or, if the existing `c` is already needed for `statChanged`, fetch a second time with the decorator inside the `if hpChange` block — see §4.6 for the call-shape decision).
2. Short-circuit on `!c.InParty()` before constructing the announce chain.
3. Delete the `Unable to announce character [%d] health to party members` debug log; let any genuine error from `ForEachByCharacterId` bubble or be logged with a more accurate message.

Resulting shape (combined with §4.6's double-fetch decision — `cd` is the party-decorated re-fetch):

```go
if hpChange {
    cd, derr := cp.GetById(cp.PartyDecorator)(c.Id())
    if derr != nil || !cd.InParty() {
        return
    }
    f := field.NewBuilder(e.WorldId, e.Body.ChannelId, cd.MapId()).Build()
    imf := party.OtherMemberInMap(f, cd.Id())
    // Use the already-decorated party from cd rather than re-fetching:
    pmp := model.FixedProvider(cd.Party())
    oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(pmp))
    _ = session.NewProcessor(l, ctx).ForEachByCharacterId(sc.Channel())(oip, session.Announce(l)(ctx)(wp)(partycb.PartyMemberHPWriter)(partycb.NewPartyMemberHP(cd.Id(), cd.Hp(), cd.MaxHp()).Encode))
}
```

### 4.5 Announce Path — `kafka/consumer/map/consumer.go:225-236`

The map-consumer site builds its own party provider chain inline. The same conversion applies: fetch the character with the decorator, short-circuit on `!c.InParty()`, drop the `Unable to announce character [%d] health to party members` log. The pre-existing fan-out at line 233 (announcing other party members' HP back to the joining character) also lives inside the same goroutine and is governed by the same `!InParty()` short-circuit.

### 4.6 Single vs Double Fetch in `handleStatusEventStatChanged`

The character is fetched once at line 82 with no decorators; then `statChanged` consumes it for the stat-broadcast packet, and the new HP-announce branch needs party data. Two options:

| Option | Cost | Downside |
|---|---|---|
| **Decorate the initial fetch** (`cp.GetById(cp.PartyDecorator)(...)` once) | One extra REST call (always, even when no HP change) | Pays the party-lookup cost on every stat change, not just HP changes. |
| **Decorate inside the `if hpChange` block** (second `cp.GetById` with decorator) | One extra REST call only when HP changes | Two GETs on HP changes (the unadorned one + the decorated one). |

Selected: **decorate inside the `if hpChange` block**. The HP-change branch is the only consumer of party data; non-HP stat changes would otherwise pay an unnecessary cost. The two-GET overhead on HP changes is bounded and runs on the consumer goroutine, not the use-item root span — so it does not affect the §8.1 perf target. To keep the GET tight, fetch only the party decorator (no other decorators):

```go
if hpChange {
    cd, err := cp.GetById(cp.PartyDecorator)(c.Id())
    if err != nil || !cd.InParty() {
        return
    }
    // …construct announce…using cd.Party()
}
```

The map-consumer site already populates a per-character `cms` map upstream of the announce branch; the implementation phase will decide whether to add the party decorator to that upstream fetch or to perform a small, scoped `cp.GetById(cp.PartyDecorator)(s.CharacterId())` inside the goroutine. Either is acceptable; the constraint is that `!InParty()` short-circuits before the announce chain is constructed.

### 4.7 Tests

- `character/processor_test.go`: `TestPartyDecorator_InParty` — fakes the party REST response with the test character as a member, asserts `m.InParty() == true` and `m.Party().Id() != 0`.
- `character/processor_test.go`: `TestPartyDecorator_Solo` — fakes an empty party list, asserts the decorator does not return an error to the caller and `m.InParty() == false`.
- `character/processor_test.go`: `TestPartyDecorator_RestError` — fakes a transport failure, asserts the decorator returns the undecorated model and the caller sees `m.InParty() == false`.
- `character/builder_test.go`: extend the existing builder/clone round-trip tests to cover the new `party` field.
- `kafka/consumer/character/consumer_test.go` (extend if exists; otherwise new): scenario where `hpChange = true` and the character is solo — assert no announce packet is constructed and no `Unable to announce` log line emitted.
- `kafka/consumer/map/consumer_test.go`: same shape for the map-join site.

---

## 5. atlas-saga-orchestrator — Nil-UUID Short-Circuit

### 5.1 Placement

**Decision: single guard inside `ProcessorImpl.AcceptEvent`** (`saga/processor.go:362`), before the `GetById` lookup. All ~30 consumer sites that funnel through `AcceptEvent` are covered automatically; future consumers inherit the guard for free.

Rationale:

- `AcceptEvent` is already the documented chokepoint ("the single gate at which a saga-tagged Kafka event is matched against the saga's pending step").
- Per-consumer guards (~30 edits) carry drift risk with no functional benefit.
- A standalone helper `IsCandidateTransactionId` adds API surface without being needed: the only call site that benefits is `AcceptEvent` itself.

### 5.2 New Skip Reason

In `saga/event_acceptance.go`, add:

```go
const (
    SkipReasonSagaNotFound       = "saga_not_found"
    SkipReasonNoPendingStep      = "no_pending_step"
    SkipReasonActionMismatch     = "action_mismatch"
    SkipReasonTemplateIdMismatch = "template_id_mismatch"
    SkipReasonUnmatchedEvent     = "unmatched_event"
    SkipReasonNilTransactionId   = "nil_transaction_id" // NEW
)
```

### 5.3 `AcceptEvent` Change

```go
func (p *ProcessorImpl) AcceptEvent(transactionId uuid.UUID, kind EventKind) (AcceptDecision, bool) {
    if transactionId == uuid.Nil {
        LogSkip(p.l, logrus.Fields{
            "event_kind": kind,
        }, SkipReasonNilTransactionId)
        return AcceptDecision{}, false
    }
    // …existing GetById and step-matching logic unchanged…
}
```

The log payload deliberately omits `transaction_id` — there is no meaningful UUID to log. `event_kind` is retained so log volume per kind can be measured if it spikes.

### 5.4 Consumer Code

No consumer code changes. Every existing call site already does `if _, ok := p.AcceptEvent(...); !ok { return }`, which now short-circuits one step earlier when the transaction id is nil. The nested vs. top-level `TransactionId` shapes (some events carry it on `e.TransactionId`, some on `e.Body.TransactionId`) both resolve to a `uuid.UUID` before reaching `AcceptEvent`, so the centralized guard handles both uniformly.

### 5.5 Tests

- `saga/accept_event_test.go`: new `TestAcceptEvent_NilTransactionId` — calls `AcceptEvent(uuid.Nil, anyKind)`, asserts `(AcceptDecision{}, false)` is returned, asserts the in-memory storage was *never* read (use a mock/spy storage that fails the test on `GetById`), asserts the structured log carries `reason=nil_transaction_id`.
- Existing `TestAcceptEvent_SagaNotFound` continues to validate the non-nil-but-unknown path.

---

## 6. atlas-consumables — Parallel Independent Reads

### 6.1 In-Scope Variants

| Function | Reads | Independence | Action |
|---|---|---|---|
| `ConsumeStandard` | `cp.GetById`, `character2.GetMap`, `cdp.GetById` | All three independent | Parallelize all three. |
| `ConsumeTownScroll` | `character2.GetMap`, `cdp.GetById`, then `_map3.GetById(m.MapId())` | First two independent; third depends on `m.MapId()` | Parallelize first two; third stays sequential after `Wait()`. |
| `ConsumeSummoningSack` | `character.GetById`, `consumable3.GetById`, then `position.GetInMap(c.MapId(), …)` | First two independent; third depends on `c` | Parallelize first two; third stays sequential after `Wait()`. |
| `ConsumePetFood` | `pp.HungriestByOwnerProvider`, `cdp.GetById` | Independent | Out of scope per PRD §4.3 ("at minimum, evaluate ConsumeTownScroll and ConsumeSummoningSack"). Add a one-line code comment noting the choice and that parallelization here is a future option. |
| `ConsumeCashPetFood` | `cash.GetById`, then dependent pet/filter chain | Not independent (filter input depends on `ci.Indexes()`) | No change. |
| `ConsumeScroll` | `cp.GetById(InventoryDecorator)`, then equipment lookup depending on the result | Not independent (PRD §2 non-goals) | No change. One-line code comment confirming. |

### 6.2 `ConsumeStandard` Shape

```go
func ConsumeStandard(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
    return func(l logrus.FieldLogger) func(ctx context.Context) error {
        return func(ctx context.Context) error {
            p := NewProcessor(l, ctx)
            cp := character.NewProcessor(l, ctx)

            pg, _ := model.NewGroup(ctx)
            fc := model.Submit(pg, func() (character.Model, error) { return cp.GetById()(characterId) })
            fm := model.Submit(pg, func() (_map.Model, error) { return character2.NewProcessor(l, ctx).GetMap(characterId) })
            fi := model.Submit(pg, func() (consumable3.Model, error) { return p.cdp.GetById(uint32(itemId)) })
            if err := pg.Wait(); err != nil {
                return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
            }
            c, m, ci := fc.Get(), fm.Get(), fi.Get()

            if err := compartment.NewProcessor(l, ctx).ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot); err != nil {
                return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
            }
            ApplyItemEffects(l, ctx, c, m, ci, characterId, itemId)
            return nil
        }
    }
}
```

Error semantics: `pg.Wait()` returns the first error among the three reads. The caller routes through `ConsumeError(..., err)` exactly as today. Concurrent execution does not change observable outcomes — if read A and read B both fail, the test environment may surface either error first; the existing tests should not assert which one wins (or should be updated to accept either).

### 6.3 `ConsumeTownScroll` Shape

```go
pg, _ := model.NewGroup(ctx)
fm := model.Submit(pg, func() (_map.Model, error) { return character2.NewProcessor(l, ctx).GetMap(characterId) })
fi := model.Submit(pg, func() (consumable3.Model, error) { return p.cdp.GetById(uint32(itemId)) })
if err := pg.Wait(); err != nil {
    return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
}
m, ci := fm.Get(), fi.Get()

// dependent reads remain sequential
toMapId := _map2.EmptyMapId
if val, ok := ci.GetSpec(consumable3.SpecTypeMoveTo); ok && val > 0 {
    toMapId = _map2.Id(val)
}
if toMapId == _map2.EmptyMapId {
    mm, err := _map3.NewProcessor(l, ctx).GetById(m.MapId())
    // …
}
```

### 6.4 `ConsumeSummoningSack` Shape

```go
pg, _ := model.NewGroup(ctx)
fc := model.Submit(pg, func() (character.Model, error) { return character.NewProcessor(l, ctx).GetById()(characterId) })
fi := model.Submit(pg, func() (consumable3.Model, error) { return consumable3.NewProcessor(l, ctx).GetById(uint32(itemId)) })
if err := pg.Wait(); err != nil {
    return NewProcessor(l, ctx).ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
}
c, ci := fc.Get(), fi.Get()

pos, err := position.NewProcessor(l, ctx).GetInMap(c.MapId(), c.X(), c.Y(), c.X(), c.Y())()
// …rest unchanged…
```

### 6.5 Code Comments for Sequential Variants

Add inline comments to `ConsumePetFood` (independent reads, deferred), `ConsumeCashPetFood` (dependent reads), and `ConsumeScroll` (dependent reads) explaining why the parallelization pattern is or is not applied. This documents the deliberate non-application so future readers don't "fix" it without thought.

### 6.6 Tenant / Trace Propagation

Each goroutine inside `model.Group` invokes a closure that captures `ctx` from the outer scope. The closure calls a processor whose constructor (e.g., `character2.NewProcessor(l, ctx)`) carries the tenant- and span-bearing context. No tenant-key extraction or context-recreation is added; the existing pattern is preserved.

### 6.7 Tests

- `consumable/processor_test.go`: extend or add `TestConsumeStandard_Success` and three error-path tests (one per read failing). Use mocks/fakes for `cp.GetById`, `character2.GetMap`, `cdp.GetById`. Asserts that `ConsumeError` is invoked with the failing read's error.
- `consumable/processor_test.go`: same shape for `ConsumeTownScroll` and `ConsumeSummoningSack` covering at least the success path and one read-failure path each.
- Concurrency proof: the `model.Group` package test in §3.5 already covers the timing invariant; per-call-site timing tests would be flaky and offer no extra signal.

---

## 7. Cross-Cutting Concerns

### 7.1 Multi-Tenancy

All concurrency added derives child contexts from the parent via `errgroup.WithContext`. The parent already carries the tenant via `tenant.MustFromContext`. No tenant boundary is crossed by the parallelization, and goroutines do not share mutable state beyond writing to their own `Future[T]`.

### 7.2 Observability

- New saga skip reason `nil_transaction_id` — countable in Loki. Appears once per nil-UUID event at debug level.
- Removed log lines: `Unable to announce character [%d] health to party members.` (two call sites). Loki queries for that string drop to zero after deploy.
- Tracing: the parallelized reads appear as overlapping siblings of the consume span. No new span enrichment — that is a separate observability task per PRD §2.

### 7.3 Backwards Compatibility

- No public REST or Kafka contract changes.
- `character.Processor` interface gains `PartyDecorator(Model) Model`. All implementers (the real `ProcessorImpl` and any test mocks) must add it. Mock updates are part of the test scope.
- `model.Group` is purely additive.

### 7.4 What Could Go Wrong

| Risk | Mitigation |
|---|---|
| `PartyDecorator` swallows a transient REST error and a downstream consumer assumes "no party" | The two known consumers in scope (`character` and `map` consumers) treat "not in party" as a no-op short-circuit. A transient error there manifests as a one-off missed HP-broadcast, which is the same behavior as today. Genuine outages of atlas-parties surface elsewhere. |
| Concurrent reads in `ConsumeStandard` change which error wins on multi-failure | The `ConsumeError` path is symmetric across the three reads (same inventory type, same slot, same transaction id). Tests must not assert error-source identity when multiple reads can fail simultaneously. Documented in §6.2. |
| `errgroup` becomes a transitive-only dep in `libs/atlas-model` and is hidden in tests | Add it as a direct require in `libs/atlas-model/go.mod`. Run `go mod tidy` in the lib. |
| Saga consumer for an event whose `TransactionId` is legitimately `uuid.Nil` (e.g., a future broadcast with no saga semantics) | This is exactly the desired behavior. By definition, a saga step is keyed by a transaction id; a nil id cannot match any saga. |
| Renaming or moving handlers breaks tests that reference internals | None of the planned changes rename existing handlers. `PartyDecorator` is additive on the interface; consumer-handler signatures are unchanged. |

---

## 8. Out-of-Scope (Reaffirmed from PRD)

- Header- or topic-level filtering of non-saga events at the Kafka consumer layer.
- Adding `partyId` to atlas-character's REST contract or storage.
- Caching party state in atlas-channel.
- Removing the GET at the start of `handleStatusEventStatChanged`.
- Parallelizing `ConsumeScroll`.
- Adding business-attribute span enrichment.
- Changing saga subscription topology or `acceptanceTable`.
- Frontend / atlas-ui changes.

---

## 9. Open Questions Resolved

The PRD's three open questions are now closed:

1. `Party()` return convention → **non-pointer `party.Model` with zero-value semantics** (§4.1).
2. Saga short-circuit placement → **single helper at the `AcceptEvent` boundary** (§5.1).
3. Whether to parallelize `ConsumeScroll` → **no, deferred** (§6.1).

---

## 10. Acceptance Criteria Mapping

| PRD criterion | Where addressed |
|---|---|
| 1. Solo HP-potion: zero "Unable to announce…" lines | §4.4, §4.5 |
| 2. Both announce paths use `cp.GetById(cp.PartyDecorator)(...)` | §4.4, §4.5, §4.6 |
| 3. `Party()` and `InParty()` with unit-test coverage | §4.2, §4.7 |
| 4. Zero `saga_not_found` for nil-UUID events; new `nil_transaction_id` reason | §5.2, §5.3, §5.5 |
| 5. `ConsumeStandard` reads concurrent; trace shows overlap | §6.2 |
| 6. Same parallelization for `ConsumeTownScroll`, `ConsumeSummoningSack`, comments elsewhere | §6.1, §6.3, §6.4, §6.5 |
| 7. p50 ≤ 200ms on root span | §2 (trace propagation), §6 (the structural change that delivers it) |
| 8. All affected services build and pass tests | §3.5, §4.7, §5.5, §6.7 |
| 9. No regression in HP application / item consumption | §6.2 (error semantics preserved) |
