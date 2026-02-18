# REST

## Endpoints

### GET /npcs/conversations

Retrieves all NPC conversation definitions for the current tenant.

- Parameters: None
- Request model: None
- Response model: `[]npc.RestModel` (JSON:API, resource type `"conversations"`)
- Error conditions:
  - 500: Internal error retrieving conversations

### GET /npcs/conversations/{conversationId}

Retrieves a specific NPC conversation definition by UUID.

- Parameters:
  - `conversationId` (path, UUID) — Conversation ID
- Request model: None
- Response model: `npc.RestModel` (JSON:API, resource type `"conversations"`)
- Error conditions:
  - 404: Conversation not found
  - 500: Internal error

### GET /npcs/{npcId}/conversations

Retrieves all NPC conversation definitions for a specific NPC.

- Parameters:
  - `npcId` (path, uint32) — NPC template ID
- Request model: None
- Response model: `[]npc.RestModel` (JSON:API, resource type `"conversations"`)
- Error conditions:
  - 500: Internal error

### POST /npcs/conversations

Creates a new NPC conversation definition.

- Parameters: None
- Request model: `npc.RestModel` (JSON:API, resource type `"conversations"`)
  - `npcId` (uint32, required)
  - `startState` (string, required)
  - `states` ([]RestStateModel, required, at least one)
- Response model: `npc.RestModel` (JSON:API, resource type `"conversations"`)
- Error conditions:
  - 400: Invalid input (extraction failure)
  - 500: Internal error

### PATCH /npcs/conversations/{conversationId}

Updates an existing NPC conversation definition.

- Parameters:
  - `conversationId` (path, UUID) — Conversation ID
- Request model: `npc.RestModel` (JSON:API, resource type `"conversations"`)
- Response model: `npc.RestModel` (JSON:API, resource type `"conversations"`)
- Error conditions:
  - 400: Invalid input
  - 500: Internal error

### DELETE /npcs/conversations/{conversationId}

Deletes an NPC conversation definition (soft delete).

- Parameters:
  - `conversationId` (path, UUID) — Conversation ID
- Request model: None
- Response model: None (204 No Content)
- Error conditions:
  - 500: Internal error

### POST /npcs/conversations/validate

Validates an NPC conversation definition without persisting it.

- Parameters: None
- Request model: `npc.RestModel` (JSON:API, resource type `"conversations"`)
- Response model: `RestValidationResult` (JSON)
  - `valid` (bool)
  - `errors` ([]RestValidationError): `stateId`, `field`, `errorType`, `message`
- Error conditions:
  - 400: Invalid input (extraction failure)

### POST /npcs/conversations/seed

Clears all NPC conversations for the current tenant and loads from JSON files on the filesystem.

- Parameters: None
- Request model: None
- Response model: `SeedResult` (JSON)
  - `deletedCount` (int)
  - `createdCount` (int)
  - `failedCount` (int)
  - `errors` ([]string)
- Error conditions:
  - 500: Internal error

### GET /quests/conversations

Retrieves all quest conversation definitions for the current tenant.

- Parameters: None
- Request model: None
- Response model: `[]quest.RestModel` (JSON:API, resource type `"quest-conversations"`)
- Error conditions:
  - 500: Internal error

### GET /quests/conversations/{conversationId}

Retrieves a specific quest conversation definition by UUID.

- Parameters:
  - `conversationId` (path, UUID) — Conversation ID
- Request model: None
- Response model: `quest.RestModel` (JSON:API, resource type `"quest-conversations"`)
- Error conditions:
  - 404: Quest conversation not found
  - 500: Internal error

### GET /quests/{questId}/conversation

Retrieves the quest conversation definition for a specific quest.

- Parameters:
  - `questId` (path, uint32) — Quest ID
- Request model: None
- Response model: `quest.RestModel` (JSON:API, resource type `"quest-conversations"`)
- Error conditions:
  - 404: Quest conversation not found for quest
  - 500: Internal error

### POST /quests/conversations

Creates a new quest conversation definition.

- Parameters: None
- Request model: `quest.RestModel` (JSON:API, resource type `"quest-conversations"`)
  - `questId` (uint32, required)
  - `npcId` (uint32, optional metadata)
  - `questName` (string, optional metadata)
  - `startStateMachine` (required): `startState` (string, required), `states` ([]RestStateModel, required)
  - `endStateMachine` (optional): `startState` (string), `states` ([]RestStateModel)
- Response model: `quest.RestModel` (JSON:API, resource type `"quest-conversations"`)
- Error conditions:
  - 400: Invalid input
  - 500: Internal error

### PATCH /quests/conversations/{conversationId}

Updates an existing quest conversation definition.

- Parameters:
  - `conversationId` (path, UUID) — Conversation ID
- Request model: `quest.RestModel` (JSON:API, resource type `"quest-conversations"`)
- Response model: `quest.RestModel` (JSON:API, resource type `"quest-conversations"`)
- Error conditions:
  - 400: Invalid input
  - 500: Internal error

### DELETE /quests/conversations/{conversationId}

Deletes a quest conversation definition (soft delete).

- Parameters:
  - `conversationId` (path, UUID) — Conversation ID
- Request model: None
- Response model: None (204 No Content)
- Error conditions:
  - 500: Internal error

### POST /quests/conversations/seed

Clears all quest conversations for the current tenant and loads from JSON files on the filesystem.

- Parameters: None
- Request model: None
- Response model: `SeedResult` (JSON)
- Error conditions:
  - 500: Internal error
