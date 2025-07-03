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
