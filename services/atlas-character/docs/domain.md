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
| ChangeMP | Modify MP |
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
