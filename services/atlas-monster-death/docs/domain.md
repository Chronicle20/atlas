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
- Evaluates drop success based on chance
- Creates item or meso drops at calculated positions

#### DistributeExperience

Distributes experience to characters who damaged the monster.

- Builds damage distribution from damage entries
- Filters characters to those still present in the map
- Calculates experience per damage based on monster HP and experience value
- Calculates personal ratio and standard deviation threshold
- Awards experience to each character based on their damage contribution

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

Produces a Kafka command to award experience to a character.

---

## Drop

### Responsibility

Represents drop information and handles drop creation logic.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| itemId | uint32 | Item identifier (0 for meso drops) |
| minimumQuantity | uint32 | Minimum drop quantity |
| maximumQuantity | uint32 | Maximum drop quantity |
| questId | uint32 | Associated quest identifier |
| chance | uint32 | Drop chance |

### Invariants

- minimumQuantity cannot exceed maximumQuantity

### Processors

#### Create

Creates a drop at a calculated position based on drop index.

#### SpawnMeso

Spawns a meso drop with randomized quantity between minimum and maximum.

#### SpawnItem

Spawns an item drop with randomized quantity between minimum and maximum.

#### SpawnDrop

Calculates final drop position and produces spawn drop command.

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

Retrieves a valid drop position within a map from the data service.

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
