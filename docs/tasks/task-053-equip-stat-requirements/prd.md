# atlas-effective-stats: honor equipment stat requirements when computing bonuses — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-03
---

## 1. Overview

`atlas-effective-stats` accumulates per-character stat bonuses from three sources — equipment, buffs, and passive skills — and exposes the sum as `Computed.maxHp`, `Computed.maxMp`, `Computed.strength`, etc. The equipment path was repaired in task-031 so per-asset HP/MP/STR/DEX/INT/LUK contributions actually reach the bonuses array. The remaining bug: **the equipment path emits a bonus for every asset in the equip compartment with `Slot < 0`, regardless of whether the wearer actually meets the item's `reqLevel` / `reqJob` / `reqStr` / `reqDex` / `reqInt` / `reqLuk` requirements**.

The v83 game client *does* enforce these requirements client-side — when the wearer fails a requirement, the client refuses to apply that item's contribution to the displayed MaxHp/MaxMp/STR/etc. So the client and atlas-effective-stats disagree by the sum of bonuses from "equipped but unqualified" items. The most user-visible consequence: `atlas-character.ChangeMP` clamps MP regen to `Computed.maxMp` (which over-counts by the unqualified-item delta), while the client's HP/MP bar shows the lower client-computed cap. Players see MP "stop regenerating" before the bar fills, or worse, regen tops out at a different value than the displayed cap.

This was discovered diagnosing PR #383 (an attempted display fix that mis-attributed the symptom to a base-vs-effective drift in atlas-channel and was reverted in PR #385). The actual root cause is `atlas-effective-stats`'s lack of requirement gating, not the `atlas-channel` display path.

## 2. Goals

Primary goals:
- For each equipped asset (`Slot < 0`) in compartment type 1, atlas-effective-stats MUST emit equipment bonuses **only when the wearer's currently-applied stats + level + job meet every populated requirement on the item's template** (`reqLevel`, `reqJob`, `reqStr`, `reqDex`, `reqInt`, `reqLuk`).
- Cross-asset qualification: an item's stat contribution is allowed to count toward *another* item's requirement (e.g., a +5 STR cape with no requirements can satisfy a +20 W.ATK weapon's `reqStr`). Use a fixed-point iteration over the set of equipped items so partial sets converge to the largest qualifying subset.
- Re-evaluate the qualifying set whenever wearer-side stats / level / job change (not just at lazy-init / equip time), so a `LUK +1` AP distribution can reactivate previously-unqualifying items, and a job change can deactivate them.
- Equipment template requirements are fetched per-tenant from atlas-data and cached indefinitely (immutable per tenant; equip stats vary across MapleStory versions).
- New unit + integration tests covering the requirement gate, the fixed-point convergence, and the re-gating triggers.
- End-to-end smoke verification on the dev cluster that the diagnostic case (overall granting +50 MP, wearer LUK below `reqLuk`) drops out of `Computed.maxMp` until LUK is raised.

Non-goals:
- No server-side enforcement of equip *requests* (atlas-inventory does not gate the underlying MOVE today; that's intentional and out of scope — players can have stale equips after a stat reset, which the client correctly displays without applying bonuses; we only need to mirror that bonus-accounting behavior in atlas-effective-stats).
- No changes to atlas-channel's display path. Once atlas-effective-stats agrees with the client, the existing base+client-equip display naturally lands on the true effective cap.
- No cash-equipment requirement gating. v83 cash equips don't carry stat requirements and don't grant stat bonuses, so the existing `Slot < 0` filter naturally excludes them with no special-case code.
- No item potential / hidden potential gating. Not modeled in the system today; revisit if/when introduced.
- No changes to the buff or passive-skill bonus paths. Those sources don't carry wearer-stat requirements (skills have their own gating which is upstream).
- No changes to atlas-data, atlas-inventory, or atlas-character REST/event surfaces. All required fields already exist.

## 3. User Stories

- As a player on the dev cluster wearing an overall granting +50 MP that I don't meet the LUK requirement for, my MP bar's max equals the cap that MP regen actually clamps to (i.e., 6330, not 6380), so MP regen tops up to the displayed max with no gap.
- As a GM looking at the Character Detail page (atlas-ui), `effective.maxMP` reflects only the equipment my character qualifies for, so the displayed `+bonus` matches what the player sees in the v83 client.
- As a player who just used AP to raise LUK from 39 to 40 (the threshold for my overall), my MP cap immediately rises by 50 once the `STAT_CHANGED` event reaches atlas-effective-stats — without re-logging or re-equipping the item.
- As a player who just job-advanced from Beginner to Magician, equipment with `reqJob = magician-class` becomes immediately qualifying without manual re-equipping.
- As a backend developer, I can call `GET /api/worlds/0/channels/0/characters/{id}/stats` and see, in the `bonuses[]` array, only entries from items the wearer actually qualifies for (the response currently emits *all* equipped items' bonuses).

## 4. Functional Requirements

### 4.1 Diagnosis (canonical reproduction)

On the dev cluster (atlas tenant, GMS / 83 / 1):

- Character `<dev character id>` has `LUK = 39` (base, after most recent AP distribution) and is wearing an overall at slot `<-5>` with templateId `<overall id>` whose template carries `reqLuk = 40` and grants `incMMP = 50`.
- Direct curl `GET /api/characters/<id>/inventory/compartments?type=1&include=assets` returns the asset with `slot=-5, mp=50`.
- Direct curl `GET /api/equipment/<templateId>` against atlas-data returns `reqLuk=40` (and other reqs zero).
- Direct curl `GET /api/worlds/0/channels/0/characters/<id>/stats` against atlas-effective-stats today returns `Computed.maxMP = base + 50` (i.e., includes the unqualified item) and a `bonuses[]` entry `{source: "equipment:<assetId>", statType: "TypeMaxMp", amount: 50}`.
- In game, the v83 client's MP bar shows the wearer's cap **without** the +50 (consistent with the client's `reqLuk` enforcement). MP regen via `atlas-character.ChangeMP` clamps to `Computed.maxMp` (with the +50), so MP exceeds the displayed bar by 50.

After this fix:
- `Computed.maxMP` returned from atlas-effective-stats no longer includes the +50 from the overall (because `reqLuk = 40 > base.luck = 39` fails).
- `bonuses[]` no longer contains the `equipment:<assetId>` entry for any of that asset's stats (HP/MP/STR/etc.).
- After a `+1 LUK` AP distribution lands (atlas-character emits `STAT_CHANGED` with `TypeLuck` and `values["luck"] = 40`), atlas-effective-stats re-evaluates, the overall now qualifies, the +50 MP bonus reappears in `Computed.maxMP` and `bonuses[]`, and ChangeMP regen clamps at the higher cap.

### 4.2 Requirement-evaluation contract

For each equipped asset (`Slot < 0`) in compartment type 1:
1. Look up the asset's `templateId`.
2. Fetch (or read from cache) the equipment template's requirements: `reqLevel`, `reqJob`, `reqStr`, `reqDex`, `reqInt`, `reqLuk`.
3. Evaluate `meetsRequirements(asset, wearer)`:
   - `wearer.level >= reqLevel` (skip check when `reqLevel == 0`)
   - `reqJob == 0` OR `(wearer.jobId & reqJob) != 0` (v83 reqJob is a bitmask — Beginner=0, Warrior=1, Magician=2, Bowman=4, Thief=8, Pirate=16; cross-class items use OR'd bitmasks. `reqJob == 0` means "no class restriction.")
   - `wearer.strength >= reqStr` (skip when `reqStr == 0`)
   - `wearer.dexterity >= reqDex` (skip when `reqDex == 0`)
   - `wearer.intelligence >= reqInt` (skip when `reqInt == 0`)
   - `wearer.luck >= reqLuk` (skip when `reqLuk == 0`)
   - All populated checks must pass for the asset to qualify.
4. If qualified, emit bonuses for every non-zero stat on the asset (existing logic in `fetchEquipmentBonuses` / `extractEquipmentBonuses`); if not, skip the asset entirely.

### 4.3 Cross-asset qualification (fixed-point iteration)

The wearer's "currently-applied stats" used in §4.2 step 3 are not just the persisted base. Other equipped items' bonuses can contribute toward an item's requirements (e.g., wearing a `+5 STR` cape with no requirement can let a `reqStr = 100` weapon qualify when base STR is 95). Therefore the qualifying set must be computed by fixed-point iteration:

```
applied  := []           // assets currently contributing bonuses
candidates := all equipped assets

repeat
    augmented_stats := wearer.base + sum(stat bonuses from `applied`) + buff bonuses + passive bonuses
    new_applied := [a in candidates if meetsRequirements(a, augmented_stats)]
until new_applied == applied
applied := new_applied
```

- An item's *own* stats DO count toward computing whether *other* items qualify, but its bonuses only enter `applied` when the item itself qualifies under the prior iteration's stats. (An item cannot bootstrap its own qualification — solving the chicken-and-egg case in v83's favor.)
- Iteration is monotonic: the qualifying set can only grow each round (because adding bonuses to `augmented_stats` cannot disqualify a previously-qualified item — every requirement is a `>=` check). Therefore convergence is bounded by `len(candidates)`, in practice at most one pass per equipment slot (~12 iterations max).
- Buff and passive bonuses are included in `augmented_stats` from iteration 1 onward (they don't have requirements themselves and are always applied), so a passive STR boost can also help qualify an equip.

Implementation sketch: rather than literally re-computing the full sum each iteration, a single forward pass that processes assets in stable order and re-checks requirements until no new asset qualifies is equivalent and O(n²) on equipment slot count (small constant — ≤12 slots).

### 4.4 Re-evaluation triggers (dynamic re-gating)

The qualifying set must be re-computed whenever any of the wearer's inputs to `meetsRequirements` change:
- `STAT_CHANGED` events with `TypeStrength`, `TypeDexterity`, `TypeIntelligence`, `TypeLuck`, `TypeLevel`, `TypeJob` (currently the `kafka/consumer/character/consumer.go` handler filters only on STR/DEX/INT/LUK + MAX_HP/MAX_MP; the filter set must expand to include LEVEL and JOB).
- `JOB_CHANGED` event (already emitted alongside `STAT_CHANGED{TypeJob}` per atlas-character `processor.go:494-495`; subscribing to either is sufficient — extending the existing `STAT_CHANGED` handler is preferred to avoid duplicate logic).
- Asset `MOVED` (equip/unequip) and `DELETED` events: existing handlers already trigger per-asset bonus add/remove. Augment them so an equip event also re-evaluates the *other* equipped items (a newly-equipped +STR cape can reactivate a previously-unqualified weapon).

Re-evaluation procedure:
1. Load the wearer's current base stats (already in the registry model after `SetBaseStats`).
2. Load the equipped-asset list from the registry's bonus map (every `source` matching `^equipment:\d+$`) — this is enough; we don't need to re-fetch atlas-inventory unless we lack per-asset stats. (Keep per-asset stat snapshots in the registry alongside bonus entries so dynamic re-gating doesn't require an inventory round-trip on every stat change. See §6.)
3. Recompute the qualifying set per §4.3.
4. Diff: items that newly qualify get their bonuses added; items that no longer qualify get their bonuses removed.
5. For HP/MP cap *decreases* (an item dropping out of the qualifying set can lower MaxHp/MaxMp), the existing `checkAndPublishClampCommands` path emits `CLAMP_HP` / `CLAMP_MP` to atlas-character — this stays correct.

### 4.5 Equipment template requirement fetch

New `external/data/equipment` client in atlas-effective-stats:
- Endpoint: `GET /api/equipment/{templateId}` against atlas-data (already exists; returns `reqLevel`, `reqJob`, `reqStr`, `reqDex`, `reqInt`, `reqLuk` per `services/atlas-data/atlas.com/data/equipment/rest.go:33-38`).
- Per-tenant indefinite cache. Key = `(tenantId, templateId)`. Value = the `EquipmentRequirements` struct (the six fields + a `loadedAt` timestamp for diagnostics; no eviction).
- Cache scope rationale: equipment template data is immutable within a tenant (atlas-data WZ files don't change at runtime), but equip stats and reqs vary across MapleStory versions, which atlas serves via different tenants (`GMS/83`, `GMS/95`, `JMS/...`). Per-tenant keying preserves correctness.
- On cold-cache miss, fetch from atlas-data; on fetch error and no cached value, **drop the asset's bonuses for this evaluation** (under-report rather than over-report) and log at WARN. On fetch error *with* a cached value, use the cached value (degraded read).

### 4.6 Persisted per-asset stat snapshot in the registry

To avoid re-fetching atlas-inventory on every wearer-stat change, persist the asset's stat snapshot (the existing `EquipableRestData` shape) inside the registry's bonus map alongside the `source = equipment:<id>` entries. New shape (conceptually):

```go
type EquippedAssetSnapshot struct {
    AssetId     uint32
    TemplateId  uint32
    Stats       stat.Bonus[]   // pre-computed flat bonuses (existing path)
}
```

The `MOVED` (equip) handler already fetches the per-asset stats; cache them keyed by assetId. The `MOVED` (unequip) and `DELETED` handlers already remove by source — extend to also remove the snapshot. Re-gating reads from this map; only requirements (template-keyed) come from the equipment-data cache.

### 4.7 Tests

- Unit: `meetsRequirements(stats, reqs) bool` covering all six predicate dimensions, zero-requirement skips, bitmask reqJob match, and edge cases (off-by-one at exactly `req`, `req-1`, `req+1`).
- Unit: fixed-point iteration on a synthetic 4-equip set where item A's bonus qualifies item B, item B's bonus qualifies item C, base stats qualify only item A → all three converge into the qualifying set; and a 2-equip cycle where A and B mutually require each other's bonuses → neither qualifies (cannot bootstrap).
- Integration: with a stub atlas-inventory returning one compartment whose `included` carries an asset at slot -5 with `mp=50` and templateId 1052095, plus a stub atlas-data returning `reqLuk=40` for that template, calling `GetEffectiveStats` for a character with `base.luck = 39` produces a `Computed.maxMp` *equal to base* (asset dropped) and a `bonuses[]` entry that does NOT contain `equipment:<assetId>`.
- Integration: with the same stubs but `base.luck = 40`, the asset qualifies — `Computed.maxMp = base + 50` and the `bonuses[]` entry is present.
- Integration: dispatch a `STAT_CHANGED` event with `TypeLuck` and `values["luck"] = 40` to the consumer and assert the registry transitions from "asset-not-qualifying" to "asset-qualifying" (asset stat snapshot preserved across the transition; no re-fetch from atlas-inventory).
- Integration: dispatch `STAT_CHANGED` with `TypeJob` and confirm a job-restricted item flips qualifying status.
- Equipment-data cache test: two consecutive `meetsRequirements` calls for the same templateId result in exactly one HTTP fetch against atlas-data; cache miss followed by atlas-data 5xx falls back to "not qualified" with WARN log.
- Smoke test post-deploy: re-run §4.1 reproduction against the dev cluster character and verify `Computed.maxMp` no longer includes the unqualified overall's +50, and that adding a LUK AP makes it reappear.

## 5. API Surface

No new endpoints. No request/response shape changes.

The existing `GET /api/worlds/{worldId}/channels/{channelId}/characters/{characterId}/stats` response remains the same shape:

```json
{
  "data": {
    "type": "effective-stats",
    "id": "12",
    "attributes": {
      "strength": 4, "dexterity": 25, "luck": 39, "intelligence": 4,
      "maxHP": 1430, "maxMP": 6330,
      "weaponAttack": 0, "weaponDefense": 0, "magicAttack": 0, "magicDefense": 0,
      "accuracy": 0, "avoidability": 0, "speed": 20, "jump": 0,
      "bonuses": [
        {"source": "passive:1002", "statType": "speed", "amount": 20}
      ]
    }
  }
}
```

After the fix, the only behavioral difference is that `bonuses[]` excludes any `equipment:<assetId>` entries whose template requirements aren't met by the wearer's current stats / level / job, and the `attributes` aggregates (`maxHP`, `maxMP`, `strength`, etc.) reflect that exclusion.

## 6. Data Model

### 6.1 In-process state (atlas-effective-stats)

Two new in-memory caches, both per-tenant:

**Equipment template cache** (new):
- Key: `(tenantId, templateId)`
- Value: `EquipmentRequirements{ReqLevel, ReqJob, ReqStr, ReqDex, ReqInt, ReqLuk}`
- Eviction: none (indefinite; bounded by total templateId count per tenant, ~5,000–10,000).
- Multi-tenancy: keyed by `(tenantId, templateId)` to handle version-dependent reqs across tenants.

**Equipped-asset snapshot map** (new, scoped to existing per-character registry model):
- Key: `assetId`
- Value: `EquippedAssetSnapshot{AssetId, TemplateId, Stats}` (per §4.6)
- Lifecycle: populated on equip / lazy-init, removed on unequip / delete. Lives alongside the existing `bonuses[]` in the registry's character model.

### 6.2 Persistence

No database changes. atlas-effective-stats is process-local (the registry rebuilds on lazy-init from atlas-character + atlas-inventory + atlas-buffs + atlas-skills); the new caches and snapshot map follow the same volatility.

## 7. Service Impact

### services/atlas-effective-stats (primary, only)

New files:
- `external/data/equipment/rest.go` — `EquipmentRestModel` mirroring atlas-data's response (only the six req fields needed; existing equip stat fields can be ignored or carried for symmetry).
- `external/data/equipment/requests.go` — `RequestById(templateId)` request builder analogous to `external/data/skill/requests.go`.
- `external/data/equipment/cache.go` (or co-located in `requests.go`) — per-tenant indefinite cache with `Get(ctx, templateId) (EquipmentRequirements, error)`.

Modified files:
- `character/initializer.go::fetchEquipmentBonuses` — wrap existing per-asset bonus emission with the fixed-point qualifying-set computation, gating each iteration on `meetsRequirements`. Populate the new equipped-asset snapshot map on each qualified asset.
- `character/processor.go::AddEquipmentBonuses` — add the asset snapshot before computing bonuses; then run the (incremental) requirement check including the rest of the qualifying set. If qualified, emit bonuses; if not, store the snapshot but emit nothing.
- `character/processor.go::RemoveEquipmentBonuses` — remove the asset snapshot in addition to the existing bonus removal.
- `character/model.go` — extend `Model` with the equipped-asset snapshot map (or place it in the registry sibling to bonuses; either is acceptable as long as it's serialized atomically with bonuses).
- `kafka/consumer/character/consumer.go::handleStatChanged` — expand the `hasRelevantStats` filter to include `LEVEL` and `JOB`. After computing the new base, trigger a re-evaluation of the qualifying set via a new `Processor.RecomputeEquipmentBonuses` or equivalent.
- `kafka/consumer/asset/consumer.go::handleItemEquipped` and `handleAssetDeleted` — call the new `Processor.RecomputeEquipmentBonuses` after the per-asset add/remove so a newly-equipped item that supplies a stat boost can promote previously-unqualified items.

### Other services

Not affected:
- atlas-data — no changes; `services/atlas-data/atlas.com/data/equipment/rest.go:33-38` already exposes all six req fields.
- atlas-inventory — no changes; the existing equip compartment + assets shape carries everything atlas-effective-stats needs (after task-031's hydration fix).
- atlas-character — no changes; emits the events atlas-effective-stats consumes (`STAT_CHANGED` covers STR/DEX/INT/LUK/LEVEL/JOB; `JOB_CHANGED` is redundant but harmless).
- atlas-channel — no changes; once `Computed.maxMp` is correct, the existing display path (which sends base MaxMp and lets the client add equip locally) lands on the correct cap.

## 8. Non-Functional Requirements

- **Performance:** Fixed-point iteration is O(n²) on equipment slot count; n ≤ 12 in v83, so worst case is 144 requirement checks per evaluation. Each check is `O(1)` arithmetic plus an `O(1)` cache lookup for template reqs (or one HTTP round-trip on cold cache, hopefully amortized to ~zero after warm-up). Re-evaluation triggers per `STAT_CHANGED` event are already handled per-event; the additional work is bounded.
- **Multi-tenancy:** Equipment template cache must key by `tenantId` (per user clarification — equip stats and reqs vary across MapleStory versions, which different tenants serve). The atlas-effective-stats request flow already propagates tenant headers via `requests.GetRequest`'s decorator chain; no new tenant-handling code needed.
- **Observability:** Add DEBUG logs on per-asset gate decisions (`asset [X] qualifies / does not qualify: <which req failed>`) and INFO logs on every fixed-point convergence (number of qualifying items vs total equipped). WARN log on equipment-data fetch failure with cold cache.
- **Backward compatibility:** No request/response shape changes. Consumers (atlas-ui, atlas-channel) parse `data.attributes` directly; values may decrease for any character wearing items they don't qualify for (this is the *intended* behavior — they were over-reported before).
- **Failure modes:**
  - atlas-data unreachable, cold cache: drop the asset's bonuses, log WARN. Wearer sees a temporarily reduced cap; on next stat-changed re-evaluation, atlas-data may be reachable again and the bonus reappears. Acceptable degradation.
  - atlas-data unreachable, warm cache: use cached value. (Indefinite TTL means warm cache stays warm forever within a process lifetime, so this is the typical post-warmup state.)
- **Logging:** Existing equipment-bonus debug logs already exist (`Fetched %d equipment bonuses for character [%d].`); after this fix, this number reflects only qualifying items. New DEBUG log on requirement-gate misses for triage.

## 9. Open Questions

- (For the design phase) Where does the equipped-asset snapshot map live? Co-located with `bonuses[]` in the character model, or in a sibling registry struct? Either works — design phase to pick the cleaner shape.
- (For the design phase) Is the asset snapshot's `Stats` field the same `[]stat.Bonus` we already produce in `extractEquipmentBonuses`, or do we keep raw `EquipableRestData` and re-extract on each evaluation? The first is denser; the second is more flexible if we ever need to reinterpret stats.
- (For the design phase) How does the new requirement-data cache interact with atlas-effective-stats's existing read patterns? Plumbing: a new package-level singleton vs a registry-scoped map? Existing patterns favor singletons via `sync.Once` (per atlas-channel's `effective_stats` package; though that's the consumer side). Picked in design.
- (For the implementation phase) Equipment reqJob bitmask semantics in v83 — verify atlas-data's `reqJob` exposes the raw bitmask (it should; `services/atlas-data/atlas.com/data/equipment/reader.go:104` reads it directly via `GetShort("reqJob", 0)`). Cross-reference to v83 client's `IsEquipPossible` if available; otherwise verify against documented v83 reqJob values (Beginner=0, Warrior=1, Magician=2, Bowman=4, Thief=8, Pirate=16, OR'd for cross-class).
- (For the implementation phase) Cyclic-dependency edge case in fixed-point iteration: A's bonus is needed for B's req, B's bonus is needed for A's req, but neither qualifies under base alone. Per §4.3, neither enters the qualifying set (correct — v83 doesn't allow self-bootstrapping pairs either). Add a unit test to lock this in.

## 10. Acceptance Criteria

- [ ] `GET /api/worlds/0/channels/0/characters/{id}/stats` for a character wearing an item the wearer doesn't qualify for returns `Computed.maxMp` (and other affected stats) **without** that item's contribution; the `bonuses[]` array does not contain an `equipment:<assetId>` entry for that asset.
- [ ] After a `STAT_CHANGED` event raises the wearer's base stat past the threshold, the same endpoint reflects the newly-qualifying item's bonus *without* requiring lazy-init / restart / re-equip.
- [ ] After a `STAT_CHANGED` event with `TypeJob` triggering a job change, items with non-zero `reqJob` flip qualifying status accordingly.
- [ ] After a `STAT_CHANGED` event with `TypeLevel` (level-up), items with `reqLevel` newly satisfied flip into the qualifying set.
- [ ] Equipping a +STR cape that lets a previously-unqualifying weapon qualify causes both items to appear in `bonuses[]` after the equip event is processed (cross-asset qualification verified end-to-end).
- [ ] Unit tests for `meetsRequirements` cover all six predicates, zero-skip semantics, and bitmask `reqJob` matching.
- [ ] Unit test for fixed-point iteration covers chain-qualification (A qualifies B qualifies C) and cyclic non-qualification (A and B mutually depend on each other; neither qualifies).
- [ ] Integration test using stub atlas-inventory + stub atlas-data verifies the canonical reproduction from §4.1.
- [ ] Per-tenant equipment template cache is hit on the second fetch of the same template within a tenant; cold-cache fetch failure falls back to "not qualified" with WARN log.
- [ ] `go build ./...` and `go test ./...` pass in `services/atlas-effective-stats/atlas.com/effective-stats`.
- [ ] Smoke verified against the dev cluster: the diagnosis character's overall +50 MP drops out of `Computed.maxMp` while LUK is below `reqLuk`, and reappears after a LUK AP distribution.
- [ ] No frontend changes required; no atlas-channel changes required.

## 11. Diagnosis Artifacts (for the fix author)

For posterity, the chain that surfaced this:

1. PR #383 (now reverted in PR #385) attempted to fix "MP regen exceeds displayed cap" by sending `effective.maxMp` (gear+buffs+passives) to the v83 client via atlas-channel's `BuildCharacterData` and `StatChanged` consumer.
2. After deploy, the user observed in-game MP bar showing **6710 / 6710** while regen capped at **6380**, vs DB base **6000** + atlas-effective-stats effective **6380** (= base + 380 equip).
3. The 6710 = 6380 (server-sent) + 330 (client's local equip-stat sum) — i.e., the v83 client adds equip stats from the items it knows about *on top of* whatever max the server sent. Sending effective double-counted the equipment.
4. PR #383 was reverted (PR #385).
5. Investigating the residual 50-unit gap (atlas-effective-stats says 380, client says 330): the user is wearing an overall granting +50 MP, but their LUK doesn't meet the overall's `reqLuk`. The v83 client correctly drops the +50; atlas-effective-stats wrongly includes it.
6. Root cause: `services/atlas-effective-stats/atlas.com/effective-stats/character/initializer.go::fetchEquipmentBonuses` and `kafka/consumer/asset/consumer.go::handleItemEquipped` accumulate every equipped asset's stats unconditionally; `external/data/equipment` doesn't even exist as a fetch path today.

This PRD covers the corrective fix.
