# Character Factory REST API

## Endpoints

### POST /api/characters/seed

Creates a new character using saga-based orchestration.

#### Parameters

None.

#### Request Headers

| Header        | Required | Description                |
|---------------|----------|----------------------------|
| TENANT_ID     | Yes      | Tenant identifier (UUID)   |
| REGION        | Yes      | Region identifier          |
| MAJOR_VERSION | Yes      | Major version number       |
| MINOR_VERSION | Yes      | Minor version number       |

#### Request Model

JSON:API resource type: `characters`

| Field        | Type   | Required | Description                    |
|--------------|--------|----------|--------------------------------|
| accountId    | uint32 | Yes      | Account ID                     |
| worldId      | byte   | Yes      | World ID                       |
| name         | string | Yes      | Character name (1-12 chars)    |
| gender       | byte   | Yes      | Gender (0 or 1)                |
| jobIndex     | uint32 | Yes      | Job index                      |
| subJobIndex  | uint32 | Yes      | Sub-job index                  |
| face         | uint32 | Yes      | Face template ID               |
| hair         | uint32 | Yes      | Hair template ID               |
| hairColor    | uint32 | Yes      | Hair color                     |
| skinColor    | byte   | Yes      | Skin color                     |
| top          | uint32 | Yes      | Top equipment template ID      |
| bottom       | uint32 | Yes      | Bottom equipment template ID   |
| shoes        | uint32 | Yes      | Shoes equipment template ID    |
| weapon       | uint32 | Yes      | Weapon equipment template ID   |
| level        | byte   | No       | Starting level                 |
| strength     | uint16 | No       | Starting strength              |
| dexterity    | uint16 | No       | Starting dexterity             |
| intelligence | uint16 | No       | Starting intelligence          |
| luck         | uint16 | No       | Starting luck                  |
| hp           | uint16 | No       | Starting HP                    |
| mp           | uint16 | No       | Starting MP                    |
| mapId        | uint32 | No       | Starting map ID                |

#### Response Model

JSON:API resource type: `characters`

| Field         | Type   | Description                        |
|---------------|--------|------------------------------------|
| transactionId | string | UUID for tracking saga progress    |

#### Error Conditions

| Status Code | Condition                                              |
|-------------|--------------------------------------------------------|
| 400         | Character name invalid (length or characters)          |
| 400         | Gender not 0 or 1                                      |
| 400         | Invalid job index                                      |
| 400         | Face not valid for job                                 |
| 400         | Hair not valid for job                                 |
| 400         | Hair color not valid for job                           |
| 400         | Skin color not valid for job                           |
| 400         | Top not valid for job                                  |
| 400         | Bottom not valid for job                               |
| 400         | Shoes not valid for job                                |
| 400         | Weapon not valid for job                               |
| 500         | Template validation configuration not found            |
| 500         | Saga creation failure                                  |
