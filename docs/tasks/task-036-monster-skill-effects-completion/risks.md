# Risks — task-036

Captured separately because several items in this task have non-local consequences and the plan phase should weigh mitigations explicitly.

---

## 1. Reflect mirror eventual consistency

**Risk.** atlas-channel's `StatusMirror` is populated by Kafka events. If the broker lags, a player's attack could land before the `STATUS_APPLIED` event for a freshly-applied reflect arrives, and the reflect window is missed. Symmetrically, an expired reflect's removal could lag behind, causing a "ghost reflect" hit on the player.

**Mitigation.**
- Apply path: the picker's existing animation delay (`processor.go:587-595`, default 100s of ms from `animation_times.skill1`) gives the consumer plenty of slack on the apply side.
- Expiration: `StatusExpirationTask` runs server-side first, the event hits Kafka almost immediately; client-side mirror lag is bounded by consumer processing time. Acceptable.
- The mirror is **not** treated as authoritative for invariants that can't tolerate drift. Reflect is fire-and-forget cosmetics-and-PvE; no money / inventory state hinges on its precision.

**Action for plan phase.** Add a regression test where a reflect status is applied, expires, and a "stale" attack referencing that monster after expiry produces no `DAMAGE_REFLECTED` event.

---

## 2. Mist explosion under heavy maps

**Risk.** Mist tick is `O(charactersInField × activeMists)` per second. A pathological boss map with 30 players and 5 simultaneous mists is 150 disease-applies per second. Each apply produces a Kafka command, so atlas-buffs will see proportional load.

**Mitigation.**
- Replacement semantics on the buff side: re-applying POISON to a character that already has POISON resets duration, doesn't duplicate. So the steady-state load is bounded by characters-in-zone, not characters × mists.
- Add a per-mist max-duration sanity in atlas-monsters (skill data can't request > 60 s mist; bail loudly if it does).

**Action for plan phase.** Benchmark the tick task at 10 mists × 50 characters; if > 50 ms per tick on dev hardware, parallelise the inner character loop.

---

## 3. Venom slot replacement edge cases

**Risk.** Concurrent applies racing into the slot allocator. Two near-simultaneous venom applies could both observe "all 3 slots free" and both insert into slot 1.

**Mitigation.**
- The whole `allocateVenomSlot → cancel-old → apply-new` sequence MUST run under a single registry write lock. Don't decompose it into separate critical sections.

**Action for plan phase.** Add a concurrency test: 10 goroutines simultaneously applying venom; assert exactly 3 slots are occupied at the end and no slot is double-assigned.

---

## 4. Reflect range axis ambiguity

**Risk.** The PRD assumes a 1-D X-axis distance check (`|attacker.X - monster.X| ≤ Range`). v83 servers commonly use Euclidean distance. If WZ data provides Y as the radius and we use it as a damage cap, reflect won't trigger correctly at edge cases.

**Mitigation.**
- §6.2 of the PRD locks the **structure**, leaves the **mapping** to the plan phase after WZ inspection.
- Plan phase TDD MUST include a manual cross-check against legacy server captures for at least one reflect skill (Pap's WeaponReflect is the canonical reference).

**Action for plan phase.** Verify by reading `mobskill.Model.X()/Y()/LtX/LtY/RbX/RbY` for skill type 145 (WEAPON_REFLECT) against the running atlas-data instance; lock the mapping in the plan.

---

## 5. cjson empty-array regressions on extension

**Risk.** Adding the four `Reflect*` fields to `StatusEffectAppliedBody` means a Lua consumer that previously decoded the body as a table-with-known-shape might choke on the new keys, or — more likely — a marshaller might serialize zero values as `null`.

**Mitigation.**
- All new numeric reflect fields are scalars, not slices — cjson empty-array gotcha doesn't apply.
- `reflectKind` is a string with default `""`; no `omitempty`.
- Audit task in §FR-4.10 covers existing fields.

**Action for plan phase.** Round-trip test for `StatusEffectAppliedBody` with a non-reflect status — assert the four reflect fields serialize as `""` and `0`, not absent.

---

## 6. Mist on instance maps

**Risk.** Instance fields (UUID-scoped) are easy to break: a mist created in instance A could be ticked against characters in instance B if the field key is mis-derived.

**Mitigation.**
- The `MistRegistry.byField` index uses `field.Model.Key()` (or equivalent) which already encodes `(world, channel, mapId, instance)`.
- `MistTickTask` MUST use `_map.CharacterIdsInFieldProvider(field)` not a map-id-only lookup.

**Action for plan phase.** Test with two instance fields of the same `mapId`; assert mist created in one does not tick characters in the other.

---

## 7. PoisonTick + Expiration race

**Risk.** PoisonTick reads from `GetPoisonCharacters` while Expiration is concurrently expiring buffs. A poison buff that expires mid-tick could produce one extra damage event after expiry.

**Mitigation.**
- `GetPoisonCharacters` already filters `b.Expired()` (see `services/atlas-buffs/atlas.com/buffs/character/registry.go:217-235`).
- Tasks run on independent goroutines; the read-then-produce window is bounded by the registry mutex.
- One extra damage tick on expiry is preferable to zero ticks on near-expiry; accept the trade-off.

**Action for plan phase.** Document the trade-off in a code comment; no test required.

---

## 8. Holy Shield bypass via mist tick

**Risk.** Mist tick re-applies disease every second. If Holy Shield's `HasImmunity` check has any window where it returns false (e.g., during buff replacement), the disease could slip through for one tick.

**Mitigation.**
- atlas-buffs apply path is single-call; `HasImmunity` reads under registry lock so there's no race.
- Existing test coverage in atlas-buffs for Holy Shield should be re-affirmed but not extended.

**Action for plan phase.** Smoke test: character with Holy Shield standing in a mist for 10 ticks receives zero POISON applies.

---

## 9. Missing affected-area packet writers

**Risk.** If `libs/atlas-packet/map/clientbound/` lacks affected-area writers, the mist arc adds a packet-layer dependency that wasn't in the original brief's mental model.

**Mitigation.**
- Open question §9-1 in the PRD acknowledges this.
- The plan adds the writers as a leaf task with its own TDD; impact is contained.

**Action for plan phase.** Verify before the mist consumer task; if writers are missing, sequence them as a prerequisite.
