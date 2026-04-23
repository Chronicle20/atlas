# Design — Reactor Type Semantics & Timer-Driven Progression

## Scope

Two coordinated changes across `atlas-data` and `atlas-reactors`:

1. **Data correctness.** `atlas-data` stops fabricating events that don't exist in the .wz and honors both `timeOut` / `timeout` casings.
2. **Decision rules.** `atlas-reactors` makes persist-vs-destroy a property of the *current state's type*, not of any event anywhere, and fires timers for states that have `timeOut` set.

## atlas-data changes (`services/atlas-data/atlas.com/data/reactor/`)

### `reader.go`

Replace the synthesis at lines 124-130:

```go
} else {
    m.StateInfo[i] = []ReactorStateRestModel{{
        Type:      999,
        NextState: i + 1,
    }}
    m.TimeoutInfo[i] = -1
}
```

with: do nothing. A state with no `event` subtree is not in `StateInfo`. `TimeoutInfo` also gets no entry for it.

Fix the timeout read at line 89:

```go
timeout := ed.GetIntegerWithDefault("timeout", -1)
```

to honor both casings, preferring `timeOut` (the 157-case majority):

```go
timeout := ed.GetIntegerWithDefault("timeOut", -1)
if timeout == -1 {
    timeout = ed.GetIntegerWithDefault("timeout", -1)
}
```

### `rest.go`

Add per-state `timeoutNextState` so reactors can know where a timer fire should transition to:

```go
type RestModel struct {
    ...
    TimeoutInfo          map[int8]int32 `json:"timeoutInfo"`
    TimeoutNextStateInfo map[int8]int8  `json:"timeoutNextStateInfo"`  // new
}
```

In the reader, populate `TimeoutNextStateInfo[i]` from the type-101 event's `state` field when one is present, else do not populate (absence = "no timer transition").

### Tests

- `reader_test.go`: for an `.img.xml` fixture with an empty terminal state, assert that state is absent from `StateInfo` / `TimeoutInfo`.
- For an .img.xml with `timeOut` (mixed case), assert `TimeoutInfo[state]` is populated.
- For an .img.xml with a type-101 event + `timeOut`, assert `TimeoutNextStateInfo[state]` equals the 101 event's `state` field.

## atlas-reactors changes (`services/atlas-reactors/atlas.com/reactors/reactor/`)

### Data model (`reactor/data/`)

Add `TimeoutNextState(state int8) (int8, bool)` to `data.Model` alongside the existing `Timeout(state)` accessor. The bool indicates "has a timer transition."

### `processor.go`

**Replace `persistsAtFinalState` (lines 259-270).** The rule becomes state-local: check the type of the event that led to this transition.

```go
// persistsAtEndState returns true if a reactor that has just transitioned
// via an event of the given type should remain alive rather than being
// destroyed. Based on the wz reactor type taxonomy:
//   100       = item-drop reactors (e.g. moonflowers)
//   5, 6, 7   = GPQ skill-gated reactors
//   101       = timer-driven cyclic reactors
// All other types (0, 1, 2) are breakable hit reactors and destroy on end.
func persistsAtEndState(eventType int32) bool {
    switch eventType {
    case 100, 101, 5, 6, 7:
        return true
    default:
        return false
    }
}
```

**Rewire `Hit` (lines 150-224)** to pass the matched event's type (the one whose nextState was chosen) into `persistsAtEndState`. The two existing call sites become:

```go
// case 1: nextState not in stateInfo
if !hasNextState {
    if persistsAtEndState(matchedEvent.Type()) {
        // persist — same body as today
    }
    return TriggerAndDestroy(...)
}

// case 2: we transitioned, but the destination state is terminal
if isTerminalState(stateInfo, nextState) {
    if persistsAtEndState(matchedEvent.Type()) {
        // persist
    }
    return TriggerAndDestroy(...)
}
```

Update `isTerminalState` only if needed — with the synthesis removed, a terminal state is now one that either (a) has no entry in `stateInfo`, or (b) all its events lead to states not in `stateInfo`. Current logic covers (b); (a) is covered by the `!hasNextState` branch.

### Timer mechanism (new file: `reactor/timer.go`)

Modeled on the existing `scheduleItemReactorActivation` / `pendingActivations` pattern in `item_reactor.go`. Pure-process timers (no Redis), safe because:

- The timer callback re-reads the reactor from Redis first; if it's gone, skip.
- A process crash loses pending timers on that process; any surviving replica's timers proceed. For cross-replica ownership, whichever replica handled the state-entry event owns the timer. This is not strictly load-balanced but is correct and simple.

Public API:

```go
func scheduleStateTimeout(l logrus.FieldLogger, ctx context.Context, r Model)
func cancelStateTimeout(reactorId uint32)
func cancelAllStateTimeouts()
```

`scheduleStateTimeout` checks `r.Data().Timeout(r.State())`; if `> 0` and `r.Data().TimeoutNextState(r.State())` exists, it arms a `time.AfterFunc` timer. On fire, it re-fetches the reactor, compares current state (bail if it changed), and issues an internal state transition equivalent to the nextState branch of `Hit`:

```go
updated := GetRegistry().Update(..., SetState(nextState))
// run Trigger so scripts can react
Trigger(l)(ctx)(updated, 0)  // characterId=0 for timer-driven
// re-arm if the new state also has a timeout
scheduleStateTimeout(l, ctx, updated)
// or if new state is terminal:
//   persistsAtEndState(101) → stay alive (101 events always persist)
//   else → Destroy(...)
```

Call sites for `scheduleStateTimeout`:

- `Create` — after successful creation, if state 0 has a timeout, arm it.
- `Hit` — after a state transition, arm for the new state.

Call sites for `cancelStateTimeout`:

- `Destroy` — when a reactor is destroyed (any code path).
- Start of `Hit` — cancel before evaluating, since a hit interrupts the timer.

Call sites for `cancelAllStateTimeouts`:

- `Teardown` alongside `CancelAllPendingActivations`.

### Tests

New cases in `processor_test.go`:

- Hit on reactor with `StateInfo = {0: [{Type:0, NextState:1}], 1: []}` (nothing synthesized) — terminal on state 1, destroys.
- Hit on reactor with type-100 event leading to terminal — persists.
- Hit on reactor with type-5 event leading to terminal — persists.

New file `timer_test.go`:

- Timer fires after configured ms and transitions state.
- Hit cancels pending timer.
- Destroy cancels pending timer.
- Re-arm after timer fire if new state also has a timeout.
- `cancelAllStateTimeouts` during teardown does not panic.

### `docs/domain.md` update

Rewrite the "State Transitions" section to reflect the new taxonomy (table of types, new persist rule, timer mechanism). Remove the old type-999 paragraph entirely.

## Risks & edge cases

1. **Existing reactor instances in Redis during a rolling deploy.** An already-created reactor in Redis predates the code change; its `data` field was populated from atlas-data at creation time. If atlas-data now returns different `stateInfo` shape (no synthesized 999), a reactor created on the old code could still hit the new code path. Mitigation: the new code gracefully handles missing state entries (`!hasNextState` already exists). Rolling deploy is safe; full test coverage on both shapes.
2. **Cross-replica timer loss.** Addressed above. Documented as known trade-off.
3. **Type 101 infinite loop.** A type-101 reactor that loops back to a state it already visited will trigger timers indefinitely. This is the desired behavior (Balrog altars cycle forever while the instance is alive). The `Teardown` / `DestroyInField` paths cancel all timers when the instance ends.
4. **Moon Bunny (9101000) with the synthesis removed.** After the change, Moon Bunny has **zero** entries in `StateInfo` (because every state in its .wz has no `event` subtree) and no `timeoutInfo` either (because its .wz has no `timeOut` that I observed in the snippet I read). This means Moon Bunny will stay at state 0 forever. This is unchanged from today's broken behavior for this reactor, and a proper fix requires richer timer data or an explicit script hook. Flag in domain.md; not a regression.

## File-by-file change summary

| file                                                           | change                                       |
|----------------------------------------------------------------|----------------------------------------------|
| `services/atlas-data/atlas.com/data/reactor/reader.go`         | remove synthesis; fix `timeOut` casing; read `timeoutNextState` |
| `services/atlas-data/atlas.com/data/reactor/rest.go`           | add `TimeoutNextStateInfo` field             |
| `services/atlas-data/atlas.com/data/reactor/reader_test.go`    | new cases                                    |
| `services/atlas-reactors/atlas.com/reactors/reactor/data/model.go` | add `TimeoutNextState(state)` accessor   |
| `services/atlas-reactors/atlas.com/reactors/reactor/data/rest.go`  | parse `timeoutNextStateInfo`             |
| `services/atlas-reactors/atlas.com/reactors/reactor/processor.go` | rewrite `persistsAtFinalState` → `persistsAtEndState(type)`; pass matched-event type to both call sites |
| `services/atlas-reactors/atlas.com/reactors/reactor/timer.go`  | new — state-timer mechanism                   |
| `services/atlas-reactors/atlas.com/reactors/reactor/timer_test.go` | new                                      |
| `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go` | new cases                            |
| `services/atlas-reactors/docs/domain.md`                       | rewrite state-transition section              |

## Phasing for the TDD plan

Broadly:

1. atlas-data changes (smallest blast radius, no downstream behavior change yet because atlas-reactors still uses old rule).
2. Update atlas-reactors `persistsAtEndState` to take an event type; wire into `Hit`; fix tests.
3. Add timer mechanism; wire into `Create`, `Hit`, `Destroy`, `Teardown`.
4. Update domain.md and verify end-to-end with a manual playtest in a test map with reactor 2001 instances.
