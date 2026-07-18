# Maple Warrior All-Stats Bonus — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Maple Warrior is the 4th-job party buff that raises a character's primary stats by X%.
In atlas-effective-stats, `stat.MapBuffStatType` maps the `MAPLE_WARRIOR` buff change to a
single `TypeStrength` multiplier bonus (`stat/model.go:432-436`, with an explicit
"for now" comment). The result: STR classes get a partial benefit, and DEX/INT/LUK
classes get no server-side stat benefit from Maple Warrior at all — their client shows
boosted stats while the server computes damage, accuracy, and requirement checks
against unboosted values.

The current mapping is wrong in two independent ways:

1. **Coverage** — the buff must add bonuses to all four primary stats (STR, DEX, INT,
   LUK), not just STR.
2. **Basis** — the engine applies multiplier bonuses to `(base + flat)` where flat
   includes equipment bonuses (`character/model.go:316`). The client applies Maple
   Warrior to the **raw base stat only**, excluding equipment (IDA-verified, §4.1).

This task fixes both: a one-to-many buff-stat mapping API that can express bonus kind,
a new base-percent bonus semantic in the computation engine, and migration of both
consumers of the old API.

## 2. Goals

Primary goals:
- Maple Warrior grants `floor(baseStat × rate / 100)` to each of STR, DEX, INT, LUK,
  matching the client's BasicStat computation byte-for-byte in basis and truncation.
- Replace the single-result `MapBuffStatType` API with a one-to-many mapping so no
  future one-to-many buff repeats this bug shape.
- Migrate both existing consumers (buff Kafka consumer, character initializer).

Non-goals:
- Changing HYPER_BODY semantics. The client multiplies accumulated MaxHP/MaxMP
  (base + equip) by the rate (v95 `BasicStat::SetFrom` HP/MP tail), which matches the
  engine's existing `(base + flat) × (1 + mult)` path. No change.
- Changing the atlas-data statup emitter or the buff service. The `MAPLE_WARRIOR`
  change type and its X-value amount are correct on the wire.
- Echo/Enrage/other percent-of-base buffs beyond Maple Warrior (none currently emit a
  `MAPLE_WARRIOR`-like type; the new API shape accommodates them later).
- Client-side display (client computes its own BasicStat; already correct).

## 3. User Stories

- As a DEX/INT/LUK-class player, I want Maple Warrior to raise my primary stat
  server-side so that my damage, accuracy, and equip/skill requirement checks reflect
  the buff the client shows me.
- As a STR-class player, I want Maple Warrior's STR gain computed from my base STR
  (not base + equipment) so that server and client stat panels agree.
- As a developer, I want the buff-stat mapping API to return a set of typed bonuses so
  that a buff affecting several stats cannot silently be reduced to one.

## 4. Functional Requirements

### 4.1 Verified client semantics (source of truth)

IDA-verified against both ends of the supported version range:

- **v83** (`MapleStory_dump.exe`, v83_Me IDB): `BasicStat::SetFrom` @ `0x77ec9f` —
  final block adds `rate × <CharacterData base stat> / 100` to each of the four
  primary-stat slots, reading the raw base stat straight from `CharacterData`
  (the same offsets the slots were initialized from), **not** the accumulated
  base+equip value. MaxHP/MaxMP are untouched by this rate. Caller
  `CWvsContext::ValidateStat` @ `0xa0843c` passes the rate from a SecondaryStat field.
- **v95** (`GMS_v95.0_U_DEVM.exe`): `CWvsContext::ValidateStat` @ `0x9e8670` reads
  `nBasicStatInc` from `m_secondaryStat.nBasicStatUp` (the Maple Warrior temporary
  stat) and passes it to `BasicStat::SetFrom` @ `0x732ba0`, which computes
  `nSTR += nBasicStatInc * characterStat.nSTR / 100` (likewise INT, DEX, LUK).
  Integer division — truncation, per stat. Equipment, set-item, and item-option
  bonuses are excluded from the basis. MaxHP/MaxMP unaffected.
- Ordering nuance: v83 applies the rate after ForcedStat overrides, v95 before.
  Irrelevant to Atlas (effective-stats has no ForcedStat analog).

Producer chain (repo-verified): atlas-data `skill/reader.go:315-318` emits statup
`TemporaryStatTypeMapleWarrior` (`"MAPLE_WARRIOR"`) with `amount = X` (the percent)
for all 14 Maple Warrior skill variants (Hero, Paladin, Dark Knight, F/P and I/L Arch
Magician, Bishop, Bowmaster, Marksman, Night Lord, Shadower, Corsair, Buccaneer,
Aran, Evan). The buff service relays these as `StatChange{Type, Amount}` in buff
status events and in the REST buff list.

### 4.2 Mapping API (stat package)

- FR-1: Replace `MapBuffStatType(buffType string) (Type, bool)` with a one-to-many
  mapping function that, for a given buff change type string, returns **all** affected
  stats together with the bonus kind for each. Three kinds must be expressible:
  - **flat** — add amount (existing behavior: WEAPON_ATTACK, SPEED, …)
  - **percent** — multiplier on `(base + flat)` (existing behavior: HYPER_BODY_HP/MP)
  - **base-percent** — `floor(base × amount / 100)` added flat (new: MAPLE_WARRIOR)
- FR-2: `MAPLE_WARRIOR` maps to four base-percent entries: `TypeStrength`,
  `TypeDexterity`, `TypeIntelligence`, `TypeLuck`.
- FR-3: All other currently-mapped buff types keep their existing single-entry
  mapping and kind, verbatim.
- FR-4: Unknown buff types return an empty result; callers keep the existing
  debug-log-and-skip behavior.
- FR-5: The old single-result function is **removed**, not aliased — the compiler
  must force every call site through the new API.

### 4.3 Computation engine (character model)

- FR-6: `Bonus` gains a base-percent dimension (field or kind discriminator — design
  phase decides the exact shape) alongside the existing `amount`/`multiplier`.
  `MarshalJSON`/`UnmarshalJSON` round-trip the new dimension; existing serialized
  bonuses (absent field) decode as zero (additive, backward compatible).
- FR-7: `ComputeEffectiveStats` (`character/model.go:269`) applies base-percent
  bonuses as `floor(baseValue × rate / 100)` added to the flat term, where
  `baseValue` is the character's base stat only (`m.baseStats`) — never equipment
  or other flat bonuses. Truncation must match the client's integer division.
  Resulting formula per stat:
  `effective = floor((base + flat + Σ floor(base × basePct_i)) × (1 + mult))`
  where for Maple Warrior `mult` contributions are zero and `basePct_i = rate/100`.
  Multiple base-percent bonuses on the same stat each truncate independently
  (client has only one such source, but the engine must be deterministic).
- FR-8: Base-percent bonuses on `TypeMaxHp`/`TypeMaxMp` are computationally valid but
  no mapping produces them (client applies no base-percent to HP/MP).

### 4.4 Consumer migration

- FR-9: `kafka/consumer/buff/consumer.go` (`handleBuffApplied`) builds the bonus list
  via the new API; a single `MAPLE_WARRIOR` change yields four bonuses that share the
  buff's source identity so `RemoveBuffBonuses` on expiry removes all four (removal is
  keyed by `sourceId` — repo-verified, no change needed there).
- FR-10: `character/initializer.go` (`fetchBuffBonuses`) builds bonuses via the new
  API with the same `buff:<sourceId>` source string, so a character logging in with
  Maple Warrior active gets identical bonuses to one who received the buff live.
- FR-11: Live-apply and initializer paths must produce byte-identical bonus sets for
  the same buff state (asserted by test).

### 4.5 Testing

- FR-12: Unit tests for the new mapping: MAPLE_WARRIOR → exactly 4 base-percent
  entries; every legacy type unchanged; unknown → empty.
- FR-13: Computation tests with truncation edge cases (e.g. base 13 × 10% → +1;
  base 4 × 10% → +0; base 100 + 30 equip STR + 10% MW → 140, **not** 143).
- FR-14: Buff-lifecycle test: apply MAPLE_WARRIOR → all four stats raised; expire →
  all four restored.
- FR-15: Existing `stat/model_test.go` coverage of `MapBuffStatType` migrated to the
  new API (project Builder pattern for setup; no `*_testhelpers.go`).

## 5. API Surface

No REST or Kafka wire changes.

- Kafka: consumes the existing buff status events; `StatChange{type: "MAPLE_WARRIOR",
  amount: X}` payload unchanged.
- REST: `GET` effective-stats resources will report the corrected computed values;
  the bonus JSON representation gains one additive field (FR-6). No new endpoints,
  no removed fields, no error-case changes.
- Go (package `stat`): `MapBuffStatType` removed; one-to-many replacement added
  (exact signature is a design-phase decision, constrained by FR-1–FR-5).

## 6. Data Model

No database entities — effective-stats state is an in-memory, tenant-scoped registry.

- `stat.Bonus` gains a base-percent dimension with JSON round-trip (FR-6).
- No migration: bonuses are recomputed from live events and login-time fetches;
  characters with Maple Warrior active during deploy self-heal on next buff event or
  login/initialization (confirmed acceptable — no backfill required).

## 7. Service Impact

- **atlas-effective-stats** — only service touched.
  - `stat/model.go` — mapping API replacement, `Bonus` extension.
  - `character/model.go` — base-percent application in `ComputeEffectiveStats`.
  - `kafka/consumer/buff/consumer.go` — consumer migration.
  - `character/initializer.go` — initializer migration.
  - `stat/model_test.go` + new tests.
- atlas-data, atlas-buffs: no changes (emitters verified correct).

## 8. Non-Functional Requirements

- **Multi-tenancy**: unchanged — registry and processors are already tenant-scoped;
  no new config, no tenant wire values (rate arrives as skill-data X per version, so
  DOM-25 is not implicated).
- **Version independence**: mechanism verified identical at v83 and v95 (§4.1); the
  per-version rate difference lives entirely in WZ skill data already served by
  atlas-data.
- **Performance**: O(1) extra work per bonus during recompute; no new allocations on
  hot paths beyond the 3 extra bonus entries per MW buff.
- **Observability**: keep existing debug logging for unknown buff types; no new
  metrics required.

## 9. Open Questions

None. (Multiplier basis was the open question from scoping; resolved by IDA
verification — base-stat-only, truncating, all four primaries, HP/MP excluded.)

## 10. Acceptance Criteria

- [ ] `MAPLE_WARRIOR` buff change maps to exactly four base-percent bonuses
      (STR/DEX/INT/LUK); no other mapping changed; unknown types map to nothing.
- [ ] Effective stat for each primary stat under Maple Warrior equals
      `base + equip + floor(base × X / 100)` — equipment excluded from the basis,
      truncation per stat (FR-13 cases pass).
- [ ] Buff expiry removes all four bonuses; live-apply and login-initializer paths
      produce identical bonus sets.
- [ ] Old `MapBuffStatType` no longer exists; both call sites migrated.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
      atlas-effective-stats; `docker buildx bake atlas-effective-stats` succeeds;
      `tools/redis-key-guard.sh` clean.
- [ ] Code review (backend-guidelines-reviewer + plan-adherence-reviewer) run before
      PR.
