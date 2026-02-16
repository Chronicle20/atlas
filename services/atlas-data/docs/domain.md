# Domain

## Data

### Responsibility

The data domain manages static game data that is parsed from WZ/XML files and served via REST endpoints. Data is tenant-aware, with per-tenant isolation.

### Core Models

#### Cash Item
Represents cash shop item data with slot limits, spec modifiers, and time windows.

#### Character Template
Defines character creation templates with faces, hair styles, hair colors, skin colors, tops, bottoms, shoes, and weapons.

#### Commodity
Represents commodity items with item ID, count, price, period, priority, gender, and sale status.

#### Consumable
Represents consumable items with trade properties, price, slot limits, level requirements, and spec modifiers including HP/MP recovery, stat buffs, morph effects, monster summons, skills, rewards, and rechargeable status.

#### Equipment
Represents equipment statistics including strength, dexterity, intelligence, luck, HP, MP, weapon attack, magic attack, weapon defense, magic defense, accuracy, avoidability, speed, jump, slots, cash status, price, time-limited status, replace item ID, replace message, and bonus experience tiers. Equipment has related equipment slots.

#### ETC Item
Represents ETC items with price, unit price, slot limits, time-limited status, replace item ID, and replace message.

#### Face
Represents face cosmetic data with cash status.

#### Hair
Represents hair cosmetic data with cash status.

#### Item String
Represents item name lookup data with item ID and name.

#### Map
Represents game maps with name, street name, return map ID, monster rate, event triggers (onFirstUserEnter, onUserEnter), field limits, mob intervals, portals, time mobs, map areas, foothold trees, areas, seats, clock status, everLast status, town status, decay HP, protect item, forced return map ID, boat status, time limits, field type, mob capacity, recovery rate, background types, X limits, reactors, NPCs, and monsters.

##### Portal (Map sub-model)
Represents portals within a map with name, target, type, position (x, y), target map ID, and script name.

##### NPC (Map sub-model)
Represents NPCs within a map with template ID, name, position (cy, x, y), facing direction (f), foothold (fh), range (rx0, rx1), and hide status.

##### Monster (Map sub-model)
Represents monster spawns within a map with template ID, mob time, team, position (cy, x, y), facing direction (f), foothold (fh), range (rx0, rx1), and hide status.

##### Reactor (Map sub-model)
Represents reactor spawns within a map with classification, name, position (x, y), delay, and direction.

##### Foothold Tree
Represents the spatial foothold structure for collision detection with quadtree nodes (NorthWest, NorthEast, SouthWest, SouthEast), foothold lists, bounding points, center, depth, and drop position limits.

#### Monster
Represents monster data with name, HP, MP, experience, level, weapon attack, weapon defense, magic attack, magic defense, friendly status, remove timer, boss status, explosive reward, FFA loot, undead status, buff to give, CP, remove on miss, changeable status, animation times, resistances, lose items, skills, revives, tag colors, fixed stance, first attack status, banish info, drop period, self-destruction info, and cool damage.

#### NPC
Represents NPC data with name, trunk put, trunk get, storebank status, hide name status, and dialog coordinates (dc_left, dc_right, dc_top, dc_bottom).

#### Pet
Represents pet data with hungry rate, cash status, life span, and skills with increase and probability values.

#### Quest
Represents quest data with name, parent, area, order, auto-start status, auto-pre-complete status, auto-complete status, time limits, selected mob status, summary, demand summary, reward summary, start requirements, end requirements, start actions, and end actions.

##### Requirements (Quest sub-model)
Represents quest requirements with NPC ID, level range, fame minimum, meso range, jobs, prerequisite quests, item requirements, mob requirements, field enter requirements, pet requirements, pet tameness minimum, day of week, time range, interval, scripts, info number, normal auto-start status, and completion count.

##### Actions (Quest sub-model)
Represents quest actions with NPC ID, experience, money, fame, item rewards, skill rewards, next quest, buff item ID, interval, and level minimum.

#### Reactor
Represents reactor data with name, bounding box (TL, BR), state info mapping state IDs to reactor states, and timeout info.

#### Setup
Represents setup items with price, slot max, recovery HP, trade block, not sale, required level, distance (X, Y), max diff, direction, time-limited status, replace item ID, and replace message.

#### Skill
Represents skill data with name, action status, element type, animation time, and skill effects including stat modifiers, durations, targets, and special properties.

#### Mob Skill
Represents monster skill data with skill ID, level, MP cost, duration, HP threshold, position (x, y), probability, interval, count, limit, bounding box (lt, rb), summon effect, and summon monster IDs. Identified by composite key of skill ID and level.

### Processors

Each data type has a processor responsible for:
- Parsing WZ/XML data files
- Registering data into in-memory registries
- Persisting data as JSON documents in the database

Processing is triggered via `POST /api/data/process`, which dispatches workers through Kafka for each data type. Workers use a pool of 10 goroutines for parallel file processing.

Processors include:
- `cash.RegisterCash`
- `templates.RegisterCharacterTemplate`
- `commodity.RegisterCommodity`
- `consumable.RegisterConsumable`
- `equipment.RegisterEquipment`
- `etc.RegisterEtc`
- `face.RegisterFace`
- `hair.RegisterHair`
- `_map.RegisterMap`
- `monster.RegisterMonster`
- `npc.RegisterNpc`
- `pet.RegisterPet`
- `quest.RegisterQuest`
- `reactor.RegisterReactor`
- `setup.RegisterSetup`
- `skill.RegisterSkill`
- `mobskill.RegisterMobSkill`
