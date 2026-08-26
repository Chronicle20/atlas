# Party EXP Sharing — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

When a monster dies, `atlas-monster-death` distributes its EXP to the characters who damaged it.
Today that distribution has no concept of a party. `produceDistribution`
(`services/atlas-monster-death/atlas.com/monster/monster/processor.go:161-163`) declares
`partyDistribution := make(map[uint32]map[uint32]uint32)` directly above a literal `// TODO parties`
and never writes to it; every damaging character is placed in `soloDistribution` instead
(`:166-171`). `DistributeExperience` (`:126-144`) consequently iterates only `d.Solo()` and calls
`distributeCharacterExperience(f, c.Id(), c.Level(), exp, 0.0, c.Level(), true, whiteExperienceGain, false)`
(`:140`) — with `partyBonusMod = 0.0`, `totalPartyLevel = own level`, `hightestPartyDamage = true`,
and `hasPartySharers = false`. Substituted into the split formula at `:231-241`, that degenerates to
`(0.8 × level / level) + 0.2 = 1.0`: the character receives exactly their own damage share, and the
`party` package that already lives in the same service (`monster/party/`) is used only for drop
ownership (`processor.go:60-64`), never for EXP.

The user-visible consequence is that party play is strictly worse than solo play. A party member who
does not personally damage a monster receives nothing. A party member who does damage it receives
only their own damage share, with no pooling across the party and no party bonus EXP. The two
mechanics that make MapleStory parties worth forming — level-weighted pooling of the party's
combined damage, and a `0.05 × memberCount` bonus on top — are both absent.

This feature implements the reference distribution from
`<cosmic>/src/main/java/server/life/Monster.java:547-600` (`distributePartyExperience`) and
`:604-680` (`distributeExperience`), which the existing Atlas code is already a partial
transliteration of — the stddev white/yellow threshold (`processor.go:196-229`), the split formula
(`:231-241`), and the shape of `DamageDistributionModel` (`monster/model.go:15-37`) are all ported
already and stay. Two correctness defects that sit directly underneath the feature are fixed in the
same change, because party pooling computed on wrong damage numbers is not worth shipping: damage
entries arrive un-aggregated (one per hit) and are silently overwritten, and total damage is
approximated by the monster's template max HP rather than damage actually dealt.

## 2. Goals

Primary goals:

- Party members co-located with a monster kill receive a level-weighted share of the party's
  **pooled** damage EXP, including members who dealt no damage themselves.
- Parties receive bonus EXP scaling with party size (`0.05 × memberCount`), delivered as the
  `PARTY` experience-distribution type that `atlas-character` already understands.
- The party's highest damager receives the MVP portion of the split; other members do not.
- A character's damage across all their hits on a monster is credited in full, not just their last
  hit.
- Total damage reflects damage actually dealt (which absorbs monster healing), not template max HP.
- Party members whose level is far outside the mob's / the contributing members' level range are
  excluded from the split and told why.
- A character with no party, or whose party lookup fails, receives EXP exactly as they do today.

Non-goals:

- Party **meso** splitting — owned by task-248-party-meso-split, which lists party EXP sharing as
  its own non-goal. The two tasks are complementary and touch disjoint services.
- Party **item/equipment** loot rules (ownership, free-for-all drops, quest-item routing).
- Changing how `atlas-rates` computes Holy Symbol. Atlas currently folds Holy Symbol into a flat
  `expRate` (`services/atlas-rates/atlas.com/rates/kafka/consumer/buff/consumer.go:56-80` →
  `buff.CalculateMultiplier`, `1 + amount/100`), which is equivalent to Cosmic's
  `USE_FULL_HOLY_SYMBOL: true`. Cosmic's alternate path divides by 500 when the attacker has no
  party sharers (`Monster.java:684-694`); Atlas deliberately does not implement that variant, and
  `atlas-rates` is untouched by this task.
- The `EXP_INCREASE` buff bonus (`Monster.java:723-726`), `increaseEquipExp`, and family reputation
  — none of these exist in Atlas and none are introduced here.
- Monster Book, quest mob counts, or event-instance kill hooks.
- Any change to the `AWARD_EXPERIENCE` command contract
  (`services/atlas-monster-death/atlas.com/monster/character/kafka.go:28-45`) — the `PARTY`
  distribution type is already defined at `:20` and already carried by the existing
  `AwardExperience(ch, characterId, white, amount, party)` signature.
- Showing the party-EXP breakdown in `atlas-ui`.

## 3. User Stories

- As a party member fighting alongside my partner, I want the EXP from monsters we kill together to
  be pooled and split by level, so that partying is at least as rewarding as soloing.
- As a low-damage support character (Priest, Bishop) in a party, I want to earn EXP from monsters I
  did not personally damage, so that a support role is viable.
- As a member of a full party, I want the party bonus EXP that scales with party size, so that
  larger parties are worth forming.
- As the highest damager in my party, I want the MVP portion of the split, so that carrying the
  party is recognized.
- As a solo player, I want my EXP to be unchanged by this feature, so that nothing regresses.
- As a player who lands many hits on a tough monster, I want every hit counted toward my damage
  share, so that my EXP reflects my actual contribution.
- As a level 30 character standing in a level 120 party's map, I want to be told why I earned no
  EXP, rather than silently receiving nothing.

## 4. Functional Requirements

### 4.1 Damage entry aggregation

**FR-1.1** — `atlas-monsters` MUST aggregate damage entries by `characterId`. `ModelBuilder.AddDamageEntry`
(`services/atlas-monsters/atlas.com/monsters/monster/builder.go:139-146`) currently appends a new
`entry` on every call, so `Model.damageEntries` holds one element per damaging hit. It MUST instead
find an existing entry for `characterId` and add to its `Damage`, appending a new entry only when
the character has none.

**FR-1.2** — Aggregation MUST preserve first-contact ordering: a character's entry keeps the
position at which that character first damaged the monster. `Model.DamageLeader()`
(`model.go:228-238`) selects the first entry with the strictly-greatest damage, so ordering is
observable.

**FR-1.3** — After FR-1.1, `Model.DamageSummary()`'s existing doc comment (`model.go:171-173`,
"Entries are now pre-aggregated by characterId at write time") becomes true. It is false today; the
comment MUST NOT be relied upon until FR-1.1 lands.

**FR-1.4** — `atlas-monster-death` MUST NOT assume the upstream event is aggregated. When building
its distribution it MUST accumulate (`+=`), not assign, per character. `processor.go:168` currently
does `soloDistribution[de.characterId] = de.damage`, which keeps only the last entry seen. This
defends the boundary independently of FR-1.1 and MUST be implemented even though FR-1.1 makes it
redundant in the normal case.

**FR-1.5** — Both the `KILLED` and `DAMAGED` status events carry `DamageEntries`
(`services/atlas-monsters/atlas.com/monsters/monster/kafka.go:117-138`), built from
`DamageSummary()` at `producer.go:67-75` and `:151-157`. FR-1.1 changes both. Any other consumer of
`damageEntries` MUST be identified during design and confirmed to be correct — or made correct —
under aggregated entries.

### 4.2 Participant bucketing

**FR-2.1** — For each damage entry whose character is present in the field
(`_map.CharacterIdsInFieldModelProvider`, `processor.go:152-159`), the distributor MUST resolve that
character's party via `party.NewProcessor(...).GetByMemberId(characterId)`.

**FR-2.2** — A character with a resolved party MUST be placed in `partyDistribution[partyId]` keyed
by `characterId`; a character with no party MUST be placed in `soloDistribution`.

**FR-2.3** — A party lookup that returns an error MUST be treated as "no party": the character falls
through to `soloDistribution` and the error is logged at warn level. A party service outage MUST
degrade to today's solo behavior, never to zero EXP.

**FR-2.4** — Party lookups MUST be issued at most once per distinct character in the damage-entry
set. A monster damaged by four characters issues at most four lookups regardless of hit count.

**FR-2.5** — `totalEntries` MUST count **participant entries**, not damage entries: one per solo
character plus one per distinct participating party. Characters that damaged the monster but are no
longer in the field MUST each still count as one entry (Cosmic's "independent party",
`Monster.java:637-638`). The current `totalEntries += 1` per damage entry (`processor.go:170`) is
wrong on both counts once entries are per-hit, and stays wrong for parties after aggregation.

**FR-2.6** — `totalEntries` is the divisor of the stddev threshold
(`calculateExperienceStandardDeviationThreshold`, `processor.go:207-221`), so FR-2.5 changes
white/yellow EXP-number selection for multi-hit kills. This is a deliberate correction, not a
regression.

### 4.3 Total damage and EXP-per-damage

**FR-3.1** — `totalDamage` MUST be the sum of all aggregated damage entries in the `KILLED` event,
replacing `totalDamage := mi.Hp()` (`processor.go:174`) and resolving the `// TODO account for
healing` at `:173`.

**FR-3.2** — Because `Model.Damage` clamps each hit to the monster's remaining HP
(`services/atlas-monsters/atlas.com/monsters/monster/model.go:215-222`), the sum of recorded damage
equals HP actually removed. Damage that a heal forced players to re-deal is therefore counted, which
is the behavior Cosmic gets from `maxHpPlusHeal` (`Monster.java:646`). No new field on the `KILLED`
event is required.

**FR-3.3** — If `totalDamage` is zero (an empty or all-zero damage-entry set), the distributor MUST
log at warn level and return without awarding EXP. It MUST NOT divide by zero.

**FR-3.4** — `experiencePerDamage` remains `float64(mi.Experience()) / float64(totalDamage)`.

### 4.4 Personal ratio and white/yellow EXP

**FR-4.1** — `personalRatio[characterId] = characterDamage / totalDamage` MUST be computed for every
participant, solo and party alike, from that character's **own** damage — unchanged in intent from
`processor.go:180-194`, but now fed by aggregated damage.

**FR-4.2** — A party contributes **one** entry to `entryExperienceRatio`, equal to the sum of its
members' personal ratios (`processor.go:186-194`, already written this way).

**FR-4.3** — A party member with zero damage has no `personalRatio` entry and therefore receives
yellow EXP (`isWhiteExperienceGain` returns `false` when the key is absent, `processor.go:223-229`).
This matches Cosmic (`Monster.java:508-515`) and is intended.

**FR-4.4** — The stddev threshold calculation (`processor.go:207-221`) is unchanged.

### 4.5 Party distribution

**FR-5.1** — For each party in `partyDistribution`, the distributor MUST compute
`partyDamage = Σ member damage` and identify the **participation MVP** as the member with the
greatest damage. On an exact tie the first such member in iteration order wins; because Go map
iteration order is non-deterministic, the implementation MUST iterate a deterministically-ordered
sequence (ascending `characterId`) so the MVP is reproducible.

**FR-5.2** — `participationExp = partyDamage × experiencePerDamage`. This is the **pooled** party
figure, and it is the `exp` argument passed to every member's split — not a per-member figure.

**FR-5.3** — The set of EXP recipients (`expMembers`) MUST be the party's **co-located members**,
not the set of damagers. A member who dealt no damage but is co-located receives a share; a member
who dealt damage but has since left the field does not.

**FR-5.4** — Co-location means the member is in the same `field` as the kill: same `worldId`, same
`channelId`, same `mapId`, and same `instance`. `MemberRestModel` already exposes `WorldId`,
`ChannelId`, `MapId`, `Instance`, and `Online`
(`services/atlas-parties/atlas.com/parties/party/rest.go:90-100`). This matches the eligibility rule
established by task-248-party-meso-split.

**FR-5.5** — A member with `Online == false` MUST be excluded regardless of stale location fields.

**FR-5.6** — `totalPartyLevel = Σ level of eligible `expMembers``. It MUST NOT include excluded
members. `totalPartyLevel` is a divisor and MUST be verified non-zero before use.

**FR-5.7** — `hasPartySharers = len(expMembers) > 1`.

**FR-5.8** — `partyBonusMod = 0.05 × len(expMembers)` when `hasPartySharers`, else `0.0`.

**FR-5.9** — For each member of `expMembers`, the distributor MUST call the split with
`experience = participationExp`, `level = member level`, `totalPartyLevel`, `partyBonusMod`,
`hightestPartyDamage = (member == participation MVP)`, `whiteExperienceGain` per FR-4, and
`hasPartySharers`.

**FR-5.10** — A party whose eligible member set is empty MUST be skipped without awarding EXP and
without error.

**FR-5.11** — A party that resolves to exactly one eligible member still goes through the party
path. With `totalPartyLevel = own level`, `hasPartySharers = false`, `partyBonusMod = 0.0`, and that
member necessarily being the MVP, the result is numerically identical to the solo path.

### 4.6 Level-range gating (leech interval)

**FR-6.1** — A party member MUST be excluded from `expMembers` when their level falls outside the
union of these intervals, per Cosmic `Monster.java:549-566`:
  - `[mobLevel − LEVEL_INTERVAL, mobLevel + LEVEL_INTERVAL]`, and
  - for each **contributing** (damage-dealing) party member, `[contributorLevel − LEACH_INTERVAL, contributorLevel + LEACH_INTERVAL]`.

**FR-6.2** — The intervals are a **union of possibly-overlapping ranges**, not a single min/max
band. Cosmic uses `IntervalBuilder` (`<cosmic>/src/main/java/tools/IntervalBuilder.java`), which
merges overlapping intervals and answers `inInterval(level)`. A party of a level-30 and a level-120
contributor against a level-125 mob admits a level-32 member (inside the level-30 contributor's
band) and rejects a level-70 member (inside neither band). An implementation that takes
`[min−5, max+5]` is incorrect and MUST NOT be used.

**FR-6.3** — The gate applies **only to the party path**. Solo distribution has no level gate, and
FR-6.1 MUST NOT be applied to `soloDistribution` — a solo character always earns from what they
damaged.

**FR-6.4** — `LEVEL_INTERVAL` and `LEACH_INTERVAL` default to `5` and `5`, matching
`<cosmic>/config.yaml:293-294`.

**FR-6.5** — The gate MUST be togglable, defaulting to **enabled** (Cosmic's
`USE_ENFORCE_MOB_LEVEL_RANGE: true`, `<cosmic>/config.yaml:243`). Where the toggle and the two
interval constants live — Go constants, service env vars, or tenant configuration — is deferred to
design (Open Question OQ-1).

**FR-6.6** — Applying the gate requires the monster's **level**, which the local
`information.Model` (`monster/information/model.go:3-14`) does not expose even though the REST
payload already carries it (`monster/information/rest.go:11`). The local model MUST be extended.

**FR-6.7** — A member excluded by FR-6.1 MUST be notified: exactly one `SHOW_HINT` command per
excluded member per kill, published to `COMMAND_TOPIC_SYSTEM_MESSAGE`
(`services/atlas-channel/atlas.com/channel/kafka/message/system_message/kafka.go:11-33`, `:62-67`).

**FR-6.8** — The hint text follows Cosmic's wording (`Character.java:9246`), rendered with the
monster's name and level:
> `You have gained #rno experience#k from defeating #e#b<name>#k#n (lv. #b<level>#k)! Take note you must have around the same level as the mob to start earning EXP from it.`

`Width` and `Height` MUST be `0` (auto-calculated). The monster's name comes from the same REST
payload as its level (`monster/information/rest.go:7`) and requires the same model extension as
FR-6.6.

**FR-6.9** — Cosmic rate-limits the notice to once per minute per character
(`Character.java:9241-9248`, `nextWarningTime`). `atlas-monster-death` is stateless and horizontally
scalable, so per-character throttle state has no natural home there. This PRD does **not** require
throttling; the notice is emitted per excluded member per kill. Whether that is acceptable spam, and
where a throttle would live if not, is Open Question OQ-2.

**FR-6.10** — A `SHOW_HINT` publish failure MUST be logged and MUST NOT abort EXP distribution for
the remaining members.

### 4.7 Solo distribution

**FR-7.1** — Solo distribution keeps its current parameters: `partyBonusMod = 0.0`,
`totalPartyLevel = own level`, `hightestPartyDamage = true`, `hasPartySharers = false`
(`processor.go:140`, matching `Monster.java:671`).

**FR-7.2** — Solo `exp = ownDamage × experiencePerDamage`.

**FR-7.3** — Solo EXP changes numerically only insofar as FR-1.4 (full damage credited instead of
last hit) and FR-3.1 (total damage instead of template max HP) change it. Both are corrections.

### 4.8 Rates and award

**FR-8.1** — The per-character `expRate` from `atlas-rates`
(`monster/rates/processor.go:58-69`) MUST be applied to the character's computed personal EXP, as
today (`processor.go:135-137`).

**FR-8.2** — Bonus (party) EXP MUST also be multiplied by the character's `expRate`, matching
Cosmic (`Monster.java:735-741`, which applies `getStatusExpMultiplier` and `getExpRate` to
`partyExp` as well as `personalExp`). Today's `distributeCharacterExperience` computes
`bonusExperience = partyBonusMod × characterExperience` from an already-rate-multiplied
`characterExperience` (`processor.go:238`), so this holds as long as the rate is applied before the
split — which is the existing ordering and MUST be preserved.

**FR-8.3** — Cosmic additionally multiplies party EXP by a server-wide `PARTY_BONUS_EXP_RATE`
(`Monster.java:747`, default `1.0` at `<cosmic>/config.yaml:297`). At the default this is a no-op;
this task MUST NOT introduce a new rate knob for it. If one is wanted later it belongs in
`atlas-rates`, not here.

**FR-8.4** — Rate lookups MUST be issued at most once per recipient character per kill.

**FR-8.5** — The award MUST continue to go through
`character.NewProcessor(...).AwardExperience(ch, characterId, white, personal, party)`
(`monster/character/processor.go:32-34`). A recipient with `party > 0` produces the `PARTY`
distribution alongside the `WHITE`/`YELLOW` one; this is existing behavior of the command body and
requires no contract change.

**FR-8.6** — EXP values are cast to `uint32` at the award boundary (`processor.go:240`). Negative
intermediate values are impossible given non-negative damage and rates, but the cast MUST be
guarded against a `NaN`/`Inf` produced by a zero divisor (see FR-3.3, FR-5.6).

### 4.9 Ordering and failure isolation

**FR-9.1** — Distribution MUST be deterministic given the same `KILLED` event: parties processed in
ascending `partyId`, members in ascending `characterId`, solo characters in ascending
`characterId`. Go map iteration order MUST NOT leak into MVP selection (FR-5.1) or award ordering.

**FR-9.2** — A failure resolving one character, party, or rate MUST NOT abort distribution for the
others. Each recipient's award is independent.

**FR-9.3** — Distribution MUST remain idempotent-per-event in the sense it is today: exactly one
`AWARD_EXPERIENCE` command per eligible recipient per `KILLED` event.

## 5. API Surface

No new HTTP endpoints. No change to any service's served REST surface.

### 5.1 Consumed REST (client-side model extensions only)

**`GET {PARTIES}/parties?filter[members.id]={characterId}`** — already called by
`monster/party/processor.go:30-33`. `atlas-monster-death`'s local `party.RestModel`
(`monster/party/rest.go:9-11`) is an id-only stub whose `GetReferencedIDs` returns nil
(`:39-42`) and whose `SetToManyReferenceIDs` is a no-op (`:49-51`). It MUST be extended to
deserialize the `members` relationship into a member struct carrying at minimum:

| Field | JSON | Type | Source |
|---|---|---|---|
| `Id` | (jsonapi id) | `uint32` | `atlas-parties` `MemberRestModel` |
| `Level` | `level` | `byte` | same |
| `WorldId` | `worldId` | `world.Id` | same |
| `ChannelId` | `channelId` | `channel.Id` | same |
| `MapId` | `mapId` | `_map.Id` | same |
| `Instance` | `instance` | `uuid.UUID` | same |
| `Online` | `online` | `bool` | same |

`atlas-parties` already serves every one of these
(`services/atlas-parties/atlas.com/parties/party/rest.go:90-100`) and needs **no change**.

The corresponding `party.Model` (`monster/party/model.go:3-9`) gains a members slice and accessors,
following the immutable-model convention (unexported fields, value receivers).

**`GET {DATA}/monsters/{monsterId}`** — already called via
`information.GetById` (`monster/monster/information/`). The local `information.Model`
(`model.go:3-14`) exposes only `Hp()` and `Experience()`; it MUST additionally expose `Level()` and
`Name()`, both of which the existing `RestModel` already deserializes (`rest.go:7`, `:11`). No
change to the data service.

**`GET {RATES}/worlds/{w}/channels/{c}/characters/{id}/rates`** — unchanged.

**`GET {CHARACTERS}/characters/{id}`** — unchanged.

### 5.2 Produced Kafka

**`COMMAND_TOPIC_CHARACTER` / `AWARD_EXPERIENCE`** — unchanged contract
(`monster/character/kafka.go:28-45`). The `PARTY` distribution type (`:20`) and the existing
`AwardExperience(..., party uint32)` parameter are already in place; this task begins populating
`party` with a non-zero value for party recipients.

**`COMMAND_TOPIC_SYSTEM_MESSAGE` / `SHOW_HINT`** — **newly produced by `atlas-monster-death`**, per
FR-6.7. Existing contract, consumed by `atlas-channel`
(`kafka/consumer/system_message/consumer.go:238-250`):

```json
{
  "transactionId": "<uuid>",
  "worldId": 0,
  "channelId": 1,
  "characterId": 12345,
  "type": "SHOW_HINT",
  "body": { "hint": "You have gained #rno experience#k from ...", "width": 0, "height": 0 }
}
```

`atlas-monster-death` gains a `system_message` producer package mirroring the existing
`character` producer, plus the `COMMAND_TOPIC_SYSTEM_MESSAGE` env var in its deployment
configuration.

### 5.3 Consumed Kafka

**`EVENT_TOPIC_MONSTER_STATUS` / `KILLED`** — contract unchanged
(`monster/kafka/consumer/monster/kafka.go:28-38`). The **semantics** of `damageEntries` change per
FR-1.1: one entry per character rather than one per hit. No field is added or removed.

## 6. Data Model

No database entities, no migrations, no persisted state. `atlas-monster-death` is stateless.

### 6.1 In-service model changes (`atlas-monster-death`)

`monster.DamageDistributionModel` (`monster/model.go:15-37`) already carries a
`party map[uint32]map[uint32]uint32` field with no accessor. It gains:

- a `Party()` accessor for the now-populated party bucket,
- a `TotalDamage()` accessor (FR-3.1 makes total damage a computed value the distributor needs),
- an `ExperiencePerDamage()`, `PersonalRatio()`, `StandardDeviationRatio()` — all already present.

`party.Model` gains `Members() []MemberModel`; a new `MemberModel` carries the FR-5.4 fields.

`information.Model` gains `Level() uint32` and `Name() string`.

All follow the repo's immutable-model convention: unexported fields, value receivers, construction
through a builder or `Extract`. Per repo convention, `world.Id`, `channel.Id`, `_map.Id`, and
`field.Model` come from `libs/atlas-constants/` rather than being redeclared.

### 6.2 In-service model changes (`atlas-monsters`)

`ModelBuilder.damageEntries` keeps its `[]entry` type; only `AddDamageEntry`'s write semantics
change (FR-1.1). No field is added.

### 6.3 Multi-tenancy

Every REST and Kafka call already flows tenant through `context.Context` via the standard
`atlas-rest`/`atlas-kafka` header decorators
(`monster/kafka/consumer/monster/consumer.go:23` registers `TenantHeaderParser` and
`EnvHeaderParser`). The new `SHOW_HINT` producer MUST use the same `producer.ProviderImpl(l)(ctx)`
path as the existing `AwardExperience` producer (`monster/character/processor.go:33`) so tenant and
span headers propagate identically. No tenant-scoped storage is introduced.

## 7. Service Impact

### `atlas-monster-death` — primary

- `monster/monster/processor.go` — `produceDistribution` gains party bucketing (FR-2), accumulating
  damage (FR-1.4), summed total damage (FR-3.1), and corrected `totalEntries` (FR-2.5).
  `DistributeExperience` gains the party branch (FR-5), the level gate (FR-6), and deterministic
  ordering (FR-9.1).
- `monster/monster/model.go` — accessors per §6.1.
- `monster/party/{model,rest}.go` — members relationship (§5.1).
- `monster/monster/information/{model,rest,builder}.go` — `Level()`, `Name()`.
- new `monster/system_message/` producer package — `SHOW_HINT` (FR-6.7).
- `main.go` / deployment env — `COMMAND_TOPIC_SYSTEM_MESSAGE`.
- Tests: `monster/monster/processor_test.go` and `characterization_test.go` exist and will need
  extension; `monster/party/mock/` and `monster/character/mock/` exist and are the seam for party
  and award assertions.

### `atlas-monsters` — secondary

- `monster/builder.go` — `AddDamageEntry` aggregates (FR-1.1).
- `monster/model.go` — the `DamageSummary()` comment becomes accurate (FR-1.3).
- Consumers of `DAMAGED`/`KILLED` `damageEntries` audited per FR-1.5.

### `atlas-parties` — read-only

No change. Already serves every field FR-5.4 needs.

### `atlas-channel` — read-only

No change. Already consumes `SHOW_HINT`.

### `atlas-character` — read-only

No change. Already handles the `PARTY` distribution type in `AWARD_EXPERIENCE`.

### `atlas-rates` — untouched

Explicit non-goal (§2).

## 8. Non-Functional Requirements

**NFR-1 (fan-out).** A kill by an N-character party must issue O(distinct characters) party lookups,
not O(damage entries) — FR-2.4. Party lookups for members of the same party MUST be de-duplicated by
`partyId`.

**NFR-2 (latency).** Distribution runs in a `routine.Go` goroutine off the `KILLED` handler
(`kafka/consumer/monster/consumer.go:63-89`) and does not block drop creation, which runs
concurrently in a sibling goroutine. Added REST round-trips MUST NOT serialize where they can be
issued concurrently; the service already uses `model.ParallelMap()` (`:67`) for this shape.

**NFR-3 (degradation).** Every new external dependency has a defined failure mode: party lookup
failure → treat as solo (FR-2.3); rates failure → defaults (existing, `rates/processor.go:62-65`);
`SHOW_HINT` failure → log and continue (FR-6.10). No new dependency may turn a kill into zero EXP.

**NFR-4 (observability).** Log at debug: per-party pooled damage, eligible member count,
`totalPartyLevel`, `partyBonusMod`, MVP id. Log at warn: party lookup failure, zero total damage,
zero `totalPartyLevel`, level-gate exclusions. Every log line MUST carry `monsterId` and the field
identifiers, matching the existing pattern at `consumer.go:53-59`.

**NFR-5 (determinism).** FR-9.1. Tests MUST assert determinism, not merely observe it once.

**NFR-6 (multi-tenancy).** §6.3.

**NFR-7 (security).** No new authenticated surface, no user-supplied input reaching the hint text
beyond a monster name sourced from the data service.

**NFR-8 (backwards compatibility).** The `KILLED` event contract is unchanged, so an
`atlas-monsters` carrying FR-1.1 and an `atlas-monster-death` without it interoperate (the latter
simply sees correct entries and overwrites them). Deploy order is unconstrained.

## 9. Open Questions

- **OQ-1** — Where do `USE_ENFORCE_MOB_LEVEL_RANGE`, `LEVEL_INTERVAL`, and `LEACH_INTERVAL` live?
  Go constants in `atlas-monster-death` are simplest and match how the service already treats
  `expSplitCommonMod = 0.8` (`processor.go:232`) and the `+0.2` MVP mod (`:235`) as inline literals.
  Service env vars allow per-deployment tuning. Tenant configuration allows per-tenant tuning but no
  precedent for EXP tuning exists in this service. **Recommendation: Go constants alongside the
  existing split mods, extracted into named constants so the existing magic numbers get named too.**
  Design phase decides.

- **OQ-2** — Should the level-gate `SHOW_HINT` be throttled (FR-6.9)? Cosmic throttles to once per
  minute per character using in-process state. `atlas-monster-death` is stateless and may run
  multiple replicas, so a faithful throttle needs either shared state or relocation of the notice.
  Un-throttled, a level-30 character parked in a level-120 grinding party receives a hint per kill,
  which is likely unacceptable. Options: accept it, add in-process per-replica throttling
  (imperfect but cheap and bounded), or move the notice elsewhere. **Recommendation: in-process
  per-replica throttle keyed by `characterId`, 1-minute window** — a replica-local approximation
  that bounds the spam without introducing shared state. Design phase decides.

- **OQ-3** — FR-1.5: which other consumers read `damageEntries` off `DAMAGED`/`KILLED`? The field's
  own doc comment (`atlas-monsters/.../kafka.go:123-127`) warns consumers not to read the last
  element as "damage this event," implying at least one consumer once did. A full consumer sweep is
  a design-phase deliverable, not a spec-phase assumption.

- **OQ-4** — Does any existing test in `atlas-monsters` assert the per-hit (append) behavior of
  `AddDamageEntry`? `builder_test.go` and `characterization_test.go` exist in `atlas-monster-death`;
  the `atlas-monsters` side must be checked. If a characterization test pins the current behavior,
  it must be updated deliberately with a note, not silently.

- **OQ-5** — Cosmic computes `expMembers` from `getPartyMembersOnSameMap()` called on an arbitrary
  contributing member (`Monster.java:571`, `:581`). Atlas resolves the party from
  `atlas-parties`, whose `MemberRestModel.MapId`/`ChannelId`/`Instance` are updated by that
  service's own location tracking (`services/atlas-parties/atlas.com/parties/location/`). How stale
  can that location be at kill time, and can a member who just changed maps be mis-credited or
  mis-excluded? Design phase should confirm the freshness guarantee.

## 10. Acceptance Criteria

### Damage aggregation
- [ ] `ModelBuilder.AddDamageEntry` called twice for the same `characterId` produces **one** entry
      with the summed damage; called for two different characters produces two entries.
- [ ] Entry order reflects first contact, and `Model.DamageLeader()` returns the highest-total
      damager (not the highest single hit).
- [ ] A test asserts a monster damaged 3×100 by character A and 1×250 by character B emits a
      `KILLED` event with exactly two entries: `{A: 300}`, `{B: 250}`, and `DamageLeader() == A`.
- [ ] `atlas-monster-death` accumulates rather than assigns, proven by feeding it a deliberately
      un-aggregated entry list and asserting the summed result (FR-1.4).

### Total damage
- [ ] `totalDamage` equals the sum of aggregated entries, not `mi.Hp()`.
- [ ] A monster with 1000 max HP healed for 500 mid-fight and killed yields `totalDamage == 1500`,
      and each participant's ratio is computed against 1500.
- [ ] A `KILLED` event with an empty damage-entry set logs a warning and awards no EXP — no panic,
      no divide-by-zero, no `NaN` reaching the award.

### Solo path (no regression)
- [ ] A partyless character who solo-kills a monster receives EXP equal to
      `monsterExp × expRate` (their damage share being 1.0), with `party == 0` in the award command.
- [ ] A party lookup that returns an error routes the character to the solo path and logs a warning;
      EXP is awarded.
- [ ] The existing `processor_test.go` and `characterization_test.go` suites pass, with any changed
      expectation accompanied by a comment naming the FR that changed it.

### Party path
- [ ] Two co-located party members, one dealing 100% of the damage and one dealing none, both
      receive EXP; the non-damager's award is non-zero.
- [ ] Both members' EXP derives from `participationExp = partyDamage × expPerDmg` (the pooled
      figure), weighted `0.8 × ownLevel / totalPartyLevel`, with `+0.2` for the damager (MVP) only.
- [ ] `hasPartySharers` is `true` and `partyBonusMod == 0.10` for a 2-member eligible party;
      `0.20` for 4 members; `0.0` for 1.
- [ ] Each party recipient's award command carries a non-zero `party` amount equal to
      `partyBonusMod × personalExp`, producing a `PARTY` distribution entry.
- [ ] A party member on a different channel, a different map, a different instance, or `Online ==
      false` is excluded and receives nothing.
- [ ] A party member who dealt damage but left the field before the kill receives nothing, yet still
      contributes their damage to `partyDamage` and to `totalEntries`.
- [ ] A one-eligible-member party produces the same EXP as the solo path for that character
      (FR-5.11), asserted numerically.
- [ ] MVP selection is deterministic on a damage tie, asserted across repeated runs.

### Level gate
- [ ] With the gate enabled, a party member 40 levels below the mob and 40 below every contributor
      is excluded from `expMembers`, is **not** counted in `totalPartyLevel`, and does **not**
      increase `partyBonusMod`.
- [ ] Interval **union** semantics are asserted, not min/max: contributors at levels 30 and 120
      against a level-125 mob admit a level-32 member and reject a level-70 member.
- [ ] Each excluded member receives exactly one `SHOW_HINT` command per kill (subject to the OQ-2
      throttle decision), addressed to that `characterId`, with `width == 0` and `height == 0`, and
      the monster's name and level interpolated into the FR-6.8 text.
- [ ] A `SHOW_HINT` publish failure is logged and the remaining members are still awarded.
- [ ] With the gate disabled, no member is excluded and no hint is emitted.
- [ ] The gate is never applied to solo distribution (FR-6.3).

### White / yellow EXP
- [ ] `totalEntries` counts one per solo character, one per participating party, and one per
      out-of-field damager — asserted directly, since it is otherwise only observable through the
      stddev threshold.
- [ ] A zero-damage party member receives **yellow** EXP.
- [ ] A dominant damager receives **white** EXP; a marginal one receives yellow.

### Cross-cutting
- [ ] Distribution is deterministic: the same `KILLED` event produces byte-identical award commands
      in the same order across runs.
- [ ] At most one party lookup per distinct party and one rate lookup per recipient per kill,
      asserted via mocks.
- [ ] Tenant and span headers propagate on the new `SHOW_HINT` producer identically to the existing
      `AWARD_EXPERIENCE` producer.
- [ ] Test setup uses the project Builder pattern; no `*_testhelpers.go` file is introduced.
- [ ] No literal `// TODO parties` or `// TODO account for healing` remains in
      `monster/monster/processor.go`.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] `backend-guidelines-reviewer` passes on both `atlas-monster-death` and `atlas-monsters`.
