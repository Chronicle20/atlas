# REST

## Endpoints

### GET /parties

Returns all parties for the tenant.

#### Parameters

None

#### Request Model

None

#### Response Model

```json
{
  "data": [
    {
      "type": "parties",
      "id": "1000000000",
      "attributes": {
        "leaderId": 12345
      },
      "relationships": {
        "members": {
          "data": [
            { "type": "members", "id": "12345" }
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
        "level": 50,
        "jobId": 100,
        "worldId": 0,
        "channelId": 1,
        "mapId": 100000000,
        "instance": "00000000-0000-0000-0000-000000000000",
        "online": true
      }
    }
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal server error |

---

### GET /parties?filter[members.id]={memberId}

Returns parties containing the specified member.

#### Parameters

| Name | Location | Type | Description |
|------|----------|------|-------------|
| memberId | query | uint32 | Character ID to filter by |

#### Request Model

None

#### Response Model

Same as GET /parties

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid memberId format |
| 500 | Internal server error |

---

### POST /parties

Creates a new party. Request is asynchronous; party creation happens via Kafka.

#### Parameters

None

#### Request Model

```json
{
  "data": {
    "type": "parties",
    "relationships": {
      "members": {
        "data": [
          { "type": "members", "id": "12345" }
        ]
      }
    }
  }
}
```

#### Response Model

None (202 Accepted)

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Members array does not contain exactly one member |
| 500 | Failed to publish command |

---

### GET /parties/{partyId}

Returns a specific party by ID.

#### Parameters

| Name | Location | Type | Description |
|------|----------|------|-------------|
| partyId | path | uint32 | Party identifier |

#### Request Model

None

#### Response Model

```json
{
  "data": {
    "type": "parties",
    "id": "1000000000",
    "attributes": {
      "leaderId": 12345
    },
    "relationships": {
      "members": {
        "data": [
          { "type": "members", "id": "12345" }
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
        "level": 50,
        "jobId": 100,
        "worldId": 0,
        "channelId": 1,
        "mapId": 100000000,
        "instance": "00000000-0000-0000-0000-000000000000",
        "online": true
      }
    }
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 404 | Party not found |
| 500 | Internal server error |

---

### PATCH /parties/{partyId}

Updates party leadership. Request is asynchronous; leadership change happens via Kafka.

#### Parameters

| Name | Location | Type | Description |
|------|----------|------|-------------|
| partyId | path | uint32 | Party identifier |

#### Request Model

```json
{
  "data": {
    "type": "parties",
    "id": "1000000000",
    "attributes": {
      "leaderId": 67890
    }
  }
}
```

#### Response Model

None (202 Accepted)

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 404 | Party not found |
| 500 | Failed to publish command |

---

### GET /parties/{partyId}/members

Returns all members of a party.

#### Parameters

| Name | Location | Type | Description |
|------|----------|------|-------------|
| partyId | path | uint32 | Party identifier |

#### Request Model

None

#### Response Model

```json
{
  "data": [
    {
      "type": "members",
      "id": "12345",
      "attributes": {
        "name": "CharacterName",
        "level": 50,
        "jobId": 100,
        "worldId": 0,
        "channelId": 1,
        "mapId": 100000000,
        "instance": "00000000-0000-0000-0000-000000000000",
        "online": true
      }
    }
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal server error |

---

### GET /parties/{partyId}/relationships/members

Alias for GET /parties/{partyId}/members.

---

### POST /parties/{partyId}/members

Adds a member to a party. Request is asynchronous; join happens via Kafka.

#### Parameters

| Name | Location | Type | Description |
|------|----------|------|-------------|
| partyId | path | uint32 | Party identifier |

#### Request Model

```json
{
  "data": {
    "type": "members",
    "id": "67890"
  }
}
```

#### Response Model

None (202 Accepted)

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Failed to publish command |

---

### GET /parties/{partyId}/members/{memberId}

Returns a specific party member.

#### Parameters

| Name | Location | Type | Description |
|------|----------|------|-------------|
| partyId | path | uint32 | Party identifier |
| memberId | path | uint32 | Character identifier |

#### Request Model

None

#### Response Model

```json
{
  "data": {
    "type": "members",
    "id": "12345",
    "attributes": {
      "name": "CharacterName",
      "level": 50,
      "jobId": 100,
      "worldId": 0,
      "channelId": 1,
      "mapId": 100000000,
      "instance": "00000000-0000-0000-0000-000000000000",
      "online": true
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 404 | Member not found in party |
| 500 | Internal server error |

---

### DELETE /parties/{partyId}/members/{memberId}

Removes a member from a party. Request is asynchronous; removal happens via Kafka.

#### Parameters

| Name | Location | Type | Description |
|------|----------|------|-------------|
| partyId | path | uint32 | Party identifier |
| memberId | path | uint32 | Character identifier |

#### Request Model

None

#### Response Model

None (202 Accepted)

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Failed to publish command |

---

## Headers

All requests require tenant identification headers:

| Header | Description |
|--------|-------------|
| TENANT_ID | Tenant UUID |
| REGION | Region identifier |
| MAJOR_VERSION | Major version number |
| MINOR_VERSION | Minor version number |
