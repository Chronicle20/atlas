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
| mapId | map.Id | Current map |
| instance | uuid.UUID | Map instance |
| spawnPoint | uint32 | Spawn point ID |
| gm | int | GM level |
| skills | []skill.Model | Character skills |

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
- Character name must match pattern `[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}`
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

### Processors

#### Processor
Handles character operations.

| Operation | Description |
|-----------|-------------|
| GetById | Retrieve character by ID |
| GetForAccountInWorld | Retrieve characters for account in world |
| GetForMapInWorld | Retrieve characters on map |
| GetForName | Retrieve characters by name |
| GetAll | Retrieve all characters |
| IsValidName | Validate character name |
| Create | Create new character |
| Delete | Delete character |
| DeleteByAccountId | Delete all characters for an account |
| Login | Process character login |
| Logout | Process character logout |
| ChangeChannel | Process channel change |
| ChangeMap | Process map change |
| ChangeJob | Change character job |
| ChangeHair | Change hair style |
| ChangeFace | Change face |
| ChangeSkin | Change skin color |
| AwardExperience | Award experience |
| AwardLevel | Award levels |
| Move | Update position |
| RequestChangeMeso | Process meso change |
| AttemptMesoPickUp | Process meso pickup |
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
Retrieves portal position data from external data service for map change positioning.

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
