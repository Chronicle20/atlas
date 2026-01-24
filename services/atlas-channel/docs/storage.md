# Storage Documentation

This service does not use persistent database storage.

## In-Memory Registries

The service maintains the following in-memory registries:

### Session Registry
- Stores active socket sessions per tenant
- Keyed by tenant ID and session UUID
- Contains connection state, encryption keys, and session metadata

### Account Registry
- Tracks logged-in accounts per tenant
- Keyed by tenant and account ID
- Used to prevent duplicate logins

### Server Registry
- Stores registered server instances
- Keyed by tenant, world ID, and channel ID
- Contains IP address and port bindings

## Data Persistence

All persistent data is managed by external services accessed via REST APIs:
- Character data: CHARACTERS service
- Inventory data: INVENTORY service
- Guild data: GUILDS service
- Party data: PARTIES service
- Map state: MAPS service
- Monster state: MONSTERS service
- Drop state: DROPS service

## Migration Rules

Not applicable - no database migrations required.
