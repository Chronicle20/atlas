# RPS Domain

## Responsibility

Runs a rock/paper/scissors minigame session for a character at an NPC: opening a session, adjudicating rounds against a server-drawn opponent throw, advancing a reward ladder rung by rung, and terminating the session (collected, quit, or disconnected) with any associated fee/prize payout.

## Core Models

### Model

Represents an active RPS session, keyed by tenant + character. Immutable; constructed via `Builder` (`NewBuilder`, `CloneBuilder`).

| Field | Type | Description |
|-------|------|--------------|
| tenant | tenant.Model | Owning tenant |
| characterId | uint32 | Player character id |
| worldId | world.Id | World id |
| channelId | channel.Id | Channel id |
| npcId | uint32 | NPC the session is opened at |
| rung | int | Current ladder rung (0 = fresh, no prize) |
| status | Status | Session lifecycle state |
| lastThrow | Throw | Most recent throw the player submitted |
| createdAt | time.Time | When the session was opened |
| updatedAt | time.Time | When the session was last modified |

`Builder.Build` requires a non-nil tenant id and a non-zero characterId; `updatedAt` is stamped on every `Build`.

### Status

Session lifecycle state: `OPEN` (session created, round not yet begun), `AWAITING_SELECT` (round open, waiting on the player's throw), `AWAITING_DECISION` (round won, waiting on the player to continue or collect), `ENDED` (terminal).

### Throw

A rock/paper/scissors selection: `ThrowRock` (0), `ThrowPaper` (1), `ThrowScissors` (2).

### Outcome

The result of an adjudicated round from the player's perspective: `OutcomeLose`, `OutcomeTie`, `OutcomeWin`.

### Ladder

The reward progression configured for a tenant.

| Field | Type | Description |
|-------|------|--------------|
| EntryCostMeso | uint32 | Meso charged to open (or retry) a session |
| ConsolationMeso | uint32 | Meso granted on a loss at rung 0 (0 disables the award) |
| Rungs | []Rung | Reward steps |

### Rung

A single step of the reward ladder. Rung indices are 1-based; rung 0 is reserved to mean "fresh, no prize" and never appears in a Ladder's `Rungs` slice.

| Field | Type | Description |
|-------|------|--------------|
| Rung | int | Rung number |
| ItemId | item.Id | Item awarded at this rung |
| Quantity | uint32 | Item quantity awarded |
| Meso | uint32 | Meso awarded at this rung |

## Invariants

- `Ladder.PrizeAt` resolves a prize by matching a rung's `Rung` field rather than by slice position; rung 0 and any rung beyond the ladder's configured rungs resolve to `ok=false`.
- `Ladder.MaxRung` returns the highest configured `Rung` value, or 0 if the ladder has no rungs. `Ladder.IsMax` is true only when the given rung equals `MaxRung` and is non-zero.
- `Adjudicate` is a pure function: equal throws tie; otherwise rock beats scissors, scissors beats paper, paper beats rock.
- `Start` disposes any stale session for the character (silent, no `GameEnded` event) before opening a fresh rung-0 `StatusOpen` session.
- `Begin` is only valid from `StatusOpen`.
- `Select` is only valid from `StatusOpen` or `StatusAwaitingSelect`.
- On a win, `Select` advances the rung, sets `StatusAwaitingDecision`, and resolves the prize at the new rung.
- On a tie, `Select` leaves the rung unchanged and returns to `StatusAwaitingSelect`.
- On a loss, `Select` sets `StatusEnded` but the session is kept in the registry (not removed); only a `RoundResult` event is buffered, not `GameEnded`. `GameEnded` is deferred to the player's subsequent `Collect` (via Exit) or `Retry`.
- `Continue` is only valid from `StatusAwaitingDecision`. If the session's rung is already the ladder's max rung, `Continue` forces a `Collect` instead of opening another round.
- `Retry` is only valid from `StatusEnded`. It re-charges `EntryCostMeso` (if positive) via a fee-deduction saga before reopening the session at rung 0 / `StatusAwaitingSelect`; a saga-submit failure blocks the restart and leaves the session `StatusEnded`.
- `submitConsolation` awards `ConsolationMeso` only when the session's rung is 0 (the player never won a round this game) and `ConsolationMeso` is non-zero; it is invoked from `Retry` and from `Collect` on a `StatusEnded` session, never at the moment of adjudication. A ladder-resolution or saga-submit failure in `submitConsolation` is logged and non-fatal.
- `Collect` behavior depends on the session's status found: from `StatusAwaitingDecision` it resolves the prize at the current rung, submits a payout saga for any non-zero prize component, removes the session, and buffers `GameEnded` with reason `collected`. From any other status (including a post-loss `StatusEnded`) it removes the session with no payout (buffering `GameEnded` with reason `quit`), awarding the deferred consolation first if the status was `StatusEnded`. If saga submission fails on the `StatusAwaitingDecision` path, the session is left in place so a retried `Collect` can attempt the payout again.
- `Quit` removes the session unconditionally and buffers `GameEnded` with reason `quit`, regardless of current status.
- `Dispose` is a no-op (no error, no event) if the session is already gone; otherwise it removes the session and buffers `GameEnded` with reason `disconnected`.
- `buildPayoutSaga` includes an `award_mesos` step only when the prize's meso is positive, and an `award_asset` step only when the prize's item id is non-zero and quantity is positive; if neither applies, no saga is built and none is submitted.
- A session is considered abandoned, and eligible for the sweep, after `defaultTTL` (5 minutes) of inactivity.
- A session swept by `SweepTask` is disposed with no payout: `GameEnded` with reason `disconnected` is emitted directly (no session remains in the registry for a `Dispose` call to find, and no payout saga is ever built for a sweep).

## State Transitions

### Session Lifecycle

1. **Start**: Disposes any stale session for the character, opens a rung-0 `StatusOpen` session, buffers `GameOpened` (carrying the ante from the resolved ladder's `EntryCostMeso`).
2. **Begin**: `StatusOpen` -> `StatusAwaitingSelect`; buffers `RoundStarted`.
3. **Select** (win): `StatusOpen`/`StatusAwaitingSelect` -> `StatusAwaitingDecision`, rung incremented; buffers `RoundResult`.
4. **Select** (tie): status unchanged (`StatusAwaitingSelect`), rung unchanged; buffers `RoundResult`.
5. **Select** (loss): -> `StatusEnded` (session kept); buffers `RoundResult` only.
6. **Continue**: `StatusAwaitingDecision` -> `StatusAwaitingSelect` (next round; buffers `RoundStarted`), or forces **Collect** if already at the ladder's max rung.
7. **Retry**: `StatusEnded` -> `StatusAwaitingSelect` at rung 0 (re-charges entry fee, awards deferred consolation); buffers `RoundStarted`.
8. **Collect**: `StatusAwaitingDecision` -> `StatusEnded` + session removed (prize paid); or any other status -> `StatusEnded` + session removed (no payout, deferred consolation awarded if the prior status was `StatusEnded`). Buffers `GameEnded`.
9. **Quit**: any status -> `StatusEnded` + session removed, no payout. Buffers `GameEnded`.
10. **Dispose**: any status -> `StatusEnded` + session removed, no payout (no-op if already gone). Buffers `GameEnded`.
11. **Sweep**: a session inactive past `defaultTTL` is popped from the registry and a `GameEnded` (reason `disconnected`) event is emitted directly, with no session mutation (the session is already gone).

## Processors

### Processor (game)

Interface defining RPS session operations. Each buffered `Method(mb, ...)` is a pure state transition that also buffers the events it produces onto a `message.Buffer`; the corresponding `MethodAndEmit(...)` wraps the buffered method so the buffered events are emitted atomically after a successful transition. Constructed via `NewProcessor` (unconfigured `LadderProvider`/`SagaSubmitter`, for callers that don't need them) or `NewProcessorWithLadder` (production/test wiring supplying a `ThrowSource`, `LadderProvider`, and `SagaSubmitter`).

**Queries:**
- `Get`: Returns the active session for a character together with the prize resolved at its current rung. Returns `ErrSessionNotFound` if no session exists.

**Commands:**
- `Start` / `StartAndEmit`: Opens a new session, disposing any stale prior one.
- `Begin` / `BeginAndEmit`: Opens the first round of an open session.
- `Select` / `SelectAndEmit`: Submits the player's throw, draws an opponent throw, adjudicates, and applies the resulting transition.
- `Continue` / `ContinueAndEmit`: Advances to the next round, or forces `Collect` at the ladder's max rung.
- `Retry` / `RetryAndEmit`: Restarts a lost game, re-charging the entry fee.
- `Collect` / `CollectAndEmit`: Ends the session, paying the resolved prize if applicable.
- `Quit` / `QuitAndEmit`: Ends the session with no payout.
- `Dispose` / `DisposeAndEmit`: Ends the session with no payout on disconnect; no-op if the session is already gone.

### LadderProvider

`func() (Ladder, error)`. Resolves the reward ladder for the processor's tenant. Injected rather than called directly to avoid an import cycle (`atlas-rps/configuration` imports `atlas-rps/game` for the `Ladder`/`Rung` types). Production wiring backs it with `configuration.NewProcessor(l, ctx).GetLadder(tenant.Id())`.

### SagaSubmitter

`func(s sharedsaga.Saga) error`. Submits a fully-built payout or fee-deduction saga to atlas-saga-orchestrator's command topic. Injected for the same reason as `LadderProvider` (avoiding coupling to the local `atlas-rps/saga` package). Production wiring backs it with `saga.NewProcessor(l, ctx).Create(s)`.

### ThrowSource

`func() Throw`. Produces a Throw, used as the opponent's throw. `DefaultThrowSource` draws uniformly at random via `math/rand`. Injectable for deterministic testing.

### Registry

Redis-backed TTL session registry (package `game`), keyed by (tenant, characterId), with a per-tenant tracking set so the sweep can fan out across tenants. `Put` stores a session and refreshes its TTL; `Get` retrieves the active session for a character; `Remove` deletes it; `PopExpired` returns and removes all sessions past `defaultTTL` (5 minutes) across every tracked tenant.

### SweepTask

Periodic task (registered by `main.go` at a 50ms interval) that pops every expired session across all tracked tenants and emits a `GameEnded` event with reason `disconnected` directly via the producer for each, with no payout saga. Implements the `routine.Task` interface (`Run`, `SleepTime`) structurally, without importing `atlas-routine`'s registration helper.

### configuration.Processor

Resolves a tenant's rps-rewards configuration (entry cost, consolation meso, reward ladder) from atlas-tenants and converts it (`Extract`) into a `game.Ladder`. A tenant with no configured record yields `ErrNoRewardConfig`. When extracting, rungs with a duplicate `Rung` number are skipped so the resulting ladder is dense and deduplicated.

### saga.Processor

Submits a fully-built `sharedsaga.Saga` to atlas-saga-orchestrator's command topic, keyed by the saga's transaction id. The composition-root-facing counterpart of `game.SagaSubmitter`.
