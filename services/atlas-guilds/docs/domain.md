# Domain

## Guild

### Responsibility

Represents a player guild with membership, emblem, titles, and capacity.

### Core Models

**Model**
- `tenantId` - Tenant identifier
- `id` - Guild identifier
- `worldId` - World identifier
- `name` - Guild name
- `notice` - Guild notice message
- `points` - Guild points
- `capacity` - Maximum member capacity
- `logo` - Emblem logo identifier
- `logoColor` - Emblem logo color
- `logoBackground` - Emblem background identifier
- `logoBackgroundColor` - Emblem background color
- `leaderId` - Leader character identifier
- `members` - Collection of guild members
- `titles` - Collection of guild titles

### Invariants

- Guild name must be unique per world
- Name "Stupid" is invalid
- Party leader must create the guild
- Party must have at least 2 members for guild creation
- All party members must not already be in a guild
- Game administrators cannot create guilds
- Only the guild leader can disband the guild
- Only the guild leader can increase capacity
- Guild cannot exceed member capacity when inviting

### Processors

**guild.Processor**
- Retrieves guilds by ID, name, or member ID
- Handles guild creation requests and agreement coordination
- Creates guilds with leader and default titles
- Updates emblem, notice, capacity, and titles
- Manages member online status and titles
- Processes member leave and join operations
- Handles guild invitation requests
- Processes guild disbanding

---

## Member

### Responsibility

Represents a character's membership in a guild.

### Core Models

**Model**
- `tenantId` - Tenant identifier
- `characterId` - Character identifier
- `guildId` - Guild identifier
- `name` - Character name
- `jobId` - Character job identifier
- `level` - Character level
- `title` - Guild title rank
- `online` - Online status
- `allianceTitle` - Alliance title rank

### Processors

**member.Processor**
- Adds members to guilds
- Removes members from guilds
- Updates member online status
- Updates member title
- Updates character-guild mapping on membership changes

---

## Title

### Responsibility

Represents a named rank within a guild.

### Core Models

**Model**
- `tenantId` - Tenant identifier
- `id` - Title identifier
- `guildId` - Guild identifier
- `name` - Title name
- `index` - Title rank index

### Processors

**title.Processor**
- Creates default titles for new guilds
- Replaces all titles for a guild
- Clears titles on guild disbanding

---

## Character

### Responsibility

Tracks character-to-guild mapping.

### Core Models

**Model**
- `tenantId` - Tenant identifier
- `characterId` - Character identifier
- `guildId` - Guild identifier

### Processors

**character.Processor**
- Gets character by ID
- Sets guild assignment for character

---

## Thread

### Responsibility

Represents a bulletin board thread within a guild.

### Core Models

**Model**
- `tenantId` - Tenant identifier
- `guildId` - Guild identifier
- `id` - Thread identifier
- `posterId` - Author character identifier
- `title` - Thread title
- `message` - Thread message content
- `emoticonId` - Associated emoticon identifier
- `notice` - Whether thread is pinned as notice
- `createdAt` - Creation timestamp
- `replies` - Collection of thread replies

### Processors

**thread.Processor**
- Retrieves all threads for a guild
- Retrieves thread by ID
- Creates threads
- Updates threads
- Deletes threads and associated replies
- Adds replies to threads
- Deletes replies from threads

---

## Reply

### Responsibility

Represents a reply to a bulletin board thread.

### Core Models

**Model**
- `id` - Reply identifier
- `posterId` - Author character identifier
- `message` - Reply message content
- `createdAt` - Creation timestamp

### Processors

**reply.Processor**
- Adds replies to threads
- Deletes replies from threads

---

## Coordinator

### Responsibility

Manages in-memory guild creation agreement state.

### Core Models

**Model**
- `tenant` - Tenant model
- `worldId` - World identifier
- `channelId` - Channel identifier
- `leaderId` - Leader character identifier
- `name` - Proposed guild name
- `requests` - List of character IDs requested to agree
- `responses` - Map of character responses
- `age` - Creation timestamp

### Processors

**coordinator.Registry**
- Initiates guild creation coordination
- Records agreement responses
- Tracks coordination timeout

---

## Invite

### Responsibility

Produces guild invitation commands.

### Processors

**invite.Processor**
- Creates guild invitations by emitting invite commands
