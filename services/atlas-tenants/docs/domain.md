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

### Invariants

- Name is required
- Region is required

### Processors

**Processor**
- `Create`: Creates a new tenant
- `CreateAndEmit`: Creates a new tenant and emits a Kafka event
- `Update`: Updates an existing tenant
- `UpdateAndEmit`: Updates an existing tenant and emits a Kafka event
- `Delete`: Deletes a tenant
- `DeleteAndEmit`: Deletes a tenant and emits a Kafka event
- `GetById`: Retrieves a tenant by ID
- `GetAll`: Retrieves all tenants
- `ByIdProvider`: Returns a provider for a tenant by ID
- `AllProvider`: Returns a provider for all tenants

---

## Configuration

### Responsibility

Manages tenant-specific configuration resources including routes and vessels.

### Core Models

**Model**
- `id` (uuid.UUID): Unique identifier
- `tenantID` (uuid.UUID): Associated tenant ID
- `resourceName` (string): Type of resource (routes, vessels)
- `resourceData` (json.RawMessage): JSON data for the resource

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
