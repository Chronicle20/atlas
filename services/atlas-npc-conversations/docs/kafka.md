# Kafka

## Topics Consumed

### EVENT_TOPIC_CHARACTER_STATUS

- Direction: Event
- Environment variable: `EVENT_TOPIC_CHARACTER_STATUS`
- Headers: Span, Tenant
- Message type: `character.StatusEvent[E]`
- Handled event types:
  - `LOGOUT` — Ends conversation for the character.
  - `CHANNEL_CHANGED` — Ends conversation for the character.
  - `MAP_CHANGED` — Ends conversation for the character.

### COMMAND_TOPIC_NPC

- Direction: Command
- Environment variable: `COMMAND_TOPIC_NPC`
- Headers: Span, Tenant
- Message type: `npc.Command[E]`
- Handled command types:
  - `START_CONVERSATION` — Body: `CommandConversationStartBody` (worldId, channelId, mapId, instance, accountId). Starts an NPC conversation.
  - `CONTINUE_CONVERSATION` — Body: `CommandConversationContinueBody` (action, lastMessageType, selection). Continues an NPC conversation.
  - `END_CONVERSATION` — Body: `CommandConversationEndBody`. Ends an NPC conversation.

### COMMAND_TOPIC_QUEST_CONVERSATION

- Direction: Command
- Environment variable: `COMMAND_TOPIC_QUEST_CONVERSATION`
- Headers: Span, Tenant
- Message type: `quest.Command[E]`
- Handled command types:
  - `START_QUEST_CONVERSATION` — Body: `StartQuestConversationCommandBody` (worldId, channelId, mapId, instance). Starts a quest conversation.

### EVENT_TOPIC_SAGA_STATUS

- Direction: Event
- Environment variable: `EVENT_TOPIC_SAGA_STATUS`
- Headers: Span, Tenant
- Message type: `saga.StatusEvent[E]`
- Handled event types:
  - `COMPLETED` — Body: `StatusEventCompletedBody`. Resumes conversation after saga completion. For craft actions, transitions to success state. For transport, party quest, party quest bonus, and gachapon actions, ends the conversation.
  - `FAILED` — Body: `StatusEventFailedBody` (errorCode, reason, failedStep). Resumes conversation after saga failure. Routes to appropriate failure state based on error code (e.g., `PQ_NOT_IN_PARTY`, `TRANSPORT_CAPACITY_FULL`, `Validation failed`).

## Topics Produced

### COMMAND_TOPIC_NPC_CONVERSATION

- Direction: Command
- Environment variable: `COMMAND_TOPIC_NPC_CONVERSATION`
- Message type: `npc.ConversationCommand[E]`
- Produced command types:
  - `SIMPLE` — Body: `CommandSimpleBody` (type). Sends a simple dialogue message.
  - `TEXT` — No body. Sends a text dialogue message (not used directly; SIMPLE with sub-type).
  - `STYLE` — Body: `CommandStyleBody` (styles []uint32). Sends a style selection dialogue.
  - `NUMBER` — Body: `CommandNumberBody` (defaultValue, minValue, maxValue). Sends a number input dialogue.
  - `SLIDE_MENU` — Body: `CommandSlideMenuBody` (menuType). Sends a slide menu dialogue.

### EVENT_TOPIC_CHARACTER_STATUS

- Direction: Event
- Environment variable: `EVENT_TOPIC_CHARACTER_STATUS`
- Message type: `npc.StatusEvent[E]`
- Produced event types:
  - `STAT_CHANGED` — Body: `StatusEventStatChangedBody` (channelId, exclRequestSent, updates, values). Emitted after certain operations.

### COMMAND_TOPIC_SAGA

- Direction: Command
- Environment variable: `COMMAND_TOPIC_SAGA`
- Produced by saga processor to send saga commands to atlas-saga-orchestrator.

### COMMAND_TOPIC_GUILD

- Direction: Command
- Environment variable: `COMMAND_TOPIC_GUILD`
- Message type: `guild.Command[E]`
- Produced command types:
  - `REQUEST_NAME` — Body: `RequestNameBody` (worldId, channelId).
  - `REQUEST_EMBLEM` — Body: `RequestEmblemBody` (worldId, channelId).
  - `REQUEST_DISBAND` — Body: `RequestDisbandBody` (worldId, channelId).
  - `REQUEST_CAPACITY_INCREASE` — Body: `RequestCapacityIncreaseBody` (worldId, channelId).

## Message Types

### npc.Command

```
NpcId       uint32
CharacterId uint32
Type        string
Body        E
```

### npc.ConversationCommand

```
WorldId        world.Id
ChannelId      channel.Id
MapId          _map.Id
Instance       uuid.UUID
CharacterId    uint32
NpcId          uint32
Speaker        string
EndChat        bool
SecondaryNpcId uint32
Message        string
Type           string
Body           E
```

### saga.StatusEvent

```
TransactionId uuid.UUID
Type          string
Body          E
```

### quest.Command

```
QuestId     uint32
NpcId       uint32
CharacterId uint32
Type        string
Body        E
```

### character.StatusEvent

```
TransactionId uuid.UUID
WorldId       world.Id
CharacterId   uint32
Type          string
Body          E
```

### guild.Command

```
CharacterId uint32
Type        string
Body        E
```

## Transaction Semantics

- Saga-based operations (craft, transport, gachapon, party quest) store a pending saga ID in the conversation context. The conversation pauses until a saga status event (COMPLETED or FAILED) is received.
- Saga status events are correlated to conversations via `GetContextBySagaId` on the conversation registry.
- Character status events (logout, channel change, map change) unconditionally end any active conversation for the character.
