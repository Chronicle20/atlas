# REST

This service does not expose any REST endpoints. It consumes REST APIs from other services.

## External API Consumption

The service makes REST API calls to the following services via the `BASE_SERVICE_URL` configuration.

### atlas-character

#### GET /characters/{characterId}

Retrieves a character by ID.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| characterId | uint32 | path | Character ID |

**Response Model**

Resource type: `characters`

| Field | Type | Description |
|-------|------|-------------|
| accountId | uint32 | Associated account ID |
| worldId | byte | World identifier |
| name | string | Character name |
| level | byte | Character level |
| jobId | uint16 | Job identifier |
| mapId | uint32 | Current map ID |
| gm | int | GM status |

#### GET /characters?name={name}

Retrieves characters by name.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| name | string | query | Character name |

**Response Model**

Array of characters matching the name.

---

### atlas-skills

#### GET /characters/{characterId}/skills

Retrieves skills for a character.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| characterId | uint32 | path | Character ID |

**Response Model**

Resource type: `skills`

| Field | Type | Description |
|-------|------|-------------|
| level | byte | Current skill level |
| masterLevel | byte | Maximum skill level |
| expiration | time.Time | Skill expiration time |
| cooldownExpiresAt | time.Time | Cooldown expiration time |

---

### atlas-maps

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/characters

Retrieves character IDs in a map instance.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| worldId | byte | path | World identifier |
| channelId | byte | path | Channel identifier |
| mapId | uint32 | path | Map identifier |
| instanceId | uuid | path | Instance identifier (use nil UUID for non-instanced maps) |

**Response Model**

Resource type: `characters`

Returns array of character references (IDs only).

---

### atlas-data

#### GET /data/maps/{mapId}

Retrieves map data.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| mapId | uint32 | path | Map identifier |

**Response Model**

Resource type: `maps`

| Field | Type | Description |
|-------|------|-------------|
| name | string | Map name |
| streetName | string | Street name |
| returnMapId | uint32 | Return map ID |

#### GET /data/equipment/{itemId}/statistics

Retrieves equipable item statistics.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| itemId | uint32 | path | Item template ID |

**Response Model**

Resource type: `statistics`

| Field | Type | Description |
|-------|------|-------------|
| strength | uint16 | Strength bonus |
| dexterity | uint16 | Dexterity bonus |
| intelligence | uint16 | Intelligence bonus |
| luck | uint16 | Luck bonus |
| hp | uint16 | HP bonus |
| mp | uint16 | MP bonus |
| weaponAttack | uint16 | Weapon attack bonus |
| magicAttack | uint16 | Magic attack bonus |
| weaponDefense | uint16 | Weapon defense bonus |
| magicDefense | uint16 | Magic defense bonus |
| accuracy | uint16 | Accuracy bonus |
| avoidability | uint16 | Avoidability bonus |
| speed | uint16 | Speed bonus |
| jump | uint16 | Jump bonus |
| slots | uint16 | Upgrade slots |
| cash | bool | Cash item flag |

#### GET /data/skills/{skillId}

Retrieves skill data.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| skillId | uint32 | path | Skill identifier |

**Response Model**

Resource type: `skills`

| Field | Type | Description |
|-------|------|-------------|
| action | bool | Has action animation |
| element | string | Element type |
| animationTime | uint32 | Animation duration |
| effects | []effect | Skill effects per level |
