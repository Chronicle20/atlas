# Domain

## Tenant

### Responsibility

Represents a game server tenant with identification, region, and version information.

### Core Models

**Model**
- `id` (uuid.UUID): Unique identifier
- `name` (string): Tenant name
- `region` (string): Tenant region
- `majorVersion` (uint16): Major version number
- `minorVersion` (uint16): Minor version number
- `environment` (string): Owning environment. Empty is the legacy value.

### Invariants

- Name is required
- Region is required
- A tenant row belongs to exactly one environment. A write (update or delete)
  is rejected unless the caller's environment matches the target row's
  environment, except that a caller with no environment (legacy) is always
  authorized. Reads are filtered to the caller's environment; a caller with
  no environment sees every tenant, unfiltered.

### Processors

**Processor**
- `Create`: Creates a new tenant
- `CreateAndEmit`: Creates a new tenant and emits a Kafka event
- `Update`: Updates an existing tenant
- `UpdateAndEmit`: Updates an existing tenant and emits a Kafka event
- `Delete`: Deletes a tenant
- `DeleteAndEmit`: Deletes a tenant and emits a Kafka event
- `GetById`: Retrieves a tenant by ID
- `ByIdProvider`: Returns a provider for a tenant by ID
- `AllProvider`: Returns a paged provider for all tenants

---

## Configuration

### Responsibility

Manages tenant-specific configuration resources including routes, vessels, instance routes, RPS rewards, MTS configs, trade configs, rankings, kite configs, and imprint configs. RPS rewards, MTS configs, trade configs, and imprint configs support seeding from JSON files on the filesystem via the Processor's seed operations below; routes, vessels, and instance routes are seeded from the shared `deploy/seed/shared/all` catalog via `libs/atlas-seeder` (see `configuration/seed`) instead — see the REST API docs for the header-scoped `POST|GET /api/tenants/configurations/<routes|vessels|instance-routes>/seed[/status]` endpoints. Rankings and kite configs have no seed operation.

### Core Models

**Model**
- `id` (uuid.UUID): Unique identifier
- `tenantID` (uuid.UUID): Associated tenant ID
- `resourceName` (string): Type of resource (routes, vessels, instance-routes, mts-configs)
- `resourceData` (json.RawMessage): JSON data for the resource

**SeedResult**
- `deletedCount` (int): Number of existing resources deleted
- `createdCount` (int): Number of resources created
- `failedCount` (int): Number of resources that failed to create
- `errors` ([]string): Error messages for failed operations

### Invariants

- TenantID is required
- ResourceName is required

### Processors

**Processor (Route Operations)**
- `CreateRoute`: Creates a new route configuration
- `CreateRouteAndEmit`: Creates a new route configuration and emits a Kafka event
- `UpdateRoute`: Updates an existing route configuration
- `UpdateRouteAndEmit`: Updates an existing route configuration and emits a Kafka event
- `DeleteRoute`: Deletes a route configuration
- `DeleteRouteAndEmit`: Deletes a route configuration and emits a Kafka event
- `GetRouteById`: Retrieves a route by ID
- `GetAllRoutes`: Retrieves all routes for a tenant
- `RouteByIdProvider`: Returns a provider for a route by ID
- `AllRoutesProvider`: Returns a provider for all routes for a tenant

**Processor (Vessel Operations)**
- `CreateVessel`: Creates a new vessel configuration
- `CreateVesselAndEmit`: Creates a new vessel configuration and emits a Kafka event
- `UpdateVessel`: Updates an existing vessel configuration
- `UpdateVesselAndEmit`: Updates an existing vessel configuration and emits a Kafka event
- `DeleteVessel`: Deletes a vessel configuration
- `DeleteVesselAndEmit`: Deletes a vessel configuration and emits a Kafka event
- `GetVesselById`: Retrieves a vessel by ID
- `GetAllVessels`: Retrieves all vessels for a tenant
- `VesselByIdProvider`: Returns a provider for a vessel by ID
- `AllVesselsProvider`: Returns a provider for all vessels for a tenant

**Processor (Instance Route Operations)**
- `CreateInstanceRoute`: Creates a new instance route configuration
- `CreateInstanceRouteAndEmit`: Creates a new instance route configuration and emits a Kafka event
- `UpdateInstanceRoute`: Updates an existing instance route configuration
- `UpdateInstanceRouteAndEmit`: Updates an existing instance route configuration and emits a Kafka event
- `DeleteInstanceRoute`: Deletes an instance route configuration
- `DeleteInstanceRouteAndEmit`: Deletes an instance route configuration and emits a Kafka event
- `GetInstanceRouteById`: Retrieves an instance route by ID
- `GetAllInstanceRoutes`: Retrieves all instance routes for a tenant
- `InstanceRouteByIdProvider`: Returns a provider for an instance route by ID
- `AllInstanceRoutesProvider`: Returns a provider for all instance routes for a tenant

**Processor (MTS Config Operations)**
- `CreateMtsConfig`: Creates a new MTS config configuration
- `CreateMtsConfigAndEmit`: Creates a new MTS config configuration and emits a Kafka event
- `UpdateMtsConfig`: Updates an existing MTS config configuration
- `UpdateMtsConfigAndEmit`: Updates an existing MTS config configuration and emits a Kafka event
- `DeleteMtsConfig`: Deletes an MTS config configuration
- `DeleteMtsConfigAndEmit`: Deletes an MTS config configuration and emits a Kafka event
- `GetMtsConfigById`: Retrieves an MTS config by ID
- `GetAllMtsConfigs`: Retrieves all MTS configs for a tenant
- `MtsConfigByIdProvider`: Returns a provider for an MTS config by ID
- `AllMtsConfigsProvider`: Returns a provider for all MTS configs for a tenant

**Processor (RPS Reward Operations)**
- `CreateRpsReward`: Creates a new RPS reward configuration
- `CreateRpsRewardAndEmit`: Creates a new RPS reward configuration and emits a Kafka event
- `UpdateRpsReward`: Updates an existing RPS reward configuration
- `UpdateRpsRewardAndEmit`: Updates an existing RPS reward configuration and emits a Kafka event
- `DeleteRpsReward`: Deletes an RPS reward configuration
- `DeleteRpsRewardAndEmit`: Deletes an RPS reward configuration and emits a Kafka event
- `GetRpsRewardById`: Retrieves an RPS reward by ID
- `GetAllRpsRewards`: Retrieves all RPS rewards for a tenant
- `RpsRewardByIdProvider`: Returns a provider for an RPS reward by ID
- `AllRpsRewardsProvider`: Returns a provider for all RPS rewards for a tenant

**Processor (Trade Config Operations)**
- `CreateTradeConfig`: Creates a new trade configuration
- `CreateTradeConfigAndEmit`: Creates a new trade configuration and emits a Kafka event
- `UpdateTradeConfig`: Updates an existing trade configuration
- `UpdateTradeConfigAndEmit`: Updates an existing trade configuration and emits a Kafka event
- `DeleteTradeConfig`: Deletes a trade configuration
- `DeleteTradeConfigAndEmit`: Deletes a trade configuration and emits a Kafka event
- `GetTradeConfigById`: Retrieves a trade configuration by ID
- `GetAllTradeConfigs`: Retrieves all trade configurations for a tenant
- `TradeConfigByIdProvider`: Returns a provider for a trade configuration by ID
- `AllTradeConfigsProvider`: Returns a provider for all trade configurations for a tenant

**Processor (Rankings Operations)**

One rankings configuration per tenant.
- `CreateRankings`: Creates the rankings configuration for a tenant
- `CreateRankingsAndEmit`: Creates the rankings configuration and emits a Kafka event
- `UpdateRankings`: Updates the rankings configuration for a tenant
- `UpdateRankingsAndEmit`: Updates the rankings configuration and emits a Kafka event
- `DeleteRankings`: Deletes the rankings configuration for a tenant
- `DeleteRankingsAndEmit`: Deletes the rankings configuration and emits a Kafka event
- `GetRankings`: Retrieves the rankings configuration for a tenant
- `RankingsProvider`: Returns a provider for the rankings configuration

**Processor (Kite Config Operations)**

One kite-configs configuration per tenant.
- `CreateKiteConfig`: Creates the kite-configs configuration for a tenant
- `CreateKiteConfigAndEmit`: Creates the kite-configs configuration and emits a Kafka event
- `UpdateKiteConfig`: Updates the kite-configs configuration for a tenant
- `UpdateKiteConfigAndEmit`: Updates the kite-configs configuration and emits a Kafka event
- `DeleteKiteConfig`: Deletes the kite-configs configuration for a tenant
- `DeleteKiteConfigAndEmit`: Deletes the kite-configs configuration and emits a Kafka event
- `GetKiteConfig`: Retrieves the kite-configs configuration for a tenant
- `KiteConfigProvider`: Returns a provider for the kite-configs configuration

**Processor (Imprint Config Operations)**
- `CreateImprintConfig`: Creates a new imprint configuration
- `CreateImprintConfigAndEmit`: Creates a new imprint configuration and emits a Kafka event
- `UpdateImprintConfig`: Updates an existing imprint configuration
- `UpdateImprintConfigAndEmit`: Updates an existing imprint configuration and emits a Kafka event
- `DeleteImprintConfig`: Deletes an imprint configuration
- `DeleteImprintConfigAndEmit`: Deletes an imprint configuration and emits a Kafka event
- `GetImprintConfigById`: Retrieves an imprint configuration by ID
- `GetAllImprintConfigs`: Retrieves all imprint configurations for a tenant
- `ImprintConfigByIdProvider`: Returns a provider for an imprint configuration by ID
- `AllImprintConfigsProvider`: Returns a provider for all imprint configurations for a tenant

**Processor (Seed Operations)**
- `SeedRpsRewards`: Deletes all existing RPS rewards for a tenant and loads them from seed files
- `SeedMtsConfigs`: Deletes all existing MTS configs for a tenant and loads them from seed files
- `SeedTradeConfigs`: Deletes all existing trade configs for a tenant and loads them from seed files
- `SeedImprintConfigs`: Deletes all existing imprint configs for a tenant and loads them from seed files

Routes, vessels, and instance routes are no longer seeded through the
`Processor` (the former `SeedRoutes` / `SeedInstanceRoutes` / `SeedVessels`
methods and their path-scoped `POST /tenants/{tenantId}/configurations/<res>/seed`
endpoints are gone). They are seeded via `libs/atlas-seeder`'s
`Subdomain`/`Group` abstractions in `configuration/seed` instead, reading
`deploy/seed/shared/all/<res>/*.json`. See the REST API docs for the
replacement endpoints.
