# REST Documentation

This service consumes REST APIs from external services. It does not expose REST endpoints.

## External Service Dependencies

### ACCOUNTS
Base URL: `BASE_SERVICE_URL` + ACCOUNTS root

#### GET /accounts/{accountId}
- Parameters: accountId (uint32)
- Request Model: None
- Response Model: `RestModel` - Account details
- Error Conditions: 404 if account not found

---

### BUDDIES
Base URL: `BASE_SERVICE_URL` + BUDDIES root

#### GET /characters/{characterId}/buddy-list
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `RestModel` - Buddy list with capacity and buddies
- Error Conditions: 404 if buddy list not found

---

### BUFFS
Base URL: `BASE_SERVICE_URL` + BUFFS root

#### GET /characters/{characterId}/buffs
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Active character buffs
- Error Conditions: 404 if character not found

---

### CASHSHOP
Base URL: `BASE_SERVICE_URL` + CASHSHOP root

#### GET /accounts/{accountId}/cash-shop/inventory/compartments
- Parameters: accountId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - All cash shop compartments for the account
- Error Conditions: None

#### GET /accounts/{accountId}/cash-shop/inventory/compartments?type={type}
- Parameters: accountId (uint32), type (byte - compartment type)
- Request Model: None
- Response Model: `RestModel` - Cash shop compartment of specified type
- Error Conditions: 404 if compartment not found

#### GET /accounts/{accountId}/cash-shop/inventory/compartments/{compartmentId}/assets
- Parameters: accountId (uint32), compartmentId (uuid)
- Request Model: None
- Response Model: `[]RestModel` - All assets in a cash shop compartment
- Error Conditions: None

#### GET /accounts/{accountId}/cash-shop/inventory/compartments/{compartmentId}/assets/{assetId}
- Parameters: accountId (uint32), compartmentId (uuid), assetId (uuid)
- Request Model: None
- Response Model: `RestModel` - Cash shop asset with nested item reference
- Error Conditions: 404 if asset not found

#### GET /accounts/{accountId}/wallet
- Parameters: accountId (uint32)
- Request Model: None
- Response Model: `RestModel` - NX wallet balances
- Error Conditions: 404 if wallet not found

#### GET /characters/{characterId}/cash-shop/wishlist
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Wishlist items
- Error Conditions: None

#### POST /characters/{characterId}/cash-shop/wishlist
- Parameters: characterId (uint32)
- Request Model: `RestModel` - Wishlist item with serialNumber
- Response Model: `RestModel` - Added wishlist item
- Error Conditions: 400 if invalid

#### DELETE /characters/{characterId}/cash-shop/wishlist
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: None
- Error Conditions: None

---

### CHAIRS
Base URL: `BASE_SERVICE_URL` + CHAIRS root

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/chairs
- Parameters: worldId, channelId, mapId
- Request Model: None
- Response Model: `[]RestModel` - Chairs in map
- Error Conditions: None

---

### CHALKBOARDS
Base URL: `BASE_SERVICE_URL` + CHALKBOARDS root

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/chalkboards
- Parameters: worldId, channelId, mapId
- Request Model: None
- Response Model: `[]RestModel` - Chalkboards in map
- Error Conditions: None

---

### CHANNELS
Base URL: `BASE_SERVICE_URL` + CHANNELS root

#### GET /worlds/{worldId}/channels/{channelId}
- Parameters: worldId, channelId
- Request Model: None
- Response Model: `RestModel` - Channel information
- Error Conditions: 404 if channel not found

#### POST /worlds/{worldId}/channels
- Parameters: worldId
- Request Model: `RestModel` - Channel to register
- Response Model: `RestModel` - Registered channel
- Error Conditions: 400 if invalid

---

### CHARACTERS
Base URL: `BASE_SERVICE_URL` + CHARACTERS root

#### GET /characters/{characterId}
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `RestModel` - Character details
- Error Conditions: 404 if character not found

#### GET /characters/{characterId}?include=inventory
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `RestModel` - Character with inventory
- Error Conditions: 404 if character not found

#### GET /characters?name={name}&include=inventory
- Parameters: name (string)
- Request Model: None
- Response Model: `[]RestModel` - Characters matching name
- Error Conditions: None

---

### CONFIGURATIONS
Base URL: `BASE_SERVICE_URL` + CONFIGURATIONS root

#### GET /services/{serviceId}
- Parameters: serviceId (uuid)
- Request Model: None
- Response Model: `RestModel` - Service configuration
- Error Conditions: 404 if not found

#### GET /tenants/{tenantId}
- Parameters: tenantId (uuid)
- Request Model: None
- Response Model: `RestModel` - Tenant configuration
- Error Conditions: 404 if not found

---

### DATA
Base URL: `BASE_SERVICE_URL` + DATA root

#### GET /maps/{mapId}
- Parameters: mapId
- Request Model: None
- Response Model: `RestModel` - Map data with portals, NPCs
- Error Conditions: 404 if map not found

#### GET /npcs/{npcId}
- Parameters: npcId (uint32)
- Request Model: None
- Response Model: `RestModel` - NPC template data
- Error Conditions: 404 if NPC not found

#### GET /maps/{mapId}/npcs
- Parameters: mapId
- Request Model: None
- Response Model: `[]RestModel` - NPCs in map
- Error Conditions: None

#### GET /maps/{mapId}/npcs?objectId={objectId}
- Parameters: mapId, objectId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - NPCs by object ID
- Error Conditions: None

#### GET /skills/{skillId}
- Parameters: skillId (uint32)
- Request Model: None
- Response Model: `RestModel` - Skill data with effects array. Each effect contains stat modifiers, resource costs, duration, cooldown, monster status effects, cure lists, and statups.
- Error Conditions: 404 if skill not found

#### GET /maps/{mapId}/portals?name={name}
- Parameters: mapId, name (string)
- Request Model: None
- Response Model: `[]RestModel` - Portals matching name in map
- Error Conditions: None

#### GET /quests/{questId}
- Parameters: questId (uint32)
- Request Model: None
- Response Model: `RestModel` - Quest definition with start/end requirements and actions
- Error Conditions: 404 if quest not found

#### GET /quests
- Parameters: None
- Request Model: None
- Response Model: `[]RestModel` - All quest definitions
- Error Conditions: None

#### GET /quests/auto-start
- Parameters: None
- Request Model: None
- Response Model: `[]RestModel` - Quests with autoStart enabled
- Error Conditions: None

---

### DROPS
Base URL: `BASE_SERVICE_URL` + DROPS root

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/drops
- Parameters: worldId, channelId, mapId
- Request Model: None
- Response Model: `[]RestModel` - Drops in map
- Error Conditions: None

---

### GUILDS
Base URL: `BASE_SERVICE_URL` + GUILDS root

#### GET /guilds?memberId={memberId}
- Parameters: memberId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Guilds with member
- Error Conditions: None

#### GET /guilds/{guildId}/members
- Parameters: guildId (uint32)
- Request Model: None
- Response Model: `[]MemberRestModel` - Guild members
- Error Conditions: None

---

### GUILD_THREADS
Base URL: `BASE_SERVICE_URL` + GUILD_THREADS root

#### GET /guilds/{guildId}/threads
- Parameters: guildId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Guild BBS threads
- Error Conditions: None

#### GET /guilds/{guildId}/threads/{threadId}
- Parameters: guildId (uint32), threadId (uint32)
- Request Model: None
- Response Model: `RestModel` - Thread with replies
- Error Conditions: 404 if thread not found

---

### INVENTORY
Base URL: `BASE_SERVICE_URL` + INVENTORY root

#### GET /characters/{characterId}/inventory
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `RestModel` - Character inventory with included compartments and assets. The response uses JSON:API relationships where the inventory contains compartments, and each compartment contains assets. Compartments and assets are extracted via `SetReferencedStructs`.
- Error Conditions: 404 if not found

#### GET /characters/{characterId}/inventory/compartments?type={type}
- Parameters: characterId (uint32), type (inventory.Type as integer: 1=equip, 2=use, 3=setup, 4=etc, 5=cash)
- Request Model: None
- Response Model: `RestModel` - A single compartment for the specified inventory type with included assets. Each asset uses the unified asset.Model (id, compartmentId, slot, templateId, and type-specific fields).
- Error Conditions: 404 if not found

---

### KEYS
Base URL: `BASE_SERVICE_URL` + KEYS root

#### GET /characters/{characterId}/keys
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Key bindings
- Error Conditions: None

#### PATCH /characters/{characterId}/keys/{key}
- Parameters: characterId (uint32), key (int32), type (int8), action (int32)
- Request Model: `RestModel` - Key binding update
- Response Model: `RestModel` - Updated key binding
- Error Conditions: 400 if invalid

---

### MAPS
Base URL: `BASE_SERVICE_URL` + MAPS root

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/characters
- Parameters: worldId, channelId, mapId
- Request Model: None
- Response Model: `[]RestModel` - Character IDs in map
- Error Conditions: None

---

### MESSENGERS
Base URL: `BASE_SERVICE_URL` + MESSENGERS root

#### GET /messengers?characterId={characterId}
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Messengers with character
- Error Conditions: None

---

### MONSTERS
Base URL: `BASE_SERVICE_URL` + MONSTERS root

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/monsters
- Parameters: worldId, channelId, mapId, instanceId (uuid)
- Request Model: None
- Response Model: `[]RestModel` - Monsters in field instance
- Error Conditions: None

#### GET /monsters/{uniqueId}
- Parameters: uniqueId (uint32)
- Request Model: None
- Response Model: `RestModel` - Monster by unique ID
- Error Conditions: 404 if not found

---

### NOTES
Base URL: `BASE_SERVICE_URL` + NOTES root

#### GET /characters/{characterId}/notes
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Notes for character
- Error Conditions: None

#### GET /notes/{noteId}
- Parameters: noteId (uint32)
- Request Model: None
- Response Model: `RestModel` - Note details
- Error Conditions: 404 if not found

---

### NPC_SHOP
Base URL: `BASE_SERVICE_URL` + NPC_SHOP root

#### GET /shops/{templateId}
- Parameters: templateId (uint32)
- Request Model: None
- Response Model: `RestModel` - NPC shop with commodities
- Error Conditions: 404 if shop not found

---

### PARTIES
Base URL: `BASE_SERVICE_URL` + PARTIES root

#### GET /parties?memberId={memberId}
- Parameters: memberId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Parties with member
- Error Conditions: None

---

### PETS
Base URL: `BASE_SERVICE_URL` + PETS root

#### GET /pets?ownerId={ownerId}
- Parameters: ownerId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Pets by owner
- Error Conditions: None

#### GET /pets/{petId}
- Parameters: petId (uint32)
- Request Model: None
- Response Model: `RestModel` - Pet details
- Error Conditions: 404 if not found

---

### REACTORS
Base URL: `BASE_SERVICE_URL` + REACTORS root

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/reactors
- Parameters: worldId, channelId, mapId
- Request Model: None
- Response Model: `[]RestModel` - Reactors in map
- Error Conditions: None

---

### ROUTES
Base URL: `BASE_SERVICE_URL` + ROUTES root

#### GET /routes
- Parameters: None
- Request Model: None
- Response Model: `[]RestModel` - Transport routes in tenant
- Error Conditions: None

#### GET /routes/{id}/schedules
- Parameters: id (string)
- Request Model: None
- Response Model: `[]TripScheduleRestModel` - Route schedules
- Error Conditions: None

#### GET /routes/{id}/state
- Parameters: id (string)
- Request Model: None
- Response Model: `RestModel` - Current route state
- Error Conditions: 404 if not found

---

### SKILLS
Base URL: `BASE_SERVICE_URL` + SKILLS root

#### GET /characters/{characterId}/skills
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Character skills
- Error Conditions: None

#### GET /characters/{characterId}/skills/{skillId}
- Parameters: characterId (uint32), skillId (uint32)
- Request Model: None
- Response Model: `RestModel` - Specific skill
- Error Conditions: 404 if not found

#### GET /characters/{characterId}/macros
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Skill macros
- Error Conditions: None

---

### STORAGE
Base URL: `BASE_SERVICE_URL` + STORAGE root

#### GET /storage/accounts/{accountId}?worldId={worldId}
- Parameters: accountId (uint32), worldId (byte)
- Request Model: None
- Response Model: `StorageRestModel` - Storage metadata (capacity, mesos) with included `AssetRestModel` items via JSON:API relationship. Each asset contains id, slot, templateId, expiration, referenceId, referenceType, and referenceData. The referenceType discriminator (equipable, cash_equipable, consumable, setup, etc, cash, pet) determines the shape of referenceData.
- Error Conditions: 404 if not found (caller returns empty storage with default capacity)

#### GET /storage/accounts/{accountId}/assets?worldId={worldId}
- Parameters: accountId (uint32), worldId (byte)
- Request Model: None
- Response Model: `[]AssetRestModel` - Storage assets for the account and world. Each asset has typed referenceData based on referenceType discriminator.
- Error Conditions: None

#### GET /storage/projections/{characterId}
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `ProjectionRestModel` - Storage projection containing characterId, accountId, worldId, storageId, capacity, mesos, npcId, and compartments (map of compartment name to raw JSON asset arrays). Each compartment's assets are parsed via `ParseCompartmentAssets()` into `[]AssetRestModel`.
- Error Conditions: 404 if no active projection

#### GET /storage/projections/{characterId}/compartments/{compartmentType}/assets/{slot}
- Parameters: characterId (uint32), compartmentType (byte), slot (int16)
- Request Model: None
- Response Model: `AssetRestModel` - A single asset from a projection compartment by type and slot
- Error Conditions: 404 if not found

---

### QUESTS
Base URL: `BASE_SERVICE_URL` + QUESTS root

#### GET /characters/{characterId}/quests
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Character quest progress
- Error Conditions: None

---

### WORLDS
Base URL: `BASE_SERVICE_URL` + WORLDS root

#### GET /worlds/{worldId}
- Parameters: worldId
- Request Model: None
- Response Model: `RestModel` - World details
- Error Conditions: 404 if not found
