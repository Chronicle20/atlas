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

### Maple Life Registry
- Tracks the in-flight Maple Life character-creation dialog for an account with no character yet
- Singleton via `sync.Once`, thread-safe via `sync.RWMutex`
- Keyed by `Key{Tenant, AccountId}`
- Holds one `Entry` (CharacterId, WorldId, ItemId, Slot, UpdateTime, Phase, TransactionId, CandidateName, At)
- Entries older than `SubmittedTTL` (30s) are removed by a per-tenant `Sweep`
- Provides Put/Get/Take/TakeByTransactionId/ClearAccount/Sweep

### Remote Merchant Registry
- Tracks characters who opened an NPC shop via a classification-545 cash item, pending the unlock of their client's exclusive-request lock
- Singleton via `sync.Once`, thread-safe via `sync.RWMutex`
- Keyed by `Key{Tenant, CharacterId}`
- Holds one `Entry` (ItemId, Slot, At)
- Entries older than `TTL` (30s) are removed by a per-tenant `Sweep`
- Provides Put/Take/ClearCharacter/Sweep

### Position Registry
- Tracks the process-local, last-known (x, y) the channel last folded out of a character's movement path
- Singleton via `sync.Once`, thread-safe via `sync.RWMutex`
- Keyed by `Key{Tenant, CharacterId}`
- No TTL and no sweeper; entries are removed on session destroy
- Provides Put/Lookup/Clear

### Ring Cache
- Per-character cache of couple/friendship ring pair halves fetched from atlas-cashshop
- Populated once per character load; a cache miss returns an empty ring set/records rather than issuing a REST call
- Keyed by tenant and character id

### Battleship Ship-HP Store
- Redis-backed counter (`redis.TenantCounter`, namespace `battleship-hp`) tracking the Corsair Battleship's remaining HP pool per character, paired with an in-process `RideMirror` (per-channel-process, tenant-scoped map of characterId to ride state)
- The Redis entry's TTL is effect-derived per ride, with a 35-minute fallback

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
- Evan dragon state: DRAGONS service
- Incubator reward rolls: GACHAPONS service
- Mini-game room state: MINI_GAMES service
- Duey parcel custody: PARCEL service
- Player trade room state: TRADES service
- Ring pair state: CASHSHOP service
- Player report submissions: BAN service
- RPS session state: RPS service

## Migration Rules

Not applicable - no database migrations required.
