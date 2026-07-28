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

### recipes

Derived index of `craftAction` states found inside NPC conversations. One row per `(tenant_id, conversation_id, state_id)` triple.

| Column                  | Type      | Constraints                                          |
|-------------------------|-----------|-------------------------------------------------------|
| id                      | uuid      | Primary key                                            |
| tenant_id               | uuid      | Not null                                               |
| conversation_id         | uuid      | Not null                                               |
| npc_id                  | uint32    | Not null                                               |
| state_id                | text      | Not null                                               |
| item_id                 | uint32    | Not null                                               |
| materials               | jsonb     | Not null, default `'[]'`                               |
| meso_cost               | uint32    | Not null, default 0                                    |
| stimulator_id           | uint32    | Not null, default 0                                    |
| stimulator_fail_chance  | float8    | Not null, default 0                                    |
| created_at              | timestamp | Not null, default CURRENT_TIMESTAMP                    |
| updated_at              | timestamp | Not null, default CURRENT_TIMESTAMP                    |

The `materials` column stores a JSON array of `{itemId, quantity}` entries. `id` is generated as a deterministic UUID v5 from `(tenant_id, conversation_id, state_id)`.

### seed_state

Tracks the most recently applied seed catalog revision per tenant/subdomain-group. Migrated from the shared `atlas-seeder` library.

| Column           | Type      | Constraints                          |
|------------------|-----------|----------------------------------------|
| tenant_id        | uuid      | Primary key (composite)                |
| group_name       | text      | Primary key (composite)                |
| catalog_revision | text      | Not null                               |
| seeded_at        | timestamp | Not null                               |
| result_summary   | jsonb     | Not null                               |

## Relationships

- `conversations.tenant_id` references the tenant. Scoped by tenant in all queries.
- `quest_conversations.tenant_id` references the tenant. Scoped by tenant in all queries.
- `recipes.tenant_id` references the tenant. Scoped by tenant in all queries.
- `recipes.conversation_id` references `conversations.id`.
- `seed_state.tenant_id` references the tenant.

## Indexes

- `conversations.deleted_at` — Indexed for soft delete queries.
- `quest_conversations(tenant_id, quest_id)` — Composite index `idx_quest_conversations_tenant_quest` for tenant-scoped quest lookup.
- `quest_conversations.npc_id` — Indexed for NPC-based lookups.
- `quest_conversations.deleted_at` — Indexed for soft delete queries.
- `recipes(tenant_id, item_id)` — Composite index `idx_recipes_tenant_item`.
- `recipes(tenant_id, npc_id)` — Composite index `idx_recipes_tenant_npc`.
- `recipes.conversation_id` — Index `idx_recipes_conversation`.
- `recipes(tenant_id, conversation_id, state_id)` — Unique composite index `idx_recipes_tenant_conv_state`.
- `seed_state(tenant_id, group_name)` — Composite primary key.

## Migration Rules

- Migrations are run via GORM `AutoMigrate` at service startup for `conversations`, `quest_conversations`, `recipes`, and `seed_state` tables.
- `conversations` and `quest_conversations` use soft deletes via the `deleted_at` column. `recipes` rows are hard-deleted.
