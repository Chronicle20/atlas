# REST

## Endpoints

### POST /api/wz/extractions

Triggers asynchronous WZ file extraction for the requesting tenant. Returns immediately with a 202 status.

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `xmlOnly` | query | `string` | No | If `"true"`, only XML serialization is performed |
| `imagesOnly` | query | `string` | No | If `"true"`, only icon extraction is performed |

Tenant identity is parsed from request headers via `ParseTenant` middleware.

#### Response

**202 Accepted**

```json
{
  "status": "started"
}
```

Extraction runs asynchronously in a background goroutine. Success or failure is logged to service logs.

#### Error Conditions

| Condition | Behavior |
|---|---|
| Missing tenant headers | Request rejected by `ParseTenant` middleware |
| No WZ files found | Logged as error; no output produced |
| Individual WZ file parse failure | Logged as error; extraction continues with remaining files |
