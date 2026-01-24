# Kafka

## Topics Consumed

### COMMAND_TOPIC_PARTY

Party command topic for processing party operations.

| Command Type | Description |
|--------------|-------------|
| CREATE | Create new party with actor as leader |
| JOIN | Actor joins specified party |
| LEAVE | Actor leaves party (force flag determines expel vs leave) |
| CHANGE_LEADER | Transfer leadership to specified character |
| REQUEST_INVITE | Request party invitation for target character |

Consumer Group: `party_command`

### EVENT_TOPIC_CHARACTER_STATUS

Character status events for synchronizing character state.

| Event Type | Description |
|------------|-------------|
| LOGIN | Character logged in |
| LOGOUT | Character logged out |
| CHANNEL_CHANGED | Character changed channel |
| MAP_CHANGED | Character changed map |
| DELETED | Character was deleted |
| LEVEL_CHANGED | Character level changed |
| JOB_CHANGED | Character job changed |

Consumer Group: `character_status_event`

### EVENT_TOPIC_INVITE_STATUS

Invite status events for processing accepted invitations.

| Event Type | Description |
|------------|-------------|
| ACCEPTED | Invitation was accepted (triggers party join) |

Consumer Group: `invite_status_event`

---

## Topics Produced

### EVENT_TOPIC_PARTY_STATUS

Party status events announcing party state changes.

| Event Type | Description |
|------------|-------------|
| CREATED | Party was created |
| JOINED | Character joined party |
| LEFT | Character left party |
| EXPEL | Character was expelled from party |
| DISBAND | Party was disbanded |
| CHANGE_LEADER | Party leadership changed |
| ERROR | Party operation error |

### EVENT_TOPIC_PARTY_MEMBER_STATUS

Party member status events for member state changes.

| Event Type | Description |
|------------|-------------|
| LOGIN | Party member logged in |
| LOGOUT | Party member logged out |
| LEVEL_CHANGED | Party member level changed |
| JOB_CHANGED | Party member job changed |

### COMMAND_TOPIC_INVITE

Invite commands for creating party invitations.

| Command Type | Description |
|--------------|-------------|
| CREATE | Create party invitation |

---

## Message Types

### Party Command

```json
{
  "actorId": uint32,
  "type": string,
  "body": object
}
```

### Party Status Event

```json
{
  "actorId": uint32,
  "worldId": byte,
  "partyId": uint32,
  "type": string,
  "body": object
}
```

### Party Member Status Event

```json
{
  "worldId": byte,
  "partyId": uint32,
  "characterId": uint32,
  "type": string,
  "body": object
}
```

### Character Status Event

```json
{
  "transactionId": uuid,
  "worldId": world.Id,
  "characterId": uint32,
  "type": string,
  "body": object
}
```

### Invite Command

```json
{
  "worldId": byte,
  "inviteType": string,
  "type": string,
  "body": {
    "originatorId": uint32,
    "targetId": uint32,
    "referenceId": uint32
  }
}
```

### Invite Status Event

```json
{
  "worldId": byte,
  "inviteType": string,
  "referenceId": uint32,
  "type": string,
  "body": object
}
```

---

## Transaction Semantics

- Messages are keyed by party ID or character ID for partition ordering
- Tenant context propagated via header parsers
- Span context propagated for distributed tracing
