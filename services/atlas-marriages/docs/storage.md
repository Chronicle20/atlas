# Marriage Storage

## Tables

### marriages

Stores marriage records.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uint32 | PRIMARY KEY, AUTO INCREMENT | Marriage identifier |
| character_id1 | uint32 | INDEX, NOT NULL | First character (proposer) |
| character_id2 | uint32 | INDEX, NOT NULL | Second character (target) |
| status | uint8 | INDEX, NOT NULL | Marriage status |
| proposed_at | timestamp | NOT NULL | Proposal timestamp |
| engaged_at | timestamp | INDEX, NULLABLE | Engagement timestamp |
| married_at | timestamp | INDEX, NULLABLE | Marriage timestamp |
| divorced_at | timestamp | INDEX, NULLABLE | Divorce timestamp |
| tenant_id | uuid | INDEX, NOT NULL | Tenant identifier |
| created_at | timestamp | NOT NULL | Record creation timestamp |
| updated_at | timestamp | NOT NULL | Record update timestamp |

### proposals

Stores proposal records.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uint32 | PRIMARY KEY, AUTO INCREMENT | Proposal identifier |
| proposer_id | uint32 | INDEX, NOT NULL | Proposing character |
| target_id | uint32 | INDEX, NOT NULL | Target character |
| status | uint8 | INDEX, NOT NULL | Proposal status |
| proposed_at | timestamp | NOT NULL | Proposal timestamp |
| responded_at | timestamp | INDEX, NULLABLE | Response timestamp |
| expires_at | timestamp | INDEX, NOT NULL | Expiration timestamp |
| rejection_count | uint32 | DEFAULT 0 | Number of rejections |
| cooldown_until | timestamp | INDEX, NULLABLE | Cooldown end timestamp |
| tenant_id | uuid | INDEX, NOT NULL | Tenant identifier |
| created_at | timestamp | NOT NULL | Record creation timestamp |
| updated_at | timestamp | NOT NULL | Record update timestamp |

### ceremonies

Stores ceremony records.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uint32 | PRIMARY KEY, AUTO INCREMENT | Ceremony identifier |
| marriage_id | uint32 | INDEX, NOT NULL | Associated marriage |
| character_id1 | uint32 | INDEX, NOT NULL | First partner |
| character_id2 | uint32 | INDEX, NOT NULL | Second partner |
| status | uint8 | INDEX, NOT NULL | Ceremony status |
| scheduled_at | timestamp | NOT NULL | Scheduled timestamp |
| started_at | timestamp | INDEX, NULLABLE | Start timestamp |
| completed_at | timestamp | INDEX, NULLABLE | Completion timestamp |
| cancelled_at | timestamp | INDEX, NULLABLE | Cancellation timestamp |
| postponed_at | timestamp | INDEX, NULLABLE | Postponement timestamp |
| invitees | text | | JSON array of character IDs |
| tenant_id | uuid | INDEX, NOT NULL | Tenant identifier |
| created_at | timestamp | NOT NULL | Record creation timestamp |
| updated_at | timestamp | NOT NULL | Record update timestamp |

## Relationships

| Table | Relationship | Related Table | Description |
|-------|--------------|---------------|-------------|
| ceremonies | belongs to | marriages | Ceremony references marriage via marriage_id |

## Indexes

### marriages

| Index | Columns | Purpose |
|-------|---------|---------|
| idx_marriages_character_id1 | character_id1 | Query marriages by first character |
| idx_marriages_character_id2 | character_id2 | Query marriages by second character |
| idx_marriages_status | status | Filter by marriage status |
| idx_marriages_engaged_at | engaged_at | Query by engagement date |
| idx_marriages_married_at | married_at | Query by marriage date |
| idx_marriages_divorced_at | divorced_at | Query by divorce date |
| idx_marriages_tenant_id | tenant_id | Tenant isolation |

### proposals

| Index | Columns | Purpose |
|-------|---------|---------|
| idx_proposals_proposer_id | proposer_id | Query proposals by proposer |
| idx_proposals_target_id | target_id | Query proposals by target |
| idx_proposals_status | status | Filter by proposal status |
| idx_proposals_responded_at | responded_at | Query by response date |
| idx_proposals_expires_at | expires_at | Find expired proposals |
| idx_proposals_cooldown_until | cooldown_until | Cooldown queries |
| idx_proposals_tenant_id | tenant_id | Tenant isolation |

### ceremonies

| Index | Columns | Purpose |
|-------|---------|---------|
| idx_ceremonies_marriage_id | marriage_id | Query ceremonies by marriage |
| idx_ceremonies_character_id1 | character_id1 | Query ceremonies by first character |
| idx_ceremonies_character_id2 | character_id2 | Query ceremonies by second character |
| idx_ceremonies_status | status | Filter by ceremony status |
| idx_ceremonies_started_at | started_at | Query by start date |
| idx_ceremonies_completed_at | completed_at | Query by completion date |
| idx_ceremonies_cancelled_at | cancelled_at | Query by cancellation date |
| idx_ceremonies_postponed_at | postponed_at | Query by postponement date |
| idx_ceremonies_tenant_id | tenant_id | Tenant isolation |

## Migration Rules

- Migrations run automatically on service startup via GORM AutoMigrate
- Tables are created in order: marriages, proposals, ceremonies
- Schema changes are additive and non-destructive
