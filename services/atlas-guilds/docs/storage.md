# Storage

## Tables

### guilds

Stores guild records.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `tenant_id` | uuid | NOT NULL | Tenant identifier |
| `id` | uint32 | PRIMARY KEY, AUTO INCREMENT | Guild identifier |
| `world_id` | byte | NOT NULL | World identifier |
| `name` | string | NOT NULL | Guild name |
| `notice` | string | NOT NULL | Guild notice message |
| `points` | uint32 | NOT NULL | Guild points |
| `capacity` | uint32 | NOT NULL, DEFAULT 30 | Maximum member capacity |
| `logo` | uint16 | NOT NULL, DEFAULT 0 | Emblem logo identifier |
| `logo_color` | byte | NOT NULL, DEFAULT 0 | Emblem logo color |
| `logo_background` | uint16 | NOT NULL, DEFAULT 0 | Emblem background identifier |
| `logo_background_color` | byte | NOT NULL, DEFAULT 0 | Emblem background color |
| `alliance_id` | uint32 | NOT NULL, DEFAULT 0 | Alliance identifier |
| `leader_id` | uint32 | NOT NULL | Leader character identifier |

### members

Stores guild member records.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `tenant_id` | uuid | NOT NULL | Tenant identifier |
| `character_id` | uint32 | PRIMARY KEY | Character identifier |
| `guild_id` | uint32 | NOT NULL | Guild identifier |
| `name` | string | NOT NULL | Character name |
| `job_id` | uint16 | NOT NULL, DEFAULT 0 | Character job identifier |
| `level` | byte | NOT NULL | Character level |
| `title` | byte | NOT NULL, DEFAULT 5 | Guild title rank |
| `online` | bool | NOT NULL, DEFAULT false | Online status |
| `alliance_title` | byte | NOT NULL, DEFAULT 5 | Alliance title rank |

### titles

Stores guild title records.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `tenant_id` | uuid | NOT NULL | Tenant identifier |
| `id` | uuid | DEFAULT uuid_generate_v4() | Title identifier |
| `guild_id` | uint32 | | Guild identifier |
| `name` | string | | Title name |
| `index` | byte | | Title rank index |

### characters

Stores character-to-guild mapping.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `tenant_id` | uuid | NOT NULL | Tenant identifier |
| `character_id` | uint32 | PRIMARY KEY | Character identifier |
| `guild_id` | uint32 | NOT NULL | Guild identifier |

### threads

Stores guild bulletin board threads.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `tenant_id` | uuid | NOT NULL | Tenant identifier |
| `guild_id` | uint32 | NOT NULL | Guild identifier |
| `id` | uint32 | PRIMARY KEY, AUTO INCREMENT | Thread identifier |
| `poster_id` | uint32 | NOT NULL | Author character identifier |
| `title` | string | NOT NULL | Thread title |
| `message` | string | NOT NULL | Thread message content |
| `emoticon_id` | uint32 | NOT NULL | Associated emoticon identifier |
| `notice` | bool | NOT NULL | Whether thread is pinned |
| `created_at` | timestamp | NOT NULL | Creation timestamp |

### replies

Stores thread reply records.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `tenant_id` | uuid | NOT NULL | Tenant identifier |
| `thread_id` | uint32 | NOT NULL | Thread identifier |
| `id` | uint32 | PRIMARY KEY, AUTO INCREMENT | Reply identifier |
| `poster_id` | uint32 | NOT NULL | Author character identifier |
| `message` | string | NOT NULL | Reply message content |
| `created_at` | timestamp | NOT NULL | Creation timestamp |

---

## Relationships

| Parent Table | Child Table | Foreign Key | Relationship |
|--------------|-------------|-------------|--------------|
| guilds | members | members.guild_id | One-to-many |
| guilds | titles | titles.guild_id | One-to-many |
| threads | replies | replies.thread_id | One-to-many |

---

## Indexes

GORM AutoMigrate creates indexes on primary keys.

---

## Migration Rules

- Migrations run via GORM AutoMigrate on service startup
- Tables: guild, title, member, character, thread, reply
- Schema changes are additive only
