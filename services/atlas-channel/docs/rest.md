# REST Documentation

This service consumes REST APIs from external services. It does not expose REST endpoints.

## External Service Dependencies

### ACCOUNTS
Base URL: `BASE_SERVICE_URL` + ACCOUNTS root

#### GET /accounts/{accountId}
- Parameters: accountId (uint32)
- Request Model: None
- Response Model: `RestModel` - Account details (id, name, password, pin, pic, loggedIn, lastLogin, gender, banned, tos, language, country, characterSlots)
- Error Conditions: 404 if account not found

---

### BUDDIES
Base URL: `BASE_SERVICE_URL` + BUDDIES root

#### GET /characters/{characterId}/buddy-list
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `RestModel` - Buddy list with tenantId, characterId, capacity, and buddies array (each with characterId, group, characterName, channelId, inShop, pending)
- Error Conditions: 404 if buddy list not found

---

### BUFFS
Base URL: `BASE_SERVICE_URL` + BUFFS root

#### GET /characters/{characterId}/buffs
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Active character buffs (sourceId, level, duration, changes with stat type/amount, createdAt, expiresAt)
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
- Parameters: accountId (uint32), type (byte - compartment type: 1=Explorer, 2=Cygnus, 3=Legend)
- Request Model: None
- Response Model: `RestModel` - Cash shop compartment of specified type with assets
- Error Conditions: 404 if compartment not found

#### GET /accounts/{accountId}/cash-shop/inventory/compartments/{compartmentId}/assets
- Parameters: accountId (uint32), compartmentId (uuid)
- Request Model: None
- Response Model: `[]RestModel` - All assets in a cash shop compartment (each with id, compartmentId, item containing id, cashId, templateId, commodityId, quantity, flag, purchasedBy, expiration)
- Error Conditions: None

#### GET /accounts/{accountId}/cash-shop/inventory/compartments/{compartmentId}/assets/{assetId}
- Parameters: accountId (uint32), compartmentId (uuid), assetId (uuid)
- Request Model: None
- Response Model: `RestModel` - Cash shop asset with nested item reference
- Error Conditions: 404 if asset not found

#### GET /accounts/{accountId}/wallet
- Parameters: accountId (uint32)
- Request Model: None
- Response Model: `RestModel` - NX wallet balances (credit, points, prepaid)
- Error Conditions: 404 if wallet not found

#### GET /characters/{characterId}/cash-shop/wishlist
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Wishlist items (characterId, serialNumber)
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
- Response Model: `[]RestModel` - Chairs in map (id, type, characterId)
- Error Conditions: None

---

### CHALKBOARDS
Base URL: `BASE_SERVICE_URL` + CHALKBOARDS root

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/chalkboards
- Parameters: worldId, channelId, mapId
- Request Model: None
- Response Model: `[]RestModel` - Chalkboards in map (id, message)
- Error Conditions: None

---

### CHANNELS
Base URL: `BASE_SERVICE_URL` + CHANNELS root

#### GET /worlds/{worldId}/channels/{channelId}
- Parameters: worldId, channelId
- Request Model: None
- Response Model: `RestModel` - Channel information (id, worldId, channelId, ipAddress, port, currentCapacity, maxCapacity, createdAt)
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
- Response Model: `RestModel` - Character details (id, accountId, worldId, name, level, experience, stats, jobId, appearance, ap, sp, mapId, spawnPoint, gm, position, meso)
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

#### GET /configurations/services/{serviceId}
- Parameters: serviceId (uuid)
- Request Model: None
- Response Model: `RestModel` - Service configuration with tasks (type, interval, duration) and tenants (id, ipAddress, worlds with channels and ports)
- Error Conditions: 404 if not found

#### GET /configurations/tenants/{tenantId}
- Parameters: tenantId (uuid)
- Request Model: None
- Response Model: `RestModel` - Tenant configuration with region, majorVersion, minorVersion, usesPin, socket (handlers with opCode/validator/handler/options, writers with opCode/writer/options), characters (templates with jobIndex/subJobIndex/mapId/gender/faces/hairs/etc), NPCs (npcId, impl), and worlds (name, flag, serverMessage, eventMessage, whyAmIRecommended)
- Error Conditions: 404 if not found

---

### DATA
Base URL: `BASE_SERVICE_URL` + DATA root

#### GET /data/maps/{mapId}
- Parameters: mapId
- Request Model: None
- Response Model: `RestModel` - Map data (clock, returnMapId, fieldLimit, town, monsterRate, seats, fieldType, timeLimit, and other metadata)
- Error Conditions: 404 if map not found

#### GET /data/npcs/{npcId}
- Parameters: npcId (uint32)
- Request Model: None
- Response Model: `RestModel` - NPC template data (id, name, trunkPut, trunkGet, storebank)
- Error Conditions: 404 if NPC not found

#### GET /data/maps/{mapId}/npcs
- Parameters: mapId
- Request Model: None
- Response Model: `[]RestModel` - NPCs in map (id, template, x, cy, f, fh, rx0, rx1)
- Error Conditions: None

#### GET /data/maps/{mapId}/npcs?objectId={objectId}
- Parameters: mapId, objectId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - NPCs by object ID
- Error Conditions: None

#### GET /data/skills/{skillId}
- Parameters: skillId (uint32)
- Request Model: None
- Response Model: `RestModel` - Skill data (id, action, element, animationTime) with effects array. Each effect contains stat modifiers, resource costs, duration, cooldown, monster status effects, cure lists, and statups.
- Error Conditions: 404 if skill not found

#### GET /data/maps/{mapId}/portals
- Parameters: mapId
- Request Model: None
- Response Model: `[]RestModel` - All portals in map (id, name, target, type, x, y, targetMapId, scriptName)
- Error Conditions: None

#### GET /data/maps/{mapId}/portals?name={name}
- Parameters: mapId, name (string)
- Request Model: None
- Response Model: `[]RestModel` - Portals matching name in map
- Error Conditions: None

#### GET /data/quests/{questId}
- Parameters: questId (uint32)
- Request Model: None
- Response Model: `RestModel` - Quest definition with start/end requirements and actions
- Error Conditions: 404 if quest not found

#### GET /data/quests
- Parameters: None
- Request Model: None
- Response Model: `[]RestModel` - All quest definitions
- Error Conditions: None

#### GET /data/quests/auto-start
- Parameters: None
- Request Model: None
- Response Model: `[]RestModel` - Quests with autoStart enabled
- Error Conditions: None

#### GET /data/cash/items/{itemId}
- Parameters: itemId (uint32)
- Request Model: None
- Response Model: `RestModel` - Cash item data (stateChangeItem, bgmPath)
- Error Conditions: 404 if not found

---

### DROPS
Base URL: `BASE_SERVICE_URL` + DROPS root

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/drops
- Parameters: worldId, channelId, mapId, instanceId (uuid)
- Request Model: None
- Response Model: `[]RestModel` - Drops in field instance (id, itemId, equipmentId, quantity, meso, type, x, y, ownerId, ownerPartyId, dropTime, dropperId, dropperX, dropperY, characterDrop, mod)
- Error Conditions: None

---

### GUILDS
Base URL: `BASE_SERVICE_URL` + GUILDS root

#### GET /guilds/{guildId}
- Parameters: guildId (uint32)
- Request Model: None
- Response Model: `RestModel` - Guild details (id, worldId, name, notice, points, capacity, logo, logoColor, logoBackground, logoBackgroundColor, leaderId, members, titles)
- Error Conditions: 404 if not found

#### GET /guilds?filter[members.id]={characterId}
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Guilds with member
- Error Conditions: None

---

### GUILD_THREADS
Base URL: `BASE_SERVICE_URL` + GUILD_THREADS root

#### GET /guilds/{guildId}/threads
- Parameters: guildId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Guild BBS threads (id, posterId, emoticonId, title, message, notice, createdAt, replies)
- Error Conditions: None

#### GET /guilds/{guildId}/threads/{threadId}
- Parameters: guildId (uint32), threadId (uint32)
- Request Model: None
- Response Model: `RestModel` - Thread with replies (id, posterId, message, createdAt)
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
- Response Model: `RestModel` - A single compartment for the specified inventory type with included assets. Each asset uses the unified asset model (id, slot, templateId, expiration, createdAt, quantity, ownerId, flag, rechargeable, equipment stats, cash fields, pet fields).
- Error Conditions: 404 if not found

---

### KEYS
Base URL: `BASE_SERVICE_URL` + KEYS root

#### GET /characters/{characterId}/keys
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Key bindings (key, type, action)
- Error Conditions: None

#### PATCH /characters/{characterId}/keys/{key}
- Parameters: characterId (uint32), key (int32)
- Request Model: `RestModel` - Key binding update (type, action)
- Response Model: `RestModel` - Updated key binding
- Error Conditions: 400 if invalid

---

### MAPS
Base URL: `BASE_SERVICE_URL` + MAPS root

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/characters
- Parameters: worldId, channelId, mapId, instanceId (uuid)
- Request Model: None
- Response Model: `[]RestModel` - Character IDs in field instance
- Error Conditions: None

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/characters
- Parameters: worldId, channelId, mapId
- Request Model: None
- Response Model: `[]RestModel` - Character IDs in map (all instances)
- Error Conditions: None

---

### MESSENGERS
Base URL: `BASE_SERVICE_URL` + MESSENGERS root

#### GET /messengers/{messengerId}
- Parameters: messengerId (uint32)
- Request Model: None
- Response Model: `RestModel` - Messenger details (id, members with id/name/worldId/channelId/online/slot)
- Error Conditions: 404 if not found

#### GET /messengers?filter[members.id]={characterId}
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
- Response Model: `[]RestModel` - Monsters in field instance (id, worldId, channelId, mapId, instance, monsterId, controlCharacterId, x, y, fh, stance, team, maxHp, hp, maxMp, mp, damageEntries, statusEffects)
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
- Response Model: `[]RestModel` - Notes for character (id, characterId, senderId, message, flag, timestamp)
- Error Conditions: None

#### GET /notes/{noteId}
- Parameters: noteId (uint32)
- Request Model: None
- Response Model: `RestModel` - Note details
- Error Conditions: 404 if not found

---

### NPC_SHOP
Base URL: `BASE_SERVICE_URL` + NPC_SHOP root

#### GET /npcs/{npcId}/shop?include=commodities
- Parameters: npcId (uint32)
- Request Model: None
- Response Model: `RestModel` - NPC shop with commodities (each with id, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimit, unitPrice, slotMax)
- Error Conditions: 404 if shop not found

---

### PARTIES
Base URL: `BASE_SERVICE_URL` + PARTIES root

#### GET /parties/{partyId}
- Parameters: partyId (uint32)
- Request Model: None
- Response Model: `RestModel` - Party details (id, leaderId, members with id/name/level/jobId/worldId/channelId/mapId/instance/online)
- Error Conditions: 404 if not found

#### GET /parties?filter[members.id]={characterId}
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Parties with member
- Error Conditions: None

#### GET /parties/{partyId}/members
- Parameters: partyId (uint32)
- Request Model: None
- Response Model: `[]MemberRestModel` - Party members
- Error Conditions: None

---

### PARTY_QUESTS
Base URL: `BASE_SERVICE_URL` + PARTY_QUESTS root

#### GET /party-quests/instances/character/{characterId}/timer
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `TimerRestModel` - Party quest timer (id, duration)
- Error Conditions: 404 if no active timer

---

### PETS
Base URL: `BASE_SERVICE_URL` + PETS root

#### GET /pets/{petId}
- Parameters: petId (uint32)
- Request Model: None
- Response Model: `RestModel` - Pet details (id, cashId, templateId, name, level, closeness, fullness, expiration, ownerId, lead, slot, x, y, stance, fh, excludes, flag, purchaseBy)
- Error Conditions: 404 if not found

#### GET /characters/{characterId}/pets
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Pets by owner
- Error Conditions: None

---

### QUESTS
Base URL: `BASE_SERVICE_URL` + QUESTS root

#### GET /characters/{characterId}/quests
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Character quest progress (id, characterId, questId, state, startedAt, completedAt, expirationTime, completedCount, forfeitCount, progress with infoNumber/progress)
- Error Conditions: None

---

### REACTORS
Base URL: `BASE_SERVICE_URL` + REACTORS root

#### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/reactors
- Parameters: worldId, channelId, mapId, instanceId (uuid)
- Request Model: None
- Response Model: `[]RestModel` - Reactors in field instance (id, worldId, channelId, mapId, instance, classification, name, state, eventState, x, y, delay, direction)
- Error Conditions: None

---

### ROUTES
Base URL: `BASE_SERVICE_URL` + ROUTES root

#### GET /transports/routes
- Parameters: None
- Request Model: None
- Response Model: `[]RestModel` - Transport routes in tenant (id, name, startMapId, stagingMapId, enRouteMapIds, destinationMapId, state, cycleInterval)
- Error Conditions: None

#### GET /transports/routes/{routeId}
- Parameters: routeId (uuid)
- Request Model: None
- Response Model: `RestModel` - Route details
- Error Conditions: 404 if not found

#### GET /transports/routes/{routeId}/state
- Parameters: routeId (uuid)
- Request Model: None
- Response Model: `RestModel` - Current route state
- Error Conditions: 404 if not found

#### GET /transports/routes/{routeId}/schedule
- Parameters: routeId (uuid)
- Request Model: None
- Response Model: `[]TripScheduleRestModel` - Route schedule (tripId, boardingOpen, boardingClosed, departure, arrival)
- Error Conditions: None

---

### SKILLS
Base URL: `BASE_SERVICE_URL` + SKILLS root

#### GET /characters/{characterId}/skills
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Character skills (id, level, masterLevel, expiration, cooldownExpiresAt)
- Error Conditions: None

#### GET /characters/{characterId}/skills/{skillId}
- Parameters: characterId (uint32), skillId (uint32)
- Request Model: None
- Response Model: `RestModel` - Specific skill
- Error Conditions: 404 if not found

#### GET /characters/{characterId}/macros
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `[]RestModel` - Skill macros (id, name, shout, skillId1, skillId2, skillId3)
- Error Conditions: None

---

### STORAGE
Base URL: `BASE_SERVICE_URL` + STORAGE root

#### GET /storage/accounts/{accountId}?worldId={worldId}
- Parameters: accountId (uint32), worldId (byte)
- Request Model: None
- Response Model: `StorageRestModel` - Storage metadata (capacity, mesos) with included `AssetRestModel` items via JSON:API relationship. Each asset contains id, slot, templateId, expiration, quantity, ownerId, flag, rechargeable, equipment stats, cashId, commodityId, purchaseBy, petId. Resource type: "storages" with "storage_assets" relationship.
- Error Conditions: 404 if not found (caller returns empty storage with default capacity)

#### GET /storage/accounts/{accountId}/assets?worldId={worldId}
- Parameters: accountId (uint32), worldId (byte)
- Request Model: None
- Response Model: `[]AssetRestModel` - Storage assets for the account and world
- Error Conditions: None

#### GET /storage/projections/{characterId}
- Parameters: characterId (uint32)
- Request Model: None
- Response Model: `ProjectionRestModel` - Storage projection containing characterId, accountId, worldId, storageId, capacity, mesos, npcId, and compartments (map of compartment name to raw JSON asset arrays). Each compartment's assets are parsed via `ParseCompartmentAssets()` into `[]AssetRestModel`. Resource type: "storage_projections".
- Error Conditions: 404 if no active projection

#### GET /storage/projections/{characterId}/compartments/{compartmentType}/assets/{slot}
- Parameters: characterId (uint32), compartmentType (byte), slot (int16)
- Request Model: None
- Response Model: `AssetRestModel` - A single asset from a projection compartment by type and slot
- Error Conditions: 404 if not found

---

### WEATHER
Base URL: `BASE_SERVICE_URL` + WEATHER root

#### GET (weather in field)
- Parameters: field (worldId, channelId, mapId, instanceId)
- Request Model: None
- Response Model: `RestModel` - Active weather (id, itemId, message). Resource type: "weather".
- Error Conditions: 404 if no active weather

---

### WORLDS
Base URL: `BASE_SERVICE_URL` + WORLDS root

#### GET /worlds/{worldId}
- Parameters: worldId
- Request Model: None
- Response Model: `RestModel` - World details (id, name, state, message, eventMessage, recommended, recommendedMessage, capacityStatus, channels, expRate, mesoRate, itemDropRate, questExpRate)
- Error Conditions: 404 if not found
