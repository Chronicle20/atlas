# REST

## Endpoints

This service does not expose any REST endpoints.

## External Dependencies

### atlas-character

| Method | Path | Description |
|--------|------|-------------|
| GET | /characters/{characterId} | Retrieve character by ID |

#### Response Model

Resource type: `characters`

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| name | string | Character name |
| level | byte | Character level |
