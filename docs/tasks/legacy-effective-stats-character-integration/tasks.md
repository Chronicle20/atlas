# Effective Stats Character Integration - Task Checklist

**Last Updated:** 2026-02-03

---

## Phase 1: Base Stats Synchronization

- [ ] **1.1** Extend `StatusEventStatChangedBody` to include stat values
  - File: `atlas-character/kafka/message/character/kafka.go`
  - Add `Values map[string]interface{}` field to body
  - Acceptance: Event body contains actual stat values, not just names

- [ ] **1.2** Modify `statChangedProvider` to include MaxHP, MaxMP, INT values
  - File: `atlas-character/character/producer.go`
  - Populate values map when stats include HP, MP, or primary stats
  - Acceptance: Published events contain current values for changed stats

- [ ] **1.3** Create Kafka consumer in atlas-effective-stats for `STAT_CHANGED`
  - File: `atlas-effective-stats/kafka/consumer/character/consumer.go` (new)
  - Register handler for `StatusEventTypeStatChanged`
  - Acceptance: Consumer receives and logs stat change events

- [ ] **1.4** Call `SetBaseStats()` when relevant stats change
  - File: `atlas-effective-stats/kafka/consumer/character/consumer.go`
  - Extract MaxHP, MaxMP, primary stats from event
  - Call `SetBaseStats()` on processor
  - Acceptance: Registry updated with new base stats on stat change

- [ ] **1.5** Test base stats sync on level-up, AP distribution
  - Verify level-up updates base MaxHP/MaxMP in effective-stats
  - Verify AP distribution to INT updates base INT in effective-stats
  - Acceptance: Effective stats reflect base stat changes within 1 second

---

## Phase 2: Effective Stats Query Integration

- [ ] **2.1** Create `external/effective-stats/rest.go` in atlas-character
  - File: `atlas-character/external/effective-stats/rest.go` (new)
  - Define `RestModel` struct matching effective-stats response
  - Acceptance: Model compiles, matches API response structure

- [ ] **2.2** Create `external/effective-stats/requests.go` with REST client
  - File: `atlas-character/external/effective-stats/requests.go` (new)
  - Function: `RequestEffectiveStats(worldId, channelId, characterId) (Computed, error)`
  - Use standard atlas REST client patterns
  - Acceptance: Successfully fetches effective stats from running service

- [ ] **2.3** Add effective-stats service URL to configuration
  - File: `atlas-character/configuration/loader.go`
  - Add `EFFECTIVE_STATS_SERVICE_URL` environment variable
  - Acceptance: Configuration loaded at service startup

- [ ] **2.4** Modify `getMaxMpGrowth()` to use effective Intelligence
  - File: `atlas-character/character/processor.go:976`
  - Fetch effective INT from service
  - Use effective INT in calculation
  - Acceptance: MP growth uses INT from buffs/equipment

- [ ] **2.5** Add graceful fallback to base Intelligence if service unavailable
  - File: `atlas-character/character/processor.go`
  - On REST error, log warning and use base INT
  - Acceptance: Level-up succeeds even if effective-stats is down

---

## Phase 3: HP/MP Bounds Using Effective Stats

- [ ] **3.1** Modify `ChangeHP()` to fetch effective MaxHP
  - File: `atlas-character/character/processor.go:1009`
  - Query effective stats before bounds check
  - Use effective MaxHP in `enforceBounds()`
  - Acceptance: HP never set above effective MaxHP with active buffs

- [ ] **3.2** Modify `ChangeMP()` to fetch effective MaxMP
  - File: `atlas-character/character/processor.go:1081`
  - Query effective stats before bounds check
  - Use effective MaxMP in `enforceBounds()`
  - Acceptance: MP never set above effective MaxMP with active buffs

- [ ] **3.3** Modify `SetHP()` to use effective MaxHP
  - File: `atlas-character/character/processor.go:1036`
  - Clamp to effective MaxHP, not base MaxHP
  - Acceptance: SetHP respects effective maximum

- [ ] **3.4** Consider caching effective stats in processor context
  - Evaluate if REST calls on every HP/MP change is acceptable
  - If not, add short TTL cache (5-10 seconds)
  - Acceptance: Performance acceptable under load

- [ ] **3.5** Test HP/MP bounds with active buffs
  - Apply Hyper Body, take damage, heal
  - Verify healing caps at effective MaxHP
  - Acceptance: No over-healing beyond buffed max

---

## Phase 4: HP/MP Clamping on Max Decrease

- [ ] **4.1** Add `CommandClampHP` and `CommandClampMP` command types
  - File: `atlas-character/kafka/message/character/kafka.go`
  - Add constants: `CommandClampHP = "CLAMP_HP"`, `CommandClampMP = "CLAMP_MP"`
  - Add body structs: `ClampHPBody`, `ClampMPBody` with `ChannelId` and `MaxValue` fields
  - Acceptance: Message types compile

- [ ] **4.2** Add handler in atlas-character for `CommandClampHP`
  - File: `atlas-character/kafka/consumer/character/consumer.go`
  - Register handler for `CommandClampHP`
  - Fetch current HP from database
  - If current > maxValue, call `SetHP(maxValue)`
  - Emit `STAT_CHANGED` if HP was clamped
  - Acceptance: HP clamped when command received

- [ ] **4.3** Add handler in atlas-character for `CommandClampMP`
  - File: `atlas-character/kafka/consumer/character/consumer.go`
  - Register handler for `CommandClampMP`
  - Same logic as 4.2 but for MP
  - Acceptance: MP clamped when command received

- [ ] **4.4** Add `previousComputed` field to effective-stats character Model
  - File: `atlas-effective-stats/character/model.go`
  - Store previous computed values before recompute
  - Add getter `PreviousComputed() Computed`
  - Acceptance: Previous values accessible after recompute

- [ ] **4.5** Modify `Recompute()` to detect MaxHP/MaxMP decreases
  - File: `atlas-effective-stats/character/model.go`
  - Compare new MaxHP/MaxMP with previous
  - Return or flag if decrease detected
  - Acceptance: Decreases correctly identified

- [ ] **4.6** Create Kafka producer in atlas-effective-stats for character commands
  - File: `atlas-effective-stats/kafka/producer/producer.go` (new)
  - Standard atlas Kafka producer setup for `COMMAND_TOPIC_CHARACTER`
  - Acceptance: Producer can publish character commands

- [ ] **4.7** Publish `CommandClampHP`/`CommandClampMP` when decrease detected
  - File: `atlas-effective-stats/character/registry.go`
  - After `Recompute()` detects decrease, publish command
  - Include characterId, worldId, channelId, new max value
  - Acceptance: Commands published on buff expiry/equipment removal

- [ ] **4.8** Test buff expiry clamping (Hyper Body)
  - Apply Hyper Body → HP increases
  - Heal to buffed max
  - Let buff expire
  - Verify HP clamped to new effective max
  - Acceptance: HP correctly reduced on buff expiry

- [ ] **4.9** Test equipment unequip clamping
  - Equip HP gear → MaxHP increases
  - Set HP to buffed max
  - Unequip gear
  - Verify HP clamped to new effective max
  - Acceptance: HP correctly reduced on unequip

---

## Verification Checklist

### Phase 1 Complete When:
- [ ] Base stat changes in atlas-character reflected in atlas-effective-stats registry
- [ ] Level-up MaxHP/MaxMP sync verified
- [ ] AP distribution to INT sync verified

### Phase 2 Complete When:
- [ ] atlas-character can query effective stats via REST
- [ ] MP growth on level-up uses effective INT
- [ ] Graceful degradation when effective-stats unavailable

### Phase 3 Complete When:
- [ ] `ChangeHP()` uses effective MaxHP
- [ ] `ChangeMP()` uses effective MaxMP
- [ ] `SetHP()` uses effective MaxHP
- [ ] Performance acceptable

### Phase 4 Complete When:
- [ ] `CommandClampHP` and `CommandClampMP` handlers work correctly
- [ ] Buff expiry triggers HP/MP clamping via commands
- [ ] Equipment removal triggers HP/MP clamping via commands
- [ ] `STAT_CHANGED` events emitted after clamping
- [ ] No character has current HP/MP > effective max

---

## Notes

- Each phase can be merged independently
- Phase 4 provides the most user-visible benefit (clamping)
- Phases 2-3 can be simplified if Phase 4 is prioritized (clamping handles edge cases)
- Uses command-driven approach: effective-stats sends `CommandClampHP`/`CommandClampMP` to character service
- Long-term: Consider consolidating Set/Change/Clamp commands into unified HP/MP mutation API
