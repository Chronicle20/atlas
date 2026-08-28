# Kafka

## Topics Consumed

None.

## Topics Produced

### tenant.status

Tenant lifecycle events.

### EVENT_TOPIC_CONFIGURATION_STATUS

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
    "minorVersion": 0,
    "environment": "string"
  }
}
```

**Event Types**
- `CREATED`: Emitted when a tenant is created
- `UPDATED`: Emitted when a tenant is updated
- `DELETED`: Emitted when a tenant is deleted

### ConfigurationStatusEvent (EVENT_TOPIC_CONFIGURATION_STATUS)

```json
{
  "tenantId": "uuid",
  "type": "ROUTE_CREATED | ROUTE_UPDATED | ROUTE_DELETED | VESSEL_CREATED | VESSEL_UPDATED | VESSEL_DELETED | INSTANCE_ROUTE_CREATED | INSTANCE_ROUTE_UPDATED | INSTANCE_ROUTE_DELETED | RPS_REWARD_CREATED | RPS_REWARD_UPDATED | RPS_REWARD_DELETED | MTS_CONFIG_CREATED | MTS_CONFIG_UPDATED | MTS_CONFIG_DELETED | TRADE_CONFIG_CREATED | TRADE_CONFIG_UPDATED | TRADE_CONFIG_DELETED | RANKINGS_CREATED | RANKINGS_UPDATED | RANKINGS_DELETED | KITE_CONFIG_CREATED | KITE_CONFIG_UPDATED | KITE_CONFIG_DELETED | IMPRINT_CONFIG_CREATED | IMPRINT_CONFIG_UPDATED | IMPRINT_CONFIG_DELETED",
  "resourceType": "route | vessel | instance-route | rps-reward | mts-config | trade-config | rankings | kite-config | imprint-config",
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
- `INSTANCE_ROUTE_CREATED`: Emitted when an instance route is created
- `INSTANCE_ROUTE_UPDATED`: Emitted when an instance route is updated
- `INSTANCE_ROUTE_DELETED`: Emitted when an instance route is deleted
- `RPS_REWARD_CREATED`: Emitted when an rps-reward is created
- `RPS_REWARD_UPDATED`: Emitted when an rps-reward is updated
- `RPS_REWARD_DELETED`: Emitted when an rps-reward is deleted
- `MTS_CONFIG_CREATED`: Emitted when an MTS config is created
- `MTS_CONFIG_UPDATED`: Emitted when an MTS config is updated
- `MTS_CONFIG_DELETED`: Emitted when an MTS config is deleted
- `TRADE_CONFIG_CREATED`: Emitted when a trade config is created
- `TRADE_CONFIG_UPDATED`: Emitted when a trade config is updated
- `TRADE_CONFIG_DELETED`: Emitted when a trade config is deleted
- `RANKINGS_CREATED`: Emitted when a rankings configuration is created
- `RANKINGS_UPDATED`: Emitted when a rankings configuration is updated
- `RANKINGS_DELETED`: Emitted when a rankings configuration is deleted
- `KITE_CONFIG_CREATED`: Emitted when a kite-config is created
- `KITE_CONFIG_UPDATED`: Emitted when a kite-config is updated
- `KITE_CONFIG_DELETED`: Emitted when a kite-config is deleted
- `IMPRINT_CONFIG_CREATED`: Emitted when an imprint config is created
- `IMPRINT_CONFIG_UPDATED`: Emitted when an imprint config is updated
- `IMPRINT_CONFIG_DELETED`: Emitted when an imprint config is deleted

## Transaction Semantics

Messages are buffered and emitted after successful database operations.
