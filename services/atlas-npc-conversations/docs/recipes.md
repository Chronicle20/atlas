# Recipes

Derived index over `craftAction` states inside NPC conversations. One row per
`(tenant, conversation, state)` triple. Maintained automatically by the
NPC-conversation Create/Update/Delete/DeleteAllForTenant/Seed processor methods
inside the same transaction as the parent write.

## Table: `recipes`

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | Deterministic UUID v5 from `(tenantId, conversationId, stateId)`. Stable across rebuilds. |
| `tenant_id` | uuid | Auto-scoped via GORM tenant callbacks. |
| `conversation_id` | uuid | FK to `conversations.id`. |
| `npc_id` | uint32 | Crafter NPC. Denormalized for fast `/npcs/{npcId}/recipes`. |
| `state_id` | text | The `craftAction` state id within the conversation. |
| `item_id` | uint32 | Output item template id. |
| `materials` | jsonb | `[{itemId, quantity}, ...]`. |
| `meso_cost` | uint32 | `0` when free. |
| `stimulator_id` | uint32 | `0` when not a stimulator recipe. |
| `stimulator_fail_chance` | float8 | `[0.0, 1.0]`. `0` when not a stimulator recipe. |

Indexes: `(tenant_id, item_id)`, `(tenant_id, npc_id)`, `(conversation_id)`, unique `(tenant_id, conversation_id, state_id)`.

## Endpoints

### `GET /api/items/{itemId}/recipes`

Returns every recipe whose output is `itemId`, for the active tenant. Empty list when none. Ordered by `(npcId, stateId)`. `400` on non-numeric `itemId`.

### `GET /api/npcs/{npcId}/recipes`

Returns every recipe owned by `npcId`, for the active tenant. Ordered by `stateId`. `400` on non-numeric `npcId`.

### `POST /api/npcs/conversations/reindex-recipes`

Admin-only. Clears the active tenant's recipe rows and rebuilds them from every NPC conversation. Idempotent — same `id`s after each run because of the deterministic UUID v5 derivation. Single-transaction; failure rolls back to the prior index state.

Response: `recipeReindexResults` resource with attributes `deletedCount`, `insertedCount`, `skippedCount`, `skippedDetails`, `conversationsScanned`.
