# Domain

## Data

### Responsibility

The data domain manages static game data that is parsed from WZ archives and served via REST endpoints. Data is tenant-aware, with per-tenant isolation, and falls back to a version-scoped canonical dataset (see [Canonical](#canonical)) when a tenant has no rows of its own.

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
Represents item name lookup data with item ID and name. Every item string is also classified into a compartment (equipment/use/setup/etc/cash) and subcategory, and — for equipment — a job-class bitmask, at write time (see `item.Classify`).

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
Represents skill data with name, action status, element type, animation time, and skill effects including stat modifiers, durations, targets, and special properties (including status-up sub-effects and card-item-up bonuses).

#### Mob Skill
Represents monster skill data with skill ID, level, MP cost, duration, HP threshold, position (x, y), probability, interval, count, limit, bounding box (lt, rb), summon effect, and summon monster IDs. Identified by composite key of skill ID and level.

### Processors

Each data type has a processor responsible for:
- Parsing a WZ archive (or an already-extracted XML tree)
- Registering data into in-memory registries
- Persisting data as JSON documents in the database, and — for search-enabled domains (Map, NPC, Monster, Reactor, Item String) — a corresponding search-index row in the same transaction

Ingest runs inside a dedicated Kubernetes Job (`MODE=ingest`), created by `POST /api/data/process` (see `docs/rest.md`). The Job fetches the target scope's WZ archives from MinIO, runs a `String` prerequisite worker (populates the item-name registry other workers resolve names from), then fans out the remaining Workers in parallel (bounded by `INGEST_MAX_PARALLEL`). Each Worker wraps one or more of the per-type processors listed below and, for some archives (Character, Map, Mob, Npc, Reactor, Skill, UI), also derives image/atlas assets to MinIO. The Worker set is:

- `workers.Item` (Item.wz) — cash, consumable, etc, pet, setup
- `workers.Mob` (Mob.wz) — monster
- `workers.Npc` (Npc.wz) — npc
- `workers.Reactor` (Reactor.wz) — reactor
- `workers.Skill` (Skill.wz) — skill, mob skill
- `workers.Quest` (Quest.wz) — quest
- `workers.String` (String.wz) — item-string search index (prerequisite)
- `workers.Map` (Map.wz) — map
- `workers.Character` (Character.wz) — equipment, face, hair, character template
- `workers.UI` (UI.wz) — world-icon assets only (no documents)
- `workers.Commodity` (Etc.wz) — commodity

Each Worker delegates to the same per-type processors as the legacy path:
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

A separate, legacy path also exists in code: a Kafka consumer on `COMMAND_TOPIC_DATA` (`START_WORKER` command) invokes `data.ProcessorImpl.StartWorker`, which parses a local `ZIP_DIR`-rooted XML tree via the same per-type `Register*` functions and, on success, emits a `DATA_UPDATED` event (see `docs/kafka.md`). This path remains registered at startup but nothing in the current codebase produces a `START_WORKER` command — `data.ProcessorImpl.ProcessData` (the only in-repo producer) has no caller.

Each processor's registration is idempotent per document id: `document.Storage.Add` upserts on `(tenant_id, type, document_id)`.

## Canonical

### Responsibility

The canonical domain defines a reserved, version-scoped "shared" tenant identity used to anchor cross-tenant baseline content (ingested via `SCOPE=shared`, published/restored via the [Baseline](#baseline) domain). It is not a real tenant.

### Core Models

#### Canonical Tenant Id
A deterministic UUID v5, derived from `(region, majorVersion, minorVersion)` via a fixed namespace. Two calls with identical arguments always return the same id; different region/version combinations return distinct ids.

#### Canonical Tenant UUID (sentinel)
The reserved all-zero UUID (`00000000-0000-0000-0000-000000000000`) that is never a valid target for a tenant-scoped destructive operation (purge), independent of the version-scoped canonical tenant id.

### Invariants

- The canonical namespace must never change once any canonical rows exist in any environment; doing so orphans every existing canonical row.
- `document.Storage`, `searchindex.ResolveTenantId`, and the various `*.AllPagedProvider` methods fall back from the active tenant to its canonical tenant id when the active tenant has no rows of its own, so a tenant provisioned after canonical ingestion still resolves reads.

## Baseline

### Responsibility

The baseline domain publishes the canonical (version-scoped, `SCOPE=shared`) subset of the searchable tables as a portable dump, lists published dumps, and restores a dump into a single target tenant.

### Core Models

#### Dump Header
Records a schema version fingerprint, the source region/majorVersion/minorVersion, the ordered table list included in the dump, and — per table — the exact ordered column list the dump's binary COPY stream was produced with.

#### Tenant Baseline
Tracks, per tenant, which published baseline (region, majorVersion, minorVersion, sha256) was last restored into it, and when.

### Invariants

- A dump covers exactly the tables `documents`, `monster_search_index`, `npc_search_index`, `reactor_search_index`, `map_search_index`, `item_string_search_index`, always scoped to the canonical tenant id for the dump's `(region, major, minor)`.
- Restore verifies the downloaded dump's sha256 against its sidecar object, and the dump header's schema version against the service's current schema version, before mutating any table.
- Restore is destructive per-table for the target tenant only: existing rows for the target tenant in every dump table are deleted and replaced from the dump.
- A restore failure at any point deletes all dump-table rows written for the target tenant so far, leaving the tenant in a "never restored" rather than a partially-restored state.

### Processors

- `Publisher.Publish` — dumps the canonical subset of every table to a tar (header + one binary COPY entry per table) and uploads it plus a sha256 sidecar to the canonical MinIO bucket.
- `Restorer.Restore` — downloads and hash-verifies a dump, validates its schema version, replaces the target tenant's rows in every dump table, `ANALYZE`s each table, and upserts the tenant's `tenant_baselines` row.
- `Lister.List` — enumerates published dumps from the canonical MinIO bucket.

## Tenant Purge

### Responsibility

The tenant purge domain deletes all per-tenant data (Postgres rows and best-effort MinIO objects) for one tenant, refusing to operate on the canonical tenant.

### Invariants

- The all-zero canonical sentinel UUID and the caller's own version-scoped canonical tenant id are both refused as purge targets.
- Purge deletes rows from `documents`, `monster_search_index`, `npc_search_index`, `reactor_search_index`, `map_search_index`, `item_string_search_index`, and `tenant_baselines` for the target tenant in a single transaction; MinIO object removal under the tenant's WZ/assets/renders prefixes is best-effort and logged (not transactional) on failure.

## WZ Input

### Responsibility

The WZ input domain accepts uploaded WZ archives (as a zip containing `.wz` entries) for a tenant or the shared canonical scope, storing each entry into MinIO under the target scope/region/version, and reports aggregate upload status for a scope.

### Invariants

- Upload entries are validated to reject path traversal (`..`, leading `/`, NUL bytes), symlink entries, and any entry not ending in `.wz`.
- `scope=shared` uploads/status reads require the `X-Atlas-Operator: 1` request header; the default (`""` or `"tenant"`) scope is always the caller's own tenant.

## Job

### Responsibility

The job domain exposes the skill list for a MapleStory job class, sourced from `libs/atlas-constants/job`. It is not a WZ-ingested document type.

### Core Models

#### Job Skills
A job id paired with the list of skill ids `libs/atlas-constants/job` associates with that job.
