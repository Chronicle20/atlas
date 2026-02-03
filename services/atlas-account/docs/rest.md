# REST API

## Endpoints

### POST /accounts/

Creates a new account via Kafka command.

#### Parameters

None.

#### Request Model

| Field | Type | JSON Key |
|-------|------|----------|
| Name | string | name |
| Password | string | password |
| Gender | byte | gender |

Resource type: `accounts`

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 202 Accepted | Command published |

---

### GET /accounts/

Retrieves all accounts for the current tenant.

#### Parameters

None.

#### Request Model

None.

#### Response Model

Array of Account resources.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | uint32 | (resource id) |
| Name | string | name |
| Pin | string | pin |
| Pic | string | pic |
| LoggedIn | byte | loggedIn |
| LastLogin | uint64 | lastLogin |
| Gender | byte | gender |
| Banned | bool | banned |
| TOS | bool | tos |
| Language | string | language |
| Country | string | country |
| CharacterSlots | int16 | characterSlots |

Resource type: `accounts`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Accounts retrieved |
| 500 Internal Server Error | Database or transformation error |

---

### GET /accounts/?name={name}

Retrieves account by name.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| name | query | string | yes |

#### Request Model

None.

#### Response Model

Single Account resource (see GET /accounts/).

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Account retrieved |
| 400 Bad Request | Missing name parameter |
| 404 Not Found | Account not found |

---

### GET /accounts/{accountId}

Retrieves account by ID.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| accountId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

Single Account resource (see GET /accounts/).

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Account retrieved |
| 400 Bad Request | Invalid account ID |
| 404 Not Found | Account not found |

---

### PATCH /accounts/{accountId}

Updates account attributes.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| accountId | path | uint32 | yes |

#### Request Model

Partial Account resource. Updatable fields:

| Field | Type | JSON Key |
|-------|------|----------|
| Pin | string | pin |
| Pic | string | pic |
| TOS | bool | tos |
| Gender | byte | gender |

#### Response Model

Updated Account resource (see GET /accounts/).

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Account updated |
| 400 Bad Request | Invalid account ID or request body |
| 404 Not Found | Account not found |
| 500 Internal Server Error | Transformation error |

---

### DELETE /accounts/{accountId}

Deletes an account.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| accountId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 204 No Content | Account deleted |
| 400 Bad Request | Invalid account ID |
| 404 Not Found | Account not found |
| 409 Conflict | Account is currently logged in |
| 500 Internal Server Error | Database error |

---

### DELETE /accounts/{accountId}/session

Logs out account by publishing logout command.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| accountId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 202 Accepted | Logout command published |
| 400 Bad Request | Invalid account ID |
