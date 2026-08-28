# REST

This service exposes one endpoint, `GET /api/chat/history`. It **is** routed
through nginx/ingress (`deploy/shared/routes.conf`) and is reachable at the
ingress host **without authentication** — it exposes captured player chat,
including whispers, to anything that can reach that host. This was an
explicit, accepted risk (see
`docs/tasks/task-145-player-reports/scope-amendment.md` Amendment 2) taken on
the basis that API authentication is coming to the fleet generally; do not
describe this endpoint as server-to-server only. It also consumes REST APIs
from other services (see below).

## Endpoints

### GET /api/chat/history

Returns recently captured chat lines authored by any of the given
characters, merged and sorted ascending by timestamp. Backed by the
`chat:recent` Redis buffer (see [Storage](storage.md)); consumed by
atlas-ban to snapshot a transcript when a player report is created.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| characterIds | comma-separated uint32 list | query | Required. Character IDs whose recent lines to include. |

A missing or malformed `characterIds` query parameter returns `400 Bad
Request`.

**Response Model**

Resource type: `chat-messages`

| Field | Type | Description |
|-------|------|-------------|
| timestamp | int64 | Unix-milli capture time |
| senderId | uint32 | Authoring character ID |
| senderName | string | Authoring character name at capture time |
| chatType | string | One of `GENERAL`, `BUDDY`, `PARTY`, `GUILD`, `ALLIANCE`, `WHISPER`, `MESSENGER` (pet echoes and system pink text are never captured) |
| text | string | Chat line text |

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
| experience | uint32 | Experience points |
| gachaponExperience | uint32 | Gachapon experience |
| strength | uint16 | Strength stat |
| dexterity | uint16 | Dexterity stat |
| intelligence | uint16 | Intelligence stat |
| luck | uint16 | Luck stat |
| hp | uint16 | Current HP |
| maxHp | uint16 | Maximum HP |
| mp | uint16 | Current MP |
| maxMp | uint16 | Maximum MP |
| meso | uint32 | Meso amount |
| hpMpUsed | int | HP/MP AP allocation used |
| jobId | uint16 | Job identifier |
| skinColor | byte | Skin color |
| gender | byte | Gender |
| fame | int16 | Fame value |
| hair | uint32 | Hair ID |
| face | uint32 | Face ID |
| ap | uint16 | Unassigned AP |
| sp | string | Comma-separated SP table |
| spawnPoint | uint32 | Spawn point |
| gm | int | GM status |
| x | int16 | X position |
| y | int16 | Y position |
| stance | byte | Stance |

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

### atlas-rates

#### GET /worlds/{worldId}/channels/{channelId}/characters/{characterId}/rates

Retrieves rates and rate factors for a character.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| worldId | byte | path | World identifier |
| channelId | byte | path | Channel identifier |
| characterId | uint32 | path | Character ID |

**Response Model**

Resource type: `rates`

| Field | Type | Description |
|-------|------|-------------|
| expRate | float64 | Experience rate multiplier |
| mesoRate | float64 | Meso rate multiplier |
| itemDropRate | float64 | Item drop rate multiplier |
| questExpRate | float64 | Quest experience rate multiplier |
| factors | []Factor | Rate factor breakdowns |

#### Factor

| Field | Type | Description |
|-------|------|-------------|
| source | string | Factor source identifier |
| rateType | string | Rate type (exp, meso, item_drop, quest_exp) |
| multiplier | float64 | Multiplier value |

---

### atlas-party-quests

#### GET /party-quests/instances/character/{characterId}

Retrieves the party quest instance for a character.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| characterId | uint32 | path | Character ID |

**Response Model**

Resource type: `instances`

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Party quest instance ID |

---

### atlas-pets

#### GET /characters/{characterId}/pets

Retrieves pets owned by a character. Paginated (page[number]/page[size] query parameters).

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| characterId | uint32 | path | Character ID |

**Response Model**

Resource type: `pets`

| Field | Type | Description |
|-------|------|-------------|
| cashId | uint64 | Cash shop item ID |
| templateId | uint32 | Pet template ID |
| name | string | Pet name |
| level | byte | Pet level |
| closeness | uint16 | Pet closeness (tameness) |
| fullness | byte | Pet fullness |
| expiration | time.Time | Pet expiration time |
| ownerId | uint32 | Owning character ID |
| slot | int8 | Pet slot |
| x | int16 | X position |
| y | int16 | Y position |
| stance | byte | Stance |
| fh | int16 | Foothold ID |
| flag | uint16 | Flag |
| purchaseBy | uint32 | Purchasing account ID |

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

#### GET /data/equipment/{itemId}

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

Retrieves skill data by ID.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| skillId | uint32 | path | Skill identifier |

**Response Model**

Resource type: `skills`

| Field | Type | Description |
|-------|------|-------------|
| name | string | Skill name |
| action | bool | Has action animation |
| element | string | Element type |
| animationTime | uint32 | Animation duration |
| effects | []effect | Skill effects per level |

#### GET /data/skills?name={name}

Retrieves skills matching a name.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| name | string | query | Skill name (URL-encoded) |

**Response Model**

Array of skills matching the name. Same resource type and fields as GET /data/skills/{skillId}.

#### POST /data/maps/{mapId}/footholds/below

Retrieves the foothold below a given position within a map.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| mapId | uint32 | path | Map identifier |

**Request Model**

Resource type: `positions`

| Field | Type | Description |
|-------|------|-------------|
| x | int16 | X position |
| y | int16 | Y position |

**Response Model**

Resource type: `footholds`

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Foothold ID |

#### GET /data/monsters/{monsterId}

Retrieves monster template data by ID.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| monsterId | uint32 | path | Monster template ID |

**Response Model**

Resource type: `monsters`

| Field | Type | Description |
|-------|------|-------------|
| name | string | Monster name |
