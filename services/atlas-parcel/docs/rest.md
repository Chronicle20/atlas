# REST

The read API and the discard/notify write paths, reached AFTER a send request has already been validated and accepted — creation, receive, restore, and remove are driven by the Kafka custody command consumer (see [Kafka](kafka.md)), not by this REST surface.

## Endpoints

### GET /api/parcels?filter[recipientId]={recipientId}&filter[worldId]={worldId}

A recipient's pending mailbox in a world.

**Parameters**
- `filter[recipientId]` (query, required): Recipient character identifier (uint32)
- `filter[worldId]` (query, required): World identifier (byte) — required whenever `filter[recipientId]` is supplied; world 0 is an ordinary real world, not a sentinel, so it is never defaulted
- `filter[status]` (query, optional): Must be `"pending"` if supplied — every list read only ever surfaces pending parcels

**Response Model**

```json
{
  "data": [
    {
      "type": "parcels",
      "id": "uuid",
      "attributes": {
        "worldId": 0,
        "senderId": 100,
        "senderAccountId": 1,
        "senderName": "Alice",
        "recipientId": 200,
        "recipientAccountId": 2,
        "recipientName": "Bob",
        "message": "here you go",
        "mesoAmount": 1000,
        "feePaid": 100,
        "itemId": 1302000,
        "itemType": 1,
        "quantity": 1,
        "itemSnapshot": {},
        "status": "pending",
        "quick": false,
        "returned": false,
        "createdAt": "2026-01-01T00:00:00Z",
        "receivableAt": "2026-01-02T00:00:00Z",
        "expiresAt": "2026-01-31T00:00:00Z"
      }
    }
  ]
}
```

**Error Conditions**
- `400 Bad Request`: `filter[recipientId]`/`filter[worldId]` malformed or missing, `filter[status]` not `"pending"`, or neither `filter[recipientId]` nor `filter[senderId]` supplied
- `500 Internal Server Error`: Database or transform error

---

### GET /api/parcels?filter[senderId]={senderId}

A sender's still-in-flight (pending) outbound parcels.

**Parameters**
- `filter[senderId]` (query, required): Sender character identifier (uint32)
- `filter[status]` (query, optional): Must be `"pending"` if supplied

**Response Model**

Same shape as the recipient list above.

**Error Conditions**
- `400 Bad Request`: `filter[senderId]` malformed, `filter[status]` not `"pending"`, or neither filter supplied
- `500 Internal Server Error`: Database or transform error

---

### GET /api/parcels/{parcelId}

Retrieves a single parcel by id.

**Parameters**
- `parcelId` (path): Parcel identifier (uuid)

**Response Model**

```json
{
  "data": {
    "type": "parcels",
    "id": "uuid",
    "attributes": { "...": "same attributes as the list response" }
  }
}
```

**Error Conditions**
- `400 Bad Request`: `parcelId` is not a valid uuid
- `404 Not Found`: No parcel exists with that id
- `500 Internal Server Error`: Database or transform error

---

### PATCH /api/parcels/{parcelId}

Marks a pending parcel discarded on behalf of the caller-supplied recipient (design §4.4/§7.3). Discard is deliberately NOT a saga — nothing leaves custody, so this route is called directly instead of going through the Kafka custody consumer.

**Parameters**
- `parcelId` (path): Parcel identifier (uuid)

**Request Model**

```json
{
  "data": {
    "type": "parcels",
    "id": "uuid",
    "attributes": {
      "recipientId": 200
    }
  }
}
```

**Response Model**

```json
{
  "data": {
    "type": "parcels",
    "id": "uuid",
    "attributes": { "...": "the now-discarded parcel" }
  }
}
```

**Error Conditions**
- `400 Bad Request`: `parcelId` is not a valid uuid
- `404 Not Found`: No parcel exists with that id
- `409 Conflict`: The caller-supplied `recipientId` is not the parcel's recipient, or the parcel is no longer pending
- `500 Internal Server Error`: Database or transform error

---

### PATCH /api/parcels/{parcelId}/notify

Stamps `lastNotified` on a parcel (task-241's SHOW_PARCEL consumer, design §5.3), dropping it out of the OPEN packet's "new arrivals" list on the next open. The body carries no attributes atlas-parcel reads — only the resource identifier, since the caller (atlas-channel) marshals a JSON:API request body regardless.

**Parameters**
- `parcelId` (path): Parcel identifier (uuid)

**Request Model**

```json
{
  "data": {
    "type": "parcels",
    "id": "uuid"
  }
}
```

**Response Model**

None — `204 No Content`.

**Error Conditions**
- `400 Bad Request`: `parcelId` is not a valid uuid
- `404 Not Found`: No parcel exists with that id
- `500 Internal Server Error`: Database error

---

### GET /api/characters/{characterId}/parcel-status

Reports whether a character has any pending parcel (as sender or recipient) — a single narrow round trip for task-26's world-transfer gate 12, rather than a full mailbox fetch.

**Parameters**
- `characterId` (path): Character identifier (uint32)

**Response Model**

```json
{
  "data": {
    "type": "parcel-statuses",
    "id": "12345",
    "attributes": {
      "inFlight": false
    }
  }
}
```

**Error Conditions**
- `400 Bad Request`: `characterId` malformed
- `500 Internal Server Error`: Database error
