# API Contracts — Item Craft Recipes

> **Surface rename (2026-04-25):** the original draft used `craftRecipes`/`craft-recipes`/`craft_recipes` everywhere. The implemented surface is `recipes`/`/items/{itemId}/recipes`/`/npcs/{npcId}/recipes`/`/npcs/conversations/reindex-recipes`/`recipes` table. User-facing UI titles ("Craftable At", "Crafts") are unchanged.

All endpoints live in **atlas-npc-conversations** under the standard `/api/` prefix. All endpoints require the four tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`).

## Resource: `recipe`

JSON:API resource type: `"recipes"`.

`id` is the `recipes.id` UUID — stable per `(tenantId, conversationId, stateId)` across rebuilds (an upsert keyed on the unique constraint preserves the id).

```json
{
  "type": "recipes",
  "id": "f3a1c4e0-12bc-4a7d-8e9f-7a5b6c2e91d4",
  "attributes": {
    "npcId": 2040020,
    "conversationId": "9b2e0f6a-b5d4-4f1c-9b40-7c4f3a2e1d09",
    "stateId": "craftWarrior0",
    "itemId": 1082007,
    "materials": [
      { "itemId": 4011000, "quantity": 3 },
      { "itemId": 4011001, "quantity": 2 },
      { "itemId": 4003000, "quantity": 15 }
    ],
    "mesoCost": 18000,
    "stimulatorId": 0,
    "stimulatorFailChance": 0
  }
}
```

Stimulator variant example:

```json
{
  "type": "recipes",
  "id": "...",
  "attributes": {
    "npcId": 2040020,
    "conversationId": "...",
    "stateId": "craftWarrior0Stim",
    "itemId": 1082007,
    "materials": [
      { "itemId": 4011000, "quantity": 3 },
      { "itemId": 4011001, "quantity": 2 },
      { "itemId": 4003000, "quantity": 15 }
    ],
    "mesoCost": 18000,
    "stimulatorId": 4020009,
    "stimulatorFailChance": 0.10
  }
}
```

### Field semantics

| Field                  | Type     | Notes |
|------------------------|----------|-------|
| `npcId`                | uint32   | NPC template id of the crafter. |
| `conversationId`       | uuid     | Parent NPC conversation id. |
| `stateId`              | string   | The `craftAction` state id within that conversation. |
| `itemId`               | uint32   | Output item template id. |
| `materials`            | array    | Aligned material list. Each element has `itemId` + `quantity`. Empty array when the recipe takes no materials (rare but possible). |
| `mesoCost`             | uint32   | `0` when free. |
| `stimulatorId`         | uint32   | `0` when this is a non-stimulator recipe. |
| `stimulatorFailChance` | float64  | `[0.0, 1.0]`. `0` when this is a non-stimulator recipe. |

## Endpoint: `GET /items/{itemId}/recipes`

Returns every recipe whose output is `itemId`, for the active tenant.

### Path params

| Name     | Type     | Required | Description                                  |
|----------|----------|----------|----------------------------------------------|
| `itemId` | uint32   | yes      | `400` if not parseable as a positive integer. |

### Responses

`200 OK` — JSON:API list. Empty `data: []` when nothing matches.

```json
{
  "data": [
    { "type": "recipes", "id": "...", "attributes": { ... } },
    { "type": "recipes", "id": "...", "attributes": { ... } }
  ]
}
```

Ordering: ascending by `npcId`, then `stateId`. Deterministic across requests.

`400 Bad Request` — `itemId` malformed.

`500 Internal Server Error` — DB failure. Logged with the inbound `itemId`.

## Endpoint: `GET /npcs/{npcId}/recipes`

Returns every recipe owned by `npcId`, for the active tenant.

### Path params

| Name    | Type   | Required | Description                                  |
|---------|--------|----------|----------------------------------------------|
| `npcId` | uint32 | yes      | `400` if not parseable as a positive integer. |

### Responses

Same body shape as `/items/{itemId}/recipes`. Ordering: ascending by `stateId`.

## Endpoint: `POST /npcs/conversations/reindex-recipes`

Admin-only. Rebuilds `recipes` for the active tenant by walking every NPC conversation and re-emitting one row per `craftAction` state. Idempotent — running it twice produces the same row set.

### Request body

None.

### Responses

`200 OK` — JSON:API single resource:

```json
{
  "data": {
    "type": "recipeReindexResults",
    "id": "<tenantId>",
    "attributes": {
      "deletedCount": 47,
      "insertedCount": 47,
      "conversationsScanned": 312
    }
  }
}
```

`500 Internal Server Error` — DB failure mid-rebuild. The handler MUST run the clear-and-rebuild inside a single transaction so a mid-rebuild failure leaves the index in its prior state.

## Headers

All three endpoints require the standard tenant headers. Missing or empty `TENANT_ID` MUST return the same error envelope as other tenant-scoped endpoints in this service (consistent with `/npcs/conversations/seed`).

## Caching

No client-side caching headers. atlas-ui controls freshness via React Query (`staleTime: 10 * 60 * 1000`, matching `useItemSellers` / `useItemDrops`).
