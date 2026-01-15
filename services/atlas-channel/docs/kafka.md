# Kafka Documentation

## Topics Consumed

### EVENT_TOPIC_ACCOUNT_STATUS
- Direction: Event
- Message Type: `StatusEvent`
- Purpose: Receives account login/logout status events to maintain account registry

### EVENT_TOPIC_ACCOUNT_SESSION_STATUS
- Direction: Event
- Message Type: `StatusEvent[StateChangedEventBody]`, `StatusEvent[ErrorStatusEventBody]`
- Purpose: Receives session state changes for player login, channel changes, and error handling

### EVENT_TOPIC_CHARACTER_STATUS
- Direction: Event
- Message Type: `StatusEvent[StatusEventStatChangedBody]`, `StatusEvent[StatusEventMapChangedBody]`, `StatusEvent[ExperienceChangedStatusEventBody]`, `StatusEvent[FameChangedStatusEventBody]`, `StatusEvent[MesoChangedStatusEventBody]`, `StatusEvent[LevelChangedStatusEventBody]`
- Purpose: Receives character stat, map, experience, fame, meso, and level change events

### EVENT_TOPIC_MAP_STATUS
- Direction: Event
- Message Type: `StatusEvent[CharacterEnter]`, `StatusEvent[CharacterExit]`
- Purpose: Receives character map entry and exit events for spawning/despawning

### EVENT_TOPIC_SESSION_STATUS
- Direction: Event
- Message Type: `StatusEvent`
- Purpose: Receives session created/destroyed events

### EVENT_TOPIC_ASSET_STATUS
- Direction: Event
- Message Type: Asset status events
- Purpose: Receives inventory asset change events

### EVENT_TOPIC_COMPARTMENT_STATUS
- Direction: Event
- Message Type: Compartment status events
- Purpose: Receives inventory compartment change events

### EVENT_TOPIC_COMPARTMENT_TRANSFER_STATUS
- Direction: Event
- Message Type: Transfer status events
- Purpose: Receives compartment transfer completion events

### EVENT_TOPIC_CHAIR_STATUS
- Direction: Event
- Message Type: Chair status events
- Purpose: Receives chair sit/stand events

### EVENT_TOPIC_CHALKBOARD_STATUS
- Direction: Event
- Message Type: Chalkboard status events
- Purpose: Receives chalkboard update events

### EVENT_TOPIC_CHARACTER_CHAT
- Direction: Event
- Message Type: Chat events
- Purpose: Receives character chat messages for broadcast

### EVENT_TOPIC_DROP_STATUS
- Direction: Event
- Message Type: Drop status events
- Purpose: Receives drop spawn/pickup events

### EVENT_TOPIC_MONSTER_STATUS
- Direction: Event
- Message Type: Monster status events
- Purpose: Receives monster spawn/death/damage events

### EVENT_TOPIC_FAME_STATUS
- Direction: Event
- Message Type: Fame status events
- Purpose: Receives fame change events

### EVENT_TOPIC_CONSUMABLE_STATUS
- Direction: Event
- Message Type: Consumable events
- Purpose: Receives consumable item use events

### EVENT_TOPIC_EXPRESSION
- Direction: Event
- Message Type: `Event`
- Purpose: Receives character expression (emote) events

### EVENT_TOPIC_CASH_SHOP_STATUS
- Direction: Event
- Message Type: Cash shop status events
- Purpose: Receives cash shop operation results

### EVENT_TOPIC_NOTE_STATUS
- Direction: Event
- Message Type: Note status events
- Purpose: Receives note send/receive events

### EVENT_TOPIC_TRANSPORT_STATUS
- Direction: Event
- Message Type: Transport status events
- Purpose: Receives transport route state changes

### EVENT_TOPIC_STORAGE_STATUS
- Direction: Event
- Message Type: Storage status events
- Purpose: Receives storage operation results

---

## Topics Produced

### COMMAND_TOPIC_ACCOUNT_SESSION
- Direction: Command
- Message Type: `Command[ProgressStateCommandBody]`, `Command[LogoutCommandBody]`
- Purpose: Issues session state progression and logout commands

### COMMAND_TOPIC_CHARACTER
- Direction: Command
- Message Type: `Command[RequestDistributeApCommandBody]`, `Command[RequestDistributeSpCommandBody]`, `Command[RequestDropMesoCommandBody]`, `Command[ChangeHPCommandBody]`, `Command[ChangeMPCommandBody]`
- Purpose: Issues character stat distribution, meso drop, and HP/MP change commands

### COMMAND_TOPIC_CHARACTER_MOVEMENT
- Direction: Command
- Message Type: `Command`
- Purpose: Issues character movement commands

### COMMAND_TOPIC_MONSTER_MOVEMENT
- Direction: Command
- Message Type: `Command`
- Purpose: Issues monster movement commands

### COMMAND_TOPIC_PET_MOVEMENT
- Direction: Command
- Message Type: `Command`
- Purpose: Issues pet movement commands

### COMMAND_TOPIC_PORTAL
- Direction: Command
- Message Type: `Command[EnterBody]`
- Purpose: Issues portal entry commands

### COMMAND_TOPIC_EXPRESSION
- Direction: Command
- Message Type: `Command`
- Purpose: Issues character expression commands

### COMMAND_TOPIC_CHARACTER_CHAT
- Direction: Command
- Message Type: Chat commands
- Purpose: Issues character chat commands

### COMMAND_TOPIC_BUDDY_LIST
- Direction: Command
- Message Type: Buddy list commands
- Purpose: Issues buddy list operation commands

### COMMAND_TOPIC_PARTY
- Direction: Command
- Message Type: Party commands
- Purpose: Issues party operation commands

### COMMAND_TOPIC_GUILD
- Direction: Command
- Message Type: Guild commands
- Purpose: Issues guild operation commands

### COMMAND_TOPIC_GUILD_THREAD
- Direction: Command
- Message Type: Guild thread commands
- Purpose: Issues guild BBS thread commands

### COMMAND_TOPIC_MESSENGER
- Direction: Command
- Message Type: Messenger commands
- Purpose: Issues messenger operation commands

### COMMAND_TOPIC_FAME
- Direction: Command
- Message Type: Fame commands
- Purpose: Issues fame change commands

### COMMAND_TOPIC_CHAIR
- Direction: Command
- Message Type: Chair commands
- Purpose: Issues chair sit/stand commands

### COMMAND_TOPIC_CHALKBOARD
- Direction: Command
- Message Type: Chalkboard commands
- Purpose: Issues chalkboard update/close commands

### COMMAND_TOPIC_DROP
- Direction: Command
- Message Type: Drop commands
- Purpose: Issues drop pickup commands

### COMMAND_TOPIC_MONSTER
- Direction: Command
- Message Type: Monster commands
- Purpose: Issues monster damage/death commands

### COMMAND_TOPIC_NPC
- Direction: Command
- Message Type: NPC commands
- Purpose: Issues NPC interaction commands

### COMMAND_TOPIC_NPC_CONVERSATION
- Direction: Command
- Message Type: NPC conversation commands
- Purpose: Issues NPC conversation state commands

### COMMAND_TOPIC_NPC_SHOP
- Direction: Command
- Message Type: NPC shop commands
- Purpose: Issues NPC shop transaction commands

### COMMAND_TOPIC_SKILL
- Direction: Command
- Message Type: Skill commands
- Purpose: Issues skill use commands

### COMMAND_TOPIC_CHARACTER_BUFF
- Direction: Command
- Message Type: Buff commands
- Purpose: Issues buff application/removal commands

### COMMAND_TOPIC_SKILL_MACRO
- Direction: Command
- Message Type: Macro commands
- Purpose: Issues skill macro update commands

### COMMAND_TOPIC_CONSUMABLE
- Direction: Command
- Message Type: Consumable commands
- Purpose: Issues consumable item use commands

### COMMAND_TOPIC_PET
- Direction: Command
- Message Type: Pet commands
- Purpose: Issues pet spawn/command/food commands

### COMMAND_TOPIC_COMPARTMENT
- Direction: Command
- Message Type: Compartment commands
- Purpose: Issues inventory compartment operation commands

### COMMAND_TOPIC_COMPARTMENT_TRANSFER
- Direction: Command
- Message Type: Transfer commands
- Purpose: Issues inventory transfer commands

### COMMAND_TOPIC_CASH_SHOP
- Direction: Command
- Message Type: Cash shop commands
- Purpose: Issues cash shop operation commands

### COMMAND_TOPIC_NOTE
- Direction: Command
- Message Type: Note commands
- Purpose: Issues note send/delete commands

### COMMAND_TOPIC_INVITE
- Direction: Command
- Message Type: Invite commands
- Purpose: Issues invite accept/reject commands

### COMMAND_TOPIC_QUEST_CONVERSATION
- Direction: Command
- Message Type: Quest commands
- Purpose: Issues quest conversation commands

### COMMAND_TOPIC_CHANNEL_STATUS
- Direction: Command
- Message Type: Channel status commands
- Purpose: Issues channel heartbeat and status commands

### COMMAND_TOPIC_SYSTEM_MESSAGE
- Direction: Command
- Message Type: System message commands
- Purpose: Issues system message broadcast commands

### COMMAND_TOPIC_STORAGE
- Direction: Command
- Message Type: Storage commands
- Purpose: Issues storage deposit/withdraw commands

### COMMAND_TOPIC_SAGA
- Direction: Command
- Message Type: Saga commands
- Purpose: Issues saga orchestration commands

---

## Message Types

### StatusEvent
Generic status event envelope with type discriminator.

### Command
Generic command envelope with type discriminator and typed body.

---

## Transaction Semantics

- Consumer group ID follows pattern: `Channel Service - {SERVICE_ID}`
- Consumers start from `LastOffset` for real-time event processing
- Tenant ID passed in Kafka headers for multi-tenant filtering
- Span context passed in headers for distributed tracing
- Messages keyed by character ID for ordering guarantees
