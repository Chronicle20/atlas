# Atlas Guilds Service

Microservice for managing guild operations including creation, membership, threads, titles, and inter-service coordination.

## Overview

The `atlas-guilds` service handles all guild-related functionality:
- Guild creation with party coordination
- Guild membership management
- Guild emblem and notice customization
- Guild title/rank configuration
- Guild bulletin board threads and replies
- Member online status tracking
- Guild disbanding and capacity management

## REST Endpoints

### Guild Endpoints

| Method | Path | Description | Query Parameters |
|--------|------|-------------|------------------|
| GET | `/guilds` | List all guilds | `filter[members.id]={memberId}` - Filter by member |
| GET | `/guilds/{guildId}` | Get guild by ID | - |

### Thread Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/guilds/{guildId}/threads` | List guild threads |
| GET | `/guilds/{guildId}/threads/{threadId}` | Get thread by ID |

## Kafka Commands (Consumed)

### Guild Commands

**Topic:** `COMMAND_TOPIC_GUILD`

| Command Type | Description | Body Fields |
|--------------|-------------|-------------|
| `REQUEST_CREATE` | Request guild creation | `worldId`, `channelId`, `mapId`, `name` |
| `CREATION_AGREEMENT` | Respond to creation agreement | `agreed` |
| `CHANGE_EMBLEM` | Update guild emblem | `guildId`, `logo`, `logoColor`, `logoBackground`, `logoBackgroundColor` |
| `CHANGE_NOTICE` | Update guild notice | `guildId`, `notice` |
| `CHANGE_TITLES` | Update guild titles | `guildId`, `titles[]` |
| `CHANGE_MEMBER_TITLE` | Update member's title | `guildId`, `targetId`, `title` |
| `REQUEST_INVITE` | Invite player to guild | `guildId`, `targetId` |
| `LEAVE` | Leave guild | `guildId`, `force` |
| `REQUEST_DISBAND` | Disband guild | `worldId`, `channelId` |
| `REQUEST_CAPACITY_INCREASE` | Increase guild capacity | `worldId`, `channelId` |

### Thread Commands

**Topic:** `COMMAND_TOPIC_GUILD_THREAD`

| Command Type | Description | Body Fields |
|--------------|-------------|-------------|
| `CREATE` | Create thread | `notice`, `title`, `message`, `emoticonId` |
| `UPDATE` | Update thread | `threadId`, `notice`, `title`, `message`, `emoticonId` |
| `DELETE` | Delete thread | `threadId` |
| `ADD_REPLY` | Add reply to thread | `threadId`, `message` |
| `DELETE_REPLY` | Delete reply | `threadId`, `replyId` |

### External Events (Consumed)

**Topic:** `EVENT_TOPIC_CHARACTER_STATUS`

| Event Type | Description | Purpose |
|------------|-------------|---------|
| `LOGIN` | Character logged in | Update member online status |
| `LOGOUT` | Character logged out | Update member online status |
| `CHANNEL_CHANGED` | Character changed channel | - |

**Topic:** `EVENT_TOPIC_INVITE_STATUS`

| Event Type | Description | Purpose |
|------------|-------------|---------|
| `ACCEPTED` | Guild invite accepted | Add member to guild |

## Kafka Events (Produced)

### Guild Status Events

**Topic:** `EVENT_TOPIC_GUILD_STATUS`

| Event Type | Description | Body Fields |
|------------|-------------|-------------|
| `CREATED` | Guild created | - |
| `DISBANDED` | Guild disbanded | `members[]` |
| `EMBLEM_UPDATED` | Emblem changed | `logo`, `logoColor`, `logoBackground`, `logoBackgroundColor` |
| `REQUEST_AGREEMENT` | Agreement requested | `actorId`, `proposedName` |
| `MEMBER_STATUS_UPDATED` | Member online status changed | `characterId`, `online` |
| `MEMBER_TITLE_UPDATED` | Member title changed | `characterId`, `title` |
| `MEMBER_LEFT` | Member left guild | `characterId`, `force` |
| `MEMBER_JOINED` | Member joined guild | `characterId`, `name`, `jobId`, `level`, `title`, `online`, `allianceTitle` |
| `NOTICE_UPDATED` | Notice changed | `notice` |
| `CAPACITY_UPDATED` | Capacity increased | `capacity` |
| `TITLES_UPDATED` | Titles changed | `guildId`, `titles[]` |
| `ERROR` | Operation error | `actorId`, `error` |

### Thread Status Events

**Topic:** `EVENT_TOPIC_GUILD_THREAD_STATUS`

| Event Type | Description | Body Fields |
|------------|-------------|-------------|
| `CREATED` | Thread created | - |
| `UPDATED` | Thread updated | - |
| `DELETED` | Thread deleted | - |
| `REPLY_ADDED` | Reply added | `replyId` |
| `REPLY_DELETED` | Reply deleted | `replyId` |

## Domain Models

### Guild
Core guild entity with:
- Identity: `tenantId`, `id`, `worldId`, `name`
- Leadership: `leaderId`
- Configuration: `capacity`, `notice`, `points`
- Emblem: `logo`, `logoColor`, `logoBackground`, `logoBackgroundColor`
- Collections: `members[]`, `titles[]`

### Member
Guild member with:
- Identity: `tenantId`, `guildId`, `characterId`, `name`
- Status: `online`, `level`, `jobId`
- Ranks: `title`, `allianceTitle`

### Title
Guild rank definition with:
- Identity: `tenantId`, `id`, `guildId`
- Configuration: `name`, `index`

### Thread
Guild bulletin board thread with:
- Identity: `tenantId`, `guildId`, `id`
- Content: `title`, `message`, `emoticonId`
- Metadata: `posterId`, `notice`, `createdAt`
- Collections: `replies[]`

### Reply
Thread reply with:
- Identity: `id`
- Content: `message`
- Metadata: `posterId`, `createdAt`

## External Dependencies

### REST Clients
- **atlas-character** - Character information retrieval
- **atlas-party** - Party membership validation

### Kafka Topics (External)
- `EVENT_TOPIC_CHARACTER_STATUS` - Character login/logout events
- `EVENT_TOPIC_INVITE_STATUS` - Invite acceptance events

## Configuration

### Environment Variables

| Variable | Description |
|----------|-------------|
| `COMMAND_TOPIC_GUILD` | Guild command topic |
| `COMMAND_TOPIC_GUILD_THREAD` | Thread command topic |
| `EVENT_TOPIC_GUILD_STATUS` | Guild status event topic |
| `EVENT_TOPIC_GUILD_THREAD_STATUS` | Thread status event topic |
| `EVENT_TOPIC_CHARACTER_STATUS` | Character status event topic |
| `EVENT_TOPIC_INVITE_STATUS` | Invite status event topic |

## Guild Creation Flow

1. Party leader sends `REQUEST_CREATE` command
2. Service validates:
   - Name not in use
   - Name is valid
   - Requester is party leader
   - Party has minimum members
   - No party members already in guild
3. Service sends `REQUEST_AGREEMENT` event to all party members
4. Each member responds with `CREATION_AGREEMENT` command
5. Once all agree, guild is created with default titles
6. Leader added with title 1, other members with title 2
7. `CREATED` event emitted
