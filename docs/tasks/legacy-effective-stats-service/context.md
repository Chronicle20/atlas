# Effective Stats Service - Context Document

**Last Updated:** 2026-02-03

---

## Key Files Reference

This document catalogs the critical files needed to understand and implement the effective stats service.

---

## Reference Implementation: atlas-rates

The `atlas-rates` service serves as the architectural template for this implementation.

### Core Architecture Files

| File | Purpose | Key Patterns |
|------|---------|--------------|
| `services/atlas-rates/atlas.com/rates/main.go` | Service entry point | Consumer registration, REST server startup |
| `services/atlas-rates/atlas.com/rates/character/registry.go` | In-memory storage | Thread-safe per-tenant registry, RWMutex locking |
| `services/atlas-rates/atlas.com/rates/character/model.go` | Domain model | Immutable model with factors, defensive copying |
| `services/atlas-rates/atlas.com/rates/character/initializer.go` | Lazy initialization | Idempotent init, service queries on first access |
| `services/atlas-rates/atlas.com/rates/character/processor.go` | Business logic | GetRates with lazy init, factor aggregation |
| `services/atlas-rates/atlas.com/rates/character/resource.go` | REST handler | JSON:API response, world/channel/character path |
| `services/atlas-rates/atlas.com/rates/rate/model.go` | Factor model | Source tracking, multiplicative computation |

### Kafka Consumer Patterns

| File | Events Handled | Pattern |
|------|----------------|---------|
| `kafka/consumer/buff/consumer.go` | APPLIED, EXPIRED | Add/remove factors on buff events |
| `kafka/consumer/asset/consumer.go` | MOVED, DELETED | Track equipped items with rate bonuses |
| `kafka/consumer/character/consumer.go` | MAP_CHANGED | Trigger initialization on map change |
| `kafka/consumer/rate/consumer.go` | RATE_CHANGED | Bulk update for world-wide rate changes |

---

## Character Service (Consumer of Effective Stats)

### Files with TODOs to Resolve

| File | Line | Issue |
|------|------|-------|
| `services/atlas-character/atlas.com/character/character/processor.go` | 976-977 | Uses base INT for max MP growth |
| `services/atlas-character/atlas.com/character/character/processor.go` | 1009 | HP bounds use base MaxHP |
| `services/atlas-character/atlas.com/character/character/processor.go` | 1081 | MP bounds use base MaxMP |

### Key Functions to Modify

```go
// processor.go:976-977 - getMaxMpGrowth()
// Currently:
resMax += uint16(math.Ceil(float64(c.Intelligence()) / 10))
// Should query effective INT from atlas-effective-stats

// processor.go:1009 - ChangeHP()
// Currently:
adjusted = enforceBounds(amount, c.HP(), c.MaxHP(), 0)
// Should use effective MaxHP

// processor.go:1081 - ChangeMP()
// Currently:
adjusted := enforceBounds(amount, c.MP(), c.MaxMP(), 0)
// Should use effective MaxMP
```

### Character Model
`services/atlas-character/atlas.com/character/character/model.go`

Base stats stored:
- `strength`, `dexterity`, `intelligence`, `luck` (uint16)
- `hp`, `mp` (uint16) - current values
- `maxHp`, `maxMp` (uint16) - base maximum values

---

## Data Source Services

### atlas-inventory (Equipment Stats)

| File | Purpose |
|------|---------|
| `services/atlas-inventory/atlas.com/inventory/equipable/model.go` | Equipment stat fields |
| `services/atlas-inventory/atlas.com/inventory/kafka/message/equipable/kafka.go` | MOVED/DELETED events |
| `services/atlas-inventory/atlas.com/inventory/asset/processor.go` | Asset operations |

**Equipment Stat Fields:**
```go
type Model struct {
    strength       uint16
    dexterity      uint16
    intelligence   uint16
    luck           uint16
    hp             uint16
    mp             uint16
    weaponAttack   uint16
    magicAttack    uint16
    weaponDefense  uint16
    magicDefense   uint16
    accuracy       uint16
    avoidability   uint16
    hands          uint16
    speed          uint16
    jump           uint16
    // ...
}
```

### atlas-buffs (Active Buffs)

| File | Purpose |
|------|---------|
| `services/atlas-buffs/atlas.com/buffs/buff/model.go` | Buff with stat changes |
| `services/atlas-buffs/atlas.com/buffs/buff/stat/model.go` | Stat change model |
| `services/atlas-buffs/atlas.com/buffs/character/registry.go` | Active buff registry |
| `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` | APPLIED/EXPIRED events |

**Stat Change Model:**
```go
type Model struct {
    statType string  // "WEAPON_ATTACK", "HYPER_BODY_HP", etc.
    amount   int32
}
```

### atlas-data (Skill Effects, Equipment Data)

| File | Purpose |
|------|---------|
| `services/atlas-data/atlas.com/data/skill/reader.go` | Skill data loading |
| `services/atlas-data/atlas.com/data/skill/rest.go` | Skill REST model |
| `services/atlas-data/atlas.com/data/skill/effect/rest.go` | Skill effect stat bonuses |
| `services/atlas-data/atlas.com/data/equipment/rest.go` | Equipment data model |

**Passive Skill Detection:**
```go
// reader.go - Skills are passive if:
skillType := xml.GetIntegerWithDefault("skillType", 0)
if skillType == 2 {
    buff = true  // Passive/buff skill
}
// Also: action=false indicates passive
```

**Effect Stat Fields:**
```go
type RestModel struct {
    WeaponAttack      int16
    MagicAttack       int16
    WeaponDefense     int16
    MagicDefense      int16
    Accuracy          int16
    Avoidability      int16
    Speed             int16
    Jump              int16
    HP                uint16
    MP                uint16
    // ...
}
```

---

## Session Tracking

### atlas-character Session

| File | Purpose |
|------|---------|
| `services/atlas-character/atlas.com/character/session/model.go` | Session state (LoggedIn, LoggedOut, Transition) |
| `services/atlas-character/atlas.com/character/session/registry.go` | Active session tracking |
| `services/atlas-character/atlas.com/character/kafka/consumer/session/consumer.go` | Session event handling |

### Session Events
**Topic:** `EVENT_TOPIC_SESSION_STATUS`
```go
type StatusEvent struct {
    SessionId   uuid.UUID
    AccountId   uint32
    CharacterId uint32
    WorldId     world.Id
    ChannelId   channel.Id
    Issuer      string  // "LOGIN" or "CHANNEL"
    Type        string  // "CREATED" or "DESTROYED"
}
```

---

## Kafka Topics Reference

### Topics to Consume

| Topic Environment Variable | Purpose | Event Types |
|---------------------------|---------|-------------|
| `EVENT_TOPIC_SESSION_STATUS` | Character login/logout | CREATED, DESTROYED |
| `EVENT_TOPIC_CHARACTER_BUFF_STATUS` | Buff application/expiration | APPLIED, EXPIRED |
| `EVENT_TOPIC_ASSET_STATUS` | Equipment equip/unequip | MOVED, DELETED, RELEASED |
| `EVENT_TOPIC_CHARACTER_STATUS` | Base stat changes | STAT_CHANGED |
| `EVENT_TOPIC_SKILL_STATUS` | Skill level changes | (May need to create) |

### Event Message Schemas

**Buff Status Event:**
```go
type StatusEvent[E any] struct {
    WorldId     byte
    CharacterId uint32
    Type        string  // "APPLIED" or "EXPIRED"
    Body        E
}

type AppliedStatusEventBody struct {
    FromId    uint32
    SourceId  int32
    Duration  int32
    Changes   []StatChange
    CreatedAt time.Time
    ExpiresAt time.Time
}
```

**Asset Status Event:**
```go
type StatusEvent[E any] struct {
    Id   uint32  // Asset ID
    Type string  // "MOVED", "DELETED", "RELEASED"
    Body E
}
```

---

## Architectural Decisions

### Decision 1: In-Memory Only (No Database)
**Rationale:** Following atlas-rates pattern. Stats are derived from other services' data. On restart, lazy initialization rebuilds state from source services.
**Trade-off:** First query after restart has higher latency.

### Decision 2: Event-Driven Updates
**Rationale:** Avoid polling. React immediately to changes.
**Implementation:** Kafka consumers update registry on each relevant event.

### Decision 3: Pre-Computed Stats
**Rationale:** Stat queries are frequent; recomputation on every query would be expensive.
**Implementation:** Compute effective stats when bonuses change, store result in registry.

### Decision 4: Source Tracking
**Rationale:** Debugging and auditing require knowing which items/skills/buffs contribute to stats.
**Implementation:** Each `StatBonus` includes a source identifier (e.g., "equipment:12345").

### Decision 5: Both Additive and Multiplicative Support
**Rationale:** Skills like Maple Warrior provide percentage bonuses to all stats.
**Implementation:** StatBonus has both `amount` (flat) and `multiplier` (percentage) fields.

---

## Dependency Graph

```
atlas-effective-stats
    │
    ├── Consumes (Kafka)
    │   ├── atlas-sessions → SESSION_STATUS
    │   ├── atlas-buffs → BUFF_STATUS
    │   ├── atlas-inventory → ASSET_STATUS
    │   └── atlas-character → STAT_CHANGED
    │
    ├── Queries (REST)
    │   ├── atlas-character → Base stats, skills
    │   ├── atlas-inventory → Equipped items
    │   ├── atlas-buffs → Active buffs
    │   └── atlas-data → Skill effects, equipment data
    │
    └── Consumed By (REST)
        ├── atlas-character → HP/MP bounds
        └── atlas-saga-orchestrator → Damage calculations
```

---

## Testing Strategy

### Unit Tests
- Stat aggregation with various bonus combinations
- Multiplicative vs additive ordering
- Registry thread safety
- Builder invariants

### Integration Tests
- Equipment equip → stat increase
- Buff apply/expire → stat change
- Session logout → registry cleanup
- Service restart → correct recomputation

### Test Data
Known MapleStory v83 equipment and skill combinations with expected results for validation.

---

## Configuration

### Environment Variables (Expected)

```bash
# Kafka
KAFKA_BROKERS=localhost:9092
COMMAND_TOPIC_*=...
EVENT_TOPIC_SESSION_STATUS=session-status
EVENT_TOPIC_CHARACTER_BUFF_STATUS=character-buff-status
EVENT_TOPIC_ASSET_STATUS=asset-status
EVENT_TOPIC_CHARACTER_STATUS=character-status

# REST
PORT=8080

# Service URLs (for REST clients)
CHARACTER_SERVICE_URL=http://atlas-character:8080
INVENTORY_SERVICE_URL=http://atlas-inventory:8080
BUFFS_SERVICE_URL=http://atlas-buffs:8080
DATA_SERVICE_URL=http://atlas-data:8080

# Tracing
JAEGER_HOST=localhost:6831
```

---

## Deployment Files

### Kubernetes
Each service has a single YAML file at its root: `services/atlas-{service}/atlas-{service}.yml`

**Reference:** `services/atlas-rates/atlas-rates.yml`

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-effective-stats
  namespace: atlas
spec:
  replicas: 1
  selector:
    matchLabels:
      app: atlas-effective-stats
  template:
    metadata:
      labels:
        app: atlas-effective-stats
    spec:
      containers:
      - name: effective-stats
        image: ghcr.io/chronicle20/atlas-effective-stats/atlas-effective-stats:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: atlas-env
        env:
        - name: LOG_LEVEL
          value: "debug"
---
apiVersion: v1
kind: Service
metadata:
  name: atlas-effective-stats
  namespace: atlas
spec:
  selector:
    app: atlas-effective-stats
  ports:
  - protocol: TCP
    port: 8080
```

**File to create:** `services/atlas-effective-stats/atlas-effective-stats.yml`

### GitHub Actions
Services are auto-detected via `.github/config/services.json`. Add entry:

```json
{
  "name": "atlas-effective-stats",
  "type": "go-service",
  "path": "services/atlas-effective-stats",
  "module_path": "services/atlas-effective-stats/atlas.com/effective-stats",
  "docker_image": "ghcr.io/chronicle20/atlas-effective-stats/atlas-effective-stats",
  "docker_context": "."
}
```

**File to modify:** `.github/config/services.json`

Workflows auto-detect changes:
- `pr-validation.yml` - Tests and builds Docker images on PR
- `main-publish.yml` - Publishes Docker images on merge to main

---

## Related Documentation

- `services/atlas-rates/docs/kafka.md` - Reference Kafka patterns
- `services/atlas-character/docs/kafka.md` - Character events
- `services/atlas-buffs/docs/kafka.md` - Buff events (if exists)
- `docs/TODO.md` - Original TODO items driving this service
