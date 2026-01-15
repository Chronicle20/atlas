# Fame REST Integration

## Endpoints

This service does not expose any REST endpoints.

## External REST Dependencies

### Character Service

The service consumes the character REST API to retrieve character data.

**GET** `{CHARACTERS}/characters/{id}`

Response model:

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| name | string | Character name |
| level | byte | Character level |

Resource type: `characters`
