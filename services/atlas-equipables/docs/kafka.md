# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Equipable Command | COMMAND_TOPIC_EQUIPABLE | Commands for equipable operations |

## Topics Produced

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Equipable Status Event | EVENT_TOPIC_EQUIPABLE_STATUS | Status events for equipable lifecycle |

## Message Types

### Command (Consumed)

```
Command[AttributeBody]
```

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Equipable ID |
| type | string | Command type |
| body | AttributeBody | Command payload |

**Command Types**

| Type | Description |
|------|-------------|
| CHANGE | Update equipable attributes |

**AttributeBody**

| Field | Type |
|-------|------|
| strength | uint16 |
| dexterity | uint16 |
| intelligence | uint16 |
| luck | uint16 |
| hp | uint16 |
| mp | uint16 |
| weaponAttack | uint16 |
| magicAttack | uint16 |
| weaponDefense | uint16 |
| magicDefense | uint16 |
| accuracy | uint16 |
| avoidability | uint16 |
| hands | uint16 |
| speed | uint16 |
| jump | uint16 |
| slots | uint16 |
| ownerName | string |
| locked | bool |
| spikes | bool |
| karmaUsed | bool |
| cold | bool |
| canBeTraded | bool |
| levelType | uint8 |
| level | uint8 |
| experience | uint32 |
| hammersApplied | uint32 |
| expiration | time.Time |

### StatusEvent (Produced)

```
StatusEvent[AttributeBody | DeletedStatusEventBody]
```

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Equipable ID |
| type | string | Event type |
| body | varies | Event payload |

**Event Types**

| Type | Body Type | Description |
|------|-----------|-------------|
| CREATED | AttributeBody | Equipable created |
| UPDATED | AttributeBody | Equipable updated |
| DELETED | DeletedStatusEventBody | Equipable deleted |

**DeletedStatusEventBody**

Empty object.

## Transaction Semantics

- All status events are produced within database transactions
- Message key is the equipable ID for partition ordering
