# REST API

## Endpoints

### GET /bans/

Retrieves all bans for the current tenant. Optionally filtered by ban type.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| type | query | int | no |
| page[number] | query | int | no (default 1) |
| page[size] | query | int | no (default 50, max 250) |

The legacy `limit` query parameter is rejected.

#### Request Model

None.

#### Response Model

Array of Ban resources.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | uint32 | (resource id) |
| BanType | byte | banType |
| Value | string | value |
| Reason | string | reason |
| ReasonCode | byte | reasonCode |
| Permanent | bool | permanent |
| ExpiresAt | time.Time | expiresAt |
| IssuedBy | string | issuedBy |

Resource type: `bans`

JSON:API `meta` block:

| Field | Type | Description |
|-------|------|--------------|
| total | int | Total count of matching bans across all pages |
| page.number | int | Current page number |
| page.size | int | Current page size |
| page.last | int | Last page number |

JSON:API `links` block: `self`, `first`, `last`, and `prev`/`next` where applicable.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Bans retrieved |
| 400 Bad Request | Invalid type parameter, or invalid page[number]/page[size] (non-integer, out of range, or legacy limit param used) |
| 500 Internal Server Error | Database or transformation error |

---

### GET /bans/{banId}

Retrieves a ban by ID.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| banId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

Single Ban resource (see GET /bans/).

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Ban retrieved |
| 400 Bad Request | Invalid ban ID |
| 404 Not Found | Ban not found |

---

### POST /bans/

Creates a new ban.

#### Parameters

None.

#### Request Model

| Field | Type | JSON Key |
|-------|------|----------|
| BanType | byte | banType |
| Value | string | value |
| Reason | string | reason |
| ReasonCode | byte | reasonCode |
| Permanent | bool | permanent |
| ExpiresAt | time.Time | expiresAt |
| IssuedBy | string | issuedBy |

Resource type: `bans`

#### Response Model

Single Ban resource (see GET /bans/).

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 201 Created | Ban created |
| 400 Bad Request | Invalid request body |
| 500 Internal Server Error | Database or transformation error |

---

### DELETE /bans/{banId}

Deletes a ban.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| banId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 204 No Content | Ban deleted |
| 400 Bad Request | Invalid ban ID |
| 500 Internal Server Error | Database error |

---

### POST /bans/{banId}/expire

Expires a temporary ban early.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| banId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

None.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 204 No Content | Ban expired |
| 400 Bad Request | Ban is permanent |
| 500 Internal Server Error | Database error |

---

### GET /bans/check

Checks if an IP address, hardware ID, or account is currently banned.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| ip | query | string | no |
| hwid | query | string | no |
| accountId | query | uint32 | no |

#### Request Model

None.

#### Response Model

Single BanCheck resource.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | uint32 | (resource id) |
| Banned | bool | banned |
| BanType | byte | banType |
| Reason | string | reason |
| ReasonCode | byte | reasonCode |
| Permanent | bool | permanent |
| ExpiresAt | time.Time | expiresAt |

Resource type: `ban-checks`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Check completed |
| 400 Bad Request | Invalid accountId parameter |
| 500 Internal Server Error | Database error |

---

### GET /history/

Retrieves login history for the current tenant. Optionally filtered by IP or HWID.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| ip | query | string | no |
| hwid | query | string | no |
| page[number] | query | int | no (default 1) |
| page[size] | query | int | no (default 50, max 250) |

The legacy `limit` query parameter is rejected.

#### Request Model

None.

#### Response Model

Array of LoginHistory resources.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | uint64 | (resource id) |
| AccountId | uint32 | accountId |
| AccountName | string | accountName |
| IPAddress | string | ipAddress |
| HWID | string | hwid |
| Success | bool | success |
| FailureReason | string | failureReason |

Resource type: `login-history`

JSON:API `meta` block:

| Field | Type | Description |
|-------|------|--------------|
| total | int | Total count of matching history entries across all pages |
| page.number | int | Current page number |
| page.size | int | Current page size |
| page.last | int | Last page number |

JSON:API `links` block: `self`, `first`, `last`, and `prev`/`next` where applicable.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | History retrieved |
| 400 Bad Request | Invalid page[number]/page[size] (non-integer, out of range, or legacy limit param used) |
| 500 Internal Server Error | Database or transformation error |

---

### GET /history/accounts/{accountId}

Retrieves login history for a specific account.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| accountId | path | uint32 | yes |
| page[number] | query | int | no (default 1) |
| page[size] | query | int | no (default 50, max 250) |

The legacy `limit` query parameter is rejected.

#### Request Model

None.

#### Response Model

Array of LoginHistory resources (see GET /history/).

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | History retrieved |
| 400 Bad Request | Invalid account ID, or invalid page[number]/page[size] (non-integer, out of range, or legacy limit param used) |
| 500 Internal Server Error | Database or transformation error |
