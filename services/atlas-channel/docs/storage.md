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

### Shop Scanner Registry
- Tracks per-character owl-of-Minerva (shop scanner) state
- Singleton via `sync.Once`, thread-safe via a single `sync.RWMutex`
- Keyed by `Key{Tenant, CharacterId}`
- Holds two maps: `lastSearch` (`SearchEntry{ItemId}` — the most recent executed search) and `pending` (`PendingEntry{ShopId, OwnerId, MapId}` — an in-flight warp-then-enter)
- Provides SetLastSearch/GetLastSearch, SetPending/GetPending/RemovePending, and ClearCharacter (invoked on session destroy)

### MTS Configuration Registry
- Lazy, per-tenant cache of MTS economic configuration (listing fee, commission, level/duration limits, price floor, page size, bid increment)
- Singleton via `sync.Once`, thread-safe via `sync.RWMutex`
- Keyed by tenant UUID
- A fetch miss or error caches and returns the default configuration so the service never hard-fails on an unconfigured tenant

### Monster Information Cache
- In-process, tenant-scoped TTL cache fronting monster template attack-pattern lookups
- Singleton via `sync.Once`, thread-safe via `sync.RWMutex`
- Keyed by tenant UUID, then monster id
- Positive entries expire after `MONSTER_INFO_CACHE_TTL` (default 5 minutes); not-found lookups are negatively cached for `MONSTER_INFO_CACHE_NEGATIVE_TTL` (default 30 seconds); transient errors are never cached
- Lazy expiry (no sweeper); evicted per-tenant via `EvictTenant` on listener drain
- Disabled entirely via `MONSTER_INFO_CACHE_ENABLED=false` (falls through to a direct upstream fetch every call)

## Data Persistence

All persistent data is managed by external services accessed via REST APIs:
- Character data: CHARACTERS service
- Inventory data: INVENTORY service
- Guild data: GUILDS service
- Party data: PARTIES service
- Map state, character location: MAPS service
- Monster state: MONSTERS service
- Monster template attack data: DATA service
- Monster-book collection and cards: MONSTER_BOOK service
- Mount progression: MOUNTS service
- Summon state: SUMMONS service
- MTS listings, holdings, transactions, wishlist: MTS service
- MTS per-tenant configuration: TENANTS service
- Session-effective character stats: EFFECTIVE_STATS service
- Drop state: DROPS service
- Door state: DOORS service
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
- Personal shop / hired merchant data (shops, listings, blacklist, visits, Frederick status, shop search): MERCHANT service
- Transport routes: ROUTES service
- Weather state: WEATHER service
- World data: WORLDS service
- Static game data: DATA service

## Migration Rules

Not applicable - no database migrations required.
