# Domain

## Monster

### Responsibility

Handles monster death processing including drop creation and experience distribution.

### Core Models

#### DamageEntryModel

Represents damage dealt by a character to a monster.

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character who dealt damage |
| damage | uint32 | Amount of damage dealt |

#### DamageDistributionModel

Represents the distribution of damage across characters for experience calculation.

| Field | Type | Description |
|-------|------|-------------|
| solo | map[uint32]uint32 | Solo character damage (characterId -> damage) |
| party | map[uint32]map[uint32]uint32 | Party damage distribution (partyId -> characterId -> damage) |
| personalRatio | map[uint32]float64 | Personal damage ratio per character |
| experiencePerDamage | float64 | Experience awarded per point of damage |
| standardDeviationRatio | float64 | Threshold for white experience gain |

### Invariants

- DamageDistributionModel requires non-nil solo, party, and personalRatio maps

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

- Builds damage distribution from damage entries
- Filters characters to those still present in the map
- Calculates experience per damage as monsterExperience / monsterHP
- Retrieves character rate multipliers and applies exp rate
- Calculates personal ratio and standard deviation threshold
- Awards experience to each character based on their damage contribution
- White experience gain is awarded when a character's personal ratio meets or exceeds the standard deviation threshold

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

Produces a Kafka command to award experience to a character. The command carries experience distributions indicating the type (WHITE or YELLOW based on white experience gain determination) and a PARTY distribution.

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

Represents party membership retrieved from external service for drop ownership assignment.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Party identifier |

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
