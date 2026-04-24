# Seed Counts — API Contracts

Companion to `prd.md`. Concrete request/response shapes for every new count endpoint.

All endpoints:

- Method: `GET`
- `Accept: application/vnd.api+json`
- Tenant headers required: `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`.
- Success: `200 OK`, `Content-Type: application/vnd.api+json`.
- Failure: `500 Internal Server Error`, `Content-Type: application/json`, body `{"error": "<detail>"}`.
- `id` field is the tenant UUID (matches pattern used by `/api/wz/input`, `/api/data/status`).

`updatedAt` is an ISO 8601 / RFC 3339 string when at least one counted row exists for the tenant and the underlying table has an `updated_at` (or equivalent) column. It is `null` when the tables are empty or no suitable column exists. The frontend does not display this field in v1; it exists for parity with the Game Data status endpoints and for possible future use.

---

## `GET /api/drops/seed/status`

Service: atlas-drop-information.

### Response 200

```json
{
  "data": {
    "type": "dropsSeedStatus",
    "id": "b1a2c3d4-5678-4abc-9def-0123456789ab",
    "attributes": {
      "monsterDropCount": 12040,
      "continentDropCount": 48,
      "reactorDropCount": 6116,
      "updatedAt": "2026-04-24T14:22:09Z"
    }
  }
}
```

### Empty tenant

```json
{
  "data": {
    "type": "dropsSeedStatus",
    "id": "b1a2c3d4-5678-4abc-9def-0123456789ab",
    "attributes": {
      "monsterDropCount": 0,
      "continentDropCount": 0,
      "reactorDropCount": 0,
      "updatedAt": null
    }
  }
}
```

---

## `GET /api/gachapons/seed/status`

Service: atlas-gachapons.

### Response 200

```json
{
  "data": {
    "type": "gachaponsSeedStatus",
    "id": "b1a2c3d4-5678-4abc-9def-0123456789ab",
    "attributes": {
      "gachaponCount": 17,
      "itemCount": 842,
      "globalItemCount": 60,
      "updatedAt": "2026-04-24T14:22:41Z"
    }
  }
}
```

---

## `GET /api/npcs/conversations/seed/status`

Service: atlas-npc-conversations.

### Response 200

```json
{
  "data": {
    "type": "npcConversationsSeedStatus",
    "id": "b1a2c3d4-5678-4abc-9def-0123456789ab",
    "attributes": {
      "conversationCount": 1284,
      "updatedAt": "2026-04-24T14:22:51Z"
    }
  }
}
```

---

## `GET /api/quests/conversations/seed/status`

Service: atlas-npc-conversations.

### Response 200

```json
{
  "data": {
    "type": "questConversationsSeedStatus",
    "id": "b1a2c3d4-5678-4abc-9def-0123456789ab",
    "attributes": {
      "conversationCount": 517,
      "updatedAt": "2026-04-24T14:22:53Z"
    }
  }
}
```

---

## `GET /api/shops/seed/status`

Service: atlas-npc-shops.

### Response 200

```json
{
  "data": {
    "type": "npcShopsSeedStatus",
    "id": "b1a2c3d4-5678-4abc-9def-0123456789ab",
    "attributes": {
      "shopCount": 148,
      "commodityCount": 2341,
      "updatedAt": "2026-04-24T14:22:58Z"
    }
  }
}
```

---

## `GET /api/portals/scripts/seed/status`

Service: atlas-portal-actions. Must not route to atlas-portals; verify ingress specificity during implementation.

### Response 200

```json
{
  "data": {
    "type": "portalScriptsSeedStatus",
    "id": "b1a2c3d4-5678-4abc-9def-0123456789ab",
    "attributes": {
      "scriptCount": 61,
      "updatedAt": "2026-04-24T14:22:59Z"
    }
  }
}
```

---

## `GET /api/reactors/actions/seed/status`

Service: atlas-reactor-actions.

### Response 200

```json
{
  "data": {
    "type": "reactorScriptsSeedStatus",
    "id": "b1a2c3d4-5678-4abc-9def-0123456789ab",
    "attributes": {
      "scriptCount": 89,
      "updatedAt": "2026-04-24T14:23:01Z"
    }
  }
}
```

---

## `GET /api/maps/actions/seed/status`

Service: atlas-map-actions.

### Response 200

```json
{
  "data": {
    "type": "mapActionScriptsSeedStatus",
    "id": "b1a2c3d4-5678-4abc-9def-0123456789ab",
    "attributes": {
      "scriptCount": 210,
      "updatedAt": "2026-04-24T14:23:02Z"
    }
  }
}
```

---

## Error shape (all endpoints)

```json
{
  "error": "failed to count monster drops: <detail>"
}
```

Status code: `500` only. No `404` — a tenant with zero seeded rows returns `200` with `0`s. No `401/403` — these endpoints inherit the project's current no-auth posture.
