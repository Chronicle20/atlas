# Instance-Based Transports - Task Checklist

**Last Updated: 2026-02-03**

---

## Phase 1: Foundation - field.Model Adoption

### 1.1 Enhance atlas-constants field.Model
- [ ] **1.1.1** Add JSON marshaling/unmarshaling to field.Model [S]
- [ ] **1.1.2** Add Kafka serialization helpers to field.Model [S]
- [ ] **1.1.3** Add field.Model equality and comparison methods [S]

### 1.2 Update atlas-maps MapKey
- [ ] **1.2.1** Add Instance (uuid.UUID) field to MapKey struct [M]
- [ ] **1.2.2** Update character registry for instance-aware keys [M]
- [ ] **1.2.3** Add instance-specific REST endpoint [M]
- [ ] **1.2.4** Update spawn point registry for instance awareness [M]
- [ ] **1.2.5** Update map status event producer with instance field [S]

### 1.3 Update Kafka Message Definitions
- [ ] **1.3.1** Update atlas-channel character status events with instance [L]
- [ ] **1.3.2** Update atlas-maps map status events with instance [M]
- [ ] **1.3.3** Update atlas-monsters status events with instance [L]
- [ ] **1.3.4** Update atlas-drops status events with instance [L]
- [ ] **1.3.5** Update atlas-reactors status events with instance [L]
- [ ] **1.3.6** Update remaining services Kafka messages (batch) [XL]
  - [ ] atlas-expressions
  - [ ] atlas-messages
  - [ ] atlas-consumables
  - [ ] atlas-npc-conversations
  - [ ] atlas-portals
  - [ ] atlas-portal-actions
  - [ ] atlas-monster-death
  - [ ] atlas-saga-orchestrator
  - [ ] atlas-query-aggregator
  - [ ] atlas-parties
  - [ ] (others as identified)

### 1.4 Update Service Processors
- [ ] **1.4.1** Update processor function signatures to use field.Model [XL]
- [ ] **1.4.2** Update provider functions for field-based lookups [L]
- [ ] **1.4.3** Update REST request builders with instance parameter [L]
- [ ] **1.4.4** Update Kafka consumers to parse instance field [L]

---

## Phase 2: Instance Transport Core

### 2.1 Instance Transport Route Model
- [ ] **2.1.1** Create InstanceTransportRoute model in atlas-transports [S]
- [ ] **2.1.2** Create instance route configuration JSON schema [S]
- [ ] **2.1.3** Implement instance route registry (singleton) [M]
- [ ] **2.1.4** Implement configuration loader for instance routes [M]

### 2.2 Instance Pool Management
- [ ] **2.2.1** Implement InstancePool struct with thread-safe fields [M]
- [ ] **2.2.2** Implement Allocate() method (returns instance or error if full) [M]
- [ ] **2.2.3** Implement Release() method (immediate return to pool) [S]
- [ ] **2.2.4** Add pool initialization from route configuration [S]
- [ ] **2.2.5** Add pool status query methods (available/total) [S]

### 2.3 Active Transport Registry
- [ ] **2.3.1** Create ActiveTransport model [S]
- [ ] **2.3.2** Implement active transport registry (singleton) [M]
- [ ] **2.3.3** Add expiration tracking (startTime + duration) [M]
- [ ] **2.3.4** Add lookup methods (by character, by instance, by route) [M]
- [ ] **2.3.5** Add cleanup method for completed/cancelled transports [S]

---

## Phase 3: Transport Execution

### 3.1 Transport Start Flow
- [ ] **3.1.1** Create Kafka consumer for start transport command [M]
- [ ] **3.1.2** Implement pool availability check in start handler [S]
- [ ] **3.1.3** Implement instance allocation in start handler [S]
- [ ] **3.1.4** Build transit field (world, channel, transitMapId, instanceId) [S]
- [ ] **3.1.5** Emit CHANGE_MAP command with instance [M]
- [ ] **3.1.6** Register active transport with expiration time [S]
- [ ] **3.1.7** Emit STARTED event to instance transport topic [S]

### 3.2 Transport Completion Flow
- [ ] **3.2.1** Implement expiration ticker (check every N seconds) [M]
- [ ] **3.2.2** Implement completion handler for expired transports [M]
- [ ] **3.2.3** Build destination field and emit CHANGE_MAP command [M]
- [ ] **3.2.4** Release instance to pool on completion [S]
- [ ] **3.2.5** Remove from active registry on completion [S]
- [ ] **3.2.6** Emit COMPLETED event [S]

### 3.3 Transport Cancellation Flow (Map Exit)
- [ ] **3.3.1** Subscribe to EVENT_TOPIC_MAP_STATUS events [S]
- [ ] **3.3.2** Filter for CHARACTER_EXIT events in transit maps [S]
- [ ] **3.3.3** Check if exiting character has active transport for that instance [S]
- [ ] **3.3.4** Release instance immediately on map exit [S]
- [ ] **3.3.5** Remove from active registry on map exit [S]
- [ ] **3.3.6** Emit CANCELLED event with reason "MAP_EXIT" [S]

### 3.4 Logout Handling
- [ ] **3.4.1** Enhance existing logout handler in atlas-transports [M]
- [ ] **3.4.2** Check active transport registry on logout [S]
- [ ] **3.4.3** Warp character to start map on logout [M]
- [ ] **3.4.4** Release instance on logout [S]
- [ ] **3.4.5** Emit CANCELLED event with reason "LOGOUT" [S]

### 3.5 Double-Transport Prevention
- [ ] **3.5.1** Check active transport registry before starting new transport [S]
- [ ] **3.5.2** Return error if character already in transport [S]

### 3.6 Stuck Transport Timeout
- [ ] **3.6.1** Add max lifetime config (e.g., 2x travel duration) [S]
- [ ] **3.6.2** Check for stuck transports in expiration ticker [S]
- [ ] **3.6.3** Force-cancel and log warning for stuck transports [M]

### 3.7 Graceful Shutdown
- [ ] **3.7.1** Register SIGTERM/SIGINT handler [S]
- [ ] **3.7.2** Stop accepting new transport commands on shutdown signal [S]
- [ ] **3.7.3** Iterate active transports and warp to start maps [M]
- [ ] **3.7.4** Release all instances and cleanup registries [S]

### 3.8 Crash Recovery
- [ ] **3.8.1** Subscribe to CHARACTER_STATUS LOGIN events [S]
- [ ] **3.8.2** Check if login mapId matches any transit map [S]
- [ ] **3.8.3** Warp stranded characters to route start map [M]

---

## Phase 4: Integration & Configuration

### 4.1 NPC Conversation Integration
- [ ] **4.1.1** Define start instance transport Kafka command schema [S]
- [ ] **4.1.2** Create capacity check endpoint for NPC pre-check [M]
- [ ] **4.1.3** Document NPC integration pattern [S]

### 4.2 Configuration Files
- [ ] **4.2.1** Create Kerning Square Train route configuration [S]
- [ ] **4.2.2** Document instance route configuration format [S]
- [ ] **4.2.3** Add configuration validation on load [M]

### 4.3 REST API Implementation
- [ ] **4.3.1** Implement GET /transports/instance-routes endpoint [M]
- [ ] **4.3.2** Implement GET /transports/instance-routes/{routeId} endpoint [S]
- [ ] **4.3.3** Implement GET /transports/instance-routes/{routeId}/pool endpoint [S]
- [ ] **4.3.4** Implement POST /transports/instance-routes/{routeId}/start endpoint [M]
- [ ] **4.3.5** Implement POST /transports/instance-routes/{routeId}/cancel endpoint [M]
- [ ] **4.3.6** Update service documentation [M]

---

## Phase 5: Testing & Migration

### 5.1 Unit Tests
- [ ] **5.1.1** Unit tests for InstancePool allocation/release [M]
- [ ] **5.1.2** Unit tests for ActiveTransport registry operations [M]
- [ ] **5.1.3** Unit tests for transport expiration logic [M]
- [ ] **5.1.4** Unit tests for cancellation scenarios [M]
- [ ] **5.1.5** Unit tests for MapKey with instance equality [S]

### 5.2 Integration Tests
- [ ] **5.2.1** Integration test: full transport flow (start → complete) [L]
- [ ] **5.2.2** Integration test: map exit cancellation (NPC and other) [L]
- [ ] **5.2.3** Integration test: logout handling [M]
- [ ] **5.2.4** Integration test: pool exhaustion [M]
- [ ] **5.2.5** Integration test: graceful shutdown [L]
- [ ] **5.2.6** Integration test: crash recovery (stranded characters) [L]
- [ ] **5.2.7** Integration test: double-transport prevention [M]
- [ ] **5.2.8** Integration test: stuck transport timeout [M]

### 5.3 Migration
- [ ] **5.3.1** Create migration documentation [M]
- [ ] **5.3.2** Add feature flag for instance transport activation [S]
- [ ] **5.3.3** Create deployment order checklist [S]
- [ ] **5.3.4** Verify backward compatibility [M]

---

## Effort Legend
- **S** = Small (< 1 hour)
- **M** = Medium (1-4 hours)
- **L** = Large (4-8 hours)
- **XL** = Extra Large (1+ days)

---

## Progress Summary

| Phase | Total | Completed | Remaining |
|-------|-------|-----------|-----------|
| Phase 1 | 18 | 0 | 18 |
| Phase 2 | 14 | 0 | 14 |
| Phase 3 | 24 | 0 | 24 |
| Phase 4 | 11 | 0 | 11 |
| Phase 5 | 16 | 0 | 16 |
| **Total** | **83** | **0** | **83** |
