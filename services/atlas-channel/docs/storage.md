# Storage Documentation

This service does not use persistent database storage.

## In-Memory Registries

The service maintains the following in-memory registries:

### Session Registry
- Stores active socket sessions per tenant
- Keyed by tenant ID and session UUID
- Contains connection state, encryption keys, field location, and session metadata
- Thread-safe via internal synchronization

### Account Registry
- Tracks logged-in accounts per tenant
- Keyed by tenant and account ID
- Used to prevent duplicate logins
- Initialized from external ACCOUNTS service on startup

### Server Registry
- Stores registered server instances
- Singleton via `sync.Once`, thread-safe via `sync.RWMutex`
- Contains a slice of `server.Model` entries
- Each entry holds tenant, channel model, IP address, and port
- Provides Register and GetAll operations

## Data Persistence

All persistent data is managed by external services accessed via REST APIs:
- Character data: CHARACTERS service
- Inventory data: INVENTORY service
- Guild data: GUILDS service
- Party data: PARTIES service
- Map state: MAPS service
- Monster state: MONSTERS service
- Drop state: DROPS service
- Reactor state: REACTORS service
- Pet data: PETS service
- Quest progress: QUESTS service
- Skill data: SKILLS service
- Storage data: STORAGE service
- Buddy list: BUDDIES service
- Buff data: BUFFS service
- Cash shop: CASHSHOP service
- Note data: NOTES service
- Messenger data: MESSENGERS service
- Chair state: CHAIRS service
- Chalkboard state: CHALKBOARDS service
- NPC shop data: NPC_SHOP service
- Transport routes: ROUTES service
- Weather state: WEATHER service
- World data: WORLDS service
- Static game data: DATA service

## Migration Rules

Not applicable - no database migrations required.
