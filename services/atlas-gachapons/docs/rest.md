# REST API

## Endpoints

### GET /gachapons

Retrieves all gachapons for the tenant.

#### Parameters

None.

#### Request Model

None.

#### Response Model

Array of Gachapon resources.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | string | (resource id) |
| Name | string | name |
| NpcIds | []uint32 | npcIds |
| CommonWeight | uint32 | commonWeight |
| UncommonWeight | uint32 | uncommonWeight |
| RareWeight | uint32 | rareWeight |

Resource type: `gachapons`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Gachapons retrieved |
| 500 Internal Server Error | Database or transformation error |

---

### GET /gachapons/{gachaponId}

Retrieves a single gachapon by ID.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| gachaponId | path | string | yes |

#### Request Model

None.

#### Response Model

Single Gachapon resource. Same fields as GET /gachapons.

Resource type: `gachapons`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Gachapon retrieved |
| 400 Bad Request | Invalid gachaponId |
| 404 Not Found | Gachapon not found |
| 500 Internal Server Error | Database or transformation error |

---

### POST /gachapons

Creates a new gachapon.

#### Parameters

None.

#### Request Model

JSON:API Gachapon resource.

| Field | Type | JSON Key | Required |
|-------|------|----------|----------|
| Id | string | (resource id) | yes |
| Name | string | name | yes |
| NpcIds | []uint32 | npcIds | yes |
| CommonWeight | uint32 | commonWeight | yes |
| UncommonWeight | uint32 | uncommonWeight | yes |
| RareWeight | uint32 | rareWeight | yes |

Resource type: `gachapons`

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 201 Created | Gachapon created |
| 400 Bad Request | Invalid input or model validation failure |
| 500 Internal Server Error | Database error |

---

### PATCH /gachapons/{gachaponId}

Updates an existing gachapon.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| gachaponId | path | string | yes |

#### Request Model

JSON:API Gachapon resource with fields to update.

| Field | Type | JSON Key |
|-------|------|----------|
| Name | string | name |
| CommonWeight | uint32 | commonWeight |
| UncommonWeight | uint32 | uncommonWeight |
| RareWeight | uint32 | rareWeight |

Resource type: `gachapons`

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 204 No Content | Gachapon updated |
| 400 Bad Request | Invalid gachaponId |
| 500 Internal Server Error | Database error |

---

### DELETE /gachapons/{gachaponId}

Deletes a gachapon.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| gachaponId | path | string | yes |

#### Request Model

None.

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 204 No Content | Gachapon deleted |
| 400 Bad Request | Invalid gachaponId |
| 500 Internal Server Error | Database error |

---

### GET /gachapons/{gachaponId}/items

Retrieves items for a gachapon. Optionally filtered by tier.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| gachaponId | path | string | yes |
| tier | query | string | no |

#### Request Model

None.

#### Response Model

Array of GachaponItem resources.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | uint32 | (resource id) |
| GachaponId | string | gachaponId |
| ItemId | uint32 | itemId |
| Quantity | uint32 | quantity |
| Tier | string | tier |

Resource type: `gachapon-items`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Items retrieved |
| 400 Bad Request | Invalid gachaponId |
| 500 Internal Server Error | Database or transformation error |

---

### POST /gachapons/{gachaponId}/items

Creates a new item for a gachapon.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| gachaponId | path | string | yes |

#### Request Model

JSON:API GachaponItem resource.

| Field | Type | JSON Key | Required |
|-------|------|----------|----------|
| ItemId | uint32 | itemId | yes |
| Quantity | uint32 | quantity | yes |
| Tier | string | tier | yes |

Resource type: `gachapon-items`

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 201 Created | Item created |
| 400 Bad Request | Invalid gachaponId or model validation failure |
| 500 Internal Server Error | Database error |

---

### DELETE /gachapons/{gachaponId}/items/{itemId}

Deletes a gachapon item.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| gachaponId | path | string | yes |
| itemId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 204 No Content | Item deleted |
| 400 Bad Request | Invalid gachaponId or itemId |
| 500 Internal Server Error | Database error |

---

### GET /global-items

Retrieves all global gachapon items. Optionally filtered by tier.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| tier | query | string | no |

#### Request Model

None.

#### Response Model

Array of GlobalGachaponItem resources.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | uint32 | (resource id) |
| ItemId | uint32 | itemId |
| Quantity | uint32 | quantity |
| Tier | string | tier |

Resource type: `global-gachapon-items`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Items retrieved |
| 500 Internal Server Error | Database or transformation error |

---

### POST /global-items

Creates a new global gachapon item.

#### Parameters

None.

#### Request Model

JSON:API GlobalGachaponItem resource.

| Field | Type | JSON Key | Required |
|-------|------|----------|----------|
| ItemId | uint32 | itemId | yes |
| Quantity | uint32 | quantity | yes |
| Tier | string | tier | yes |

Resource type: `global-gachapon-items`

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 201 Created | Item created |
| 400 Bad Request | Model validation failure |
| 500 Internal Server Error | Database error |

---

### DELETE /global-items/{itemId}

Deletes a global gachapon item.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| itemId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 204 No Content | Item deleted |
| 400 Bad Request | Invalid itemId |
| 500 Internal Server Error | Database error |

---

### POST /gachapons/{gachaponId}/rewards/select

Selects a random reward from a gachapon using weighted tier selection.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| gachaponId | path | string | yes |

#### Request Model

None.

#### Response Model

Single GachaponReward resource.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | string | (resource id) |
| ItemId | uint32 | itemId |
| Quantity | uint32 | quantity |
| Tier | string | tier |
| GachaponId | string | gachaponId |

Resource type: `gachapon-rewards`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Reward selected |
| 400 Bad Request | Invalid gachaponId |
| 500 Internal Server Error | Gachapon not found, empty pool, or selection error |

---

### GET /gachapons/{gachaponId}/prize-pool

Retrieves the merged prize pool for a gachapon. Optionally filtered by tier.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| gachaponId | path | string | yes |
| tier | query | string | no |

#### Request Model

None.

#### Response Model

Array of GachaponReward resources. Same fields as POST /gachapons/{gachaponId}/rewards/select.

Resource type: `gachapon-rewards`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Prize pool retrieved |
| 400 Bad Request | Invalid gachaponId |
| 500 Internal Server Error | Database or transformation error |

---

### POST /gachapons/seed

Triggers asynchronous seed data loading for the tenant. Deletes existing data and loads gachapons, items, and global items from JSON files.

#### Parameters

None.

#### Request Model

None.

#### Response Model

None. Seeding runs asynchronously.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 202 Accepted | Seed operation started |
