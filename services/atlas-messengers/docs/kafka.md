# Kafka

## Topics Consumed

### COMMAND_TOPIC_MESSENGER

Messenger commands.

| Type | Description |
|------|-------------|
| CREATE | Create a new messenger |
| JOIN | Join an existing messenger |
| LEAVE | Leave a messenger |
| REQUEST_INVITE | Request to invite another character |

### EVENT_TOPIC_CHARACTER_STATUS

Character status events.

| Type | Description |
|------|-------------|
| LOGIN | Character logged in |
| LOGOUT | Character logged out |
| CHANNEL_CHANGED | Character changed channel |

### EVENT_TOPIC_INVITE_STATUS

Invite status events.

| Type | Description |
|------|-------------|
| ACCEPTED | Messenger invite was accepted |

---

## Topics Produced

### EVENT_TOPIC_MESSENGER_STATUS

Messenger status events.

| Type | Description |
|------|-------------|
| CREATED | Messenger was created |
| JOINED | Character joined messenger |
| LEFT | Character left messenger |
| ERROR | Error occurred |

### EVENT_TOPIC_MESSENGER_MEMBER_STATUS

Member status events.

| Type | Description |
|------|-------------|
| LOGIN | Member logged in |
| LOGOUT | Member logged out |

### COMMAND_TOPIC_INVITE

Invite commands.

| Type | Description |
|------|-------------|
| CREATE | Create an invitation |

---

## Message Types

### CommandEvent (Messenger)

```go
type CommandEvent[E any] struct {
    TransactionID uuid.UUID `json:"transactionId"`
    ActorId       uint32    `json:"actorId"`
    Type          string    `json:"type"`
    Body          E         `json:"body"`
}
```

#### CreateCommandBody

Empty body.

#### JoinCommandBody

| Field | Type | Description |
|-------|------|-------------|
| messengerId | uint32 | Messenger to join |

#### LeaveCommandBody

| Field | Type | Description |
|-------|------|-------------|
| messengerId | uint32 | Messenger to leave |

#### RequestInviteBody

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character to invite |

---

### StatusEvent (Messenger)

```go
type StatusEvent[E any] struct {
    TransactionID uuid.UUID `json:"transactionId"`
    ActorId       uint32    `json:"actorId"`
    WorldId       world.Id  `json:"worldId"`
    MessengerId   uint32    `json:"messengerId"`
    Type          string    `json:"type"`
    Body          E         `json:"body"`
}
```

#### CreatedEventBody

Empty body.

#### JoinedEventBody

| Field | Type | Description |
|-------|------|-------------|
| slot | byte | Assigned slot |

#### LeftEventBody

| Field | Type | Description |
|-------|------|-------------|
| slot | byte | Vacated slot |

#### ErrorEventBody

| Field | Type | Description |
|-------|------|-------------|
| type | string | Error type code |
| characterName | string | Related character name |

---

### MemberStatusEvent

```go
type MemberStatusEvent[E any] struct {
    TransactionID uuid.UUID `json:"transactionId"`
    WorldId       world.Id  `json:"worldId"`
    MessengerId   uint32    `json:"messengerId"`
    CharacterId   uint32    `json:"characterId"`
    Type          string    `json:"type"`
    Body          E         `json:"body"`
}
```

#### MemberLoginEventBody

Empty body.

#### MemberLogoutEventBody

Empty body.

---

### StatusEvent (Character)

```go
type StatusEvent[E any] struct {
    TransactionID uuid.UUID `json:"transactionId"`
    WorldId       world.Id  `json:"worldId"`
    CharacterId   uint32    `json:"characterId"`
    Type          string    `json:"type"`
    Body          E         `json:"body"`
}
```

#### StatusEventLoginBody

| Field | Type | Description |
|-------|------|-------------|
| channelId | channel.Id | Channel ID |
| mapId | map.Id | Map ID |

#### StatusEventLogoutBody

| Field | Type | Description |
|-------|------|-------------|
| channelId | channel.Id | Channel ID |
| mapId | map.Id | Map ID |

#### StatusEventChannelChangedBody

| Field | Type | Description |
|-------|------|-------------|
| channelId | channel.Id | New channel ID |
| oldChannelId | channel.Id | Previous channel ID |
| mapId | map.Id | Map ID |

---

### CommandEvent (Invite)

```go
type CommandEvent[E any] struct {
    TransactionID uuid.UUID `json:"transactionId"`
    WorldId       world.Id  `json:"worldId"`
    InviteType    string    `json:"inviteType"`
    Type          string    `json:"type"`
    Body          E         `json:"body"`
}
```

#### CreateCommandBody

| Field | Type | Description |
|-------|------|-------------|
| originatorId | uint32 | Inviting character ID |
| targetId | uint32 | Invited character ID |
| referenceId | uint32 | Messenger ID |

---

### StatusEvent (Invite)

```go
type StatusEvent[E any] struct {
    TransactionID uuid.UUID `json:"transactionId"`
    WorldId       world.Id  `json:"worldId"`
    InviteType    string    `json:"inviteType"`
    ReferenceId   uint32    `json:"referenceId"`
    Type          string    `json:"type"`
    Body          E         `json:"body"`
}
```

#### AcceptedEventBody

| Field | Type | Description |
|-------|------|-------------|
| originatorId | uint32 | Inviting character ID |
| targetId | uint32 | Invited character ID |

---

## Transaction Semantics

- All messages include transactionId for correlation
- Messages are keyed by messengerId or characterId for partition ordering
- Required headers: SpanHeader, TenantHeader
