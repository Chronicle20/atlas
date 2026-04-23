# Effective Stats Service Integration with Character Service

**Task:** Integrate `atlas-effective-stats` into `atlas-character` for effective stat queries and HP/MP clamping
**Last Updated:** 2026-02-03
**Status:** Planned

---

## 1. Executive Summary

The `atlas-effective-stats` service was recently implemented (commit `d33f9d4e4`) to maintain computed character stats that include bonuses from equipment, buffs, and passive skills. However, this service is not yet integrated with `atlas-character` where effective stats are needed for:

1. **MP Growth Calculations** - Intelligence with buffs/weapons affects MP gain on level-up
2. **HP Bounds Enforcement** - `ChangeHP()` should use effective MaxHP, not base MaxHP
3. **MP Bounds Enforcement** - `ChangeMP()` should use effective MaxMP, not base MaxMP
4. **HP/MP Clamping** - When effective MaxHP/MP decreases (buff expires, equipment removed), current HP/MP must be clamped to the new maximum

### Goals

1. Enable `atlas-character` to query effective stats from `atlas-effective-stats`
2. Use effective MaxHP/MaxMP when enforcing HP/MP bounds
3. Implement HP/MP clamping when effective max values decrease
4. Keep base stats synchronized with effective-stats service

---

## 2. Current State Analysis

### 2.1 Effective Stats Service Architecture

The `atlas-effective-stats` service:
- Maintains an in-memory registry of character effective stats
- Consumes Kafka events for buff application/expiration and equipment equip/unequip
- Computes effective stats by aggregating base stats with bonuses (flat + multipliers)
- Exposes REST API: `GET /worlds/{worldId}/channels/{channelId}/characters/{characterId}/stats`

**Key Files:**
- `character/processor.go` - Core interface with `GetEffectiveStats()`, `SetBaseStats()`, bonus management
- `character/registry.go` - In-memory singleton storing `Model` per character per tenant
- `stat/model.go` - `Computed` struct with all effective stat values
- `kafka/consumer/buff/consumer.go` - Handles buff applied/expired events
- `kafka/consumer/asset/consumer.go` - Handles equipment equip/unequip events

### 2.2 Character Service TODOs

**processor.go:976** - `getMaxMpGrowth()`:
```go
// TODO this needs to incorporate computed total intelligence (buffs, weapons, etc)
resMax += uint16(math.Ceil(float64(c.Intelligence()) / 10))
```

**processor.go:1009** - `ChangeHP()`:
```go
// TODO consider effective (temporary) Max HP.
adjusted = enforceBounds(amount, c.HP(), c.MaxHP(), 0)
```

**processor.go:1081** - `ChangeMP()`:
```go
// TODO consider effective (temporary) Max MP.
adjusted := enforceBounds(amount, c.MP(), c.MaxMP(), 0)
```

### 2.3 Missing: HP/MP Clamping Mechanism

When effective MaxHP/MP decreases (e.g., Hyper Body buff expires reducing MaxHP from 1200 to 800), if current HP is 1000, it must be clamped to 800.

**Current Event Flow:**
```
Buff Expires → atlas-channel publishes BUFF_EXPIRED event
            → atlas-effective-stats removes buff bonuses, recomputes stats
            → ??? (no notification to atlas-character)
```

**No current mechanism** for atlas-character to know when to clamp HP/MP.

---

## 3. Proposed Architecture

### 3.1 Command-Driven HP/MP Clamping

When effective MaxHP or MaxMP decreases, atlas-effective-stats sends a **command** to atlas-character via the existing `COMMAND_TOPIC_CHARACTER`:

```
COMMAND_TOPIC_CHARACTER
Type: CLAMP_HP / CLAMP_MP

Body:
{
  "channelId": 1,
  "maxValue": 800
}
```

atlas-character handles the command:
1. Fetches current HP/MP from database
2. If current > maxValue, sets current = maxValue
3. Publishes `STAT_CHANGED` event

**Pros:**
- Uses existing Kafka infrastructure (no new topic)
- Character service remains sole authority on HP/MP mutations
- No REST lookup needed - effective-stats just sends the new max
- Follows existing command patterns

**Cons:**
- New command types needed (can consolidate with set/change later)

### 3.2 REST for MP Growth Calculation

atlas-character queries effective-stats via REST for level-up MP growth:

```
GET /worlds/{worldId}/channels/{channelId}/characters/{characterId}/stats
```

This is acceptable because level-up is infrequent.

### 3.3 Summary

1. **Command-Driven for HP/MP Clamping** - atlas-effective-stats sends `CommandClampHP`/`CommandClampMP`
2. **REST for MP Growth Calculation** - One-time query during level-up (low frequency)
3. **Synchronize Base Stats** - atlas-character publishes base stat updates; atlas-effective-stats consumes

---

## 4. Implementation Phases

### Phase 1: Base Stats Synchronization

**Goal:** Keep atlas-effective-stats informed of base stat changes from atlas-character

1.1 Modify atlas-character to publish base stat updates on existing `EVENT_TOPIC_CHARACTER_STATUS`
1.2 Add consumer in atlas-effective-stats for character stat changes
1.3 Call `SetBaseStats()` when base MaxHP/MaxMP/INT changes

### Phase 2: Effective Stats Query Integration

**Goal:** Enable atlas-character to query effective stats for MP growth calculation

2.1 Create external REST client in atlas-character for effective-stats
2.2 Modify `getMaxMpGrowth()` to fetch effective Intelligence
2.3 Add caching/fallback if effective-stats unavailable

### Phase 3: HP/MP Bounds Using Effective Stats

**Goal:** Use effective MaxHP/MaxMP in `ChangeHP()` and `ChangeMP()`

3.1 Modify `ChangeHP()` to query effective MaxHP
3.2 Modify `ChangeMP()` to query effective MaxMP
3.3 Optionally add effective stats to processor context

### Phase 4: HP/MP Clamping on Max Decrease

**Goal:** Clamp current HP/MP when effective max decreases

4.1 Add `CommandClampHP` and `CommandClampMP` to character Kafka message types
4.2 Modify atlas-effective-stats registry to track previous computed values
4.3 Detect MaxHP/MaxMP decreases in `Recompute()`
4.4 Create Kafka producer in atlas-effective-stats for character commands
4.5 Publish `CommandClampHP`/`CommandClampMP` when decrease detected
4.6 Add handler in atlas-character for clamp commands
4.7 Implement clamping logic: if current > maxValue, set current = maxValue
4.8 Emit `STAT_CHANGED` event after clamping

---

## 5. Detailed Task Breakdown

### Phase 1: Base Stats Synchronization (Effort: M)

| Task | Description | Effort | Dependencies |
|------|-------------|--------|--------------|
| 1.1 | Extend `StatusEventStatChangedBody` to include stat values, not just names | S | - |
| 1.2 | Modify `statChangedProvider` to include MaxHP, MaxMP, INT values | S | 1.1 |
| 1.3 | Create Kafka consumer in atlas-effective-stats for `STAT_CHANGED` | M | 1.2 |
| 1.4 | Call `SetBaseStats()` when relevant stats change | S | 1.3 |
| 1.5 | Test base stats sync on level-up, AP distribution | S | 1.4 |

### Phase 2: Effective Stats Query (Effort: M)

| Task | Description | Effort | Dependencies |
|------|-------------|--------|--------------|
| 2.1 | Create `external/effective-stats/rest.go` in atlas-character | S | - |
| 2.2 | Create `external/effective-stats/requests.go` with REST client | M | 2.1 |
| 2.3 | Add effective-stats service URL to configuration | S | 2.2 |
| 2.4 | Modify `getMaxMpGrowth()` to use effective Intelligence | S | 2.3 |
| 2.5 | Add graceful fallback to base Intelligence if service unavailable | S | 2.4 |

### Phase 3: HP/MP Bounds with Effective Stats (Effort: M)

| Task | Description | Effort | Dependencies |
|------|-------------|--------|--------------|
| 3.1 | Modify `ChangeHP()` to fetch effective MaxHP | M | 2.2 |
| 3.2 | Modify `ChangeMP()` to fetch effective MaxMP | S | 3.1 |
| 3.3 | Modify `SetHP()` to use effective MaxHP | S | 3.2 |
| 3.4 | Consider caching effective stats in processor context | M | 3.3 |
| 3.5 | Test HP/MP bounds with active buffs | S | 3.4 |

### Phase 4: HP/MP Clamping (Effort: M)

| Task | Description | Effort | Dependencies |
|------|-------------|--------|--------------|
| 4.1 | Add `CommandClampHP` and `CommandClampMP` to `kafka/message/character/kafka.go` | S | - |
| 4.2 | Add handler in atlas-character for `CommandClampHP` | M | 4.1 |
| 4.3 | Add handler in atlas-character for `CommandClampMP` | S | 4.2 |
| 4.4 | Add `previousComputed` field to effective-stats character Model | S | - |
| 4.5 | Modify `Recompute()` to detect MaxHP/MaxMP decreases | M | 4.4 |
| 4.6 | Create Kafka producer in atlas-effective-stats for character commands | M | 4.5 |
| 4.7 | Publish `CommandClampHP`/`CommandClampMP` when decrease detected | S | 4.6 |
| 4.8 | Test buff expiry clamping (Hyper Body) | M | 4.7 |
| 4.9 | Test equipment unequip clamping | M | 4.8 |

---

## 6. Risk Assessment

### 6.1 Race Conditions

**Risk:** Buff expires → effective-stats sends ClampHP command → atlas-character clamps HP → but HP was already modified by another operation

**Mitigation:** Use database transactions with optimistic locking. The clamp operation should be atomic and idempotent.

### 6.2 Command Ordering

**Risk:** ClampHP command arrives before atlas-character has processed an HP change that caused current > max

**Mitigation:** Clamp command processing always re-fetches current HP/MP from database before clamping. The check `if current > maxValue` handles natural ordering - if HP is already below max, clamp is a no-op.

### 6.3 Service Unavailability

**Risk:** effective-stats service is down when atlas-character needs effective stats

**Mitigation:**
- Phase 2/3: Fall back to base stats with warning log
- Phase 4: Kafka commands are persisted; clamping will occur when consumer catches up

### 6.4 Performance Impact

**Risk:** REST calls to effective-stats on every HP/MP change could impact performance

**Mitigation:**
- Consider caching effective stats in processor with TTL
- Command-driven clamping (Phase 4) removes need for REST in hot path
- Level-up (Phase 2) is infrequent

---

## 7. Success Metrics

1. **MP Growth Accuracy** - MP gained on level-up correctly incorporates INT from equipment/buffs
2. **HP/MP Bounds** - `ChangeHP()` and `ChangeMP()` never set values above effective max
3. **Clamp Latency** - HP/MP clamped within 500ms of buff expiry/equipment removal
4. **Zero Over-Max States** - No character ever has current HP/MP > effective MaxHP/MaxMP
5. **Graceful Degradation** - System continues functioning if effective-stats is temporarily unavailable

---

## 8. Dependencies

### External Services
- `atlas-effective-stats` - Must be deployed and accessible
- Kafka - For event publishing/consumption

### Internal Dependencies
- `atlas-character` processor has access to Kafka producer
- `atlas-effective-stats` registry tracks computed values

### Configuration
- `EFFECTIVE_STATS_SERVICE_URL` environment variable in atlas-character
- `COMMAND_TOPIC_CHARACTER` - existing topic, used by effective-stats to send clamp commands

---

## 9. Testing Strategy

### Unit Tests
- `getMaxMpGrowth()` with mocked effective-stats response
- `Recompute()` decrease detection logic
- Clamping edge cases (current = max, current > max, current < max)

### Integration Tests
- Buff application → effective stats increase
- Buff expiry → effective stats decrease → HP/MP clamped
- Equipment equip/unequip → effective stats change → HP/MP clamped
- Level-up with buffs active → correct MP growth

### Manual Testing
- Apply Hyper Body (60% HP buff) → HP increases to new max → buff expires → HP clamped
- Equip HP gear → HP capped at new max → unequip → HP clamped

---

## 10. Open Questions

1. **Should we cache effective stats in atlas-character?** - Could reduce REST calls but adds complexity
2. **What about SetHP() being called with value > effectiveMax?** - Should it clamp immediately or reject?
3. **Long-term: Consolidate Set/Change/Clamp commands?** - Three commands with overlapping semantics; consider unifying later
4. **How to handle login/character creation?** - Base stats should be synced on session start (already handled by effective-stats session consumer)
