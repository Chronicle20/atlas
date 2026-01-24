# REST

## Endpoints

### GET /api/messengers

Returns all messengers.

**Parameters**

| Name | In | Type | Description |
|------|-----|------|-------------|
| filter[members.id] | query | uint32 | Filter by member character ID |

**Request Model**

None.

**Response Model**

```json
{
  "data": [
    {
      "type": "messengers",
      "id": "1000000001",
      "relationships": {
        "members": {
          "data": [
            {"type": "members", "id": "12345"}
          ]
        }
      }
    }
  ],
  "included": [
    {
      "type": "members",
      "id": "12345",
      "attributes": {
        "name": "CharacterName",
        "worldId": 0,
        "channelId": 1,
        "online": true,
        "slot": 0
      }
    }
  ]
}
```

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid member ID filter |
| 500 | Internal server error |

---

### GET /api/messengers/{messengerId}

Returns a messenger by ID.

**Parameters**

| Name | In | Type | Description |
|------|-----|------|-------------|
| messengerId | path | uint32 | Messenger ID |

**Request Model**

None.

**Response Model**

```json
{
  "data": {
    "type": "messengers",
    "id": "1000000001",
    "relationships": {
      "members": {
        "data": [
          {"type": "members", "id": "12345"}
        ]
      }
    }
  },
  "included": [
    {
      "type": "members",
      "id": "12345",
      "attributes": {
        "name": "CharacterName",
        "worldId": 0,
        "channelId": 1,
        "online": true,
        "slot": 0
      }
    }
  ]
}
```

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 404 | Messenger not found |

---

### GET /api/messengers/{messengerId}/members

Returns members of a messenger.

**Parameters**

| Name | In | Type | Description |
|------|-----|------|-------------|
| messengerId | path | uint32 | Messenger ID |

**Request Model**

None.

**Response Model**

```json
{
  "data": [
    {
      "type": "members",
      "id": "12345",
      "attributes": {
        "name": "CharacterName",
        "worldId": 0,
        "channelId": 1,
        "online": true,
        "slot": 0
      }
    }
  ]
}
```

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 500 | Messenger not found or internal error |

---

### GET /api/messengers/{messengerId}/relationships/members

Returns members of a messenger (JSON:API relationship format).

**Parameters**

| Name | In | Type | Description |
|------|-----|------|-------------|
| messengerId | path | uint32 | Messenger ID |

**Request Model**

None.

**Response Model**

```json
{
  "data": [
    {
      "type": "members",
      "id": "12345",
      "attributes": {
        "name": "CharacterName",
        "worldId": 0,
        "channelId": 1,
        "online": true,
        "slot": 0
      }
    }
  ]
}
```

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 500 | Messenger not found or internal error |

---

### POST /api/messengers/{messengerId}/members

Adds a member to a messenger by producing a JOIN command to Kafka.

**Parameters**

| Name | In | Type | Description |
|------|-----|------|-------------|
| messengerId | path | uint32 | Messenger ID |

**Request Model**

```json
{
  "data": {
    "type": "members",
    "id": "12345"
  }
}
```

**Response Model**

None. Returns 202 Accepted.

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 500 | Failed to produce Kafka message |

---

### GET /api/messengers/{messengerId}/members/{memberId}

Returns a specific member of a messenger.

**Parameters**

| Name | In | Type | Description |
|------|-----|------|-------------|
| messengerId | path | uint32 | Messenger ID |
| memberId | path | uint32 | Member character ID |

**Request Model**

None.

**Response Model**

```json
{
  "data": {
    "type": "members",
    "id": "12345",
    "attributes": {
      "name": "CharacterName",
      "worldId": 0,
      "channelId": 1,
      "online": true,
      "slot": 0
    }
  }
}
```

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 404 | Member not found |
| 500 | Messenger not found or internal error |

---

### DELETE /api/messengers/{messengerId}/members/{memberId}

Removes a member from a messenger by producing a LEAVE command to Kafka.

**Parameters**

| Name | In | Type | Description |
|------|-----|------|-------------|
| messengerId | path | uint32 | Messenger ID |
| memberId | path | uint32 | Member character ID |

**Request Model**

None.

**Response Model**

None. Returns 202 Accepted.

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 500 | Failed to produce Kafka message |
