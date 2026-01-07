# atlas-npc-conversations
Mushroom game NPC Conversations Service

## Table of Contents
1. [Overview](#overview)
2. [Technical Stack](#technical-stack)
3. [Key Features](#key-features)
4. [Conversation Model](#conversation-model)
5. [Setup Instructions](#setup-instructions)
6. [Environment Variables](#environment-variables)
7. [Integration](#integration)
8. [API](#api)
9. [Example Conversation](#example-conversation)
10. [Testing](#testing)

## Overview

A RESTful resource which provides NPC conversation services. This service implements a JSON-driven NPC conversation system that integrates with:

- **atlas-saga-orchestrator** for performing distributed transactions (e.g., item rewards, warps, job changes)
- **atlas-query-aggregator** for character state validations (e.g., job ID, mesos, inventory checks)

The service follows the Atlas microservice architecture and style guide, leveraging domain-driven design, GORM for PostgreSQL, and tenant-aware storage throughout the service.

## Technical Stack

### Go Version
- Go 1.24.2 (latest)

### Key Dependencies
- **Database**: gorm.io/gorm with PostgreSQL driver
- **Messaging**: segmentio/kafka-go for Kafka integration
- **API**: gorilla/mux for routing, api2go/jsonapi for JSON:API implementation
- **Observability**: opentracing/opentracing-go, uber/jaeger-client-go, sirupsen/logrus
- **Utilities**: google/uuid for unique identifiers
- **Internal Libraries**: Atlas libraries (atlas-constants, atlas-kafka, atlas-model, atlas-rest, atlas-tenant)

## Key Features

- **JSON-Driven Conversations**: Store structured NPC conversation trees in PostgreSQL, with each tree represented as a single JSON blob per NPC.
- **Tenant Awareness**: Fully tenant-aware across all database operations, caching, and runtime logic.
- **State Machine**: Interpret player conversations using a JSON state machine.
- **Condition Evaluation**: Evaluate conditions using local checks and the atlas-query-aggregator.
- **Operation Execution**: Execute operations directly or via the atlas-saga-orchestrator.
- **Kafka Integration**: Emit Kafka events using the Provider pattern.

## Conversation Model

The conversation system is built around a state machine model with the following components:

- **Conversation**: The top-level container for an NPC's conversation tree.
- **States**: Individual states in the conversation, each with a unique ID.
- **State Types**:
  - **Dialogue**: Present text and choices to the player.
  - **GenericAction**: Execute operations and evaluate conditions.
  - **CraftAction**: Handle crafting mechanics.
  - **ListSelection**: Present a list of options to the player.
  - **AskStyle**: Present cosmetic style selection interface to the player (hair, face, skin).
- **Operations**: Actions that can be performed during a conversation (e.g., award items, mesos, experience).
- **Conditions**: Criteria that must be met to progress in the conversation.

## Input Specification

### Conversation Input Structure

The create (POST) and update (PATCH) endpoints accept the following data structure:

```json
{
  "data": {
    "type": "conversations",
    "attributes": {
      "npcId": 9010000,              // uint32 - Required
      "startState": "greeting",       // string - Required
      "states": []                    // Array of states - At least one required
    }
  }
}
```

### State Types

Each state in the `states` array must have:
- `id` (string): Unique identifier for the state - Required
- `type` (string): One of "dialogue", "genericAction", "craftAction", "listSelection" - Required
- One of: `dialogue`, `genericAction`, `craftAction`, or `listSelection` object based on type

#### Dialogue State

```json
{
  "id": "greeting",
  "type": "dialogue",
  "dialogue": {
    "dialogueType": "sendYesNo",    // Required: "sendOk", "sendYesNo", "sendNext", "sendNextPrev", "sendPrev", or "sendAcceptDecline"
    "text": "Hello!",               // Required: Dialogue text
    "choices": [                    // Required based on dialogueType:
      {                             // - sendOk: exactly 2 choices
        "text": "Yes",              // - sendYesNo: exactly 3 choices
        "nextState": "reward",      // - sendAcceptDecline: exactly 3 choices
        "context": {                // - sendNext: exactly 2 choices
          "key": "value"            // - sendNextPrev: exactly 3 choices
        }                           // - sendPrev: exactly 2 choices
      }                             // Note: For menu selections, use listSelection state type
    ]
  }
}
```

#### Generic Action State

```json
{
  "id": "reward",
  "type": "genericAction",
  "genericAction": {
    "operations": [],               // Array of operations to execute
    "outcomes": []                  // Array of outcomes determining next state
  }
}
```

#### Craft Action State

```json
{
  "id": "craft",
  "type": "craftAction",
  "craftAction": {
    "itemId": "2000000",          // string - Item to craft - Required
    "materials": [4000000, 4000001], // []uint32 - Material item IDs - At least one required
    "quantities": [10, 5],          // []uint32 - Material quantities - Must match materials length
    "mesoCost": 1000,               // uint32 - Meso cost
    "stimulatorId": 0,              // uint32 - Optional stimulator item
    "stimulatorFailChance": 0.0,    // float64 - Optional failure chance
    "successState": "craftSuccess", // string - Required
    "failureState": "craftFail",    // string - Required
    "missingMaterialsState": "noMats" // string - Required
  }
}
```

#### List Selection State

```json
{
  "id": "selection",
  "type": "listSelection",
  "listSelection": {
    "title": "Select an option:",   // string - Required
    "choices": [                    // Same as dialogue choices
      {
        "text": "Option 1",
        "nextState": "option1"
      }
    ]
  }
}
```

#### Ask Style State

Presents a cosmetic style selection interface (hair styles, hair colors, face styles, skin colors):

```json
{
  "id": "selectHairStyle",
  "type": "askStyle",
  "askStyle": {
    "text": "Choose your new hairstyle!",     // string - Required
    "stylesContextKey": "availableStyles",  // string - Required: Context key containing styles array
    "contextKey": "selectedStyle",          // string - Required: Context key to store selection
    "nextState": "applyStyle"               // string - Required: Next state after selection
  }
}
```

**Note**: The styles must be pre-populated in context using local operations like `local:generate_hair_styles` or `local:generate_hair_colors`.

### Operations

Operations are actions executed during a `genericAction` state:

```json
{
  "type": "operation_type",         // string - Required
  "params": {                       // map[string]string - Parameters vary by type
    "key": "value"
  }
}
```

#### Available Operations

##### Operations (executed via saga orchestrator)
- `award_item` - Award an item to the character
  - Params: `itemId`, `quantity`
- `award_mesos` - Award mesos (game currency)
  - Params: `amount`, `actorId` (optional), `actorType` (optional, default "NPC")
- `award_exp` - Award experience points
  - Params: `amount`, `type` (optional, default "WHITE"), `attr1` (optional, default 0)
- `award_level` - Award character levels
  - Params: `amount`
- `warp_to_map` - Warp character to specific map and portal
  - Params: `mapId`, `portalId`
- `warp_to_random_portal` - Warp character to random portal in map
  - Params: `mapId`
- `change_job` - Change character's job
  - Params: `jobId`
- `create_skill` - Create a new skill for character
  - Params: `skillId`, `level` (optional, default 1), `masterLevel` (optional, default 1)
- `update_skill` - Update an existing skill
  - Params: `skillId`, `level` (optional, default 1), `masterLevel` (optional, default 1)
- `destroy_item` - Remove items from inventory
  - Params: `itemId`, `quantity`
- `change_hair` - Change character's hair style
  - Params: `styleId` (hair style ID, can use context references like `{context.selectedHair}`)
- `change_face` - Change character's face style
  - Params: `styleId` (face style ID, can use context references like `{context.selectedFace}`)
- `change_skin` - Change character's skin color
  - Params: `styleId` (skin color ID 0-9, can use context references like `{context.selectedSkin}`)
- `increase_buddy_capacity` - Increase character's buddy list capacity
  - Params: `amount` (byte, capacity increase amount)
- `gain_closeness` - Increase pet closeness/intimacy
  - Params: `petId` (uint32) or `petIndex` (int8, slot position), `amount` (uint16)
- `spawn_monster` - Spawn monsters at a location (foothold resolved automatically by saga-orchestrator)
  - Params: `monsterId` (monster template ID), `x` (x coordinate), `y` (y coordinate), `count` (optional, default 1), `team` (optional, default 0)
- `complete_quest` - Complete a quest for the character (stub implementation - no quest service yet)
  - Params: `questId` (quest ID to complete), `npcId` (optional, defaults to conversation NPC)

##### Local Operations (executed within npc-conversations service)
- `local:generate_hair_styles` - Generate available hair styles for character
  - Params:
    - `baseStyles` (comma-separated hair base IDs)
    - `genderFilter` (optional, "true" to filter by character gender)
    - `preserveColor` (optional, "true" to preserve current hair color)
    - `validateExists` (optional, "true" to validate styles exist in WZ data)
    - `excludeEquipped` (optional, "true" to exclude current hair)
    - `outputContextKey` (required, context key to store results)
- `local:generate_hair_colors` - Generate available hair colors for character
  - Params:
    - `colors` (comma-separated color IDs 0-7)
    - `validateExists` (optional, "true" to validate colors exist)
    - `excludeEquipped` (optional, "true" to exclude current color)
    - `outputContextKey` (required, context key to store results)
- `local:generate_face_styles` - Generate available face styles for character
  - Params:
    - `baseStyles` (comma-separated face base IDs)
    - `genderFilter` (optional, "true" to filter by character gender)
    - `validateExists` (optional, "true" to validate styles exist in WZ data)
    - `excludeEquipped` (optional, "true" to exclude current face)
    - `outputContextKey` (required, context key to store results)
- `local:select_random_cosmetic` - Randomly select a cosmetic from a styles array
  - Params:
    - `stylesContextKey` (required, context key containing styles array)
    - `outputContextKey` (required, context key to store selected style)
- `local:fetch_map_player_counts` - Fetch current player counts for multiple maps
  - Params:
    - `mapIds` (comma-separated string of map IDs, supports context references)
  - Stores results in context with keys: `playerCount_{mapId}` for each map
- `local:calculate_lens_coupon` - Calculate one-time lens item ID from selected face
  - Params:
    - `selectedFaceContextKey` (required, context key containing the selected face ID)
    - `outputContextKey` (required, context key to store calculated lens item ID)
  - Formula: `lensItemId = 5152100 + (selectedFace / 100) % 10`
  - Maps face colors (0-7) to items 5152100-5152107
- `local:log` - Log an informational message
  - Params:
    - `message` (string, supports context references)
- `local:debug` - Log a debug message
  - Params:
    - `message` (string, supports context references)

### Conditions

Conditions are evaluated to determine the next state in `outcomes`:

```json
{
  "type": "condition_type",         // string - Required
  "operator": "=",                  // string - Required: "=", ">", "<", ">=", "<="
  "value": "100",                   // string - Required
  "itemId": "0"                   // string - Required only for "item" type
}
```

#### Available Condition Types
- `jobId` - Check character's job ID
- `meso` - Check character's meso amount
- `mapId` - Check character's current map ID
- `fame` - Check character's fame level
- `gender` - Check character's gender (0 = male, 1 = female)
- `level` - Check character's level
- `reborns` - Check character's rebirth count
- `dojoPoints` - Check character's Mu Lung Dojo points
- `vanquisherKills` - Check character's vanquisher kill count
- `gmLevel` - Check character's GM level
- `guildId` - Check character's guild ID (0 = not in guild)
- `guildLeader` - Check if character is guild leader (0 = not leader, 1 = is leader)
- `guildRank` - Check character's guild rank
- `questStatus` - Check quest status (requires `referenceId` field with quest ID)
  - Values: 0 = UNDEFINED, 1 = NOT_STARTED, 2 = STARTED, 3 = COMPLETED
- `questProgress` - Check quest progress (requires `referenceId` and `step` fields)
- `hasUnclaimedMarriageGifts` - Check for unclaimed marriage gifts (0 = false, 1 = true)
- `strength` - Check character's strength stat
- `dexterity` - Check character's dexterity stat
- `intelligence` - Check character's intelligence stat
- `luck` - Check character's luck stat
- `buddyCapacity` - Check character's buddy list capacity
- `petCount` - Check number of pets character has
- `mapCapacity` - Check player count in a specific map (requires `referenceId` with map ID)
- `item` - Check if character has specific item (requires `referenceId` field with item template ID)

### Outcomes

Outcomes determine state transitions based on conditions:

```json
{
  "conditions": [],                 // Array of conditions to evaluate
  "nextState": "state1",           // string - Optional
  "successState": "success",       // string - Optional
  "failureState": "failure"        // string - Optional
}
```

**Note**: At least one of `nextState`, `successState`, or `failureState` must be provided.

### Context References

Operation parameters can reference conversation context values using the format `context.{key}`:

```json
{
  "type": "award_item",
  "params": {
    "itemId": "context.selectedItem",    // References context value
    "quantity": "context.rewardAmount"    // References context value
  }
}
```

This allows dynamic values to be passed between conversation states.

### Arithmetic Expressions

Both operation parameters and condition values support arithmetic expressions, enabling dynamic calculations based on context values. This is particularly useful for bulk crafting scenarios where material requirements scale with quantity.

**Supported Operators**: `*`, `/`, `+`, `-`

**Examples**:

Bulk crafting with multiplied material requirements:
```json
{
  "type": "genericAction",
  "genericAction": {
    "operations": [
      {
        "type": "destroy_item",
        "params": {
          "itemId": "4000003",
          "quantity": "10 * {context.quantity}"  // If quantity=5, destroys 50 items
        }
      },
      {
        "type": "award_item",
        "params": {
          "itemId": "4003001",
          "quantity": "{context.quantity}"  // Awards 5 items
        }
      }
    ],
    "outcomes": [
      {
        "conditions": [
          {
            "type": "item",
            "operator": ">=",
            "value": "10 * {context.quantity}",  // Validates player has enough materials
            "referenceId": 4000003
          }
        ],
        "nextState": "craftSuccess"
      }
    ]
  }
}
```

**How It Works**:
1. **Context substitution happens first**: `{context.quantity}` is replaced with the actual value (e.g., `"5"`)
2. **Expression evaluation happens second**: `"10 * 5"` is evaluated to `50`
3. **Result is used**: The operation uses the calculated value

**Evaluation Order**: Expressions are evaluated left-to-right without operator precedence. For complex calculations, use multiple steps or pre-calculate values.

## Setup Instructions

### Prerequisites
- Go 1.24.2 or later
- PostgreSQL database
- Kafka cluster
- Jaeger (for distributed tracing)

## Environment Variables

The service is configured using the following environment variables:

- **JAEGER_HOST** - Jaeger [host]:[port] for distributed tracing
- **LOG_LEVEL** - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)
- **CONFIG_FILE** - Location of service configuration file
- **BOOTSTRAP_SERVERS** - Kafka [host]:[port]
- **BASE_SERVICE_URL** - [scheme]://[host]:[port]/api/
- **COMMAND_TOPIC_GUILD** - Kafka topic for transmitting Guild commands
- **COMMAND_TOPIC_NPC** - Kafka topic for transmitting NPC commands 
- **COMMAND_TOPIC_NPC_CONVERSATION** - Kafka topic for transmitting NPC Conversation commands
- **COMMAND_TOPIC_SAGA** - Kafka topic for transmitting Saga commands
- **EVENT_TOPIC_CHARACTER_STATUS** - Kafka Topic for receiving Character status events
- **WORLD_ID** - World ID for the service instance

## Integration

### atlas-query-aggregator

For conversation state conditions requiring character validations, the service:

- Synchronously invokes POST /api/validations on the atlas-query-aggregator.
- Passes structured conditions defined in the conversation state.
- Handles pass/fail results to drive state transitions.

### atlas-saga-orchestrator

For complex conversation actions (e.g., crafting, job changes, warps), the service:

- Generates SagaCommand messages and emits them to the COMMAND_TOPIC_SAGA.
- Populates steps based on the conversation-defined operations.
- Ensures saga payloads conform to the supported actions in atlas-saga-orchestrator.

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Endpoints

The service provides the following RESTful endpoints for managing NPC conversations:

#### Get All Conversations

Retrieves all NPC conversation definitions.

```
GET /npcs/conversations
```

#### Get Conversation by ID

Retrieves a specific NPC conversation definition by its UUID.

```
GET /npcs/conversations/{conversationId}
```

#### Get Conversations by NPC ID

Retrieves all NPC conversation definitions for a specific NPC.

```
GET /npcs/{npcId}/conversations
```

#### Create Conversation

Creates a new NPC conversation definition.

```
POST /npcs/conversations
{
  "data": {
    "type": "conversations",
    "attributes": {
      "npcId": 9010000,
      "startState": "greeting",
      "states": [
        {
          "id": "greeting",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendYesNo",
            "text": "Hello! Would you like to receive a reward?",
            "choices": [
              {
                "text": "Yes",
                "nextState": "reward"
              },
              {
                "text": "No",
                "nextState": "goodbye"
              }
            ]
          }
        },
        // Additional states...
      ]
    }
  }
}
```

#### Update Conversation

Updates an existing NPC conversation definition.

```
PATCH /npcs/conversations/{conversationId}
{
  "data": {
    "type": "conversations",
    "id": "{conversationId}",
    "attributes": {
      "npcId": 9010000,
      "startState": "greeting",
      "states": [
        // Updated states...
      ]
    }
  }
}
```

#### Delete Conversation

Deletes an NPC conversation definition.

```
DELETE /npcs/conversations/{conversationId}
```

## Example Conversation

Here's a simplified example of a conversation tree:

```json
{
  "data": {
    "type": "conversations",
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "attributes": {
      "npcId": 9010000,
      "startState": "greeting",
      "states": [
        {
          "id": "greeting",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendYesNo",
            "text": "Hello! Would you like to receive a reward?",
            "choices": [
              {
                "text": "Yes",
                "nextState": "reward"
              },
              {
                "text": "No",
                "nextState": "goodbye"
              }
            ]
          }
        },
        {
          "id": "reward",
          "type": "genericAction",
          "genericAction": {
            "operations": [
              {
                "type": "award_item",
                "params": {
                  "itemId": "2000000",
                  "quantity": "10"
                }
              }
            ],
            "outcomes": [
              {
                "conditions": [
                  {
                    "type": "constant",
                    "operator": "eq",
                    "value": "true"
                  }
                ],
                "nextState": "thanks"
              }
            ]
          }
        },
        {
          "id": "thanks",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendOk",
            "text": "Here's your reward! Thanks for visiting!",
            "choices": []
          }
        },
        {
          "id": "goodbye",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendOk",
            "text": "Goodbye! Come back soon!",
            "choices": []
          }
        }
      ]
    }
  }
}
```

## Cosmetic System

The service supports character cosmetic changes (hair, face, skin) through a comprehensive system that integrates with the Atlas saga orchestrator for distributed transaction handling.

### Architecture

Cosmetic changes flow through the following architecture:

1. **NPC Conversation** → 2. **atlas-saga-orchestrator** → 3. **atlas-character** (via Kafka) → 4. **Database Update** → 5. **Event Emission** → 6. **atlas-channel**

This ensures:
- **Transactional Integrity**: Changes are part of a saga with rollback support
- **Event-Driven**: Other services are notified of cosmetic changes
- **Audit Trail**: All changes are logged and traceable

### Cosmetic Change Workflow

#### Player Selection (e.g., Hair Salon NPC 1012103)

1. **Generate Available Styles**:
   ```json
   {
     "type": "genericAction",
     "operations": [{
       "type": "local:generate_hair_styles",
       "params": {
         "baseStyles": "30060,30140,30200,30210",
         "genderFilter": "true",
         "preserveColor": "true",
         "validateExists": "true",
         "excludeEquipped": "true",
         "outputContextKey": "availableStyles"
       }
     }]
   }
   ```

2. **Present Style Selection**:
   ```json
   {
     "type": "askStyle",
     "askStyle": {
       "text": "Choose your new hairstyle!",
       "stylesContextKey": "availableStyles",
       "contextKey": "selectedHair",
       "nextState": "checkCoupon"
     }
   }
   ```

3. **Validate and Apply**:
   ```json
   {
     "type": "genericAction",
     "operations": [
       {
         "type": "destroy_item",
         "params": {"itemId": "5150001", "quantity": "1"}
       },
       {
         "type": "change_hair",
         "params": {"styleId": "{context.selectedHair}"}
       }
     ]
   }
   ```

#### Random Selection (e.g., Brittany NPC 1012104)

1. **Generate Styles** (same as above)

2. **Random Selection**:
   ```json
   {
     "type": "genericAction",
     "operations": [{
       "type": "local:select_random_cosmetic",
       "params": {
         "stylesContextKey": "availableStyles",
         "outputContextKey": "selectedHair"
       }
     }]
   }
   ```

3. **Apply** (same as player selection)

### Saga Integration

The `change_hair`, `change_face`, and `change_skin` operations create saga steps that:

1. **Emit Kafka Command** to atlas-character service
2. **Update Database** in a transaction (capturing old value)
3. **Emit Status Events**:
   - `HairChanged` / `FaceChanged` / `SkinColorChanged` (with old/new values)
   - `StatChanged` (triggers client update)
4. **Support Rollback** (compensation logic in saga-orchestrator)

### Rollback Support

The saga compensator handles failed cosmetic changes:

- **Current Implementation**: Acknowledges compensation without reverting (character keeps new cosmetic)
- **Future Enhancement**: Could store old cosmetic value in saga payload for full rollback
- **Justification**: Similar to `CreateCharacter` - partial rollback is acceptable for non-critical cosmetics

See: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go`

### Example NPCs

- **NPC 1012103** (Head Hair Salon): Player-selected haircuts and hair dye
- **NPC 1012104** (Brittany): Random haircuts and hair dye

Both NPCs demonstrate the full cosmetic change workflow with coupon consumption and validation.
