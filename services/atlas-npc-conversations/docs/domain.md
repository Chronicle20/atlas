# Domain

## Conversation

### Responsibility

Shared domain models and processing logic for NPC and quest conversation state machines. Manages conversation lifecycle (start, continue, end), state processing, condition evaluation, operation execution, and context tracking.

### Core Models

- **StateContainer** — Interface for state machine containers. Provides `StartState()` and `FindState(stateId)`. Implemented by `npc.Model` and `quest.StateMachine`.
- **NpcConversation** — Extends `StateContainer` with `NpcId()` and `States()`. Implemented by `npc.Model`.
- **StateModel** — A state in a conversation. Has `id`, `stateType`, and exactly one type-specific model (dialogue, genericAction, craftAction, transportAction, gachaponAction, partyQuestAction, partyQuestBonusAction, listSelection, askNumber, askStyle, askSlideMenu).
- **DialogueModel** — Dialogue state. Has `dialogueType`, `text`, `speaker`, `endChat`, `secondaryNpcId`, and `choices`. Speaker is `"NPC"` or `"CHARACTER"`. EndChat defaults to true.
- **ChoiceModel** — A choice in a dialogue or list selection. Has `text`, `nextState`, and `context` (key-value map merged into conversation context on selection).
- **GenericActionModel** — Executes operations and evaluates outcomes. Has `operations` and `outcomes`.
- **CraftActionModel** — Crafting via saga. Has `itemId`, `materials`, `quantities`, `mesoCost`, `stimulatorId`, `stimulatorFailChance`, `successState`, `failureState`, `missingMaterialsState`.
- **TransportActionModel** — Instance transport via saga. Has `routeName`, `failureState`, `capacityFullState`, `alreadyInTransitState`, `routeNotFoundState`, `serviceErrorState`.
- **GachaponActionModel** — Gachapon via saga. Has `gachaponId`, `ticketItemId`, `failureState`.
- **PartyQuestActionModel** — Party quest registration via saga. Has `questId`, `failureState`, `notInPartyState`, `notLeaderState`.
- **PartyQuestBonusActionModel** — Party quest bonus entry via saga. Has `failureState`.
- **ListSelectionModel** — List selection. Has `title` and `choices`.
- **AskNumberModel** — Number input. Has `text`, `defaultValue`, `minValue`, `maxValue`, `contextKey` (default `"quantity"`), `nextState`.
- **AskStyleModel** — Style selection. Has `text`, `styles` (static) or `stylesContextKey` (dynamic from context), `contextKey` (default `"selectedStyle"`), `nextState`.
- **AskSlideMenuModel** — Slide menu selection. Has `title`, `menuType`, `contextKey` (default `"selectedOption"`), `choices`.
- **OperationModel** — An operation in a generic action. Has `operationType` and `params` (string key-value map).
- **ConditionModel** — A condition for outcome evaluation. Has `conditionType`, `operator`, `value`, `referenceId`, `step`, `worldId`, `channelId`, `includeEquipped`.
- **OutcomeModel** — Determines state transition based on conditions. Has `conditions` and `nextState`.
- **ConversationContext** — Runtime state for an active conversation. Has `field`, `characterId`, `npcId`, `currentState`, `conversation` (StateContainer), `context` (key-value map), `pendingSagaId`, `conversationType` (`"npc"` or `"quest"`), `sourceId`.

### Invariants

- StateModel requires a non-empty `id` and exactly one type-specific model matching its `stateType`.
- DialogueModel requires non-empty `dialogueType` and `text`. Choice count is validated per dialogue type: sendOk requires 2, sendNext requires 2, sendNextPrev requires 3, sendPrev requires 3, sendYesNo requires 3, sendAcceptDecline requires 3.
- CraftActionModel requires non-empty `itemId`, at least one material, quantities matching materials length, and non-empty `successState`, `failureState`, `missingMaterialsState`.
- TransportActionModel requires non-empty `routeName` and `failureState`.
- GachaponActionModel requires non-empty `gachaponId`, non-zero `ticketItemId`, and non-empty `failureState`.
- PartyQuestActionModel requires non-empty `questId` and `failureState`.
- PartyQuestBonusActionModel requires non-empty `failureState`.
- AskNumberModel requires `minValue <= defaultValue <= maxValue` and `maxValue > 0`.
- AskStyleModel requires either static `styles` or `stylesContextKey` (not both empty), and non-empty `nextState`.
- AskSlideMenuModel requires at least one choice.
- Only one conversation per character at a time. Starting a conversation fails if one already exists.
- ConversationContext defaults to `NpcConversationType` if not set.

### State Transitions

State types are: `dialogue`, `genericAction`, `craftAction`, `transportAction`, `gachaponAction`, `partyQuestAction`, `partyQuestBonusAction`, `listSelection`, `askNumber`, `askStyle`, `askSlideMenu`.

Dialogue types are: `sendOk`, `sendYesNo`, `sendAcceptDecline`, `sendNext`, `sendNextPrev`, `sendPrev`.

- **dialogue** — Sends dialogue to client. Waits for player input. Player action resolves to a choice via `ChoiceFromAction`. Empty `nextState` ends the conversation.
- **genericAction** — Executes operations sequentially. Evaluates outcomes in order. First outcome whose conditions pass determines the next state. If no outcome matches, conversation ends.
- **craftAction** — Builds a saga with validation, material destruction, meso deduction, and item award steps. Stores saga ID and state references in context. Waits for saga completion/failure.
- **transportAction** — Builds a saga with a single `start_instance_transport` step. Stores failure state variants in context. Waits for saga completion/failure. On success, conversation ends (player is warped).
- **gachaponAction** — Builds a saga to destroy ticket and select reward. Waits for saga completion/failure. On success, conversation ends.
- **partyQuestAction** — Builds a saga with a single `register_party_quest` step. Waits for saga completion/failure. On success, conversation ends (party is warped).
- **partyQuestBonusAction** — Builds a saga with a single `enter_party_quest_bonus` step. Waits for saga completion/failure.
- **listSelection** — Sends list to client. Waits for player selection.
- **askNumber** — Sends number input to client. Waits for player input. Stores result in context using `contextKey`.
- **askStyle** — Sends style selection to client. Resolves styles from static array or context key. Waits for player selection. Stores result in context using `contextKey`.
- **askSlideMenu** — Sends slide menu to client. Waits for player selection.

### Processors

- **Processor** (conversation) — Interface: `Start`, `StartQuest`, `Continue`, `End`.
- **ProcessorImpl** (conversation) — Implements Processor. Uses Evaluator for condition evaluation, OperationExecutor for operation execution, and NpcConversationProvider for NPC conversation lookup. Processes states in a loop until a waiting state (dialogue, list, number, style, slide menu, saga) or end is reached.
- **Evaluator** — Interface for condition evaluation. Evaluates conditions by sending validation requests to atlas-query-aggregator. Supports context references and arithmetic expressions in values.
- **OperationExecutor** — Executes operations. Local operations (`local:*` prefix) execute within the service. Remote operations execute via saga orchestrator.
- **Validator** — Validates conversation structure: state references, reachability from start state, circular reference detection.
- **Registry** — Thread-safe in-memory conversation context store. Per-tenant, per-character. Singleton via `sync.Once`. Supports lookup by saga ID for saga resumption.

## NPC Conversation

### Responsibility

NPC conversation definitions. Each definition associates an NPC ID with a conversation state machine.

### Core Models

- **Model** — An NPC conversation definition. Has `id` (UUID), `npcId` (uint32), `startState` (string), `states` ([]StateModel), `createdAt`, `updatedAt`. Implements `StateContainer` and `NpcConversation`.

### Invariants

- `npcId` must be non-zero.
- `startState` must be non-empty.
- At least one state is required.

### Processors

- **Processor** (npc) — Interface: `Create`, `Update`, `Delete`, `ByIdProvider`, `ByNpcIdProvider`, `AllByNpcIdProvider`, `AllProvider`, `DeleteAllForTenant`, `Seed`.
- **ProcessorImpl** (npc) — Implements Processor. Tenant-scoped CRUD operations. `Seed` clears all conversations for the tenant and loads from JSON files on the filesystem.

## Quest Conversation

### Responsibility

Quest conversation definitions. Each definition associates a quest ID with dual state machines for quest acceptance and completion.

### Core Models

- **Model** — A quest conversation definition. Has `id` (UUID), `questId` (uint32), `npcId` (uint32, metadata), `questName` (string, metadata), `startStateMachine`, `endStateMachine` (optional), `createdAt`, `updatedAt`.
- **StateMachine** — A state machine within a quest conversation. Has `startState` (string) and `states` ([]StateModel). Implements `StateContainer`.

### Invariants

- `questId` must be non-zero.
- `startStateMachine` must have a non-empty `startState` and at least one state.
- `endStateMachine` is optional (nil if quest only has start dialogue).

### Processors

- **Processor** (quest) — Interface: `Create`, `Update`, `Delete`, `ByIdProvider`, `ByQuestIdProvider`, `AllProvider`, `DeleteAllForTenant`, `Seed`, `GetStateMachineForCharacter`.
- **ProcessorImpl** (quest) — Implements Processor. Tenant-scoped CRUD operations. `GetStateMachineForCharacter` routes to startStateMachine (quest NOT_STARTED) or endStateMachine (quest STARTED) based on quest status queried from atlas-query-aggregator. `Seed` clears all quest conversations for the tenant and loads from JSON files.

## Saga

### Responsibility

Saga model types and saga creation for distributed operations.

### Core Models

- Re-exports types from `atlas-script-core/saga`: `Type`, `Saga`, `Status`, `Action`, and payload types.
- Local payload types: `ShowGuideHintPayload`, `ShowIntroPayload`, `SetHPPayload`, `ResetStatsPayload`, `StartQuestPayload`, `StartInstanceTransportPayload`, `RegisterPartyQuestPayload`, `WarpPartyQuestMembersToMapPayload`, `LeavePartyQuestPayload`, `StageClearAttemptPqPayload`, `EnterPartyQuestBonusPayload`, `ValidateCharacterStatePayload`.

### Processors

- **Processor** (saga) — Creates sagas by emitting saga commands via Kafka.
- **Builder** (saga) — Builds saga instances with transaction ID, type, initiator, and steps.

## Cosmetic

### Responsibility

Hair, face, and skin style generation for cosmetic change conversations.

### Processors

- Generates available styles based on character gender, current cosmetics, and WZ data validation.
- Supports hair styles, hair colors, face styles, face colors.

## NPC Talk

### Responsibility

Sends NPC dialogue messages to the client via Kafka.

### Processors

- **Processor** (npc) — Sends dialogue commands: `SendSimple`, `SendNext`, `SendNextPrevious`, `SendPrevious`, `SendOk`, `SendYesNo`, `SendAcceptDecline`, `SendNumber`, `SendStyle`, `SendSlideMenu`, `Dispose`.
