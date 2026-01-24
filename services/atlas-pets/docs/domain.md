# Pet Domain

## Responsibility

Manages pet lifecycle, attributes, and state within the game. Pets are companion entities owned by characters that can be spawned, despawned, and have attributes modified over time.

## Core Models

### Pet Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Unique pet identifier |
| cashId | uint64 | Cash shop identifier |
| templateId | uint32 | Pet template reference |
| name | string | Pet name |
| level | byte | Pet level (1-30) |
| closeness | uint16 | Pet closeness value (0-30000) |
| fullness | byte | Pet fullness value (0-100) |
| expiration | time.Time | Pet expiration timestamp |
| ownerId | uint32 | Owning character identifier |
| slot | int8 | Spawn slot (-1 = not spawned, 0-2 = spawned) |
| excludes | []Exclude | Items excluded from pet auto-loot |
| flag | uint16 | Pet flags |
| purchaseBy | uint32 | Character who purchased the pet |

### Exclude Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Unique exclude identifier |
| itemId | uint32 | Excluded item identifier |

### Temporal Data

In-memory tracking for pet position and stance.

| Field | Type | Description |
|-------|------|-------------|
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| stance | byte | Pet stance |
| fh | int16 | Foothold identifier |

## Invariants

- `templateId` is required
- `ownerId` is required
- `name` is required
- `level` must be between 1 and 30
- `fullness` must be between 0 and 100
- `slot` must be -1 (despawned) or between 0 and 2 (spawned)
- `itemId` is required for excludes

## State Transitions

### Spawn States

| From | To | Condition |
|------|----|-----------|
| slot = -1 | slot = 0-2 | Spawn command received, owner has fewer than 3 spawned pets |
| slot = 0-2 | slot = -1 | Despawn command received or fullness <= 5 |

### Multi-Pet Spawning

- Maximum 3 pets can be spawned simultaneously (slots 0, 1, 2)
- Spawning more than 1 pet requires the multi-pet skill
- When a pet spawns as lead (slot 0), existing spawned pets shift to higher slots
- When a pet despawns, remaining pets shift to lower slots

### Hunger Mechanics

- Fullness decreases over time based on pet template hunger value
- When fullness reaches 5 or below, the pet is automatically despawned

### Closeness and Leveling

- Closeness is awarded through commands and interactions
- Level increases when closeness reaches experience thresholds
- Experience thresholds: 1, 1, 3, 6, 14, 31, 60, 108, 181, 287, 434, 632, 891, 1224, 1642, 2161, 2793, 3557, 4467, 5542, 6801, 8263, 9950, 11882, 14084, 16578, 19391, 22547, 26074, 30000

## Processors

### Pet Processor

| Method | Description |
|--------|-------------|
| GetById | Retrieves a pet by identifier |
| GetByOwner | Retrieves all pets for an owner |
| Create | Creates a new pet |
| Delete | Deletes a pet |
| DeleteForCharacter | Deletes all pets for a character |
| DeleteOnRemove | Deletes a pet when the inventory item is removed |
| Move | Updates pet position in temporal registry |
| Spawn | Spawns a pet to an active slot |
| Despawn | Despawns a pet to inactive state |
| AttemptCommand | Executes a pet command (trick) |
| EvaluateHunger | Evaluates and decreases pet fullness |
| ClearPositions | Clears temporal position data for owner's pets |
| AwardCloseness | Awards closeness to a pet |
| AwardFullness | Awards fullness to a pet |
| AwardLevel | Awards levels to a pet |
| SetExclude | Sets excluded items for pet auto-loot |

### Temporal Registry

| Method | Description |
|--------|-------------|
| Update | Updates position, stance, and foothold for a pet |
| UpdatePosition | Updates position and foothold for a pet |
| UpdateStance | Updates stance for a pet |
| GetById | Retrieves temporal data for a pet |
| Remove | Removes temporal data for a pet |

### Hunger Task

Background task that runs at a configured interval to evaluate pet hunger for all logged-in characters.
