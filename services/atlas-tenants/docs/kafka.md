# Kafka

## Topics Consumed

None.

## Topics Produced

### tenant.status

Tenant lifecycle events.

### configuration.status

Configuration resource lifecycle events.

## Message Types

### StatusEvent (tenant.status)

```json
{
  "tenantId": "uuid",
  "type": "CREATED | UPDATED | DELETED",
  "body": {
    "name": "string",
    "region": "string",
    "majorVersion": 0,
    "minorVersion": 0
  }
}
```

**Event Types**
- `CREATED`: Emitted when a tenant is created
- `UPDATED`: Emitted when a tenant is updated
- `DELETED`: Emitted when a tenant is deleted

### ConfigurationStatusEvent (configuration.status)

```json
{
  "tenantId": "uuid",
  "type": "ROUTE_CREATED | ROUTE_UPDATED | ROUTE_DELETED | VESSEL_CREATED | VESSEL_UPDATED | VESSEL_DELETED",
  "resourceType": "route | vessel",
  "resourceId": "string"
}
```

**Event Types**
- `ROUTE_CREATED`: Emitted when a route is created
- `ROUTE_UPDATED`: Emitted when a route is updated
- `ROUTE_DELETED`: Emitted when a route is deleted
- `VESSEL_CREATED`: Emitted when a vessel is created
- `VESSEL_UPDATED`: Emitted when a vessel is updated
- `VESSEL_DELETED`: Emitted when a vessel is deleted

## Transaction Semantics

Messages are buffered and emitted after successful database operations.
