# atlas-parties
Mushroom game parties Service

## Overview

A RESTful resource which provides parties services. The service automatically handles character deletion events from Kafka to maintain consistent party state and prevent orphaned references.

## Features

### Character Deletion Handling
The service listens for character deletion events via Kafka and automatically:
- Removes deleted characters from any parties they belong to
- Elects new party leaders when a deleted character was the party leader
- Disbands parties when the last member is deleted
- Handles duplicate events idempotently to prevent double-processing
- Logs all deletion operations for audit trails

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- EVENT_TOPIC_CHARACTER_STATUS - Kafka topic for character status events (defaults to configured topic)

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Requests

#### [GET] Get Parties

```/api/parties```

#### [GET] Get Parties - By Member Id

```/api/parties?filter[members.id]={memberId}```

#### [POST] Create Party

```/api/parties```

#### [GET] Get Party - By Id

```/api/parties/{partyId}```

#### [PATCH] Update Party

```/api/parties/{partyId}```

#### [GET] Get Party Members

```/api/parties/{partyId}/members```

#### [POST] Add Party Member

```/api/parties/{partyId}/members```

#### [GET] Get Party Member

```/api/parties/{partyId}/members/{memberId}```

## Automatic Operations

### Character Deletion Event Processing

The service automatically processes character deletion events from Kafka to maintain party integrity:

#### Event Source
- **Topic**: `EVENT_TOPIC_CHARACTER_STATUS`
- **Event Type**: `DELETED`
- **Consumer Group**: `character_status_event`

#### Processing Flow
1. **Event Validation**: Validates incoming deletion events for required fields and proper format
2. **Party Lookup**: Efficiently locates any party containing the deleted character
3. **Character Removal**: Removes the character from party membership
4. **Leader Election**: If deleted character was party leader, automatically elects new leader from remaining members
5. **Party Disbanding**: Disbands party if no members remain after character removal
6. **Event Emission**: Publishes party state change events for other services

#### Error Handling
- **Idempotency**: Duplicate deletion events are safely ignored using transaction ID tracking
- **Malformed Events**: Invalid events are logged and skipped without affecting service stability
- **Panic Recovery**: Graceful recovery from unexpected errors during processing
- **Non-Party Members**: Characters not in parties are handled as no-op operations

#### Monitoring and Logging

##### Structured Logging
All deletion operations include structured logging with:
- Transaction IDs for request tracing
- Character and party identifiers
- Operation outcomes and timing
- Error details for failed operations

##### Cache Metrics
- Cache hit/miss rates for character-party lookups
- Cache size and cleanup statistics
- Performance metrics for party operations

##### Audit Trail
Complete audit trail of:
- Character deletion event processing
- Party membership changes
- Leader election outcomes
- Party disbanding events

#### Performance Characteristics
- **Event Processing**: < 100ms per deletion event
- **Character Lookup**: O(1) with caching, O(n) worst case
- **Leader Election**: O(m) where m is party member count
- **Concurrent Processing**: Thread-safe with race condition protection