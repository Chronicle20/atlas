# Storage

## Tables

### conversations

Stores NPC conversation definitions.

| Column     | Type         | Constraints                  |
|------------|--------------|------------------------------|
| id         | uuid         | Primary key                  |
| tenant_id  | uuid         | Not null                     |
| npc_id     | uint32       | Not null                     |
| data       | jsonb        | Not null                     |
| created_at | timestamp    | Not null, default NOW()      |
| updated_at | timestamp    | Not null, default NOW()      |
| deleted_at | timestamp    | Nullable, indexed (soft delete) |

The `data` column stores the full conversation definition (startState, states) as a JSON blob.

### quest_conversations

Stores quest conversation definitions.

| Column     | Type         | Constraints                  |
|------------|--------------|------------------------------|
| id         | uuid         | Primary key                  |
| tenant_id  | uuid         | Not null                     |
| quest_id   | uint32       | Not null                     |
| npc_id     | uint32       | Indexed                      |
| data       | jsonb        | Not null                     |
| created_at | timestamp    | Not null, default NOW()      |
| updated_at | timestamp    | Not null, default NOW()      |
| deleted_at | timestamp    | Nullable, indexed (soft delete) |

The `data` column stores the full quest conversation definition (questName, startStateMachine, endStateMachine) as a JSON blob.

## Relationships

- `conversations.tenant_id` references the tenant. Scoped by tenant in all queries.
- `quest_conversations.tenant_id` references the tenant. Scoped by tenant in all queries.

## Indexes

- `conversations.deleted_at` — Indexed for soft delete queries.
- `quest_conversations(tenant_id, quest_id)` — Composite index `idx_quest_conversations_tenant_quest` for tenant-scoped quest lookup.
- `quest_conversations.npc_id` — Indexed for NPC-based lookups.
- `quest_conversations.deleted_at` — Indexed for soft delete queries.

## Migration Rules

- Migrations are run via GORM `AutoMigrate` at service startup for both `conversations` and `quest_conversations` tables.
- Both tables use soft deletes via the `deleted_at` column.
