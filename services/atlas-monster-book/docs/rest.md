# REST

## Endpoints

### GET /characters/{characterId}/monster-book

Retrieves a character's monster-book collection stats.

#### Parameters

| Name | Location | Type | Description |
|------|----------|------|--------------|
| characterId | path | uint32 | Character identifier |

#### Request Model

None.

#### Response Model

Resource type: `monster-book`

| Field | Type | Description |
|-------|------|--------------|
| id | character.Id | Character identifier |
| bookLevel | uint16 | Monster book level |
| normalCount | uint16 | Count of owned normal cards |
| specialCount | uint16 | Count of owned special cards |
| totalUniqueCards | uint16 | normalCount + specialCount |
| coverCardId | item.Id | Selected cover card item id (0 = no cover) |
| coverMonsterId | monster.Id | Monster id represented by the cover card |
| expBonusPercent | uint16 | Party EXP bonus percentage granted by the book |

If no collection row exists for the character, a default response (bookLevel=1, all counts/cover fields zero) is returned.

#### Error Conditions

| Condition | Response |
|-----------|----------|
| Database failure | Error response per `server.WriteErrorResponse` |

### PATCH /characters/{characterId}/monster-book

Sets or clears the character's monster-book cover card.

#### Parameters

| Name | Location | Type | Description |
|------|----------|------|--------------|
| characterId | path | uint32 | Character identifier |

#### Request Model

Resource type: `monster-book`

| Field | Type | Description |
|-------|------|--------------|
| coverCardId | item.Id | Cover card item id to set; 0 clears the cover |

#### Response Model

Same as `GET /characters/{characterId}/monster-book`, reflecting the collection state after the update.

#### Error Conditions

| Condition | Response |
|-----------|----------|
| coverCardId is nonzero and does not classify as a monster-book card item | 422 Unprocessable Entity |
| coverCardId is nonzero and not owned by the character at level >= 1 | 422 Unprocessable Entity |
| Database failure | Error response per `server.WriteErrorResponse` |

### GET /characters/{characterId}/monster-book/cards

Retrieves a character's owned monster cards, paginated.

#### Parameters

| Name | Location | Type | Description |
|------|----------|------|--------------|
| characterId | path | uint32 | Character identifier |
| page[number] | query | int | Page number |
| page[size] | query | int | Page size (default and maximum 250) |
| filter[isSpecial] | query | bool | Optional filter restricting results to special or normal cards |

Cards are sorted ascending by cardId before pagination.

#### Request Model

None.

#### Response Model

Resource type: `monster-book-card`, JSON:API paginated collection.

| Field | Type | Description |
|-------|------|--------------|
| id | item.Id | Card item identifier |
| level | uint8 | Card level (1-5) |
| isSpecial | bool | Whether the card is a special card |
| firstAcquiredAt | time.Time | Timestamp the card was first acquired |

#### Error Conditions

| Condition | Response |
|-----------|----------|
| Invalid page[number]/page[size] (including a legacy `limit` query param) | 400 Bad Request |
| Database failure | Error response per `server.WriteErrorResponse` |

### GET /characters/{characterId}/monster-book/cards/{cardId}

Retrieves a single card owned by a character.

#### Parameters

| Name | Location | Type | Description |
|------|----------|------|--------------|
| characterId | path | uint32 | Character identifier |
| cardId | path | uint32 | Card item identifier |

#### Request Model

None.

#### Response Model

Resource type: `monster-book-card`

| Field | Type | Description |
|-------|------|--------------|
| id | item.Id | Card item identifier |
| level | uint8 | Card level (1-5) |
| isSpecial | bool | Whether the card is a special card |
| firstAcquiredAt | time.Time | Timestamp the card was first acquired |

#### Error Conditions

| Condition | Response |
|-----------|----------|
| Card not found for the character | 404 Not Found |
| Database failure | Error response per `server.WriteErrorResponse` |

## External Dependencies

### atlas-data

| Method | Path | Description |
|--------|------|-------------|
| GET | data/consumables/{itemId} | Retrieve a consumable by item id, used to resolve a monster-book cover card to its represented monster |

#### Response Model

Resource type: `consumables`

| Field | Type | Description |
|-------|------|-------------|
| monsterBook | bool | Whether the item is a monster-book card |
| monsterId | uint32 | Monster id represented by the card |
