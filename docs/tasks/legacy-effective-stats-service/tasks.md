# Effective Stats Service - Task Checklist

**Last Updated:** 2026-02-03

---

## Phase 1: Core Service Foundation

### 1.1 Create Service Scaffolding
**Effort:** M | **Status:** [ ] Not Started

- [ ] Create service directory `services/atlas-effective-stats/`
- [ ] Create `atlas.com/effective-stats/main.go` with initialization
- [ ] Create `go.mod` with dependencies
- [ ] Create `service/teardown.go` for graceful shutdown
- [ ] Create `logger/logger.go`
- [ ] Create `tracing/tracing.go`
- [ ] Create `Dockerfile`
- [ ] Verify service starts and shuts down cleanly

### 1.2 Define Domain Models
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 1.1

- [ ] Create `stat/model.go` with StatType enum
- [ ] Create `stat/bonus.go` with StatBonus struct
- [ ] Create `character/model.go` with effective stats Model
- [ ] Create `character/builder.go` with fluent builder
- [ ] Implement accessor methods with defensive copying
- [ ] Add unit tests for model immutability

### 1.3 Implement Registry Pattern
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 1.2

- [ ] Create `character/registry.go` singleton
- [ ] Implement per-tenant map structure
- [ ] Implement per-tenant RWMutex locking
- [ ] Implement `GetOrCreate(tenant, charId)` method
- [ ] Implement `Update(tenant, charId, model)` method
- [ ] Implement `Delete(tenant, charId)` method
- [ ] Implement `GetAll(tenant)` method
- [ ] Add unit tests for concurrent access

### 1.4 Create Base Processor
**Effort:** S | **Status:** [ ] Not Started | **Depends:** 1.3

- [ ] Create `character/processor.go` interface
- [ ] Create `character/provider.go` for dependencies
- [ ] Implement `GetEffectiveStats(characterId)` method
- [ ] Implement `Recompute(characterId)` method
- [ ] Add logging for all operations

---

## Phase 2: Data Source Integration

### 2.1 Character Service REST Client
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 1.4

- [ ] Create `character/requests.go` for REST client
- [ ] Create `character/rest.go` for response models
- [ ] Implement `GetCharacterBaseStats(charId)` function
- [ ] Implement `GetCharacterSkills(charId)` function
- [ ] Add error handling for service unavailability
- [ ] Add unit tests with mock responses

### 2.2 Inventory Service REST Client
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 1.4

- [ ] Create `inventory/requests.go` for REST client
- [ ] Create `inventory/rest.go` for response models
- [ ] Implement `GetEquippedItems(charId)` function
- [ ] Implement `GetEquipmentData(templateId)` function
- [ ] Filter for equipped compartment only
- [ ] Add unit tests with mock responses

### 2.3 Buffs Service REST Client
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 1.4

- [ ] Create `buffs/requests.go` for REST client
- [ ] Create `buffs/rest.go` for response models
- [ ] Implement `GetActiveBuffs(charId)` function
- [ ] Map buff stat types to StatType enum
- [ ] Handle multiplicative buffs (MAPLE_WARRIOR, HYPER_BODY)
- [ ] Add unit tests with mock responses

### 2.4 Passive Skill Processor
**Effort:** L | **Status:** [ ] Not Started | **Depends:** 2.1

- [ ] Create `passive/processor.go`
- [ ] Create `passive/requests.go` for skill data
- [ ] Implement passive skill identification logic
- [ ] Extract stat bonuses from skill effects
- [ ] Support additive bonuses from effects
- [ ] Support multiplicative bonuses (Maple Warrior)
- [ ] Cache skill effect data to reduce queries
- [ ] Add unit tests with known skill data

### 2.5 Stat Aggregation Engine
**Effort:** L | **Status:** [ ] Not Started | **Depends:** 2.1, 2.2, 2.3, 2.4

- [ ] Create `character/aggregator.go`
- [ ] Implement aggregation formula:
  - [ ] Sum additive bonuses by source
  - [ ] Multiply multiplicative bonuses
  - [ ] Apply in correct order (add then multiply)
  - [ ] Floor final values
- [ ] Handle missing/null data gracefully
- [ ] Add debug logging for aggregation steps
- [ ] Add unit tests with complex bonus scenarios

---

## Phase 3: Event-Driven Updates

### 3.1 Session Status Consumer
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 2.5

- [ ] Create `kafka/consumer/session/consumer.go`
- [ ] Create `kafka/message/session/kafka.go`
- [ ] Register consumer in main.go
- [ ] Handle `CREATED` event (Issuer="CHANNEL"):
  - [ ] Mark character for lazy initialization
- [ ] Handle `DESTROYED` event:
  - [ ] Remove character from registry
- [ ] Add idempotency checks
- [ ] Add unit tests

### 3.2 Buff Status Consumer
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 2.5

- [ ] Create `kafka/consumer/buff/consumer.go`
- [ ] Create `kafka/message/buff/kafka.go`
- [ ] Register consumer in main.go
- [ ] Handle `APPLIED` event:
  - [ ] Parse stat changes from event
  - [ ] Add buff bonuses to character model
  - [ ] Recompute effective stats
- [ ] Handle `EXPIRED` event:
  - [ ] Remove buff bonuses from character model
  - [ ] Recompute effective stats
- [ ] Map buff stat types to StatBonus
- [ ] Add unit tests

### 3.3 Asset Status Consumer
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 2.5

- [ ] Create `kafka/consumer/asset/consumer.go`
- [ ] Create `kafka/message/asset/kafka.go`
- [ ] Register consumer in main.go
- [ ] Handle `MOVED` event:
  - [ ] If moved TO equipped slot: add equipment bonuses
  - [ ] If moved FROM equipped slot: remove equipment bonuses
  - [ ] Recompute effective stats
- [ ] Handle `DELETED`/`RELEASED` events:
  - [ ] Remove equipment bonuses if was equipped
  - [ ] Recompute effective stats
- [ ] Filter for equipment compartment
- [ ] Add unit tests

### 3.4 Character Status Consumer
**Effort:** S | **Status:** [ ] Not Started | **Depends:** 2.5

- [ ] Create `kafka/consumer/character/consumer.go`
- [ ] Create `kafka/message/character/kafka.go`
- [ ] Register consumer in main.go
- [ ] Handle `STAT_CHANGED` event:
  - [ ] Check if relevant stat types changed
  - [ ] Update base stats in model
  - [ ] Recompute effective stats
- [ ] Add unit tests

### 3.5 Skill Status Consumer
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 2.5

- [ ] Verify EVENT_TOPIC_SKILL_STATUS exists or create
- [ ] Create `kafka/consumer/skill/consumer.go`
- [ ] Create `kafka/message/skill/kafka.go`
- [ ] Register consumer in main.go
- [ ] Handle skill level change:
  - [ ] Check if passive skill
  - [ ] Update passive bonuses
  - [ ] Recompute effective stats
- [ ] Handle skill reset:
  - [ ] Remove all passive bonuses
  - [ ] Recompute effective stats
- [ ] Add unit tests

---

## Phase 4: REST API & Deployment

### 4.1 REST Resource Handler
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 2.5

- [ ] Create `character/resource.go`
- [ ] Create `stat/rest.go` with JSON:API model
- [ ] Implement GET handler for stats endpoint
- [ ] Path: `/api/worlds/{worldId}/channels/{channelId}/characters/{characterId}/stats`
- [ ] Add lazy initialization on query
- [ ] Add optional `?include=breakdown` parameter
- [ ] Return 404 for non-existent character
- [ ] Add unit tests with mock processor

### 4.2 Update Ingress Configuration
**Effort:** S | **Status:** [ ] Not Started | **Depends:** 4.1

- [ ] Open `atlas-ingress.yml`
- [ ] Add location block to route `.../stats` to effective-stats service
- [ ] Ensure no conflict with existing character service routes
- [ ] Follow pattern from atlas-rates routing
- [ ] Verify routing in local environment

### 4.3 Kubernetes Deployment
**Effort:** S | **Status:** [ ] Not Started | **Depends:** 4.1

- [ ] Create `services/atlas-effective-stats/atlas-effective-stats.yml`
- [ ] Include Deployment spec (replicas, container, ports, envFrom)
- [ ] Include Service spec (selector, port 8080)
- [ ] Reference `atlas-env` ConfigMap for environment variables
- [ ] Follow pattern from `services/atlas-rates/atlas-rates.yml`
- [ ] Verify deployment in local/dev cluster

### 4.4 GitHub Actions Configuration
**Effort:** S | **Status:** [ ] Not Started | **Depends:** 4.3

- [ ] Add service entry to `.github/config/services.json`:
  - name: "atlas-effective-stats"
  - type: "go-service"
  - path: "services/atlas-effective-stats"
  - module_path: "services/atlas-effective-stats/atlas.com/effective-stats"
  - docker_image: "ghcr.io/chronicle20/atlas-effective-stats/atlas-effective-stats"
  - docker_context: "."
- [ ] Verify PR validation workflow detects the service
- [ ] Verify Docker image builds successfully

### 4.5 Service Documentation
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 4.1

- [ ] Create `README.md` in service root
- [ ] Document service purpose
- [ ] Document REST endpoints with examples
- [ ] Create `docs/kafka.md`
- [ ] Document consumed Kafka topics
- [ ] Document event handling behavior

---

## Phase 5: Consumer Integration

### 5.1 Update Character Service
**Effort:** M | **Status:** [ ] Not Started | **Depends:** 4.1

- [ ] Create `effectivestats/requests.go` in character service
- [ ] Create `effectivestats/rest.go` for response model
- [ ] Update `processor.go:976-977` (getMaxMpGrowth):
  - [ ] Query effective INT from effective-stats service
  - [ ] Use effective INT in calculation
  - [ ] Remove TODO comment
- [ ] Update `processor.go:1009` (ChangeHP):
  - [ ] Query effective Max HP from effective-stats service
  - [ ] Use effective Max HP in bounds check
  - [ ] Remove TODO comment
- [ ] Update `processor.go:1081` (ChangeMP):
  - [ ] Query effective Max MP from effective-stats service
  - [ ] Use effective Max MP in bounds check
  - [ ] Remove TODO comment
- [ ] Add fallback to base stats if service unavailable
- [ ] Add unit tests for new integration

### 5.2 Integration Testing
**Effort:** L | **Status:** [ ] Not Started | **Depends:** 5.1

- [ ] Create integration test suite
- [ ] Test: Equip item → stats increase correctly
- [ ] Test: Unequip item → stats decrease correctly
- [ ] Test: Apply buff → stats increase correctly
- [ ] Test: Buff expires → stats decrease correctly
- [ ] Test: Level up passive skill → stats increase correctly
- [ ] Test: Login → stats computed correctly
- [ ] Test: Logout → registry cleaned up
- [ ] Test: Service restart → first query correct
- [ ] Test: Concurrent updates → thread-safe
- [ ] Document test results

---

## Final Checklist

### Pre-Deployment

- [ ] All unit tests pass (`go test ./... -count=1`)
- [ ] All integration tests pass
- [ ] Service builds successfully (`go build`)
- [ ] Ingress configuration updated
- [ ] Service documentation complete
- [ ] TODO items removed from character service

### Post-Deployment Verification

- [ ] Service starts successfully in cluster
- [ ] REST endpoint accessible via ingress
- [ ] Kafka consumers connected and processing
- [ ] Character service queries effective stats correctly
- [ ] HP/MP bounds respect effective max values
- [ ] Memory usage within expected bounds
- [ ] Query latency within expected bounds

---

## Progress Summary

| Phase | Tasks | Completed | Progress |
|-------|-------|-----------|----------|
| Phase 1: Core Foundation | 4 | 0 | 0% |
| Phase 2: Data Integration | 5 | 0 | 0% |
| Phase 3: Event Updates | 5 | 0 | 0% |
| Phase 4: REST API & Deployment | 5 | 0 | 0% |
| Phase 5: Integration | 2 | 0 | 0% |
| **Total** | **21** | **0** | **0%** |
