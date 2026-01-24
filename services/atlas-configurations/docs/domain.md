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

**Socket**
- `Handlers` - List of socket handlers with opcode, validator, handler name, and options
- `Writers` - List of socket writers with opcode, writer name, and options

**Characters**
- `Templates` - List of character creation templates defining job index, map, gender, appearance options, and starting items/skills

**NPCs**
- `NPCId` - NPC identifier
- `Impl` - Implementation name

**Worlds**
- `Name` - World name
- `Flag` - World flag
- `ServerMessage` - Server message
- `EventMessage` - Event message
- `WhyAmIRecommended` - Recommendation text

### Processors

**templates.Processor**
- `GetAll` - Retrieves all templates
- `GetById` - Retrieves template by UUID
- `GetByRegionAndVersion` - Retrieves template by region, major version, and minor version
- `Create` - Creates a new template
- `UpdateById` - Updates an existing template
- `DeleteById` - Deletes a template

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

### Invariants

- Updates and deletions create history records before modifying data

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
- `Tasks` - List of task configurations
- `Tenants` - List of login tenant configurations with ID and port

**ChannelRestModel**
- `Id` - UUID identifier
- `Tasks` - List of task configurations
- `Tenants` - List of channel tenant configurations with ID, IP address, and world/channel mappings

**GenericRestModel**
- `Id` - UUID identifier
- `Tasks` - List of task configurations

**Task**
- `Type` - Task type identifier
- `Interval` - Task interval in milliseconds
- `Duration` - Task duration in milliseconds

### Processors

**services.Processor**
- `GetAll` - Retrieves all service configurations
- `GetById` - Retrieves service configuration by UUID

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
