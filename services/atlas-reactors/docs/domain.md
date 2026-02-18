# Reactor Domain

## Responsibility

Manages reactor instances as in-memory volatile game objects. Reactors are interactive objects within maps that respond to player actions. This domain handles reactor lifecycle (creation, hit, destruction), state transitions, cooldown management, and item-reactor activation.

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

### ReactorKey

Composite key for reactor cooldown tracking.

| Field          | Type   | Description                    |
|----------------|--------|--------------------------------|
| Classification | uint32 | Reactor type/classification ID |
| X              | int16  | X coordinate position          |
| Y              | int16  | Y coordinate position          |

## Invariants

- Classification is required when building a reactor model
- Reactor IDs are assigned from a running counter starting at 1000000001
- Reactor IDs wrap around to 1000000001 if they exceed 2000000000
- A reactor cannot be created at the same location (classification, x, y) while on cooldown
- Cooldown is recorded when a reactor is destroyed, based on its delay value
- Cooldowns are cleared when a reactor is successfully created at that location
- Item reactor activation type is identified by state event type 100
- Reactors with state event type 100 or type 999 persist at their final state rather than being destroyed

## State Transitions

Reactors transition through states based on hits:

1. **Initial State**: Reactor starts at configured initial state
2. **Hit Processing**: When hit, the reactor checks its current state's events
3. **Next State**: If a matching event exists (skill match or no skill requirement), transition to nextState
4. **Terminal State**: If no valid next state exists, or the next state has no further transitions, the reactor triggers and is destroyed

A state is considered terminal when:
- No events defined for that state
- All events lead to states not defined in stateInfo

Reactors that contain state event type 100 (item reactor) or type 999 persist at their final state. When such a reactor reaches a terminal state, it triggers but is not destroyed.

## Processors

### GetById

Retrieves a reactor by its unique ID from the in-memory registry.

### GetInField

Retrieves all reactors in a specific field (world/channel/map/instance combination).

### Create

Creates a new reactor instance:
1. Checks cooldown status for the location
2. Retrieves reactor game data from atlas-data service
3. Sets reactor name from game data if not provided
4. Registers reactor in the in-memory registry
5. Clears any cooldown for that location
6. Emits CREATED status event

### Destroy

Destroys a reactor instance:
1. Cancels any pending item-reactor activation
2. Records cooldown based on reactor delay
3. Removes reactor from registry
4. Emits DESTROYED status event

### Hit

Processes a hit on a reactor:
1. Retrieves reactor from registry
2. Emits HIT command to atlas-reactor-actions
3. Evaluates state transitions based on current state and skill
4. If no state events exist or no matching transition, triggers and destroys
5. If next state is not in state info or is terminal:
   - If reactor persists at final state (type 100 or 999), updates state, triggers, and emits HIT status event without destroying
   - Otherwise, triggers and destroys
6. If next state has further transitions, updates state and emits HIT status event

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

Runs every 60 seconds to remove expired cooldown entries from the registry.
