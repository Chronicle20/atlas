# Domain

## Character

### Responsibility
Manages player character state including identity, stats, appearance, position, and progression.

### Core Models

#### Model
Immutable representation of a character.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| accountId | uint32 | Associated account |
| worldId | world.Id | World assignment |
| name | string | Character name |
| level | byte | Current level |
| experience | uint32 | Current experience |
| gachaponExperience | uint32 | Gachapon experience |
| strength | uint16 | STR stat |
| dexterity | uint16 | DEX stat |
| intelligence | uint16 | INT stat |
| luck | uint16 | LUK stat |
| hp | uint16 | Current HP |
| mp | uint16 | Current MP |
| maxHp | uint16 | Maximum HP |
| maxMp | uint16 | Maximum MP |
| meso | uint32 | Currency |
| hpMpUsed | int | AP spent on HP/MP |
| jobId | job.Id | Current job |
| skinColor | byte | Skin color |
| gender | byte | Gender (0=male, 1=female) |
| fame | int16 | Fame points |
| hair | uint32 | Hair style ID |
| face | uint32 | Face ID |
| ap | uint16 | Available AP |
| sp | string | Available SP (comma-separated) |
| spawnPoint | uint32 | Spawn point ID |
| gm | int | GM level |
| skills | []skill.Model | Character skills |

Map assignment (mapId, instance) is not part of this Model; atlas-maps owns character location state.

#### Temporal Data
Transient position/stance data, held in a Redis-backed registry keyed by character ID (not persisted with the character record).

| Field | Type | Description |
|-------|------|-------------|
| x | int16 | X position |
| y | int16 | Y position |
| fh | int16 | Foothold |
| stance | byte | Current stance |

#### Builder
Creates new character models with configurable stat allocation.

| Configuration | Type | Description |
|---------------|------|-------------|
| useStarting4AP | bool | Enable 4 starting AP mode |
| useAutoAssignStartersAP | bool | Auto-assign starter stats |
| defaultInventoryCapacity | uint32 | Default inventory size |

#### ExperienceModel
Represents experience distribution.

| Field | Type | Description |
|-------|------|-------------|
| experienceType | string | Experience source type |
| amount | uint32 | Experience amount |
| attr1 | uint32 | Additional attribute |

#### Distribution
Represents AP distribution request.

| Field | Type | Description |
|-------|------|-------------|
| Ability | string | Target stat |
| Amount | int8 | Points to distribute |

### Invariants
- Character name must match anchored pattern `^[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}$` (length and character set apply to the whole name)
- Character name must be unique within tenant
- Level must be between 1 and 200
- Gender must be 0 or 1
- Skin color must be between 0 and 9
- Hair ID must be between 30000 and 35000
- Face ID must be between 20000 and 25000
- GM level must be non-negative
- HP cannot exceed maxHp
- MP cannot exceed maxMp
- Meso cannot overflow uint32
- AP rebalance target floors must be at least 4 (the base primary stat value)
- AP transfer (point reset) validates source and target against per-job primary stat caps (4-32767) and HP/MP pool caps (30000) before applying

### Processors

#### Processor
Handles character operations.

| Operation | Description |
|-----------|-------------|
| GetById | Retrieve character by ID |
| GetForAccountInWorld | Retrieve characters for account in world |
| GetForName | Retrieve characters by name |
| GetAll | Retrieve all characters |
| IsValidName | Validate character name |
| CheckNameValidity | Validate character name and uniqueness within a world, returning a reason/detail |
| Create | Create new character |
| Delete | Delete character |
| DeleteByAccountId | Delete all characters for an account |
| DeleteForSagaCompensation | Delete character as a saga compensation step (idempotent on missing rows) |
| Login | Process character login |
| Logout | Process character logout |
| ChangeJob | Change character job |
| ChangeHair | Change hair style |
| ChangeFace | Change face |
| ChangeSkin | Change skin color |
| AwardExperience | Award experience |
| AwardLevel | Award levels |
| Move | Update position |
| RequestChangeMeso | Process meso change |
| AwardPickedUpMeso | Credit a picked-up meso share to a character and, for the picker, emit the pick-up completion command |
| RequestDropMeso | Process meso drop |
| RequestChangeFame | Process fame change |
| RequestDistributeAp | Process AP distribution |
| RequestDistributeSp | Process SP distribution |
| ChangeHP | Modify HP |
| SetHP | Set HP to specific value |
| ClampHP | Clamp HP to max value |
| ChangeMP | Modify MP |
| ClampMP | Clamp MP to max value |
| DeductExperience | Deduct experience |
| ResetStats | Reset character stats |
| RebalanceAP | Reclaim AP spent on primary stats above base and reassign to target floors |
| TransferAP | Transfer one already-spent AP point between a stat/pool source and target (AP Reset) |
| ProcessLevelChange | Apply level-up bonuses |
| ProcessJobChange | Apply job-change bonuses |
| Update | Update character properties |

---

## Skill

### Responsibility
Manages character skill data retrieval and modification requests.

### Core Models

#### Model
Immutable representation of a character skill.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Skill identifier |
| level | byte | Current skill level |
| masterLevel | byte | Master level |
| expiration | time.Time | Skill expiration |

### Processors

#### Processor
Handles skill operations.

| Operation | Description |
|-----------|-------------|
| GetByCharacterId | Retrieve skills for character |
| RequestCreate | Request skill creation |
| RequestUpdate | Request skill update |

---

## Drop

### Responsibility
Coordinates drop creation and pickup requests with external drop service.

### Processors

#### Processor
Handles drop coordination.

| Operation | Description |
|-----------|-------------|
| CreateForMesos | Request meso drop creation |
| RequestPickUp | Request drop pickup |
| CancelReservation | Cancel drop reservation |

---

## Data Portal

### Responsibility
Retrieves portal position data from the external data service by map ID and portal ID.

### Core Models

#### Model
Immutable representation of a map portal.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Portal identifier |
| name | string | Portal name |
| target | string | Target portal name |
| portalType | uint8 | Portal type |
| x | int16 | X position |
| y | int16 | Y position |
| targetMapId | map.Id | Target map |
| scriptName | string | Script name |

### Processors

#### Processor
Handles portal data retrieval from external data service.

| Operation | Description |
|-----------|-------------|
| GetInMapById | Retrieve portal by map and portal ID |

---

## Data Skill

### Responsibility
Retrieves skill definition and effect data from external data service for stat calculations.

### Core Models

#### Model
Immutable representation of a skill definition.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Skill identifier |
| action | bool | Has action |
| element | string | Element type |
| animationTime | uint32 | Animation time |
| effects | []effect.Model | Skill effects by level |

#### Effect Model
Immutable representation of a skill effect at a given level.

Selected fields used by this service:

| Field | Type | Description |
|-------|------|-------------|
| x | int16 | X value (used for MP bonus) |
| y | int16 | Y value (used for HP bonus) |

### Processors

#### Processor
Handles skill data retrieval from external data service.

| Operation | Description |
|-----------|-------------|
| GetById | Retrieve skill definition by ID |
| GetEffect | Retrieve skill effect for a given skill and level |

---

## Session History

### Responsibility
Records and queries character session history for playtime tracking.

### Core Models

#### Model
Immutable representation of a session history entry.

| Field | Type | Description |
|-------|------|-------------|
| id | uint64 | History entry identifier |
| characterId | uint32 | Character |
| worldId | world.Id | World |
| channelId | channel.Id | Channel |
| loginTime | time.Time | Login timestamp |
| logoutTime | *time.Time | Logout timestamp (null if active) |

### Invariants
- A character may have at most one active session (logoutTime is null)
- loginTime is always set on creation
- logoutTime is set when the session ends

### Processors

#### Processor
Handles session history operations.

| Operation | Description |
|-----------|-------------|
| StartSession | Create new session record |
| EndSession | Close active session |
| GetActiveSession | Get current active session |
| GetSessionsSince | Get sessions since timestamp |
| GetSessionsInRange | Get sessions in time range |
| ComputePlaytimeSince | Compute total playtime since timestamp |
| ComputePlaytimeInRange | Compute total playtime in range |

---

## Saved Location

### Responsibility
Manages saved map locations per character by location type.

### Core Models

#### Model
Immutable representation of a saved location.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Location identifier |
| characterId | uint32 | Character |
| locationType | string | Type of saved location |
| mapId | map.Id | Map |
| portalId | uint32 | Portal |

### Invariants
- A character may have at most one saved location per location type

### Processors

#### Processor
Handles saved location operations.

| Operation | Description |
|-----------|-------------|
| Put | Upsert saved location |
| Get | Get saved location by character and type |
| Delete | Delete saved location by character and type |

---

## Session

### Responsibility
Tracks character login state for session management.

### State Transitions

| From State | To State | Trigger |
|------------|----------|---------|
| (none) | LoggedIn | Session created (issuer: CHANNEL) |
| LoggedIn | Transition | Session destroyed |
| Transition | LoggedIn | Session created (channel change) |
| Transition | LoggedOut | Timeout |

---

## Pending Change

### Responsibility
Manages cash-shop-initiated character name-change and world-transfer requests: eligibility gating, request lifecycle, coupon consumption/refund, and expiry.

### Core Models

#### Model
Immutable representation of a pending change request.

| Field | Type | Description |
|-------|------|--------------|
| id | uuid.UUID | Request identifier |
| characterId | uint32 | Character |
| changeType | string | NAME_CHANGE or WORLD_TRANSFER |
| status | string | PENDING, APPLIED, CANCELLED, REJECTED, or EXPIRED |
| requestedName | string | Requested name (NAME_CHANGE only) |
| destinationWorldId | world.Id | Destination world (WORLD_TRANSFER only) |
| sourceWorldId | world.Id | World the character occupied at request time |
| assetId | uint32 | Coupon item template ID consumed at acceptance (item purchase path only) |
| hasAsset | bool | Whether assetId is set |
| reason | string | Rejection/expiry/cancel reason |
| transactionId | uuid.UUID | Correlation ID for the originating request |
| createdAt | time.Time | Creation timestamp |
| expiresAt | time.Time | Deadline after which the sweep expires the request |
| resolvedAt | *time.Time | Terminal-transition timestamp (null while PENDING) |
| notifiedAt | *time.Time | Timestamp the resolution notification was last emitted (null if unnotified) |

### Invariants
- changeType is one of NAME_CHANGE, WORLD_TRANSFER
- status is one of PENDING, APPLIED, CANCELLED, REJECTED, EXPIRED
- A character may have at most one PENDING request per changeType
- A PENDING NAME_CHANGE reserves its requested name tenant-wide (case-insensitive) until it leaves PENDING
- A transition out of PENDING is one-way; a terminal record is never reopened
- expiresAt defaults to the tenant's configured pending-expiry (see Configuration domain), 168h fallback
- A WORLD_TRANSFER request is admitted only if every eligibility gate passes, evaluated in order: destination world differs from the current world; character is not GM; destination world exists and is not full; the account has a free character slot in the destination world; the requested name is available; the account is not banned; the character is not a guild master; the character is not in a family; the character has no open trade; the character has no open hired merchant; the character holds no live MTS listings or bids; the character has no parcel in flight. A gate whose remote dependency fails is reported as check_unavailable rather than the gate's own reason
- A dependency-error outcome from any gate fails closed (the request is rejected)

### Processors

#### Processor
Handles pending-change lifecycle operations.

| Operation | Description |
|-----------|-------------|
| Create / CreateAndEmit | Validate eligibility and create a PENDING request |
| Resolve / ResolveAndEmit | Transition a PENDING request to a terminal status, exactly once |
| CancelForCharacterAndType | Cancel the calling character's own PENDING request of a given type |
| ApplyForCharacter | Apply every PENDING request the character holds (driven by LOGOUT) |
| RenotifyForCharacter | Re-emit resolution notifications not yet delivered (driven by LOGIN) |
| Sweep | Expire every PENDING request whose deadline has passed |
| GetByCharacterId | Retrieve requests for a character |
| GetById | Retrieve a request by ID |
| NameReserved | Report whether a live PENDING NAME_CHANGE holds a given name |
| CheckTransferEligibility | Evaluate the full gate table for a character and destination world |
| CheckTransferEligibilityIndependent | Evaluate only the destination-independent gates |

---

## Equip Slot

### Responsibility
Manages purchased equipped-inventory slot extensions per character.

### Core Models

#### Model
Immutable representation of one active equip-slot extension.

| Field | Type | Description |
|-------|------|--------------|
| id | uuid.UUID | Extension record identifier |
| characterId | uint32 | Character |
| slotIndex | int16 | Atlas canonical equipped-inventory position the extension unlocks |
| expiresAt | time.Time | Expiration timestamp |

### Invariants
- A character has at most one extension row per slotIndex
- Extend adds period to max(now, current expiresAt) rather than duplicating or resetting the row
- Extend is idempotent per transactionId: a repeat call carrying the same non-zero transactionId as the row's last-applied call returns the current expiry unchanged
- An expired row is not deleted; only rows with expiresAt in the future are considered active

### Processors

#### Processor
Handles equip-slot extension operations.

| Operation | Description |
|-----------|-------------|
| Extend | Extend (or create) a character's slot extension by a period |
| GetActive | Retrieve a character's currently-active extensions |

---

## Teleport Rock

### Responsibility
Manages a character's saved teleport-rock map lists (regular and VIP).

### Core Models

#### Model
Immutable representation of both of a character's saved-map lists, unpadded and ordered.

| Field | Type | Description |
|-------|------|--------------|
| characterId | uint32 | Character |
| regular | []map.Id | Regular list entries |
| vip | []map.Id | VIP list entries |

### Invariants
- The regular list holds at most 5 entries; the VIP list holds at most 10
- A map is eligible for registration only if mapId/100000000 != 0 and (mapId/1000000)%100 != 9
- A map may appear at most once per list
- Removing a map compacts the remaining entries of that list to a contiguous prefix

### Processors

#### Processor
Handles teleport-rock list operations.

| Operation | Description |
|-----------|-------------|
| GetByCharacterId | Retrieve both lists for a character |
| AddMap / AddMapAndEmit | Register the character's current map on a list; buffers/emits an error event on rejection |
| Add | REST-facing add that returns the typed validation error instead of buffering it |
| RemoveMap / RemoveMapAndEmit | Remove a map from a list; buffers/emits an error event on rejection |
| Remove | REST-facing remove that returns the typed validation error instead of buffering it |

---

## Configuration

### Responsibility
Resolves and caches per-tenant pending-change configuration fetched from atlas-tenants.

### Core Models

#### Model
Immutable per-tenant imprint (pending-change) configuration.

| Field | Type | Description |
|-------|------|--------------|
| pendingExpiry | time.Duration | How long a pending request survives before the sweep expires and refunds it |

### Invariants
- A non-positive fetched expiry value falls back to the shipped default of 168 hours
- A tenant with no imprint-configs resource resolves to the default configuration
- A tenant's resolved configuration is fetched once and cached for the process lifetime

### Processors

#### Registry
Resolves and caches configuration.

| Operation | Description |
|-----------|-------------|
| Get | Return the cached configuration for the request's tenant, fetching and caching on first access |

---

## External Effective Stats

### Responsibility
Retrieves a character's computed effective stats from the external atlas-effective-stats service.

### Core Models

#### Model
Immutable representation of a character's effective stats as reported by atlas-effective-stats.

| Field | Type | Description |
|-------|------|--------------|
| strength | uint32 | Effective STR |
| dexterity | uint32 | Effective DEX |
| luck | uint32 | Effective LUK |
| intelligence | uint32 | Effective INT |
| maxHp | uint32 | Effective max HP |
| maxMp | uint32 | Effective max MP |
| weaponAttack | uint32 | Effective weapon attack |
| weaponDefense | uint32 | Effective weapon defense |
| magicAttack | uint32 | Effective magic attack |
| magicDefense | uint32 | Effective magic defense |
| accuracy | uint32 | Effective accuracy |
| avoidability | uint32 | Effective avoidability |
| speed | uint32 | Effective speed |
| jump | uint32 | Effective jump |
