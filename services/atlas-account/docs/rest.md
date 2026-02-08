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
| PinAttempts | int | pinAttempts |
| PicAttempts | int | picAttempts |
| LoggedIn | byte | loggedIn |
| LastLogin | uint64 | lastLogin |
| Gender | byte | gender |
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
| PinAttempts | int | pinAttempts |
| PicAttempts | int | picAttempts |
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

### POST /accounts/{accountId}/pin-attempts

Records a PIN attempt result. On failure, increments the PIN attempt counter. If the configured limit is reached, issues a temporary ban via Kafka and resets the counter. On success, resets the counter to 0.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| accountId | path | uint32 | yes |

#### Request Model

| Field | Type | JSON Key |
|-------|------|----------|
| Success | bool | success |

Resource type: `pin-attempts`

#### Response Model

| Field | Type | JSON Key |
|-------|------|----------|
| Id | string | (resource id) |
| Attempts | int | attempts |
| LimitReached | bool | limitReached |

Resource type: `pin-attempts`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Attempt recorded |
| 400 Bad Request | Invalid account ID |
| 500 Internal Server Error | Database or processing error |

---

### POST /accounts/{accountId}/pic-attempts

Records a PIC attempt result. On failure, increments the PIC attempt counter. If the configured limit is reached, issues a temporary ban via Kafka and resets the counter. On success, resets the counter to 0.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| accountId | path | uint32 | yes |

#### Request Model

| Field | Type | JSON Key |
|-------|------|----------|
| Success | bool | success |

Resource type: `pic-attempts`

#### Response Model

| Field | Type | JSON Key |
|-------|------|----------|
| Id | string | (resource id) |
| Attempts | int | attempts |
| LimitReached | bool | limitReached |

Resource type: `pic-attempts`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Attempt recorded |
| 400 Bad Request | Invalid account ID |
| 500 Internal Server Error | Database or processing error |

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
