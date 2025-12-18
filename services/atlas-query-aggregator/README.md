# atlas-query-aggregator
Mushroom game query-aggregator Service

## Overview

A RESTful service that provides composite character state validation in the Atlas ecosystem. This service queries dependent services directly to validate conditions against character state.

### Features

- Validates character state against specified conditions
- Supports various comparison operators (=, >, <, >=, <=)
- Returns detailed validation results with pass/fail status
- JSON:API-compliant API design

### Supported Validations

- Job ID validation
- Meso (currency) validation
- Map ID validation
- Fame validation
- Gender validation (0 = male, 1 = female)
- Level validation
- Rebirth count validation
- Dojo points validation
- Vanquisher kills validation
- GM level validation
- Guild leader validation (0 = not a leader, 1 = is a leader)
- Guild membership validation
- Guild rank validation
- Quest status validation
- Quest progress validation
- Marriage gift validation
- Character stats validation (strength, dexterity, intelligence, luck)

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace

### External Service Dependencies

The atlas-query-aggregator service requires connectivity to multiple external Atlas microservices to provide comprehensive character validation capabilities. Each service provides specific data domains required for validation conditions.

#### Character Service (`CHARACTERS` environment variable)
**Base URL**: Configured via `requests.RootUrl("CHARACTERS")`  
**Endpoint**: `GET /characters/{characterId}`  
**Purpose**: Primary source for character-specific data and attributes  

**Data Fields Provided**:
- **Basic Attributes**: `id`, `name`, `accountId`, `worldId`, `level`, `jobId`
- **Stats**: `strength`, `dexterity`, `intelligence`, `luck`, `hp`, `maxHp`, `mp`, `maxMp`, `ap`, `sp`, `experience`
- **Character State**: `mapId`, `x`, `y`, `stance`, `meso`, `fame`, `gender`, `skinColor`, `face`, `hair`
- **Special Fields**: `gm` (GM level), `reborns`, `dojoPoints`, `vanquisherKills`
- **Guild Information**: Embedded guild data including `guild.id` and `guild.rank`

**Integration Notes**:
- Character data is fetched via REST API using JSON:API format
- Guild information is embedded within character response
- GM level validation uses the `gm` field for privilege checks
- All numeric character attributes support comparison operators (`=`, `>`, `<`, `>=`, `<=`)

#### Inventory Service (`INVENTORY` environment variable)
**Base URL**: Configured via `requests.RootUrl("INVENTORY")`  
**Endpoint**: `GET /characters/{characterId}/inventory`  
**Purpose**: Provides item quantity and equipment data for inventory-based validations  

**Data Fields Provided**:
- **Item Quantities**: Template ID to quantity mapping for all character items
- **Equipment Data**: Currently equipped items with slot positions
- **Compartment Data**: Organized inventory compartments (equipable, consumable, setup, etc., cash)

**Integration Notes**:
- Inventory data is lazily loaded via the `InventoryDecorator` when item validations are required
- Supports item quantity checks using `referenceId` parameter to specify template ID
- Equipment and cash equipment are processed separately for proper slot mapping
- Integration occurs through the character processor's `SetInventory()` method

#### Quest Service (*Future Implementation*)
**Status**: Service interface defined, external integration pending  
**Purpose**: Provides quest status and progress tracking for quest-based validations  

**Planned Data Fields**:
- **Quest Status**: `UNDEFINED`, `NOT_STARTED`, `STARTED`, `COMPLETED` (enum values 0-3)
- **Quest Progress**: Numeric progress values for specific quest steps
- **Quest Metadata**: Quest ID, character assignment, completion timestamps

**Integration Design**:
- `GetQuestStatus(characterId, questId)` returns quest completion state
- `GetQuestProgress(characterId, questId, step)` returns progress for specific quest steps
- Uses `referenceId` parameter for quest ID specification
- Supports both status equality checks and progress comparisons

**Current Implementation**: Placeholder methods return default values (`UNDEFINED` status, `0` progress)

#### Marriage Service (*Future Implementation*)
**Status**: Service interface defined, external integration pending  
**Purpose**: Provides marriage-related data for relationship and gift validations  

**Planned Data Fields**:
- **Gift Status**: Boolean indicator for unclaimed marriage gifts
- **Gift Count**: Numeric count of unclaimed gifts
- **Marriage State**: Character relationship status and partner information

**Integration Design**:
- `HasUnclaimedGifts(characterId)` returns boolean gift availability
- `GetUnclaimedGiftCount(characterId)` returns numeric gift count
- Supports boolean equality operations for gift presence validation

**Current Implementation**: Placeholder methods return default values (`false` for gift presence)
## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Requests

#### POST /api/validations

Validates a set of conditions against a character's state.

**Request Body (Structured Format - Recommended):**
```json
{
  "data": {
    "id": "56",
    "type": "validations",
    "attributes": {
      "conditions": [
        {
          "type": "jobId",
          "operator": "=",
          "value": 100
        },
        {
          "type": "meso",
          "operator": ">=",
          "value": 10000
        },
        {
          "type": "level",
          "operator": ">=",
          "value": 30
        },
        {
          "type": "mapId",
          "operator": "=",
          "value": 2000
        },
        {
          "type": "fame",
          "operator": ">=",
          "value": 50
        },
        {
          "type": "gender",
          "operator": "=",
          "value": 0
        },
        {
          "type": "guildId",
          "operator": "=",
          "value": 12345
        },
        {
          "type": "guildLeader",
          "operator": "=",
          "value": 1
        },
        {
          "type": "item",
          "operator": ">=",
          "value": 10,
          "referenceId": 2000001
        },
        {
          "type": "reborns",
          "operator": ">=",
          "value": 1
        },
        {
          "type": "dojoPoints",
          "operator": ">=",
          "value": 1000
        },
        {
          "type": "questStatus",
          "operator": "=",
          "value": 2,
          "referenceId": 1001
        }
      ]
    }
  }
}
```

**Response:**
```json
{
  "data": {
    "type": "validations",
    "id": "56",
    "attributes": {
      "passed": true,
      "details": [
        "Passed: Job ID = 100",
        "Passed: Meso >= 10000",
        "Passed: Level >= 30",
        "Passed: Map ID = 2000",
        "Passed: Fame >= 50",
        "Passed: Gender = 0",
        "Passed: Guild Leader = 1",
        "Passed: Guild ID = 12345",
        "Passed: Item 2000001 quantity >= 10",
        "Passed: Reborns >= 1",
        "Passed: Dojo Points >= 1000",
        "Passed: Quest 1001 Status = 2"
      ],
      "results": [
        {
          "passed": true,
          "description": "Job ID = 100",
          "type": "jobId",
          "operator": "=",
          "value": 100,
          "actualValue": 100
        },
        {
          "passed": true,
          "description": "Meso >= 10000",
          "type": "meso",
          "operator": ">=",
          "value": 10000,
          "actualValue": 15000
        },
        {
          "passed": true,
          "description": "Level >= 30",
          "type": "level",
          "operator": ">=",
          "value": 30,
          "actualValue": 45
        },
        {
          "passed": true,
          "description": "Map ID = 2000",
          "type": "mapId",
          "operator": "=",
          "value": 2000,
          "actualValue": 2000
        },
        {
          "passed": true,
          "description": "Fame >= 50",
          "type": "fame",
          "operator": ">=",
          "value": 50,
          "actualValue": 75
        },
        {
          "passed": true,
          "description": "Gender = 0",
          "type": "gender",
          "operator": "=",
          "value": 0,
          "actualValue": 0
        },
        {
          "passed": true,
          "description": "Guild ID = 12345",
          "type": "guildId",
          "operator": "=",
          "value": 12345,
          "actualValue": 12345
        },
        {
          "passed": true,
          "description": "Guild Leader = 1",
          "type": "guildLeader",
          "operator": "=",
          "value": 1,
          "actualValue": 1
        },
        {
          "passed": true,
          "description": "Item 2000001 quantity >= 10",
          "type": "item",
          "operator": ">=",
          "value": 10,
          "itemId": 2000001,
          "actualValue": 15
        },
        {
          "passed": true,
          "description": "Reborns >= 1",
          "type": "reborns",
          "operator": ">=",
          "value": 1,
          "actualValue": 2
        },
        {
          "passed": true,
          "description": "Dojo Points >= 1000",
          "type": "dojoPoints",
          "operator": ">=",
          "value": 1000,
          "actualValue": 1500
        },
        {
          "passed": true,
          "description": "Quest 1001 Status = 2",
          "type": "questStatus",
          "operator": "=",
          "value": 2,
          "actualValue": 2
        }
      ]
    }
  }
}
```

**Supported Conditions:**

| Condition       | Expression Format Example | Source                                                        |
|-----------------|---------------------------|---------------------------------------------------------------|
| Job             | jobId=100                 | Character Service (character.JobId)                           |
| Meso (Currency) | meso>=10000               | Character Service (character.Meso)                            |
| Map             | mapId=2000                | Character Service (character.MapId)                           |
| Fame            | fame>=50                  | Character Service (character.Fame)                            |
| Gender          | gender=0                  | Character Service (character.Gender) - 0=male, 1=female       |
| Level           | level>=30                 | Character Service (character.Level)                           |
| Reborns         | reborns>=1                | Character Service (character.Reborns)                         |
| Dojo Points     | dojoPoints>=1000          | Character Service (character.DojoPoints)                      |
| Vanquisher Kills| vanquisherKills>=10       | Character Service (character.VanquisherKills)                |
| GM Level        | gmLevel>=1                | Character Service (character.GmLevel)                         |
| Guild ID        | guildId=12345             | Character Service (character.Guild.Id)                        |
| Guild Leader    | guildLeader=1             | Guild Service (guild.IsLeader) - 0=not a leader, 1=is a leader|
| Guild Rank      | guildRank>=2              | Character Service (character.Guild.Rank)                      |
| Quest Status    | questStatus=2             | Quest Service (quest.Status) - 0=UNDEFINED, 1=NOT_STARTED, 2=STARTED, 3=COMPLETED |
| Quest Progress  | questProgress>=5          | Quest Service (quest.Progress) - requires referenceId and step |
| Marriage Gifts  | hasUnclaimedMarriageGifts=1 | Marriage Service (marriage.HasUnclaimedGifts) - 0=false, 1=true |
| Strength        | strength>=100             | Character Service (character.Strength)                        |
| Dexterity       | dexterity>=100            | Character Service (character.Dexterity)                       |
| Intelligence    | intelligence>=100         | Character Service (character.Intelligence)                    |
| Luck            | luck>=100                 | Character Service (character.Luck)   
| Inventory Item  | item[2000001]>=10         | Inventory Service (quantity of item with template ID 2000001) |

**Supported Operators:**
- `=` (equals)
- `>` (greater than)
- `<` (less than)
- `>=` (greater than or equal to)
- `<=` (less than or equal to)

**Additional Parameters:**
- `referenceId` (uint32): Required for quest and item validations. Specifies the quest ID or item template ID to validate against.
- `step` (string): Required for quest progress validations. Specifies the specific quest step to check progress for.

**Quest Status Values:**
- `0` = UNDEFINED
- `1` = NOT_STARTED  
- `2` = STARTED
- `3` = COMPLETED

**Examples of Advanced Validations:**

```json
{
  "type": "questStatus",
  "operator": "=",
  "value": 2,
  "referenceId": 1001
}
```

```json
{
  "type": "questProgress", 
  "operator": ">=",
  "value": 5,
  "referenceId": 1001,
  "step": "mobsKilled"
}
```

```json
{
  "type": "item",
  "operator": ">=", 
  "value": 10,
  "referenceId": 2000001
}
```

## NPC Conversation Validation Examples

The following examples demonstrate how to use the validation API for common NPC conversation scenarios, corresponding to typical `cm` scripting functions used in MapleStory server development.

### Job Advancement NPC

Validate if a character meets requirements for job advancement:

```json
{
  "data": {
    "id": "job-advancement-check",
    "type": "validations",
    "attributes": {
      "conditions": [
        {
          "type": "level",
          "operator": ">=",
          "value": 30
        },
        {
          "type": "jobId",
          "operator": "=",
          "value": 100
        },
        {
          "type": "questStatus",
          "operator": "=",
          "value": 3,
          "referenceId": 1002
        }
      ]
    }
  }
}
```

### Guild Master NPC

Check if player can access guild-specific content:

```json
{
  "data": {
    "id": "guild-master-check",
    "type": "validations",
    "attributes": {
      "conditions": [
        {
          "type": "guildId",
          "operator": "=",
          "value": 12345
        },
        {
          "type": "guildRank",
          "operator": "<=",
          "value": 2
        },
        {
          "type": "level",
          "operator": ">=",
          "value": 50
        }
      ]
    }
  }
}
```

### Wedding NPC

Validate marriage-related interactions:

```json
{
  "data": {
    "id": "wedding-npc-check",
    "type": "validations",
    "attributes": {
      "conditions": [
        {
          "type": "hasUnclaimedMarriageGifts",
          "operator": "=",
          "value": 1
        },
        {
          "type": "meso",
          "operator": ">=",
          "value": 100000
        }
      ]
    }
  }
}
```

### Dojo Master NPC

Check Mu Lung Dojo related requirements:

```json
{
  "data": {
    "id": "dojo-master-check",
    "type": "validations",
    "attributes": {
      "conditions": [
        {
          "type": "dojoPoints",
          "operator": ">=",
          "value": 1000
        },
        {
          "type": "level",
          "operator": ">=",
          "value": 25
        },
        {
          "type": "reborns",
          "operator": ">=",
          "value": 1
        }
      ]
    }
  }
}
```

### GM Event NPC

Validate GM privileges and event participation:

```json
{
  "data": {
    "id": "gm-event-check",
    "type": "validations",
    "attributes": {
      "conditions": [
        {
          "type": "gmLevel",
          "operator": ">=",
          "value": 1
        },
        {
          "type": "vanquisherKills",
          "operator": ">=",
          "value": 10
        }
      ]
    }
  }
}
```

### Quest Progress NPC

Check specific quest progress for multi-step quests:

```json
{
  "data": {
    "id": "quest-progress-check",
    "type": "validations",
    "attributes": {
      "conditions": [
        {
          "type": "questProgress",
          "operator": ">=",
          "value": 5,
          "referenceId": 1001,
          "step": "mobsKilled"
        },
        {
          "type": "questStatus",
          "operator": "=",
          "value": 2,
          "referenceId": 1001
        }
      ]
    }
  }
}
```

### Complex Multi-Condition NPC

Example of a high-level NPC that requires multiple conditions (e.g., End Game Boss NPC):

```json
{
  "data": {
    "id": "endgame-boss-check",
    "type": "validations",
    "attributes": {
      "conditions": [
        {
          "type": "level",
          "operator": ">=",
          "value": 200
        },
        {
          "type": "reborns",
          "operator": ">=",
          "value": 3
        },
        {
          "type": "strength",
          "operator": ">=",
          "value": 500
        },
        {
          "type": "questStatus",
          "operator": "=",
          "value": 3,
          "referenceId": 2001
        },
        {
          "type": "item",
          "operator": ">=",
          "value": 1,
          "referenceId": 4001000
        },
        {
          "type": "vanquisherKills",
          "operator": ">=",
          "value": 100
        }
      ]
    }
  }
}
```

### Corresponding cm Scripting Functions

These validation conditions correspond to common `cm` functions used in NPC scripts:

| Validation Type | cm Function Equivalent |
|----------------|------------------------|
| `level` | `cm.getLevel()` |
| `reborns` | `cm.getReborns()` |
| `dojoPoints` | `cm.getDojoPoints()` |
| `vanquisherKills` | `cm.getVanquisherKills()` |
| `gmLevel` | `cm.gmLevel()` |
| `questStatus` | `cm.getQuestStatus(questId)` |
| `questProgress` | `cm.getQuestProgress(questId, step)` |
| `hasUnclaimedMarriageGifts` | `cm.getUnclaimedMarriageGifts()` |
| `guildId` | `cm.getGuild()?.getId()` |
| `guildRank` | `cm.getGuild()?.getRank()` |

**Error Responses:**
- 400 Bad Request: Invalid condition format or unsupported condition type
- 500 Internal Server Error: Failed to retrieve character data or other server errors
