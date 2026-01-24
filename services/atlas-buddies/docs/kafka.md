# Kafka

## Topics Consumed

### EVENT_TOPIC_CHARACTER_STATUS
Character status events from external character service.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| CREATED | CreatedStatusEventBody | Character created - initializes buddy list with capacity 30 |
| DELETED | DeletedStatusEventBody | Character deleted - removes buddy list and cleans up buddy references |
| LOGIN | LoginStatusEventBody | Character logged in - updates channel across buddy lists |
| LOGOUT | LogoutStatusEventBody | Character logged out - sets channel to -1 across buddy lists |
| CHANNEL_CHANGED | ChannelChangedStatusEventBody | Character changed channel - updates channel across buddy lists |

### EVENT_TOPIC_INVITE_STATUS
Invite status events from external invite service.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| ACCEPTED | AcceptedEventBody | Buddy invite accepted - creates mutual buddy relationship |
| REJECTED | RejectedEventBody | Buddy invite rejected - removes pending buddy entry |

Filtered by `inviteType: "BUDDY"`.

### EVENT_TOPIC_CASH_SHOP_STATUS
Cash shop status events from external cash shop service.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| CHARACTER_ENTER | MovementBody | Character entered cash shop - sets inShop to true across buddy lists |
| CHARACTER_EXIT | MovementBody | Character exited cash shop - sets inShop to false across buddy lists |

### COMMAND_TOPIC_BUDDY_LIST
Buddy list commands.

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| CREATE | CreateCommandBody | Creates a new buddy list with specified capacity |
| REQUEST_ADD | RequestAddBuddyCommandBody | Requests to add a buddy |
| REQUEST_DELETE | RequestDeleteBuddyCommandBody | Requests to remove a buddy |
| INCREASE_CAPACITY | IncreaseCapacityCommandBody | Increases buddy list capacity |

---

## Topics Produced

### EVENT_TOPIC_BUDDY_LIST_STATUS
Buddy list status events.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| BUDDY_ADDED | BuddyAddedStatusEventBody | Buddy successfully added |
| BUDDY_REMOVED | BuddyRemovedStatusEventBody | Buddy successfully removed |
| BUDDY_UPDATED | BuddyUpdatedStatusEventBody | Buddy information updated |
| BUDDY_CHANNEL_CHANGE | BuddyChannelChangeStatusEventBody | Buddy channel changed |
| CAPACITY_CHANGE | BuddyCapacityChangeStatusEventBody | Buddy list capacity changed |
| ERROR | ErrorStatusEventBody | Operation failed |

### COMMAND_TOPIC_INVITE
Invite commands to external invite service.

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| CREATE | CreateCommandBody | Creates a buddy invite |
| REJECT | RejectCommandBody | Rejects a buddy invite |

All invite commands use `inviteType: "BUDDY"`.

---

## Message Types

### Command Messages

#### Command[E]
```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 12345,
  "type": "COMMAND_TYPE",
  "body": {}
}
```

#### CreateCommandBody
```json
{
  "capacity": 50
}
```

#### RequestAddBuddyCommandBody
```json
{
  "characterId": 67890,
  "characterName": "BuddyName",
  "group": "Friends"
}
```

#### RequestDeleteBuddyCommandBody
```json
{
  "characterId": 67890
}
```

#### IncreaseCapacityCommandBody
```json
{
  "newCapacity": 100
}
```

### Status Event Messages

#### StatusEvent[E]
```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "EVENT_TYPE",
  "body": {}
}
```

#### BuddyAddedStatusEventBody
```json
{
  "characterId": 67890,
  "group": "Friends",
  "characterName": "BuddyName",
  "channelId": 1
}
```

#### BuddyRemovedStatusEventBody
```json
{
  "characterId": 67890
}
```

#### BuddyUpdatedStatusEventBody
```json
{
  "characterId": 67890,
  "group": "Friends",
  "characterName": "BuddyName",
  "channelId": 1,
  "inShop": false
}
```

#### BuddyChannelChangeStatusEventBody
```json
{
  "characterId": 67890,
  "channelId": 2
}
```

#### BuddyCapacityChangeStatusEventBody
```json
{
  "capacity": 100,
  "transactionId": "uuid"
}
```

#### ErrorStatusEventBody
```json
{
  "error": "ERROR_CODE"
}
```

Error codes:
- `BUDDY_LIST_FULL`: Requester's buddy list is at capacity
- `OTHER_BUDDY_LIST_FULL`: Target's buddy list is at capacity
- `ALREADY_BUDDY`: Characters are already buddies
- `CANNOT_BUDDY_GM`: Attempted to buddy a game master
- `CHARACTER_NOT_FOUND`: Character not found
- `INVALID_CAPACITY`: New capacity not greater than current
- `UNKNOWN_ERROR`: Unexpected error

---

## Transaction Semantics

- Buddy list commands include optional `transactionId` for saga coordination
- Status events include `transactionId` when command included one
- All database operations within a single command are wrapped in a transaction
- Events are buffered and emitted after successful transaction commit
