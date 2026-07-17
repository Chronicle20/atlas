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

**Socket**
- `Handlers` - List of socket handlers with opcode, validator, handler name, and options
- `Writers` - List of socket writers with opcode, writer name, and options

**Characters**
- `Templates` - List of character creation templates defining job index, map, gender, appearance options, and starting items/skills
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

**Commodities**
- `HourlyExpirations` - List of hourly expiration entries with template ID and hours

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

### Processors

**templates.Processor**
- `GetAll` - Retrieves all templates
- `GetById` - Retrieves template by UUID
- `GetByRegionAndVersion` - Retrieves template by region, major version, and minor version
- `Create` - Creates a new template
- `UpdateById` - Updates an existing template
- `DeleteById` - Deletes a template

**preset.Validator**
- `Validate` - Validates a list of character presets against the invariant rules above, returning the (possibly mutated) list and any validation errors

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

### Invariants

- Updates and deletions create history records before modifying data
- On update, `Characters.Presets` is validated against the same preset rules described under the Templates domain's Invariants; violations prevent the update

### Processors

**tenants.Processor**
- `GetAll` - Retrieves all tenants
- `GetById` - Retrieves tenant by UUID
- `GetByRegionAndVersion` - Retrieves tenant by region, major version, and minor version
- `Create` - Creates a new tenant (accepts optional ID)
- `UpdateById` - Updates an existing tenant (creates history record)
- `DeleteById` - Deletes a tenant (creates history record)

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

**ChannelRestModel**
- `Id` - UUID identifier
- `Type` - Service type
- `Tasks` - List of task configurations
- `Tenants` - List of channel tenant configurations with ID, IP address, and world/channel mappings

**GenericRestModel**
- `Id` - UUID identifier
- `Type` - Service type
- `Tasks` - List of task configurations

**Task**
- `Type` - Task type identifier
- `Interval` - Task interval in milliseconds
- `Duration` - Task duration in milliseconds

### Invariants

- Updates and deletions create history records before modifying data
- Service type must be one of the valid types (`login-service`, `channel-service`, `drops-service`)

### Processors

**services.Processor**
- `GetAll` - Retrieves all service configurations
- `GetById` - Retrieves service configuration by UUID
- `Create` - Creates a new service configuration (accepts optional ID)
- `UpdateById` - Updates an existing service configuration (creates history record)
- `DeleteById` - Deletes a service configuration (creates history record)

---

## Seeder

### Responsibility

Imports template configurations from JSON files on startup.

### Processors

**seeder.Seeder**
- `Run` - Executes the seeding process if enabled
- Discovers JSON files in `{SEED_DATA_PATH}/templates/`
- Checks if template exists by region and version
- Skips existing templates
- Imports new templates via templates.Processor
