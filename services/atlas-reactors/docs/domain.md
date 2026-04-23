# Reactor Domain

## Responsibility

Manages reactor instances as volatile game objects backed by Redis. Reactors are interactive objects within maps that respond to player actions. This domain handles reactor lifecycle (creation, hit, destruction), state transitions, cooldown management, and item-reactor activation.

## Core Models

### Model

Represents a reactor instance.

| Field          | Type         | Description                            |
|----------------|--------------|----------------------------------------|
| tenant         | tenant.Model | Tenant context                         |
| id             | uint32       | Unique reactor instance ID             |
| worldId        | byte         | World identifier                       |
| channelId      | byte         | Channel identifier                     |
| mapId          | uint32       | Map identifier                         |
| instance       | uuid.UUID    | Map instance identifier                |
| classification | uint32       | Reactor type/classification ID         |
| name           | string       | Reactor name                           |
| data           | data.Model   | Reactor game data (state info)         |
| state          | int8         | Current reactor state                  |
| eventState     | byte         | Event state                            |
| delay          | uint32       | Respawn delay in milliseconds          |
| direction      | byte         | Facing direction                       |
| x              | int16        | X coordinate position                  |
| y              | int16        | Y coordinate position                  |
| updateTime     | time.Time    | Last update timestamp                  |

### data.Model

Represents reactor game data retrieved from atlas-data service.

| Field       | Type                      | Description                          |
|-------------|---------------------------|--------------------------------------|
| name        | string                    | Reactor name from game data          |
| tl          | point.Model               | Top-left bounding point              |
| br          | point.Model               | Bottom-right bounding point          |
| stateInfo   | map[int8][]state.Model    | State transition definitions         |
| timeoutInfo | map[int8]int32            | Timeout per state                    |

### state.Model

Represents a state transition event.

| Field        | Type        | Description                          |
|--------------|-------------|--------------------------------------|
| theType      | int32       | Event type                           |
| reactorItem  | *item.Model | Associated item (optional)           |
| activeSkills | []uint32    | Skills that can trigger transition   |
| nextState    | int8        | State to transition to               |

### item.Model

Represents an item associated with a reactor state event.

| Field    | Type   | Description    |
|----------|--------|----------------|
| itemId   | uint32 | Item ID        |
| quantity | uint16 | Item quantity  |

### point.Model

Represents a 2D coordinate.

| Field | Type  | Description    |
|-------|-------|----------------|
| x     | int16 | X coordinate   |
| y     | int16 | Y coordinate   |

### MapKey

Composite key for map-scoped operations.

| Field     | Type      | Description          |
|-----------|-----------|----------------------|
| worldId   | byte      | World identifier     |
| channelId | byte      | Channel identifier   |
| mapId     | uint32    | Map identifier       |
| instance  | uuid.UUID | Instance identifier  |

## Invariants

- Classification is required when building a reactor model
- Reactor IDs are assigned via atomic Redis increment starting at 1000000001
- Reactor IDs wrap around to 1000000001 if they exceed 2000000000
- A reactor cannot be created at the same location (classification, x, y) while on cooldown
- Cooldown is recorded when a reactor is destroyed, based on its delay value
- Cooldowns expire automatically via Redis TTL
- Cooldowns are cleared when a reactor is successfully created at that location
- Item reactor activation type is identified by state event type 100
- Reactors with state event type 100 persist at their final state rather than being destroyed
- State event type 101 (timer-driven) reactors persist at their final state (including terminal states)

## State Transitions

Reactors advance through states via two mechanisms:

1. **Hit** — a player attack triggers an event on the current state. The matched event's `nextState` becomes the reactor's new state. A hit can deliver a skill id; if a state's event lists `activeSkills`, only a matching skill id (or an event with empty `activeSkills`) is eligible.
2. **State timeout** — if a state has a `timeout` set and a paired `timeoutNextState`, an in-process `time.AfterFunc` timer advances the reactor automatically. Timers are cancelled on hit, destroy, or teardown.

### Event-type taxonomy

Each event carries a `type` field that determines what happens on a terminal transition:

| type    | meaning                               | end-state behavior   |
|---------|---------------------------------------|----------------------|
| 0       | hit by any attack (default breakable) | destroy + cooldown   |
| 1, 2    | directional hit                       | destroy + cooldown   |
| 5, 6, 7 | GPQ skill-gated                       | **persist**          |
| 100     | item-drop trigger                     | **persist**          |
| 101     | timer-driven cyclic                   | **persist (cyclic)** |

The persist-vs-destroy decision is **state-local**: it is based on the type of the event that led to the terminal transition, not on whether a persist-type event appears anywhere in the reactor's state machine.

### Timer-driven reactors

State-timeout timers are armed whenever a reactor enters a state with `timeout > 0` and a configured `timeoutNextState`. Arming happens in:

- `Create` (for the reactor's initial state).
- `Hit` (for the new state after a transition that keeps the reactor alive).
- The timer callback itself (re-arming for the new state after a fire).

Cancellation happens in:

- `Hit` (on function entry — a hit interrupts the timer).
- `Destroy` / `DestroyInField` (any code path removing the reactor).
- `Teardown` (`cancelAllStateTimeouts` alongside `CancelAllPendingActivations`).

When a timer fires, the callback re-reads the reactor from the registry, bails if it no longer exists or its state has changed since arming, transitions to the configured `nextState`, **re-arms the timer for the new state if applicable**, then emits a TRIGGER and a HIT-status event. Re-arm precedes emit so that a slow or unreachable Kafka cannot stall the chained timer sequence. Type-101 is always a persist type, so the reactor stays alive even at terminal states.

### Notes & caveats

- Timers are process-local. A replica that owns the timer at arming time owns the fire. Process crashes lose pending timers on that process; other replicas' timers proceed independently. This is not load-balanced, but it is simple and correct.
- Reactors whose `.wz` defines states with no `event` subtree are represented with no entry for that state in `StateInfo` (atlas-data no longer synthesises placeholders). A hit on such a state flows through the "no state events" branch and destroys the reactor unless the matched event type is a persist type — which, in this branch, it cannot be, because there was no matched event.
- Moon Bunny (`9101000`) currently has neither events nor a `timeOut` in its `.wz` snippet and so will remain at state 0 until teardown. A proper fix requires richer per-state data or an explicit script hook and is tracked separately.

## Processors

### GetById

Retrieves a reactor by its unique ID from the registry.

### GetInField

Retrieves all reactors in a specific field (world/channel/map/instance combination).

### Create

Creates a new reactor instance:
1. Checks cooldown status for the location
2. Retrieves reactor game data from atlas-data service
3. Sets reactor name from game data if not provided
4. Registers reactor in the registry
5. Clears any cooldown for that location
6. Arms state-timeout timer for the initial state if applicable
7. Emits CREATED status event

### Destroy

Destroys a reactor instance:
1. Cancels any pending item-reactor activation
2. Records cooldown based on reactor delay
3. Removes reactor from registry
4. Emits DESTROYED status event

### Hit

Processes a hit on a reactor:
1. Cancels any pending state-timeout timer
2. Retrieves reactor from registry
3. Emits HIT command to atlas-reactor-actions
4. Evaluates state transitions based on current state and skill
5. If no state events exist or no matching transition, destroys if event type is non-persist; otherwise updates state
6. If next state is not in state info or is terminal:
   - If the triggering event type is a persist type (100, 101, 5, 6, 7), updates state, triggers, and emits HIT status event without destroying
   - Otherwise, triggers and destroys
7. If next state has further transitions, updates state, arms state-timeout timer if applicable, and emits HIT status event

### Trigger

Emits a TRIGGER command to atlas-reactor-actions without destroying the reactor.

### TriggerAndDestroy

Triggers reactor script execution and destroys the reactor:
1. Emits TRIGGER command to atlas-reactor-actions
2. Calls Destroy processor

### DestroyInField

Destroys all reactors in a specific field (world/channel/map/instance combination):
1. Retrieves all reactors in the field
2. For each reactor: cancels pending item-reactor activation, removes from registry, emits DESTROYED status event
3. Clears all cooldowns for that map/instance

### DestroyAll

Destroys all reactors across all tenants. Used during service shutdown.

### ActivateItemReactors

Handles item-drop-triggered reactor activation:
1. Finds reactors in the same field as the dropped item
2. Matches reactors with item-type state events (type 100) by item ID, quantity, and position within reactor bounding area
3. Schedules a delayed activation (configurable via ITEM_REACTOR_ACTIVATION_DELAY_MS, default 5000ms)
4. On activation, emits a CONSUME command for the drop, then hits the reactor

### Teardown

Cancels all pending item-reactor activations and destroys all reactors during service shutdown.

### data.GetById

Retrieves reactor game data from atlas-data service by classification ID.

## Background Tasks

### CooldownCleanup

Runs every 60 seconds. No-op since Redis TTL handles cooldown expiration automatically.
