# Domain

## Monster

### Responsibility

Handles monster death processing including drop creation and experience distribution.

### Core Models

#### DamageEntryModel

Represents damage dealt by a character to a monster, as received from the KILLED event.

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character who dealt damage |
| damage | uint32 | Amount of damage dealt |

#### DamageInput

One aggregated damage entry, exactly one per damaging character (in-field or not), produced by folding the KILLED event's damage entries.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character who dealt damage |
| Damage | uint32 | Total damage dealt by the character |

#### SoloInput

An in-field damager with no party.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character identifier |
| Level | byte | Character level |

#### MemberInput

One co-located party member (present in the field where the kill happened), whether or not they dealt damage.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character identifier |
| Level | byte | Character level |

#### PartyInput

One participating party and its co-located members.

| Field | Type | Description |
|-------|------|-------------|
| PartyId | uint32 | Party identifier |
| Members | []MemberInput | Co-located members |

#### ExperienceInput

Aggregate input to the experience planner.

| Field | Type | Description |
|-------|------|-------------|
| MonsterExperience | uint32 | Monster's base experience value |
| MonsterLevel | uint32 | Monster's level |
| Damages | []DamageInput | Every damager, including those who left the field |
| Solos | []SoloInput | In-field damagers with no party |
| Parties | []PartyInput | Participating parties |

#### Recipient

One character who will receive an experience award.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character identifier |
| Level | byte | Character level |
| PartyId | uint32 | Party identifier (0 for solo) |
| PooledExp | float64 | Participation experience pooled for the character (solo) or the party (shared) |
| TotalPartyLevel | uint32 | Sum of levels of party members sharing the pool (own level for solo) |
| PartyBonusMod | float64 | Party bonus modifier applied to the personal award |
| IsMvp | bool | Whether the character is the highest damage-dealing member (always true for solo) |
| White | bool | Whether the award is white experience |

#### Exclusion

A co-located party member the level gate kept out of the award.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character identifier |

#### ExperiencePlan

Result of planning an experience distribution.

| Field | Type | Description |
|-------|------|-------------|
| Recipients | []Recipient | Characters to award |
| Exclusions | []Exclusion | Characters excluded by the level gate |
| TotalDamage | uint32 | Sum of damage across all damagers |
| TotalEntries | int | Count of solos, parties, and out-of-field damagers |
| ExperiencePerDamage | float64 | MonsterExperience / TotalDamage |
| StandardDeviationRatio | float64 | Threshold for white experience gain |

#### ExperienceConfig

Tunable settings for experience distribution.

| Field | Type | Description |
|-------|------|-------------|
| EnforceMobLevelRange | bool | Whether the level gate is applied |
| LevelInterval | uint32 | Band width around the monster's level admitted by the gate |
| LeachInterval | uint32 | Band width around each contributor's level admitted by the gate |
| SplitCommonMod | float64 | Common share modifier applied to a party member's split |
| MvpMod | float64 | Additional share modifier applied to the party MVP |
| PartyBonusPerMember | float64 | Party bonus modifier applied per sharing member |

Defaults: EnforceMobLevelRange=true, LevelInterval=5, LeachInterval=5, SplitCommonMod=0.8, MvpMod=0.2, PartyBonusPerMember=0.05. EnforceMobLevelRange, LevelInterval, and LeachInterval are overridable via environment variables; the split modifiers are not.

### Invariants

- aggregateDamageEntries sums damage per character across all entries for a given KILLED event
- planDistribution returns an empty plan (nil Recipients and Exclusions) when total damage is zero
- The level gate (when EnforceMobLevelRange is set) is a union of level bands: one band around the monster's level, plus one band around each contributor's level; it is not a single min/max band
- partyDamage, participationExp, the contributor list, and the interval set are computed from the ungated contributor list before the level gate is applied; a gated-out member still widens the interval and still contributes to a pool they do not share in
- A recipient with TotalPartyLevel of 0 receives a personal and bonus award of 0
- computeAward guards non-finite values to 0 and clamps values at or above MaxUint32 to MaxUint32 before conversion to the award's uint32 wire type
- intervalSet clamps a band's lower bound at 0

### Processors

#### CreateDrops

Evaluates monster drop tables and creates drops for a killed monster.

- Retrieves drop information for the monster
- Filters quest-specific drops based on character's started quests
- Retrieves character rate multipliers
- Evaluates drop success based on chance and item drop rate (adjustedChance = chance * itemDropRate, success if rand < adjustedChance out of 999999)
- Retrieves the killer's party membership for drop ownership assignment
- Creates item or meso drops at calculated positions

#### DistributeExperience

Distributes experience to characters who damaged the monster.

- Aggregates the KILLED event's damage entries into one entry per character; returns without distributing if total damage is zero
- Concurrently retrieves monster information and the character IDs currently present in the field
- Resolves parties for in-field damagers; an out-of-field damager is never party-resolved. A party-service error, or a party of id 0, is treated as solo
- Plans the distribution via planDistribution
- Awards each planned recipient: retrieves the character's rate multipliers, computes the personal and party-bonus amounts via computeAward, and produces an experience award command. One recipient's failure does not abort the others
- After all awards are attempted, sends a level-gate hint to each excluded character, subject to the hint throttle; hints are sent last so a hint failure cannot affect an award

#### planDistribution

Pure planner that computes an ExperiencePlan from an ExperienceInput and an ExperienceConfig.

- Computes experience per damage as MonsterExperience / TotalDamage
- Computes each solo's and each party's personal damage ratio, and a standard-deviation threshold across all entries' ratios (calculateExperienceStandardDeviationThreshold)
- Every solo damager with a non-zero level is a recipient, treated as the MVP of their own single-member "party" and awarded the full experience-per-damage pool for their damage
- For each party: contributors are members who dealt damage; participation experience is contributor damage total multiplied by experience per damage; if EnforceMobLevelRange is set, members outside the level-gate interval union are excluded and recorded as Exclusions; the remaining members share the pool by level, with the highest-damage member marked MVP and, when more than one member shares the pool, a per-member party bonus modifier applied
- White experience gain (isWhiteExperienceGain) is set on a recipient when their personal damage ratio meets or exceeds the standard deviation threshold

#### computeAward

Converts a Recipient's pooled experience into a personal and a party-bonus award amount, applying the character's experience rate before the split and reporting whether either value had to be guarded to a representable uint32.

#### filterByQuestState

Filters drops based on character's quest state.

- Returns all drops unchanged if no quest-specific drops exist
- Fetches started quest IDs for the character from quest service
- Includes drops with questId == 0 (non-quest items)
- Includes drops with questId matching a started quest
- Excludes drops with questId not matching any started quest
- On quest service error, excludes all quest-specific drops

---

## Character

### Responsibility

Represents character information retrieved from external service and produces experience award commands.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| level | byte | Character level |

### Processors

#### AwardExperience

Produces a Kafka command to award experience to a character, given the personal amount and a party bonus amount. The command carries experience distributions indicating the type (WHITE or YELLOW based on white experience gain determination) and a PARTY distribution.

---

## Drop

### Responsibility

Represents drop information and handles drop creation logic including inline equipment statistics generation for equipment drops.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| itemId | uint32 | Item identifier (0 for meso drops) |
| minimumQuantity | uint32 | Minimum drop quantity |
| maximumQuantity | uint32 | Maximum drop quantity |
| questId | uint32 | Associated quest identifier |
| chance | uint32 | Drop chance (out of 999999) |

#### EquipmentData

Carries inline equipment statistics for equipment drops. Populated when the dropped item is an equipment (itemId / 1000000 == 1). Statistics are randomized from base values fetched from the data service.

| Field | Type | Description |
|-------|------|-------------|
| strength | uint16 | STR stat |
| dexterity | uint16 | DEX stat |
| intelligence | uint16 | INT stat |
| luck | uint16 | LUK stat |
| hp | uint16 | HP stat |
| mp | uint16 | MP stat |
| weaponAttack | uint16 | Weapon attack |
| magicAttack | uint16 | Magic attack |
| weaponDefense | uint16 | Weapon defense |
| magicDefense | uint16 | Magic defense |
| accuracy | uint16 | Accuracy |
| avoidability | uint16 | Avoidability |
| hands | uint16 | Hands |
| speed | uint16 | Speed |
| jump | uint16 | Jump |
| slots | uint16 | Upgrade slots (not randomized) |

### Invariants

- minimumQuantity cannot exceed maximumQuantity
- Default minimumQuantity and maximumQuantity are both 1

### Processors

#### Create

Creates a drop at a calculated position based on drop index. Drops are spread alternating left/right from the monster position using a spacing factor (25 for normal drops, 40 for drop type 3). Even indices offset right, odd indices offset left.

#### SpawnMeso

Spawns a meso drop with randomized quantity between minimum and maximum. Applies the meso rate multiplier to the base amount.

#### SpawnItem

Spawns an item drop with randomized quantity between minimum and maximum. For equipment items (itemId / 1000000 == 1), fetches base statistics from the data service and generates randomized equipment data. Each non-zero stat is varied within +/- 10% of the base value (capped at a per-stat maximum of 5 or 10). Slots are copied directly without randomization.

#### SpawnDrop

Calculates final drop position using the data service's drop position endpoint (called twice for refinement) and produces a spawn drop command via Kafka.

---

## Drop Position

### Responsibility

Represents a calculated drop position retrieved from external service.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| x | int16 | X coordinate |
| y | int16 | Y coordinate |

### Processors

#### GetInMap

Retrieves a valid drop position within a map from the data service. Accepts initial coordinates and fallback coordinates; returns the fallback on error.

---

## Monster Information

### Responsibility

Represents monster static data retrieved from external service.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| hp | uint32 | Monster hit points |
| experience | uint32 | Base experience value |
| level | uint32 | Monster level |
| name | string | Monster name |

---

## Equipment Statistics

### Responsibility

Represents base equipment statistics retrieved from the data service. Used to generate randomized equipment data for equipment drops.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| strength | uint16 | Base STR |
| dexterity | uint16 | Base DEX |
| intelligence | uint16 | Base INT |
| luck | uint16 | Base LUK |
| hp | uint16 | Base HP |
| mp | uint16 | Base MP |
| weaponAttack | uint16 | Base weapon attack |
| magicAttack | uint16 | Base magic attack |
| weaponDefense | uint16 | Base weapon defense |
| magicDefense | uint16 | Base magic defense |
| accuracy | uint16 | Base accuracy |
| avoidability | uint16 | Base avoidability |
| hands | uint16 | Base hands |
| speed | uint16 | Base speed |
| jump | uint16 | Base jump |
| slots | uint16 | Upgrade slots |

### Processors

#### GetById

Retrieves base equipment statistics by item ID from the data service.

---

## Party

### Responsibility

Represents party membership retrieved from external service for drop ownership assignment and for party experience distribution.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Party identifier |
| leaderId | uint32 | Party leader's character identifier |
| members | []MemberModel | Party members |

#### MemberModel

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| name | string | Character name |
| level | byte | Character level |
| jobId | job.Id | Character job identifier |
| field | field.Model | Field the character is currently in |
| online | bool | Whether the character is online |

### Processors

#### GetByMemberId

Retrieves the party a character belongs to by querying with the character's ID. Returns the first matching party.

---

## Quest

### Responsibility

Represents quest state information retrieved from external service for quest-aware drop filtering.

### Core Models

#### State

Quest state enumeration.

| Value | Name | Description |
|-------|------|-------------|
| 0 | StateNotStarted | Quest not started |
| 1 | StateStarted | Quest in progress |
| 2 | StateCompleted | Quest completed |

#### Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| questId | uint32 | Quest identifier |
| state | State | Quest state |

### Processors

#### GetStartedQuestIds

Retrieves a set of started quest IDs for a character. Returns a map of questId to bool for efficient lookup.

---

## Rates

### Responsibility

Represents character rate multipliers retrieved from external service.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| expRate | float64 | Experience rate multiplier |
| mesoRate | float64 | Meso rate multiplier |
| itemDropRate | float64 | Item drop rate multiplier |
| questExpRate | float64 | Quest experience rate multiplier |

### Processors

#### GetForCharacter

Retrieves computed rates for a character. Returns default rates (all 1.0) if the rate service is unavailable.

---

## System Message

### Responsibility

Produces a command to show a hint box to a character, and bounds how often a given character may be sent a hint.

### Core Models

#### ShowHintBody

| Field | Type | Description |
|-------|------|-------------|
| Hint | string | Hint text to display |
| Width | uint16 | Width of the hint box (0 for auto-calculation) |
| Height | uint16 | Height of the hint box (0 for auto-calculation) |

#### Throttle

Bounds how often a given (tenant, character) pair may be sent a hint, within an in-process window and capacity. State is per-process: with multiple replicas the effective bound is one hint per replica per window.

### Invariants

- A hint for a given (tenant, character) pair is allowed only if the window since the prior hint for that pair has elapsed
- When the tracked-key count reaches capacity, entries older than the window are evicted before recording a new one
- The process-wide hint throttle (GetHintThrottle) uses a one-minute window and a 4096-key capacity

### Processors

#### ShowHint

Produces a Kafka command to show a hint box for a character.
