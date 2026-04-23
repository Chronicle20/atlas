# Effective Stats Service Implementation Plan

**Last Updated:** 2026-02-03

---

## Executive Summary

This document outlines the implementation plan for a new `atlas-effective-stats` service that computes and maintains effective (temporary) character statistics in real-time. The service addresses a critical gap in the Atlas architecture where stat calculations currently use only base character statistics, ignoring equipment bonuses, passive skill bonuses, and active buff modifiers.

The service will:
- Maintain an in-memory projection of effective stats for all logged-in characters
- React to Kafka events for equipment changes, buff applications, and stat updates
- Expose a REST API for other services to query effective stats
- Support both additive and multiplicative stat bonuses
- Rebuild state lazily on service restart (no database persistence)

**Target Stats (14 Core Stats):**
- Strength, Dexterity, Luck, Intelligence
- Maximum HP, Maximum MP
- Weapon Attack, Weapon Defense
- Magic Attack, Magic Defense
- Accuracy, Avoidability
- Speed, Jump

---

## Current State Analysis

### Problem Statement

The character service calculates HP/MP bounds and other stat-dependent values using only base statistics stored in the character model. This creates incorrect behavior in several scenarios:

1. **Max MP Calculation** (`processor.go:976-977`): Uses `c.Intelligence()` which returns base INT only
2. **HP Change Bounds** (`processor.go:1009`): Clamps HP to `c.MaxHP()` ignoring buff-enhanced max HP
3. **MP Change Bounds** (`processor.go:1081`): Clamps MP to `c.MaxMP()` ignoring buff-enhanced max MP

### Current Architecture Gaps

| Component | Current State | Gap |
|-----------|---------------|-----|
| Equipment bonuses | Stored in `atlas-inventory` equipable model | NOT integrated into character stat calculations |
| Buff modifiers | Managed by `atlas-buffs` service | NOT incorporated into effective stats |
| Passive skills | Effects defined in `atlas-data` skill effects | NOT applied to character totals |
| Stat queries | Services query `atlas-character` for base stats | No way to get computed effective stats |

### Reference Architecture: atlas-rates

The `atlas-rates` service provides an excellent template for this implementation:
- **Registry Pattern**: Thread-safe per-tenant character model storage
- **Factor Aggregation**: Combines multiple sources (world, buffs, items) into computed values
- **Lazy Initialization**: Rebuilds state on first query after restart
- **Kafka Consumers**: Reacts to buff, asset, and session events
- **REST API**: Simple query endpoint for computed values

---

## Proposed Future State

### Architecture Overview

```
                                    ┌─────────────────────────┐
                                    │   atlas-effective-stats │
                                    │                         │
    Kafka Events ─────────────────▶ │  ┌─────────────────┐    │
    - SESSION_STATUS (login/logout) │  │    Registry     │    │
    - BUFF_STATUS (apply/expire)    │  │  (In-Memory)    │    │
    - ASSET_STATUS (equip/unequip)  │  │                 │    │
    - STAT_CHANGED (base stat mod)  │  │  tenant -> map  │    │
    - SKILL_CHANGED (skill up)      │  │  charId -> Model│    │
                                    │  └────────┬────────┘    │
                                    │           │             │
                                    │           ▼             │
                                    │  ┌─────────────────┐    │
                                    │  │   Processor     │    │
                                    │  │  - Aggregate()  │    │◀──── REST Query
                                    │  │  - GetStats()   │    │      GET /api/effective-stats/
                                    │  └─────────────────┘    │      characters/{id}/stats
                                    │                         │
                                    └─────────────────────────┘
```

### Domain Model

```go
// Effective stats for a single character
type Model struct {
    tenant       tenant.Model
    worldId      byte
    channelId    byte
    characterId  uint32

    // Base stats (from character service)
    baseStats    BaseStats

    // Bonuses by source
    equipmentBonuses  []StatBonus
    passiveBonuses    []StatBonus
    buffBonuses       []StatBonus

    // Cached computed totals
    computed     ComputedStats
    lastUpdated  time.Time
}

type StatBonus struct {
    source     string       // "equipment:12345", "passive:1000001", "buff:2311003"
    statType   StatType     // STR, DEX, LUK, INT, etc.
    amount     int32        // Flat bonus (+20)
    multiplier float64      // Percentage bonus (1.10 = +10%)
}

type ComputedStats struct {
    Strength       uint32
    Dexterity      uint32
    Luck           uint32
    Intelligence   uint32
    MaxHP          uint32
    MaxMP          uint32
    WeaponAttack   uint32
    WeaponDefense  uint32
    MagicAttack    uint32
    MagicDefense   uint32
    Accuracy       uint32
    Avoidability   uint32
    Speed          uint32
    Jump           uint32
}
```

### Calculation Formula

For each stat type, the effective value is computed as:

```
effective = floor((base + equipment_flat + passive_flat + buff_flat) * (1.0 + equipment_mult + passive_mult + buff_mult))
```

Where:
- `base` = Character's base stat from `atlas-character`
- `*_flat` = Sum of all additive bonuses from that source
- `*_mult` = Sum of all percentage bonuses from that source (as decimals, e.g., 10% = 0.10)

**Example for Strength:**
```
Base STR: 50
Equipment: Sword (+15 STR)
Passive: None
Buff: Maple Warrior (+10% all stats)

effective_str = floor((50 + 15) * (1.0 + 0.10)) = floor(65 * 1.10) = floor(71.5) = 71
```

**Example for Max HP with multiple multipliers:**
```
Base Max HP: 5000
Equipment: Medal (+500 HP)
Passive: Improved Max HP (+200 HP)
Buff: Hyper Body (+60% Max HP)
Buff: Maple Warrior (+10% all stats)

effective_maxhp = floor((5000 + 500 + 200) * (1.0 + 0.60 + 0.10))
                = floor(5700 * 1.70)
                = floor(9690)
                = 9690
```

---

## Implementation Phases

### Phase 1: Core Service Foundation (Effort: L)

Establish the base service structure following Atlas backend conventions.

**Deliverables:**
- Service scaffolding with main.go, service lifecycle
- Domain models (model.go, builder.go)
- In-memory registry with tenant isolation
- Basic processor functions

### Phase 2: Data Source Integration (Effort: XL)

Implement providers to fetch data from dependent services.

**Deliverables:**
- REST client for `atlas-character` (base stats)
- REST client for `atlas-inventory` (equipped items)
- REST client for `atlas-buffs` (active buffs)
- REST client for `atlas-data` (passive skill effects)
- Data aggregation processor

### Phase 3: Event-Driven Updates (Effort: L)

React to changes via Kafka consumers.

**Deliverables:**
- Session status consumer (initialize on login, cleanup on logout)
- Buff status consumer (apply/expire buff bonuses)
- Asset status consumer (equip/unequip equipment bonuses)
- Character stat changed consumer (base stat updates)
- Skill changed consumer (passive skill level changes)

### Phase 4: REST API & Deployment (Effort: L)

Expose effective stats via JSON:API REST endpoint and configure deployment.

**Deliverables:**
- REST resource handler
- JSON:API response model
- Ingress configuration
- Kubernetes deployment YAML
- GitHub Actions workflow updates
- Service documentation

### Phase 5: Consumer Integration (Effort: M)

Integrate the service with existing consumers.

**Deliverables:**
- Update `atlas-character` to query effective stats
- Update `atlas-saga-orchestrator` for damage calculations
- Resolve TODO items in character service

---

## Detailed Tasks

### Phase 1: Core Service Foundation

#### 1.1 Create Service Scaffolding
**Effort:** M | **Priority:** P0 | **Blocks:** All subsequent tasks

**Acceptance Criteria:**
- [ ] Service directory created at `services/atlas-effective-stats/`
- [ ] `main.go` with standard Atlas initialization pattern
- [ ] `go.mod` with required dependencies
- [ ] Service teardown manager for graceful shutdown
- [ ] Logger and tracing initialization
- [ ] Dockerfile following existing service patterns

**Files to Create:**
```
services/atlas-effective-stats/
├── atlas.com/effective-stats/
│   ├── main.go
│   ├── service/
│   │   └── teardown.go
│   ├── logger/
│   │   └── logger.go
│   └── tracing/
│       └── tracing.go
├── Dockerfile
└── go.mod
```

#### 1.2 Define Domain Models
**Effort:** M | **Priority:** P0 | **Depends On:** 1.1

**Acceptance Criteria:**
- [ ] Immutable `Model` struct with all stat types
- [ ] `StatBonus` struct for source-tracked bonuses
- [ ] `ComputedStats` struct for final computed values
- [ ] `StatType` enum for the 14 core stats
- [ ] Builder pattern for model construction
- [ ] All accessors return copies (defensive copying)

**Files to Create:**
- `stat/model.go` - Stat types and bonus model
- `character/model.go` - Character effective stats model
- `character/builder.go` - Fluent builder

#### 1.3 Implement Registry Pattern
**Effort:** M | **Priority:** P0 | **Depends On:** 1.2

**Acceptance Criteria:**
- [ ] Thread-safe registry with per-tenant isolation
- [ ] RWMutex locking strategy matching atlas-rates
- [ ] `GetOrCreate()` for lazy initialization
- [ ] `Update()` for modifying character stats
- [ ] `Delete()` for session cleanup
- [ ] `GetAll()` for debugging/admin purposes

**Files to Create:**
- `character/registry.go`

#### 1.4 Create Base Processor
**Effort:** S | **Priority:** P0 | **Depends On:** 1.3

**Acceptance Criteria:**
- [ ] Processor interface with stat retrieval methods
- [ ] `GetEffectiveStats(characterId)` method
- [ ] `Recompute(characterId)` method
- [ ] Provider pattern for data access
- [ ] Pure functions for stat computation

**Files to Create:**
- `character/processor.go`
- `character/provider.go`

---

### Phase 2: Data Source Integration

#### 2.1 Character Service REST Client
**Effort:** M | **Priority:** P0 | **Depends On:** 1.4

**Acceptance Criteria:**
- [ ] REST client to fetch character base stats
- [ ] REST client to fetch character skills
- [ ] Request model matching character service response
- [ ] Error handling for service unavailability
- [ ] Caching strategy for base stat queries

**Files to Create:**
- `character/requests.go`
- `character/rest.go`

#### 2.2 Inventory Service REST Client
**Effort:** M | **Priority:** P0 | **Depends On:** 1.4

**Acceptance Criteria:**
- [ ] REST client to fetch equipped items
- [ ] REST client to fetch equipment stats (from atlas-data)
- [ ] Filter for currently equipped items only
- [ ] Extract stat bonuses from equipment model

**Files to Create:**
- `inventory/requests.go`
- `inventory/rest.go`

#### 2.3 Buffs Service REST Client
**Effort:** M | **Priority:** P0 | **Depends On:** 1.4

**Acceptance Criteria:**
- [ ] REST client to fetch active buffs
- [ ] REST client to fetch skill effects (from atlas-data)
- [ ] Map buff stat changes to StatBonus model
- [ ] Handle multiplicative buffs (Maple Warrior, etc.)

**Files to Create:**
- `buffs/requests.go`
- `buffs/rest.go`

#### 2.4 Passive Skill Processor
**Effort:** L | **Priority:** P0 | **Depends On:** 2.1

**Acceptance Criteria:**
- [ ] Identify passive skills from character skill list
- [ ] Fetch skill effects from atlas-data
- [ ] Extract stat bonuses from passive effects
- [ ] Support additive and multiplicative bonuses
- [ ] Cache passive skill effects to reduce queries

**Files to Create:**
- `passive/processor.go`
- `passive/requests.go`

#### 2.5 Stat Aggregation Engine
**Effort:** L | **Priority:** P0 | **Depends On:** 2.1, 2.2, 2.3, 2.4

**Acceptance Criteria:**
- [ ] Aggregate function combining all bonus sources
- [ ] Correct ordering: additive first, then multiplicative
- [ ] Floor rounding for final values
- [ ] Handle missing/unavailable data gracefully
- [ ] Logging for aggregation steps (debug level)

**Files to Create:**
- `character/aggregator.go`

---

### Phase 3: Event-Driven Updates

#### 3.1 Session Status Consumer
**Effort:** M | **Priority:** P0 | **Depends On:** 2.5

**Acceptance Criteria:**
- [ ] Consumer for `EVENT_TOPIC_SESSION_STATUS`
- [ ] On `CREATED`: Initialize character stats (lazy)
- [ ] On `DESTROYED`: Remove character from registry
- [ ] Filter by `Issuer == "CHANNEL"` like atlas-rates
- [ ] Idempotent handling for duplicate events

**Files to Create:**
- `kafka/consumer/session/consumer.go`
- `kafka/message/session/kafka.go`

#### 3.2 Buff Status Consumer
**Effort:** M | **Priority:** P0 | **Depends On:** 2.5

**Acceptance Criteria:**
- [ ] Consumer for `EVENT_TOPIC_CHARACTER_BUFF_STATUS`
- [ ] On `APPLIED`: Add buff bonuses, recompute
- [ ] On `EXPIRED`: Remove buff bonuses, recompute
- [ ] Map buff stat changes to StatBonus model
- [ ] Handle Maple Warrior and similar % buffs

**Files to Create:**
- `kafka/consumer/buff/consumer.go`
- `kafka/message/buff/kafka.go`

#### 3.3 Asset Status Consumer
**Effort:** M | **Priority:** P0 | **Depends On:** 2.5

**Acceptance Criteria:**
- [ ] Consumer for `EVENT_TOPIC_ASSET_STATUS`
- [ ] On `MOVED` (to equipped slot): Add equipment bonuses
- [ ] On `MOVED` (from equipped slot): Remove equipment bonuses
- [ ] On `DELETED`/`RELEASED`: Remove if equipped
- [ ] Filter for equipment compartment only

**Files to Create:**
- `kafka/consumer/asset/consumer.go`
- `kafka/message/asset/kafka.go`

#### 3.4 Character Status Consumer
**Effort:** S | **Priority:** P1 | **Depends On:** 2.5

**Acceptance Criteria:**
- [ ] Consumer for `EVENT_TOPIC_CHARACTER_STATUS`
- [ ] On `STAT_CHANGED`: Update base stats, recompute
- [ ] Filter for relevant stat types only
- [ ] Avoid unnecessary recomputation

**Files to Create:**
- `kafka/consumer/character/consumer.go`
- `kafka/message/character/kafka.go`

#### 3.5 Skill Status Consumer
**Effort:** M | **Priority:** P1 | **Depends On:** 2.5

**Acceptance Criteria:**
- [ ] Consumer for `EVENT_TOPIC_SKILL_STATUS` (may need to create)
- [ ] On skill level change: Recompute passive bonuses
- [ ] Filter for passive skills only
- [ ] Handle skill reset scenarios

**Files to Create:**
- `kafka/consumer/skill/consumer.go`
- `kafka/message/skill/kafka.go`

---

### Phase 4: REST API & Deployment

#### 4.1 REST Resource Handler
**Effort:** M | **Priority:** P0 | **Depends On:** 2.5

**Acceptance Criteria:**
- [ ] Endpoint: `GET /api/worlds/{worldId}/channels/{channelId}/characters/{characterId}/stats`
- [ ] JSON:API response format
- [ ] Lazy initialization on first query
- [ ] Include breakdown by source (optional query param)
- [ ] Error response for non-existent character

**Files to Create:**
- `character/resource.go`
- `stat/rest.go`

#### 4.2 Update Ingress Configuration
**Effort:** S | **Priority:** P0 | **Depends On:** 4.1

**Acceptance Criteria:**
- [ ] Add nginx location block to route `/api/worlds/.../stats` to effective-stats service
- [ ] Must not conflict with existing character service routes
- [ ] Follow existing pattern from atlas-rates routing
- [ ] Verify routing works in local environment

**Files to Modify:**
- `atlas-ingress.yml`

#### 4.3 Kubernetes Deployment
**Effort:** S | **Priority:** P0 | **Depends On:** 4.1

**Acceptance Criteria:**
- [ ] Single YAML file with Deployment and Service specs (following atlas-rates pattern)
- [ ] Deployment: replicas=1, container image, port 8080, envFrom atlas-env ConfigMap
- [ ] Service: selector matching app label, TCP port 8080
- [ ] Proper namespace (atlas) and labels

**Files to Create:**
- `services/atlas-effective-stats/atlas-effective-stats.yml`

**Reference:** `services/atlas-rates/atlas-rates.yml`

#### 4.4 GitHub Actions Configuration
**Effort:** S | **Priority:** P0 | **Depends On:** 4.3

**Acceptance Criteria:**
- [ ] Add service entry to `.github/config/services.json`
- [ ] Entry includes: name, type, path, module_path, docker_image, docker_context
- [ ] Workflows auto-detect service (no workflow file changes needed)
- [ ] Verify PR validation builds and tests the service

**Files to Modify:**
- `.github/config/services.json`

#### 4.5 Service Documentation
**Effort:** M | **Priority:** P1 | **Depends On:** 4.1

**Acceptance Criteria:**
- [ ] README.md with service overview
- [ ] REST endpoint documentation
- [ ] Kafka topic documentation
- [ ] Example requests and responses

**Files to Create:**
- `services/atlas-effective-stats/atlas.com/effective-stats/README.md`
- `services/atlas-effective-stats/docs/kafka.md`

---

### Phase 5: Consumer Integration

#### 5.1 Update Character Service
**Effort:** M | **Priority:** P0 | **Depends On:** 4.1

**Acceptance Criteria:**
- [ ] Add REST client for effective stats service
- [ ] Update `getMaxMpGrowth()` to use effective INT (line 976)
- [ ] Update `ChangeHP()` to use effective Max HP (line 1009)
- [ ] Update `ChangeMP()` to use effective Max MP (line 1081)
- [ ] Remove corresponding TODOs

**Files to Modify:**
- `services/atlas-character/atlas.com/character/character/processor.go`
- New file: `services/atlas-character/atlas.com/character/effectivestats/requests.go`

#### 5.2 Integration Testing
**Effort:** L | **Priority:** P1 | **Depends On:** 5.1

**Acceptance Criteria:**
- [ ] Test equipment bonus application
- [ ] Test buff bonus application/expiration
- [ ] Test passive skill bonus calculation
- [ ] Test service restart recovery
- [ ] Test concurrent update handling

---

## Risk Assessment and Mitigation

### Risk 1: Circular Dependency
**Severity:** High
**Description:** Character service queries effective stats, which queries character service for base stats.
**Mitigation:** Effective stats service caches base stats and updates via Kafka events. Initial query fetches base stats but subsequent updates come from events.

### Risk 2: Event Ordering
**Severity:** Medium
**Description:** Equipment or buff events may arrive before session events.
**Mitigation:** Lazy initialization pattern - if character not in registry when event arrives, either initialize immediately or ignore (depending on event type).

### Risk 3: Service Restart Data Loss
**Severity:** Low
**Description:** On restart, all effective stats are lost until queries trigger recomputation.
**Mitigation:** Lazy initialization from source services. First query after restart may have higher latency but will return correct data.

### Risk 4: Performance Under Load
**Severity:** Medium
**Description:** Many concurrent stat queries could overwhelm the service.
**Mitigation:** Stats are pre-computed on event receipt; queries are simple map lookups. If needed, add read replicas or caching layer.

### Risk 5: Incorrect Bonus Calculation
**Severity:** High
**Description:** Multiplicative vs additive bonus ordering could produce incorrect values.
**Mitigation:** Clear aggregation formula documented and tested. Unit tests for known skill/equipment combinations.

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Query latency (p99) | < 10ms | Prometheus metrics |
| Event processing latency | < 100ms | Kafka consumer lag |
| Restart recovery time | < 5s for 1000 characters | Manual testing |
| Calculation accuracy | 100% match to client display | Integration tests |
| Memory usage | < 50MB for 10000 characters | Prometheus metrics |

---

## Required Resources and Dependencies

### Service Dependencies
- `atlas-character` - Base stat queries
- `atlas-inventory` - Equipped item queries
- `atlas-buffs` - Active buff queries
- `atlas-data` - Skill effect data
- Kafka cluster - Event streaming

### Library Dependencies
- `atlas-rest` - REST client utilities
- `atlas-kafka` - Kafka consumer/producer
- `atlas-tenant` - Multi-tenancy support
- `atlas-model` - Provider patterns

### New Kafka Topics (if not existing)
- May need `EVENT_TOPIC_SKILL_STATUS` for passive skill changes

---

## Appendix: Stat Bonus Sources

### Equipment Bonuses (Additive)
Source: `atlas-inventory` equipped items → `atlas-data` equipment stats

| Equipment Stat | Maps To |
|----------------|---------|
| strength | STR |
| dexterity | DEX |
| luck | LUK |
| intelligence | INT |
| hp | Max HP |
| mp | Max MP |
| weaponAttack | WATK |
| weaponDefense | WDEF |
| magicAttack | MATK |
| magicDefense | MDEF |
| accuracy | ACC |
| avoidability | AVOID |
| speed | Speed |
| jump | Jump |

### Buff Bonuses (Additive and Multiplicative)
Source: `atlas-buffs` active buffs

| Buff Stat Type | Maps To | Type |
|----------------|---------|------|
| WEAPON_ATTACK | WATK | Additive |
| MAGIC_ATTACK | MATK | Additive |
| WEAPON_DEFENSE | WDEF | Additive |
| MAGIC_DEFENSE | MDEF | Additive |
| ACCURACY | ACC | Additive |
| AVOIDABILITY | AVOID | Additive |
| SPEED | Speed | Additive |
| JUMP | Jump | Additive |
| HYPER_BODY_HP | Max HP | Multiplicative |
| HYPER_BODY_MP | Max MP | Multiplicative |
| MAPLE_WARRIOR | All stats | Multiplicative |

### Passive Skill Bonuses
Source: Character skills → `atlas-data` skill effects

Identified by:
- `skillType == 2` in skill data
- `action == false` in skill model
- No `hit` or `ball` elements in skill XML

Effect fields map directly to stat bonuses.
