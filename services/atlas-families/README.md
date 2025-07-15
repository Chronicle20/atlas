# Atlas Family Service

A comprehensive microservice for managing hierarchical character relationships, reputation accumulation, and benefit redemption in the Atlas game server ecosystem.

## Overview

The Atlas Family Service enables players to form hierarchical family relationships where seniors can recruit up to 2 juniors, creating a tree-like structure that facilitates gameplay collaboration. The service manages reputation (Rep) accumulation through junior activities and provides mechanisms for benefit redemption.

### Key Features

- **Hierarchical Family Structure**: Senior-junior relationships with max 2 juniors per senior
- **Reputation System**: Activity-based reputation accumulation with daily limits
- **Level-Based Constraints**: Max 20 level difference and same-map requirements for linking
- **Event-Driven Architecture**: Kafka integration for inter-service communication
- **REST API**: JSON:API compliant endpoints for family management
- **Scheduled Operations**: Daily reputation reset with configurable timing
- **Audit Trail**: Comprehensive logging and event emission for all operations

### Business Rules

- **Family Structure**: Maximum 1 senior and 2 juniors per character
- **Linking Requirements**: Same map location and ≤20 level difference
- **Reputation Limits**: 5,000 daily Rep cap per junior for offline accumulation
- **Activity-Based Rep**: 2 Rep per 5 mob kills, expedition rewards × 10
- **Level Penalties**: Halved Rep gain if junior outlevels senior
- **Cycle Prevention**: No circular family relationships allowed

## Architecture

The service follows Atlas microservice architecture patterns:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   REST Layer    │    │   Kafka Layer   │    │  Scheduler      │
│   (JSON:API)    │    │   (Commands/    │    │  (Daily Reset)  │
│                 │    │    Events)      │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌─────────────────┐    ┌─────────────────┐
         │   Processor     │    │ Administrator   │
         │  (Business      │    │  (Data Coord.)  │
         │   Logic)        │    │                 │
         └─────────────────┘    └─────────────────┘
                                 │
         ┌─────────────────┐    ┌─────────────────┐
         │   Provider      │    │   Database      │
         │  (Data Access)  │    │  (PostgreSQL)   │
         │                 │    │                 │
         └─────────────────┘    └─────────────────┘
```

### Core Components

- **Domain Model**: Immutable FamilyMember with business logic validation
- **Processor**: Curried business operations (AddJunior, BreakLink, AwardRep, etc.)
- **Administrator**: High-level coordination and transaction management
- **Provider**: Database access with functional composition patterns
- **Producer**: Kafka event emission for audit and integration
- **Consumer**: Command processing from external services
- **Scheduler**: Daily reputation reset with configurable timing

## Prerequisites

- **Go 1.24+**: Required for generics support
- **PostgreSQL 12+**: Primary database (SQLite supported for testing)
- **Kafka 2.8+**: Message broker for event-driven architecture
- **Docker**: For containerized deployment
- **Make**: For build automation (optional)

## Installation

### Local Development

1. **Clone the repository**:
   ```bash
   git clone https://github.com/Chronicle20/atlas-families.git
   cd atlas-families
   ```

2. **Install dependencies**:
   ```bash
   cd atlas.com/family
   go mod download
   ```

3. **Set up environment variables**:
   ```bash
   export DB_HOST=localhost
   export DB_PORT=5432
   export DB_USER=atlas
   export DB_PASSWORD=password
   export DB_NAME=atlas_family
   export DB_SCHEMA=public
   
   export KAFKA_BROKERS=localhost:9092
   export COMMAND_TOPIC_FAMILY=family-commands
   export EVENT_TOPIC_FAMILY_STATUS=family-status-events
   export EVENT_TOPIC_FAMILY_REPUTATION=family-reputation-events
   
   export LOG_LEVEL=info
   export JAEGER_HOST=localhost:14268
   ```

4. **Run database migrations**:
   ```bash
   # Database migrations are automatically run on service startup
   go run main.go
   ```

### Docker Deployment

1. **Build the Docker image**:
   ```bash
   docker build -t atlas-family:latest .
   ```

2. **Run with Docker Compose**:
   ```bash
   docker-compose up -d
   ```

### Production Deployment

1. **Build for production**:
   ```bash
   CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o atlas-family main.go
   ```

2. **Configure environment variables** (see Environment section below)

3. **Deploy with your orchestration platform** (Kubernetes, Docker Swarm, etc.)

## Configuration

### Environment Variables

#### Database Configuration
- `DB_HOST`: Database host (default: localhost)
- `DB_PORT`: Database port (default: 5432)
- `DB_USER`: Database username (required)
- `DB_PASSWORD`: Database password (required)
- `DB_NAME`: Database name (required)
- `DB_SCHEMA`: Database schema (default: public)
- `DB_SSL_MODE`: SSL mode (default: disable)

#### Kafka Configuration
- `KAFKA_BROKERS`: Comma-separated Kafka brokers (required)
- `COMMAND_TOPIC_FAMILY`: Family command topic name
- `EVENT_TOPIC_FAMILY_STATUS`: Family status event topic name
- `EVENT_TOPIC_FAMILY_REPUTATION`: Family reputation event topic name
- `EVENT_TOPIC_FAMILY_ERRORS`: Family error event topic name

#### Scheduler Configuration
- `REPUTATION_RESET_HOUR`: Hour for daily reset (0-23, default: 0)
- `REPUTATION_RESET_MINUTE`: Minute for daily reset (0-59, default: 0)
- `REPUTATION_RESET_TIMEZONE`: Timezone for reset (default: UTC)

#### Logging & Monitoring
- `LOG_LEVEL`: Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace, default: Info)
- `JAEGER_HOST`: Jaeger tracer host:port for distributed tracing

#### Multi-tenancy Headers
- `TENANT_ID`: Tenant identifier (required in requests)
- `REGION`: Game region (required in requests)
- `MAJOR_VERSION`: Game major version (required in requests)
- `MINOR_VERSION`: Game minor version (required in requests)

## Usage

### Starting the Service

```bash
# Development mode
go run main.go

# Production mode
./atlas-family
```

### Service Health

The service provides health check endpoints:
- Service starts on port 8080 (configurable)
- Database connection is validated on startup
- Kafka connectivity is verified during initialization
- Scheduler status is logged on startup

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests (requires database)
go test -tags=integration ./...

# Run specific test suites
go test ./family/... -v
```

## API Documentation

### REST Endpoints

The service provides JSON:API compliant endpoints for family management. All endpoints follow the JSON:API specification and require specific headers for authentication and transaction tracking.

#### Required Headers

All requests require the following headers:
```
Content-Type: application/json
X-Tenant-Id: {tenant-id}
X-Transaction-Id: {transaction-id}
X-World-Id: {world-id}
```

#### Authentication

The service integrates with the Atlas authentication system through:
- Multi-tenant headers for request isolation
- Transaction ID tracking for idempotency
- Audit trail for all operations

---

### 1. Add Junior to Family

Add a junior member to a senior's family.

**Endpoint:** `POST /api/families/{characterId}/juniors`

**Path Parameters:**
- `characterId` (uint32): The senior character's ID

**Request Body:**
```json
{
  "data": {
    "type": "familyMembers",
    "attributes": {
      "juniorId": 12345
    }
  }
}
```

**Success Response (201 Created):**
```json
{
  "data": {
    "id": "67890",
    "type": "familyMembers",
    "characterId": 67890,
    "tenantId": "550e8400-e29b-41d4-a716-446655440000",
    "seniorId": null,
    "juniorIds": [12345],
    "rep": 150,
    "dailyRep": 25,
    "level": 45,
    "world": 1,
    "createdAt": "2025-01-15T10:30:00Z",
    "updatedAt": "2025-01-15T14:22:00Z"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid character ID, missing junior ID, or self-reference
- `404 Not Found`: Senior or junior character not found
- `409 Conflict`: Senior has too many juniors, junior already linked, level difference too large, or not on same map

**Example cURL:**
```bash
curl -X POST "https://api.atlas.com/api/families/67890/juniors" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Transaction-Id: tx-12345-add-junior" \
  -H "X-World-Id: 1" \
  -d '{
    "data": {
      "type": "familyMembers",
      "attributes": {
        "juniorId": 12345
      }
    }
  }'
```

---

### 2. Break Family Link

Break a family link for a character (either as senior or junior).

**Endpoint:** `DELETE /api/families/links/{characterId}`

**Path Parameters:**
- `characterId` (uint32): The character's ID whose link to break

**Query Parameters:**
- `reason` (string, optional): Reason for breaking the link

**Success Response (200 OK):**
```json
{
  "data": [
    {
      "id": "67890",
      "type": "familyMembers",
      "characterId": 67890,
      "tenantId": "550e8400-e29b-41d4-a716-446655440000",
      "seniorId": null,
      "juniorIds": [],
      "rep": 150,
      "dailyRep": 25,
      "level": 45,
      "world": 1,
      "createdAt": "2025-01-15T10:30:00Z",
      "updatedAt": "2025-01-15T14:25:00Z"
    }
  ]
}
```

**Error Responses:**
- `400 Bad Request`: Invalid character ID or missing transaction ID
- `404 Not Found`: Character not found
- `409 Conflict`: No link to break

**Example cURL:**
```bash
curl -X DELETE "https://api.atlas.com/api/families/links/67890?reason=Player%20requested" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Transaction-Id: tx-12345-break-link" \
  -H "X-World-Id: 1"
```

---

### 3. Get Family Tree

Retrieve the complete family tree for a character.

**Endpoint:** `GET /api/families/tree/{characterId}`

**Path Parameters:**
- `characterId` (uint32): The character's ID to get the tree for

**Success Response (200 OK):**
```json
{
  "data": {
    "id": "67890",
    "type": "familyTrees",
    "members": [
      {
        "id": "54321",
        "type": "familyMembers",
        "characterId": 54321,
        "tenantId": "550e8400-e29b-41d4-a716-446655440000",
        "seniorId": null,
        "juniorIds": [67890],
        "rep": 500,
        "dailyRep": 100,
        "level": 60,
        "world": 1,
        "createdAt": "2025-01-10T09:00:00Z",
        "updatedAt": "2025-01-15T14:20:00Z"
      },
      {
        "id": "67890",
        "type": "familyMembers",
        "characterId": 67890,
        "tenantId": "550e8400-e29b-41d4-a716-446655440000",
        "seniorId": 54321,
        "juniorIds": [12345],
        "rep": 150,
        "dailyRep": 25,
        "level": 45,
        "world": 1,
        "createdAt": "2025-01-15T10:30:00Z",
        "updatedAt": "2025-01-15T14:22:00Z"
      },
      {
        "id": "12345",
        "type": "familyMembers",
        "characterId": 12345,
        "tenantId": "550e8400-e29b-41d4-a716-446655440000",
        "seniorId": 67890,
        "juniorIds": [],
        "rep": 0,
        "dailyRep": 0,
        "level": 25,
        "world": 1,
        "createdAt": "2025-01-15T14:22:00Z",
        "updatedAt": "2025-01-15T14:22:00Z"
      }
    ]
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid character ID
- `404 Not Found`: Character not found

**Example cURL:**
```bash
curl -X GET "https://api.atlas.com/api/families/tree/67890" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Transaction-Id: tx-12345-get-tree" \
  -H "X-World-Id: 1"
```

---

### 4. Process Reputation Activity

Process reputation-generating activities (mob kills, expeditions).

**Endpoint:** `POST /api/families/reputation/activities`

**Request Body:**
```json
{
  "data": {
    "type": "activities",
    "attributes": {
      "characterId": 12345,
      "activityType": "mob_kill",
      "amount": 10
    }
  }
}
```

**Activity Types:**
- `mob_kill`: Awards 2 Rep per 5 kills to the character's senior
- `expedition`: Awards coin amount × 10 Rep to the character's senior

**Success Response (200 OK):**
```json
{
  "data": {
    "id": "67890",
    "type": "familyMembers",
    "characterId": 67890,
    "tenantId": "550e8400-e29b-41d4-a716-446655440000",
    "seniorId": 54321,
    "juniorIds": [12345],
    "rep": 154,
    "dailyRep": 29,
    "level": 45,
    "world": 1,
    "createdAt": "2025-01-15T10:30:00Z",
    "updatedAt": "2025-01-15T14:30:00Z"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid character ID, missing activity type, or invalid amount
- `404 Not Found`: Character not found
- `409 Conflict`: Daily reputation cap exceeded

**Example cURL:**
```bash
curl -X POST "https://api.atlas.com/api/families/reputation/activities" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Transaction-Id: tx-12345-activity" \
  -H "X-World-Id: 1" \
  -d '{
    "data": {
      "type": "activities",
      "attributes": {
        "characterId": 12345,
        "activityType": "mob_kill",
        "amount": 10
      }
    }
  }'
```

---

### 5. Redeem Reputation

Redeem reputation for buffs or teleportation.

**Endpoint:** `POST /api/families/reputation/redeem`

**Request Body:**
```json
{
  "data": {
    "type": "redemption",
    "attributes": {
      "characterId": 67890,
      "amount": 50,
      "reason": "2x EXP buff for 1 hour"
    }
  }
}
```

**Success Response (200 OK):**
```json
{
  "data": {
    "id": "67890",
    "type": "familyMembers",
    "characterId": 67890,
    "tenantId": "550e8400-e29b-41d4-a716-446655440000",
    "seniorId": 54321,
    "juniorIds": [12345],
    "rep": 100,
    "dailyRep": 29,
    "level": 45,
    "world": 1,
    "createdAt": "2025-01-15T10:30:00Z",
    "updatedAt": "2025-01-15T14:35:00Z"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid character ID, missing amount, or missing reason
- `404 Not Found`: Character not found
- `409 Conflict`: Insufficient reputation

**Example cURL:**
```bash
curl -X POST "https://api.atlas.com/api/families/reputation/redeem" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Transaction-Id: tx-12345-redeem" \
  -H "X-World-Id: 1" \
  -d '{
    "data": {
      "type": "redemption",
      "attributes": {
        "characterId": 67890,
        "amount": 50,
        "reason": "2x EXP buff for 1 hour"
      }
    }
  }'
```

---

### 6. Get Reputation Info

Get current reputation information for a character.

**Endpoint:** `GET /api/families/reputation/{characterId}`

**Path Parameters:**
- `characterId` (uint32): The character's ID

**Success Response (200 OK):**
```json
{
  "data": {
    "id": "67890",
    "type": "reputation",
    "characterId": 67890,
    "availableRep": 100,
    "dailyRep": 29,
    "dailyRepLimit": 5000,
    "remainingDailyRep": 4971
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid character ID
- `404 Not Found`: Character not found

**Example cURL:**
```bash
curl -X GET "https://api.atlas.com/api/families/reputation/67890" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Transaction-Id: tx-12345-get-rep" \
  -H "X-World-Id: 1"
```

---

### 7. Get Member Location

Get the current location and online status of a family member.

**Endpoint:** `GET /api/families/location/{characterId}`

**Path Parameters:**
- `characterId` (uint32): The character's ID

**Success Response (200 OK):**
```json
{
  "data": {
    "id": "67890",
    "type": "locations",
    "characterId": 67890,
    "world": 1,
    "channel": 3,
    "map": 100000001,
    "online": true
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid character ID
- `404 Not Found`: Character not found

**Example cURL:**
```bash
curl -X GET "https://api.atlas.com/api/families/location/67890" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Transaction-Id: tx-12345-get-location" \
  -H "X-World-Id: 1"
```

---

### Error Response Format

All error responses follow the JSON:API error format:

```json
{
  "error": {
    "status": 400,
    "title": "Bad Request",
    "detail": "Junior ID is required"
  }
}
```

### Common HTTP Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request format or missing required fields
- `404 Not Found`: Resource not found
- `409 Conflict`: Business rule violation or constraint failure
- `500 Internal Server Error`: Server error

### Rate Limiting

The API implements rate limiting to prevent abuse:
- 100 requests per minute per IP address
- 500 requests per minute per tenant
- Burst limit of 20 requests per second

Rate limit headers are included in all responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1641024000
```

## Kafka Integration

### Topics and Message Formats

The Atlas Family Service uses Kafka for event-driven communication with other services. All messages follow a consistent generic structure with type-safe message bodies.

#### Topic Configuration

Topics are configured via environment variables:

| Topic Variable | Default Topic Name | Purpose |
|----------------|-------------------|---------|
| `COMMAND_TOPIC_FAMILY` | `family-commands` | Commands from other services |
| `EVENT_TOPIC_FAMILY_STATUS` | `family-status-events` | Family relationship events |
| `EVENT_TOPIC_FAMILY_REPUTATION` | `family-reputation-events` | Reputation change events |
| `EVENT_TOPIC_FAMILY_ERRORS` | `family-error-events` | Error events |

#### Message Structure

All Kafka messages use generic wrappers for type safety:

**Command Structure:**
```go
type Command[E any] struct {
    TransactionId string `json:"transactionId"`
    TenantId      string `json:"tenantId"`
    WorldId       byte   `json:"worldId"`
    CharacterId   uint32 `json:"characterId"`
    Type          string `json:"type"`
    Body          E      `json:"body"`
}
```

**Event Structure:**
```go
type Event[E any] struct {
    WorldId     byte   `json:"worldId"`
    CharacterId uint32 `json:"characterId"`
    Type        string `json:"type"`
    Body        E      `json:"body"`
}
```

---

### Commands (Consumed)

These are commands that the Family Service consumes from other services:

#### 1. ADD_JUNIOR
**Purpose**: Add a junior to a senior's family  
**Command Type**: `ADD_JUNIOR`

**Body Structure:**
```json
{
    "juniorId": 12345,
    "seniorLevel": 50,
    "seniorWorld": 1,
    "juniorLevel": 30,
    "juniorWorld": 1
}
```

**Example Message:**
```json
{
    "transactionId": "tx-add-junior-12345",
    "tenantId": "550e8400-e29b-41d4-a716-446655440000",
    "worldId": 1,
    "characterId": 67890,
    "type": "ADD_JUNIOR",
    "body": {
        "juniorId": 12345,
        "seniorLevel": 50,
        "seniorWorld": 1,
        "juniorLevel": 30,
        "juniorWorld": 1
    }
}
```

#### 2. REMOVE_MEMBER
**Purpose**: Remove a member from the family  
**Command Type**: `REMOVE_MEMBER`

**Body Structure:**
```json
{
    "targetId": 12345,
    "reason": "Inactive player"
}
```

#### 3. BREAK_LINK
**Purpose**: Break a family link  
**Command Type**: `BREAK_LINK`

**Body Structure:**
```json
{
    "reason": "Player requested"
}
```

#### 4. DEDUCT_REP
**Purpose**: Deduct reputation for buff/teleport usage  
**Command Type**: `DEDUCT_REP`

**Body Structure:**
```json
{
    "amount": 100,
    "reason": "2x EXP buff for 1 hour"
}
```

#### 5. REGISTER_KILL_ACTIVITY
**Purpose**: Register mob kill activity for reputation  
**Command Type**: `REGISTER_KILL_ACTIVITY`

**Body Structure:**
```json
{
    "killCount": 10,
    "timestamp": "2025-01-15T14:30:00Z"
}
```

**Reputation Calculation**: 2 Rep per 5 kills awarded to the character's senior

#### 6. REGISTER_EXPEDITION_ACTIVITY
**Purpose**: Register expedition activity for reputation  
**Command Type**: `REGISTER_EXPEDITION_ACTIVITY`

**Body Structure:**
```json
{
    "coinReward": 5000,
    "timestamp": "2025-01-15T14:30:00Z"
}
```

**Reputation Calculation**: Coin reward × 10 Rep awarded to the character's senior

---

### Events (Produced)

These are events that the Family Service produces for other services:

#### Status Events (EVENT_TOPIC_FAMILY_STATUS)

##### 1. LINK_CREATED
**Purpose**: Notify when a family link is created  
**Event Type**: `LINK_CREATED`

**Body Structure:**
```json
{
    "seniorId": 67890,
    "juniorId": 12345,
    "timestamp": "2025-01-15T14:30:00Z"
}
```

##### 2. LINK_BROKEN
**Purpose**: Notify when a family link is broken  
**Event Type**: `LINK_BROKEN`

**Body Structure:**
```json
{
    "seniorId": 67890,
    "juniorId": 12345,
    "reason": "Player requested",
    "timestamp": "2025-01-15T14:30:00Z"
}
```

##### 3. TREE_DISSOLVED
**Purpose**: Notify when an entire family tree is dissolved  
**Event Type**: `TREE_DISSOLVED`

**Body Structure:**
```json
{
    "seniorId": 67890,
    "affectedIds": [12345, 54321],
    "reason": "Senior removed",
    "timestamp": "2025-01-15T14:30:00Z"
}
```

#### Reputation Events (EVENT_TOPIC_FAMILY_REPUTATION)

##### 1. REP_GAINED
**Purpose**: Notify when reputation is gained  
**Event Type**: `REP_GAINED`

**Body Structure:**
```json
{
    "repGained": 4,
    "dailyRep": 104,
    "source": "mob_kill",
    "timestamp": "2025-01-15T14:30:00Z"
}
```

##### 2. REP_REDEEMED
**Purpose**: Notify when reputation is redeemed  
**Event Type**: `REP_REDEEMED`

**Body Structure:**
```json
{
    "repRedeemed": 100,
    "reason": "2x EXP buff for 1 hour",
    "timestamp": "2025-01-15T14:30:00Z"
}
```

##### 3. REP_CAPPED
**Purpose**: Notify when daily reputation cap is reached  
**Event Type**: `REP_CAPPED`

**Body Structure:**
```json
{
    "attemptedAmount": 10,
    "dailyRep": 5000,
    "source": "mob_kill",
    "timestamp": "2025-01-15T14:30:00Z"
}
```

##### 4. REP_RESET
**Purpose**: Notify when daily reputation is reset  
**Event Type**: `REP_RESET`

**Body Structure:**
```json
{
    "previousDailyRep": 2500,
    "timestamp": "2025-01-16T00:00:00Z"
}
```

#### Error Events (EVENT_TOPIC_FAMILY_ERRORS)

##### 1. REP_ERROR
**Purpose**: Notify about reputation operation errors  
**Event Type**: `REP_ERROR`

**Body Structure:**
```json
{
    "errorCode": "INSUFFICIENT_REP",
    "errorMessage": "Character has insufficient reputation",
    "amount": 100,
    "timestamp": "2025-01-15T14:30:00Z"
}
```

##### 2. LINK_ERROR
**Purpose**: Notify about family link operation errors  
**Event Type**: `LINK_ERROR`

**Body Structure:**
```json
{
    "seniorId": 67890,
    "juniorId": 12345,
    "errorCode": "LEVEL_DIFFERENCE_TOO_LARGE",
    "errorMessage": "Level difference exceeds maximum allowed",
    "timestamp": "2025-01-15T14:30:00Z"
}
```

#### Buff Events (EVENT_TOPIC_FAMILY_BUFFS)

##### 1. BUFF_REDEEMED
**Purpose**: Notify when a buff is redeemed with reputation  
**Event Type**: `BUFF_REDEEMED`

**Body Structure:**
```json
{
    "buffType": "2x_exp",
    "repCost": 100,
    "duration": 3600,
    "timestamp": "2025-01-15T14:30:00Z"
}
```

##### 2. TELEPORT_USED
**Purpose**: Notify when teleport is used with reputation  
**Event Type**: `TELEPORT_USED`

**Body Structure:**
```json
{
    "targetId": 12345,
    "repCost": 50,
    "world": 1,
    "map": 100000001,
    "timestamp": "2025-01-15T14:30:00Z"
}
```

---

### Message Partitioning

Messages are partitioned by `characterId` to ensure:
- Ordered processing of operations for each character
- Balanced load distribution across Kafka partitions
- Consistent routing for family-related operations

### Consumer Groups

The service uses the following consumer groups:
- `family-command-processor`: Processes commands from other services
- `family-activity-processor`: Processes activity-based reputation events

### Producer Configuration

Events are produced with:
- **Idempotency**: All operations use transactionId for deduplication
- **Ordering**: Per-character ordering maintained through partitioning
- **Reliability**: At-least-once delivery with proper error handling
- **Headers**: Span tracing and tenant context included

### Error Handling

The service implements comprehensive error handling:
- **Validation Errors**: Sent to error topic with detailed error codes
- **Business Rule Violations**: Logged and sent as error events
- **System Errors**: Logged for monitoring and alerting
- **Dead Letter Queue**: Failed messages sent to DLQ for manual inspection

---

## Development

### Architecture Guidelines

- **Immutable Models**: Domain objects are immutable with private fields
- **Functional Composition**: Curried functions enable flexible composition
- **Provider Pattern**: Lazy evaluation for database operations
- **Event-Driven**: Kafka integration for inter-service communication
- **Pure Functions**: Business logic separated from side effects

### Code Organization

```
atlas.com/family/
├── main.go                  # Service entry point
├── database/               # Database connection and transactions
├── family/                 # Core domain implementation
│   ├── model.go           # Immutable domain models
│   ├── entity.go          # Database entities
│   ├── builder.go         # Fluent builders
│   ├── processor.go       # Business logic
│   ├── provider.go        # Data access
│   ├── administrator.go   # High-level coordination
│   ├── producer.go        # Kafka producers
│   ├── resource.go        # REST endpoints
│   └── rest.go           # REST models
├── kafka/                 # Kafka integration
│   ├── consumer/         # Command consumers
│   ├── producer/         # Event producers
│   └── message/          # Message definitions
├── scheduler/            # Scheduled operations
└── logger/              # Logging configuration
```

### Contributing

1. Follow Atlas microservice architecture patterns
2. Maintain immutability in domain models
3. Use functional composition where appropriate
4. Add comprehensive tests for new features
5. Follow Go code style guidelines
6. Update documentation for API changes

## Monitoring

### Metrics

The service provides metrics for:
- Request latency and throughput
- Database operation performance
- Kafka message processing rates
- Scheduler execution status
- Error rates and types

### Logging

Structured logging includes:
- Request/response correlation IDs
- Transaction tracking
- Business rule validation results
- Error context and stack traces
- Performance metrics

### Tracing

Distributed tracing through Jaeger:
- Request flow across services
- Database query performance
- Kafka message processing
- Scheduler execution traces

## Troubleshooting

### Common Issues

1. **Database Connection Failures**
   - Verify database credentials and connectivity
   - Check database schema existence
   - Validate migration completion

2. **Kafka Connectivity Issues**
   - Verify broker addresses and ports
   - Check topic existence and permissions
   - Validate consumer group configurations

3. **Scheduler Not Running**
   - Check timezone configuration
   - Verify database connectivity
   - Review scheduler logs for errors

4. **Transaction Failures**
   - Ensure unique transaction IDs
   - Check for concurrent operations
   - Review business rule validations

### Performance Optimization

- **Database Indexing**: Ensure proper indexes on characterId, tenantId, and seniorId
- **Connection Pooling**: Configure appropriate database connection limits
- **Kafka Partitioning**: Use characterId for message partitioning
- **Batch Operations**: Utilize batch reputation reset for performance

## License

This project is part of the Atlas game server ecosystem. See the main Atlas repository for licensing information.

## Support

For issues and questions:
- Create issues in the GitHub repository
- Check the Atlas documentation
- Contact the Atlas development team

## Database Schema

### Table: `family_members`

The core table that stores all family member information and relationships.

#### Schema Definition

```sql
CREATE TABLE family_members (
    id SERIAL PRIMARY KEY,
    character_id INTEGER NOT NULL UNIQUE,
    tenant_id UUID NOT NULL,
    senior_id INTEGER,
    junior_ids INTEGER[],
    rep INTEGER DEFAULT 0,
    daily_rep INTEGER DEFAULT 0,
    level SMALLINT NOT NULL,
    world SMALLINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### Field Descriptions

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | `SERIAL` | PRIMARY KEY | Auto-incrementing unique identifier |
| `character_id` | `INTEGER` | NOT NULL, UNIQUE | Game character ID (unique across all family members) |
| `tenant_id` | `UUID` | NOT NULL | Multi-tenant identifier for data isolation |
| `senior_id` | `INTEGER` | NULL, FOREIGN KEY | Reference to senior's character_id (null for root members) |
| `junior_ids` | `INTEGER[]` | NULL | Array of junior character IDs (max 2 elements) |
| `rep` | `INTEGER` | DEFAULT 0, >= 0 | Total accumulated reputation points |
| `daily_rep` | `INTEGER` | DEFAULT 0, >= 0, <= 5000 | Daily reputation gained (resets daily) |
| `level` | `SMALLINT` | NOT NULL, > 0 | Character level for link validation |
| `world` | `SMALLINT` | NOT NULL | Game world/server identifier |
| `created_at` | `TIMESTAMP` | NOT NULL | Record creation timestamp |
| `updated_at` | `TIMESTAMP` | NOT NULL | Last modification timestamp |

#### Indexes

Performance-optimized indexes for common query patterns:

```sql
-- Composite index for tenant-specific character lookups
CREATE INDEX idx_family_members_tenant_character 
ON family_members(tenant_id, character_id);

-- Index for senior-based queries (partial index excluding nulls)
CREATE INDEX idx_family_members_senior_id 
ON family_members(senior_id) WHERE senior_id IS NOT NULL;

-- Index for world-based queries
CREATE INDEX idx_family_members_world 
ON family_members(world);

-- Index for temporal queries and daily reset operations
CREATE INDEX idx_family_members_updated_at 
ON family_members(updated_at);
```

#### Database Constraints

Business rule enforcement at the database level:

```sql
-- Ensure maximum 2 juniors per senior
ALTER TABLE family_members 
ADD CONSTRAINT check_junior_count 
CHECK (array_length(junior_ids, 1) IS NULL OR array_length(junior_ids, 1) <= 2);

-- Ensure reputation is non-negative
ALTER TABLE family_members 
ADD CONSTRAINT check_rep_non_negative 
CHECK (rep >= 0);

-- Ensure daily reputation is non-negative
ALTER TABLE family_members 
ADD CONSTRAINT check_daily_rep_non_negative 
CHECK (daily_rep >= 0);

-- Enforce daily reputation limit (5000 per day)
ALTER TABLE family_members 
ADD CONSTRAINT check_daily_rep_limit 
CHECK (daily_rep <= 5000);

-- Ensure level is positive
ALTER TABLE family_members 
ADD CONSTRAINT check_level_positive 
CHECK (level > 0);

-- Prevent self-referential senior relationships
ALTER TABLE family_members 
ADD CONSTRAINT check_no_self_senior 
CHECK (senior_id != character_id);
```

### Relationships

#### Hierarchical Structure

The family system implements a tree-like hierarchy using self-referential relationships:

```
┌─────────────────┐
│   Root Member   │ <- senior_id: NULL
│   (No Senior)   │
└─────────────────┘
         │
         ├─────────────────┐
         │                 │
┌─────────────────┐ ┌─────────────────┐
│   Junior #1     │ │   Junior #2     │ <- senior_id: Root's character_id
│                 │ │                 │
└─────────────────┘ └─────────────────┘
         │
         ├─────────────────┐
         │                 │
┌─────────────────┐ ┌─────────────────┐
│ Junior #1's     │ │ Junior #1's     │ <- senior_id: Junior #1's character_id
│ Junior #1       │ │ Junior #2       │
└─────────────────┘ └─────────────────┘
```

#### Relationship Types

1. **Senior-Junior Relationship**
   - Each member can have at most 1 senior (`senior_id`)
   - Each member can have at most 2 juniors (`junior_ids[]`)
   - Relationship is bidirectional but stored in both directions

2. **Root Members**
   - Members with `senior_id = NULL` are root members
   - Root members can still have juniors

3. **Leaf Members**
   - Members with empty `junior_ids[]` are leaf members
   - Leaf members can still have a senior

#### Relationship Constraints

- **Level Difference**: Maximum 20 levels between senior and junior
- **Same World**: Senior and junior must be in the same world for initial linking
- **No Cycles**: Circular relationships are prevented by business logic
- **Cascade Operations**: Removing a senior dissolves all junior relationships

### Query Patterns

#### Common Queries

1. **Get Family Member by Character ID**
```sql
SELECT * FROM family_members 
WHERE tenant_id = ? AND character_id = ?;
```

2. **Get All Juniors of a Senior**
```sql
SELECT * FROM family_members 
WHERE tenant_id = ? AND senior_id = ?;
```

3. **Get Complete Family Tree**
```sql
-- Recursive CTE to get entire family tree
WITH RECURSIVE family_tree AS (
    -- Base case: start with the target character
    SELECT * FROM family_members 
    WHERE tenant_id = ? AND character_id = ?
    
    UNION ALL
    
    -- Recursive case: find all related members
    SELECT fm.* FROM family_members fm
    JOIN family_tree ft ON (
        fm.senior_id = ft.character_id OR 
        ft.senior_id = fm.character_id OR
        fm.character_id = ANY(ft.junior_ids) OR
        ft.character_id = ANY(fm.junior_ids)
    )
    WHERE fm.tenant_id = ?
)
SELECT DISTINCT * FROM family_tree;
```

4. **Get Members for Daily Reset**
```sql
SELECT character_id, daily_rep FROM family_members 
WHERE tenant_id = ? AND daily_rep > 0;
```

5. **Get Members by World**
```sql
SELECT * FROM family_members 
WHERE tenant_id = ? AND world = ?;
```

#### Performance Considerations

- **Indexing**: All common queries use indexed fields for optimal performance
- **Partitioning**: Consider partitioning by `tenant_id` for large-scale deployments
- **Array Operations**: PostgreSQL array operations on `junior_ids` are optimized with GIN indexes if needed
- **Batch Operations**: Daily reset operations use batch updates for efficiency

### Data Integrity

#### Referential Integrity

- **Foreign Key Simulation**: `senior_id` references `character_id` but not enforced as FK to avoid circular dependencies
- **Application-Level Validation**: Business rules enforced in the service layer
- **Cascade Handling**: Removal operations handled by service logic, not database cascades

#### Consistency Guarantees

- **Transaction Isolation**: All family operations use database transactions
- **Idempotent Operations**: Transaction IDs ensure operations can be safely retried
- **Atomic Updates**: Complex operations (like dissolving subtrees) use transactions

#### Audit Trail

- **Timestamps**: `created_at` and `updated_at` track record lifecycle
- **Event Emission**: All changes emit Kafka events for external audit systems
- **Transaction Tracking**: All operations include transaction IDs for traceability

### Migration Strategy

#### Initial Schema Creation

The schema is automatically created on service startup through GORM migrations:

```go
// Migration creates the family_members table with proper indexes and constraints
func Migration(db *gorm.DB) error {
    err := db.AutoMigrate(&Entity{})
    if err != nil {
        return err
    }
    
    // Add indexes and constraints...
    return nil
}
```

#### Schema Evolution

Future schema changes should:
1. Be backward-compatible where possible
2. Use database migrations for structural changes
3. Maintain index performance
4. Consider tenant data isolation
5. Test with production-like data volumes

---

**Atlas Family Service** - Enabling collaborative gameplay through hierarchical family relationships.
