# Domain

## character

### Responsibility

Orchestrates expiration checks across character inventory, account storage, and cash shop for a given character session.

### Core Models

No local domain models. Consumes REST models from external services:
- `inventory.RestModel`, `inventory.CompartmentRestModel`, `inventory.AssetRestModel`
- `storage.AssetRestModel`
- `cashshop.CompartmentRestModel`, `cashshop.ItemRestModel`

### Invariants

None. Service performs read-only checks and emits commands.

### Processors

#### CheckAndExpire

Checks all items for a character across inventory, storage, and cash shop. Emits expire commands for items past their expiration time.

Parameters: `characterId`, `accountId`, `worldId`

Delegates to three internal functions:
- `checkInventory`: Iterates all compartments and their assets for the character
- `checkStorage`: Iterates all storage assets for the account and world
- `checkCashshop`: Iterates all cash shop items across compartments for the account

Each emits a service-specific expire command via Kafka for every expired item found.

---

## expiration

### Responsibility

Provides pure utility functions for expiration time checks.

### Core Models

None.

### Invariants

- Zero time value means no expiration (item never expires)
- Item is expired if current time is after expiration time

### Processors

#### IsExpired

Returns true if expiration time is set and current time is after expiration.

#### HasExpiration

Returns true if expiration time is not zero.

---

## session

### Responsibility

Tracks online character sessions for periodic expiration checks.

### Core Models

#### Session

| Field | Type |
|-------|------|
| CharacterId | uint32 |
| AccountId | uint32 |
| Channel | channel.Model |
| TenantId | uuid.UUID |
| Region | string |
| MajorVersion | uint16 |
| MinorVersion | uint16 |

### Invariants

- Sessions are keyed by CharacterId
- Tracker is a singleton via `sync.Once`
- Thread-safe via `sync.RWMutex`

### Processors

#### Tracker.Add

Adds or updates a session in the tracker.

#### Tracker.Remove

Removes a session by character ID.

#### Tracker.GetAll

Returns a snapshot of all tracked sessions.

#### Tracker.Get

Returns a session by character ID.

#### Tracker.Count

Returns the number of tracked sessions.

---

## data

### Responsibility

Retrieves item replacement information from atlas-data based on template ID ranges.

### Core Models

#### ReplaceInfo

| Field | Type |
|-------|------|
| ReplaceItemId | uint32 |
| ReplaceMessage | string |

### Invariants

- Equipment: template IDs 1000000-1999999
- Consumables: template IDs 2000000-2999999
- Setup: template IDs 3000000-3999999
- Etc: template IDs 4000000-4999999
- Cash items and unknown ranges return empty ReplaceInfo

### Processors

#### GetReplaceInfo

Returns replacement item information for a given template ID. Routes to the appropriate atlas-data endpoint based on template ID range.

---

## task

### Responsibility

Runs periodic expiration checks at configurable intervals for all online sessions.

### Core Models

#### PeriodicTask

| Field | Type |
|-------|------|
| l | logrus.FieldLogger |
| interval | time.Duration |
| stopCh | chan struct{} |
| wg | *sync.WaitGroup |

### Invariants

- Default interval is 60 seconds if not configured or invalid
- Iterates all sessions from the Tracker on each tick
- Reconstructs tenant context per session for Kafka header propagation

### Processors

#### Start

Launches the periodic check goroutine.

#### Stop

Signals the goroutine to stop and waits for completion.
