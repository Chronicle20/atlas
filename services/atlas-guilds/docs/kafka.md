# Kafka

## Topics Consumed

### COMMAND_TOPIC_GUILD

Guild command topic.

**Message Types**

| Type | Body | Description |
|------|------|-------------|
| `REQUEST_CREATE` | `RequestCreateBody` | Request guild creation |
| `CREATION_AGREEMENT` | `CreationAgreementBody` | Respond to creation agreement |
| `CHANGE_EMBLEM` | `ChangeEmblemBody` | Change guild emblem |
| `CHANGE_NOTICE` | `ChangeNoticeBody` | Change guild notice |
| `LEAVE` | `LeaveBody` | Leave or expel from guild |
| `REQUEST_INVITE` | `RequestInviteBody` | Request to invite a character |
| `CHANGE_TITLES` | `ChangeTitlesBody` | Change guild titles |
| `CHANGE_MEMBER_TITLE` | `ChangeMemberTitleBody` | Change a member's title |
| `REQUEST_DISBAND` | `RequestDisbandBody` | Request guild disbanding |
| `REQUEST_CAPACITY_INCREASE` | `RequestCapacityIncreaseBody` | Request capacity increase |

**Required Headers**
- Tenant header
- Span header

### COMMAND_TOPIC_GUILD_THREAD

Thread command topic.

**Message Types**

| Type | Body | Description |
|------|------|-------------|
| `CREATE` | `CreateCommandBody` | Create a thread |
| `UPDATE` | `UpdateCommandBody` | Update a thread |
| `DELETE` | `DeleteCommandBody` | Delete a thread |
| `ADD_REPLY` | `AddReplyCommandBody` | Add a reply |
| `DELETE_REPLY` | `DeleteReplyCommandBody` | Delete a reply |

**Required Headers**
- Tenant header
- Span header

### EVENT_TOPIC_CHARACTER_STATUS

Character status event topic.

**Message Types**

| Type | Body | Description |
|------|------|-------------|
| `DELETED` | `StatusEventDeletedBody` | Character deleted |
| `LOGIN` | `StatusEventLoginBody` | Character logged in |
| `LOGOUT` | `StatusEventLogoutBody` | Character logged out |

**Required Headers**
- Tenant header
- Span header

### EVENT_TOPIC_INVITE_STATUS

Invite status event topic.

**Message Types**

| Type | Body | Description |
|------|------|-------------|
| `ACCEPTED` | `AcceptedEventBody` | Guild invite accepted |

**Required Headers**
- Tenant header
- Span header

---

## Topics Produced

### EVENT_TOPIC_GUILD_STATUS

Guild status event topic.

**Message Types**

| Type | Body | Description |
|------|------|-------------|
| `REQUEST_AGREEMENT` | `StatusEventRequestAgreementBody` | Request creation agreement |
| `CREATED` | `StatusEventCreatedBody` | Guild created |
| `DISBANDED` | `StatusEventDisbandedBody` | Guild disbanded |
| `EMBLEM_UPDATED` | `StatusEventEmblemUpdatedBody` | Emblem changed |
| `MEMBER_STATUS_UPDATED` | `StatusEventMemberStatusUpdatedBody` | Member online status changed |
| `MEMBER_TITLE_UPDATED` | `StatusEventMemberTitleUpdatedBody` | Member title changed |
| `MEMBER_LEFT` | `StatusEventMemberLeftBody` | Member left guild |
| `MEMBER_JOINED` | `StatusEventMemberJoinedBody` | Member joined guild |
| `NOTICE_UPDATED` | `StatusEventNoticeUpdatedBody` | Notice changed |
| `CAPACITY_UPDATED` | `StatusEventCapacityUpdatedBody` | Capacity changed |
| `TITLES_UPDATED` | `StatusEventTitlesUpdatedBody` | Titles changed |
| `ERROR` | `StatusEventErrorBody` | Operation error |

**Ordering**
- Keyed by guild ID or character ID

### EVENT_TOPIC_GUILD_THREAD_STATUS

Thread status event topic.

**Message Types**

| Type | Body | Description |
|------|------|-------------|
| `CREATED` | `CreatedStatusEventBody` | Thread created |
| `UPDATED` | `UpdatedStatusEventBody` | Thread updated |
| `DELETED` | `DeletedStatusEventBody` | Thread deleted |
| `REPLY_ADDED` | `ReplyAddedStatusEventBody` | Reply added |
| `REPLY_DELETED` | `ReplyDeletedStatusEventBody` | Reply deleted |

**Ordering**
- Keyed by guild ID

### COMMAND_TOPIC_INVITE

Invite command topic.

**Message Types**

| Type | Body | Description |
|------|------|-------------|
| `CREATE` | `createCommandBody` | Create a guild invite |

**Ordering**
- Keyed by reference ID

---

## Message Types

### Guild Command Bodies

```go
type RequestCreateBody struct {
    WorldId   byte
    ChannelId byte
    MapId     uint32
    Name      string
}

type CreationAgreementBody struct {
    Agreed bool
}

type ChangeEmblemBody struct {
    GuildId             uint32
    Logo                uint16
    LogoColor           byte
    LogoBackground      uint16
    LogoBackgroundColor byte
}

type ChangeNoticeBody struct {
    GuildId uint32
    Notice  string
}

type LeaveBody struct {
    GuildId uint32
    Force   bool
}

type RequestInviteBody struct {
    GuildId  uint32
    TargetId uint32
}

type ChangeTitlesBody struct {
    GuildId uint32
    Titles  []string
}

type ChangeMemberTitleBody struct {
    GuildId  uint32
    TargetId uint32
    Title    byte
}

type RequestDisbandBody struct {
    WorldId   byte
    ChannelId byte
}

type RequestCapacityIncreaseBody struct {
    WorldId   byte
    ChannelId byte
}
```

### Guild Status Event Bodies

```go
type StatusEventRequestAgreementBody struct {
    ActorId      uint32
    ProposedName string
}

type StatusEventCreatedBody struct {}

type StatusEventDisbandedBody struct {
    Members []uint32
}

type StatusEventEmblemUpdatedBody struct {
    Logo                uint16
    LogoColor           byte
    LogoBackground      uint16
    LogoBackgroundColor byte
}

type StatusEventMemberStatusUpdatedBody struct {
    CharacterId uint32
    Online      bool
}

type StatusEventMemberTitleUpdatedBody struct {
    CharacterId uint32
    Title       byte
}

type StatusEventMemberLeftBody struct {
    CharacterId uint32
    Force       bool
}

type StatusEventMemberJoinedBody struct {
    CharacterId   uint32
    Name          string
    JobId         uint16
    Level         byte
    Title         byte
    Online        bool
    AllianceTitle byte
}

type StatusEventNoticeUpdatedBody struct {
    Notice string
}

type StatusEventCapacityUpdatedBody struct {
    Capacity uint32
}

type StatusEventTitlesUpdatedBody struct {
    GuildId uint32
    Titles  []string
}

type StatusEventErrorBody struct {
    ActorId uint32
    Error   string
}
```

### Thread Command Bodies

```go
type CreateCommandBody struct {
    Notice     bool
    Title      string
    Message    string
    EmoticonId uint32
}

type UpdateCommandBody struct {
    ThreadId   uint32
    Notice     bool
    Title      string
    Message    string
    EmoticonId uint32
}

type DeleteCommandBody struct {
    ThreadId uint32
}

type AddReplyCommandBody struct {
    ThreadId uint32
    Message  string
}

type DeleteReplyCommandBody struct {
    ThreadId uint32
    ReplyId  uint32
}
```

### Thread Status Event Bodies

```go
type CreatedStatusEventBody struct {}

type UpdatedStatusEventBody struct {}

type DeletedStatusEventBody struct {}

type ReplyAddedStatusEventBody struct {
    ReplyId uint32
}

type ReplyDeletedStatusEventBody struct {
    ReplyId uint32
}
```

---

## Transaction Semantics

- Guild commands include a `transactionId` field for correlation
- Status events include `transactionId` for response correlation
- Thread commands do not include transaction IDs
