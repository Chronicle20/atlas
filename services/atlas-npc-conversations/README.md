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
    "dialogueType": "sendYesNo",    // Required: "sendOk", "sendYesNo", "sendSimple", "sendNext", or "sendNextPrev"
    "text": "Hello!",               // Required: Dialogue text
    "choices": [                    // Required based on dialogueType:
      {                             // - sendOk: exactly 2 choices
        "text": "Yes",              // - sendYesNo: exactly 3 choices
        "nextState": "reward",      // - sendSimple: at least 1 choice
        "context": {                // - sendNext: exactly 2 choices
          "key": "value"            // - sendNextPrev: exactly 3 choices
        }                           // Optional: context data
      }
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
- `mapId` - Check character's current map
- `fame` - Check character's fame level
- `item` - Check if character has specific item (requires `itemId` field)

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
