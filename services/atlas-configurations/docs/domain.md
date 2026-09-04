# Domain

## Templates

### Responsibility

Manages version-specific configuration templates that define schemas for game regions and versions.

### Core Models

**RestModel**
- `Id` - UUID identifier
- `Region` - Game region identifier
- `MajorVersion` - Major version number
- `MinorVersion` - Minor version number
- `UsesPin` - Whether PIN authentication is enabled
- `Socket` - Socket handler and writer configurations
- `Characters` - Character creation templates
- `NPCs` - NPC implementation mappings
- `Worlds` - World configuration list
- `CashShop` - Cash shop configuration
- `MapleLife` - Maple Life character-sale dialog configuration
- `Environment` - Execution environment the row belongs to; server-owned and read-only (always reflects the persisted row, never a client-supplied value)

**ViewRestModel** (read-only projection returned by GET/POST/reseed responses)
- Embeds `RestModel`
- `ShippedRevision` - content hash of the seed file this image ships for the template's region/version; empty when no such file ships
- `StoredRevision` - content hash of the template's stored content
- `SeedDrift` - `true` when `ShippedRevision` and `StoredRevision` disagree; always `false` when no file ships

**Socket**
- `Handlers` - List of socket handlers with opcode, validator, handler name, informational client function name (`fname`), options, and service scopes
- `Writers` - List of socket writers with opcode, writer name, informational client function name (`fname`), options, and service scopes
- `Unsupported` - Names of handlers/writers audited and confirmed absent for this region/version

**Characters**
- `Templates` - List of character creation templates defining job index, sub-job index, map, gender, appearance options, and starting items/skills
- `Presets` - List of character presets (id, and attributes: name, description, tags, jobId, gender, face, hair, hairColor, skinColor, mapId, level, meso, gm, stats, defaultName, equipment, inventory, skills)

**NPCs**
- `NPCId` - NPC identifier
- `Impl` - Implementation name

**Worlds**
- `Name` - World name
- `Flag` - World flag
- `ServerMessage` - Server message
- `EventMessage` - Event message
- `WhyAmIRecommended` - Recommendation text
- `ExpRate` - Experience rate multiplier
- `MesoRate` - Meso rate multiplier
- `ItemDropRate` - Item drop rate multiplier
- `QuestExpRate` - Quest experience rate multiplier

**CashShop**
- `Commodities` - Commodity configuration
- `Surprise` - Cash Shop Surprise box configuration

**Commodities**
- `HourlyExpirations` - List of hourly expiration entries with template ID and hours

**Surprise**
- `BoxTemplateIds` - Cash item template IDs that open as a Surprise box

**MapleLife**
- `Looks` - Per-gender selectable appearance option lists (faces, hairs, hair colors, skin colors)
- `Classes` - List of class entries, one per (class ordinal, gender): job id, level, map id, stats, unspent AP/SP, optional pre-level skill id, starting meso, equipment, and inventory

### Invariants

- On update, presets with an empty `Id` are assigned a generated UUID before validation
- Presets are validated against the following rules; violations are collected and prevent the update:
  - `name` length must be 1..64 characters
  - `description` length must be ≤512 characters
  - `jobId` must be a known job id
  - `gender` must be 0 or 1
  - `level` must be in [1,250]
  - each equipment entry's `templateId` must exist and be equippable (skipped when no tenant context is available)
  - equipment entries must not collide on slot (slot bucket = `templateId / 10000`)
  - each inventory entry's `templateId` must exist (skipped when no tenant context is available)
  - each inventory entry's `quantity` must be ≥1
  - each skill entry's `skillId` must exist (skipped when no tenant context is available)
  - each skill entry's `level` must be in [1,maxLevel] for that skill (skipped when no tenant context is available)
- On create and update, the socket document is validated; violations are collected and prevent the write:
  - each handler/writer entry's name is required
  - each entry's opcode must match `0x`/`0X` followed by 1-4 hex digits
  - a name bound twice to the same numeric opcode within its collection is rejected (the same name at distinct opcodes is allowed)
  - each handler entry requires a non-empty validator (writers do not)
  - each entry's service scopes must be one of the known socket services
  - an unsupported entry's name is required, must not also appear in the defined handlers/writers, and must not be listed twice
- `Id` and `Environment` are excluded from the content hash used to detect drift between a stored template and the file this image ships for its region/version
- A template row belongs to exactly one execution environment. Reads fall back to the row's registered baseline environment when the caller's environment has no row for that region/version key; a UUID lookup is visible to the caller's own environment and its baseline only
- A write (update or delete) is rejected unless the caller's environment matches the target row's environment; a caller with no environment is always authorized (legacy behavior)
- Re-seeding a template replaces its stored content with the file this image ships for its region/version; it fails when the template id does not exist or when no shipped file exists for the row's region/version

### Processors

**templates.Processor**
- `GetAll` - Retrieves all templates
- `GetById` - Retrieves template by UUID
- `GetByRegionAndVersion` - Retrieves template by region, major version, and minor version
- `Create` - Creates a new template
- `UpdateById` - Updates an existing template
- `DeleteById` - Deletes a template
- `ReseedById` - Replaces a template's stored content with the file this image ships for its region/version
- `AllViewProvider` / `ViewByIdProvider` / `ViewByRegionAndVersionProvider` - Read paths that additionally compute `ShippedRevision`, `StoredRevision`, and `SeedDrift` against the shipped-template catalog

**templates.Catalog**
- The set of templates baked into the running image, keyed by (region, majorVersion, minorVersion), loaded once at startup from the seed data directory
- Used both to compute drift on read and to source the content a re-seed writes back

**preset.Validator**
- `Validate` - Validates a list of character presets against the invariant rules above, returning the (possibly mutated) list and any validation errors

---

## drift

**drift**
- The one definition of a comparable revision, shared by both sides of every tenant-vs-template comparison. Operates on marshaled JSON, never on either package's Go types, so structurally identical documents in distinct Go types hash identically
- Sections: `properties` (every comparable key not otherwise named — today `usesPin`), `socket`, `characters`, `npcs`, `cashShop`, `mapleLife`
- Excluded from every revision: `id`, `environment`, `region`, `majorVersion`, `minorVersion`, `worlds`, `diagnostics`
- Empty values (`null`, `[]`, `{}`) are pruned recursively before hashing, so a document that omits a collection and one that carries an empty collection are the same document. `false`, `0` and `""` are values and are never pruned
- A `drift` hash is not comparable with `templates.Revision`: one hashes a struct in field order, the other a map in key order

---

## Tenants

### Responsibility

Manages tenant-specific configurations derived from templates. Maintains history of configuration changes.

### Core Models

**RestModel**
- `Id` - UUID identifier
- `Region` - Game region identifier
- `MajorVersion` - Major version number
- `MinorVersion` - Minor version number
- `UsesPin` - Whether PIN authentication is enabled
- `Socket` - Socket handler and writer configurations
- `Characters` - Character creation templates
- `NPCs` - NPC implementation mappings
- `Worlds` - World configuration list
- `CashShop` - Cash shop configuration
- `MapleLife` - Maple Life character-sale dialog configuration
- `Environment` - Execution environment the row belongs to; server-owned and read-only (always reflects the persisted row, never a client-supplied value)

**ViewRestModel**
- Embeds `RestModel` and adds five read-only computed attributes: `BaselineTemplateId`, `BaselineRevision`, `StoredRevision`, `TemplateDrift`, `SectionDrift`
- `SectionDrift` is a map keyed by section name, always fully populated with all six keys
- Read paths only. The write model bound by POST/PATCH is unchanged, so the computed attributes are ignored on write by omission

### Invariants

- Updates and deletions create history records before modifying data
- On update, `Characters.Presets` is validated against the same preset rules described under the Templates domain's Invariants; violations prevent the update
- On create and update, the socket document is validated against the same rules described under the Templates domain's Invariants; violations prevent the write
- A tenant row belongs to exactly one execution environment. Reads (list, by id, by region/version) are scoped to the caller's environment; a caller with no environment sees every row (legacy behavior)
- A write (update or delete) is rejected unless the caller's environment matches the target row's environment; a caller with no environment is always authorized (legacy behavior)
- A tenant's baseline is the template row matching its `(region, majorVersion, minorVersion)` as it stands at read time, resolved through the templates overlay/baseline environment fallback. No baseline resolves to the unknown state — empty revisions, empty baseline id, every drift flag `false` — never to `true`
- Baseline resolution never fails a read: a missing or unreadable template yields the unknown state, not a 404 or 500
- A reset writes only the `tenants` row. The tenant's `id`, `region`, `majorVersion`, `minorVersion`, `environment`, `worlds` and `diagnostics` survive verbatim at every scope
- A reset records history before writing and enqueues the tenant status outbox message in the same transaction, identically to an update
- A reset validates the merged document with the same validators an update runs, but discards the preset validator's id-assignment mutation, so the persisted document is byte-identical to the baseline's content

### Processors

**tenants.Processor**
- `GetAll` - Retrieves all tenants
- `GetById` - Retrieves tenant by UUID
- `GetByRegionAndVersion` - Retrieves tenant by region, major version, and minor version
- `Create` - Creates a new tenant (accepts optional ID)
- `UpdateById` - Updates an existing tenant (creates history record)
- `DeleteById` - Deletes a tenant (creates history record)
- `ResetById` - Replaces the tenant's stored content, for the requested sections, with its baseline template's (creates history record, enqueues status)
- `AllViewProvider` / `ViewByIdProvider` - Read paths that additionally compute `BaselineTemplateId`, `BaselineRevision`, `StoredRevision`, `TemplateDrift` and `SectionDrift` against the resolved baseline template. The list resolves each distinct region/version baseline once per request

---

## Services

### Responsibility

Manages service-specific configurations with type-specific data models.

### Core Models

**ServiceType**
- `login-service` - Login service configuration
- `channel-service` - Channel service configuration
- `drops-service` - Drops service configuration

**LoginRestModel**
- `Id` - UUID identifier
- `Type` - Service type
- `Tasks` - List of task configurations
- `Tenants` - List of login tenant configurations with ID and port
- `Environment` - Execution environment the row belongs to; server-owned and read-only

**ChannelRestModel**
- `Id` - UUID identifier
- `Type` - Service type
- `Tasks` - List of task configurations
- `Tenants` - List of channel tenant configurations with ID, IP address, and world/channel mappings
- `Environment` - Execution environment the row belongs to; server-owned and read-only

**GenericRestModel**
- `Id` - UUID identifier
- `Type` - Service type
- `Tasks` - List of task configurations
- `Environment` - Execution environment the row belongs to; server-owned and read-only

**Task**
- `Type` - Task type identifier
- `Interval` - Task interval in milliseconds
- `Duration` - Task duration in milliseconds

### Invariants

- Updates and deletions create history records before modifying data
- Service type must be one of the valid types (`login-service`, `channel-service`, `drops-service`)
- A service row belongs to exactly one execution environment. Reads (list, by id) are scoped to the caller's environment; a caller with no environment sees every row (legacy behavior)
- A write (update or delete) is rejected unless the caller's environment matches the target row's environment; a caller with no environment is always authorized (legacy behavior)
- At most one service configuration row may exist per (type, environment) pair

### Processors

**services.Processor**
- `GetAll` - Retrieves all service configurations
- `GetById` - Retrieves service configuration by UUID
- `Create` - Creates a new service configuration (accepts optional ID)
- `UpdateById` - Updates an existing service configuration (creates history record)
- `DeleteById` - Deletes a service configuration (creates history record)

---

## Environments

### Responsibility

Manages the list of execution environments (e.g. the main deployment, a per-PR sparse environment) that templates, tenants, and services are scoped against.

### Core Models

**RestModel**
- `Id` - UUID identifier
- `Name` - Environment's wire identity (e.g. `main`, `pr-123`); the resource's addressable key
- `Baseline` - Name of the environment this one falls back to / derives from
- `Namespace` - Deployment namespace the environment runs in
- `Tenant` - Tenant identifier associated with the environment
- `Overrides` - Map of service name to namespace, naming which services this environment serves out of its own namespace rather than the baseline's
- `Phase` - Lifecycle phase: `PROVISIONING`, `ACTIVE`, `DEACTIVATING`, or `DELETED`

### Invariants

- `Name` is required and must be a well-formed environment id
- `Phase` must be one of `PROVISIONING`, `ACTIVE`, `DEACTIVATING`, `DELETED`
- On update, a phase transition is legal only as a no-op or one step forward along `PROVISIONING` -> `ACTIVE` -> `DEACTIVATING` -> `DELETED`; skipping or reverting is rejected
- On update, a field omitted from the request body retains its previously stored value rather than being zeroed
- On update, the record's `Name` always follows the URL path identity, never the request body

### Processors

**environments.Processor**
- `GetByName` - Retrieves an environment by name
- `AllProvider` - Retrieves all environments, paginated
- `Create` - Creates a new environment
- `UpdateByName` - Updates an existing environment
- `Republish` - Re-emits the persisted record for a name unchanged; used by the heartbeat so consumers can treat topic arrival as liveness

---

## Seeder

### Responsibility

Imports template configurations from JSON files on startup.

### Processors

**seeder.Seeder**
- `Run` - Executes the seeding process if enabled
- Reads shipped template files from a catalog loaded once at startup (`{SEED_DATA_PATH}/templates/`); a file that fails to parse is skipped
- Checks if template exists by region and version
- Skips existing templates; never overwrites an existing row's content (drift correction is the operator-triggered re-seed endpoint, not a startup side effect)
- Imports new templates via templates.Processor
- After template seeding, re-publishes every existing service and tenant row into the outbox so a cold-start or a cluster recovering from a wiped Kafka topic has a complete snapshot to publish
