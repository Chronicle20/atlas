# atlas-marriages
Mushroom game marriages Service

## Overview

The `atlas-marriages` service manages the complete lifecycle of character marriages within the game world. It handles proposals, engagement tracking, ceremony coordination, and historical data storage following Atlas microservice architecture patterns.

### Key Features

- **Marriage Lifecycle Management**: Proposals, engagements, ceremonies, and divorces
- **Eligibility Validation**: Level requirements, relationship constraints, and cooldown management
- **Ceremony Orchestration**: Scheduling, invitee management, and state transitions
- **Event-Driven Architecture**: Kafka messaging for inter-service communication
- **Historical Tracking**: Complete audit trail of all marriage-related activities

## Environment Variables

- `JAEGER_HOST` - Jaeger [host]:[port]
- `LOG_LEVEL` - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- `COMMAND_TOPIC_MARRIAGE` - Kafka topic for marriage commands
- `EVENT_TOPIC_MARRIAGE_STATUS` - Kafka topic for marriage events

## Deployment and Configuration Guide

### Prerequisites

Before deploying the atlas-marriages service, ensure you have the following dependencies available:

- **PostgreSQL Database**: Primary data store for marriage, proposal, and ceremony data
- **Apache Kafka**: Message broker for event-driven communication
- **Jaeger**: Distributed tracing system for observability
- **Docker**: Container runtime for deployment

### Environment Configuration

The service requires the following environment variables to be configured:

#### Core Configuration
```bash
# Service Configuration
LOG_LEVEL=Info
REST_PORT=8080

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=atlas_marriages
DB_USER=atlas_user
DB_PASSWORD=atlas_password
DB_SSL_MODE=disable

# Kafka Configuration
KAFKA_BROKERS=localhost:9092
COMMAND_TOPIC_MARRIAGE=command.marriage
EVENT_TOPIC_MARRIAGE_STATUS=event.marriage.status

# Tracing Configuration
JAEGER_HOST=localhost:14268
JAEGER_SERVICE_NAME=atlas-marriages
```

#### Multi-Tenant Configuration
```bash
# Tenant Context (injected by infrastructure)
TENANT_ID=083839c6-c47c-42a6-9585-76492795d123
REGION=GMS
MAJOR_VERSION=83
MINOR_VERSION=1
```

### Database Setup

1. **Create Database Schema**:
```sql
CREATE DATABASE atlas_marriages;
CREATE USER atlas_user WITH ENCRYPTED PASSWORD 'atlas_password';
GRANT ALL PRIVILEGES ON DATABASE atlas_marriages TO atlas_user;
```

2. **Migration**: The service automatically runs database migrations on startup. The following tables will be created:
   - `marriages` - Stores marriage records and states
   - `proposals` - Tracks proposal history and cooldowns
   - `ceremonies` - Manages ceremony scheduling and states
   - `invitees` - Stores ceremony invitee information

### Kafka Topic Configuration

Create the required Kafka topics with appropriate partitioning:

```bash
# Command topic (for receiving marriage commands)
kafka-topics --create --topic command.marriage --partitions 12 --replication-factor 3

# Event topic (for publishing marriage events)
kafka-topics --create --topic event.marriage.status --partitions 12 --replication-factor 3
```

### Docker Deployment

#### Using Docker Compose

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  atlas-marriages:
    image: atlas-marriages:latest
    ports:
      - "8080:8080"
    environment:
      - LOG_LEVEL=Info
      - REST_PORT=8080
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=atlas_marriages
      - DB_USER=atlas_user
      - DB_PASSWORD=atlas_password
      - DB_SSL_MODE=disable
      - KAFKA_BROKERS=kafka:9092
      - COMMAND_TOPIC_MARRIAGE=command.marriage
      - EVENT_TOPIC_MARRIAGE_STATUS=event.marriage.status
      - JAEGER_HOST=jaeger:14268
      - JAEGER_SERVICE_NAME=atlas-marriages
    depends_on:
      - postgres
      - kafka
      - jaeger

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=atlas_marriages
      - POSTGRES_USER=atlas_user
      - POSTGRES_PASSWORD=atlas_password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  kafka:
    image: confluentinc/cp-kafka:latest
    environment:
      - KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181
      - KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092
      - KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1
    depends_on:
      - zookeeper

  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      - ZOOKEEPER_CLIENT_PORT=2181

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
    environment:
      - COLLECTOR_OTLP_ENABLED=true

volumes:
  postgres_data:
```

#### Building the Docker Image

```bash
# Build the production image
docker build -t atlas-marriages:latest .

# Or build the debug image for development
docker build -f Dockerfile.debug -t atlas-marriages:debug .
```

### Kubernetes Deployment

#### ConfigMap for Environment Variables

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: atlas-marriages-config
data:
  LOG_LEVEL: "Info"
  REST_PORT: "8080"
  DB_HOST: "postgres-service"
  DB_PORT: "5432"
  DB_NAME: "atlas_marriages"
  DB_SSL_MODE: "disable"
  KAFKA_BROKERS: "kafka-service:9092"
  COMMAND_TOPIC_MARRIAGE: "command.marriage"
  EVENT_TOPIC_MARRIAGE_STATUS: "event.marriage.status"
  JAEGER_HOST: "jaeger-service:14268"
  JAEGER_SERVICE_NAME: "atlas-marriages"
```

#### Deployment Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-marriages
spec:
  replicas: 3
  selector:
    matchLabels:
      app: atlas-marriages
  template:
    metadata:
      labels:
        app: atlas-marriages
    spec:
      containers:
      - name: atlas-marriages
        image: atlas-marriages:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: atlas-marriages-config
        - secretRef:
            name: atlas-marriages-secrets
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: atlas-marriages-service
spec:
  selector:
    app: atlas-marriages
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

#### Secret for Sensitive Data

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: atlas-marriages-secrets
type: Opaque
data:
  DB_USER: YXRsYXNfdXNlcg==  # base64 encoded
  DB_PASSWORD: YXRsYXNfcGFzc3dvcmQ=  # base64 encoded
```

### Health Checks

The service exposes health check endpoints:

- **Liveness**: `GET /health` - Returns 200 if service is alive
- **Readiness**: `GET /ready` - Returns 200 if service is ready to accept requests

### Monitoring and Observability

#### Metrics

The service exposes metrics in Prometheus format at `/metrics`:

- `marriage_proposals_total` - Counter of total proposals created
- `marriage_ceremonies_total` - Counter of total ceremonies completed
- `marriage_divorces_total` - Counter of total divorces processed
- `proposal_cooldowns_active` - Gauge of active proposal cooldowns
- `ceremony_timeouts_total` - Counter of ceremony timeouts

#### Logging

Structured logging is configured via the `LOG_LEVEL` environment variable:

- `Panic` - Only panic messages
- `Fatal` - Fatal errors that cause service termination
- `Error` - Error conditions
- `Warn` - Warning messages
- `Info` - General information (recommended for production)
- `Debug` - Debug information
- `Trace` - Detailed tracing information

#### Distributed Tracing

The service integrates with Jaeger for distributed tracing:

- Configure `JAEGER_HOST` to point to your Jaeger collector
- Set `JAEGER_SERVICE_NAME` to identify the service in traces
- All incoming requests and outgoing Kafka messages are traced

### Scaling Considerations

#### Horizontal Scaling

- The service is stateless and can be scaled horizontally
- Use multiple replicas behind a load balancer
- Ensure database connection pooling is configured appropriately

#### Database Scaling

- Use read replicas for read-heavy workloads
- Consider partitioning large tables by tenant or time
- Monitor database connection pool usage

#### Kafka Scaling

- Increase topic partitions for higher throughput
- Use appropriate consumer group sizing
- Monitor consumer lag and processing times

### Security

#### Database Security

- Use SSL/TLS for database connections in production
- Implement database connection encryption
- Use secrets management for database credentials

#### Network Security

- Deploy behind a reverse proxy or API gateway
- Use TLS for all HTTP communications
- Implement proper firewall rules

#### Authentication and Authorization

- The service relies on tenant context from headers
- Implement proper authentication at the API gateway level
- Use RBAC for service-to-service communication

### Troubleshooting

#### Common Issues

1. **Database Connection Errors**
   - Check database connectivity and credentials
   - Verify database is running and accessible
   - Check connection pool settings

2. **Kafka Connection Issues**
   - Verify Kafka broker accessibility
   - Check topic existence and permissions
   - Monitor consumer group lag

3. **High Memory Usage**
   - Check for memory leaks in application logs
   - Monitor garbage collection metrics
   - Adjust resource limits if needed

4. **Slow Response Times**
   - Check database query performance
   - Monitor Kafka producer/consumer performance
   - Verify adequate resource allocation

#### Debugging

Enable debug logging for troubleshooting:

```bash
LOG_LEVEL=Debug
```

Check service logs for detailed information about:
- Database operations
- Kafka message processing
- HTTP request handling
- Business logic execution

## API Documentation

### Headers

All RESTful requests require the following header information to identify the server instance:

```
Content-Type: application/json
TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123
REGION: GMS
MAJOR_VERSION: 83
MINOR_VERSION: 1
```

## REST API Endpoints

### GET /api/characters/{characterId}/marriage

Returns the current marriage information for a character.

**Parameters:**
- `characterId` (path, required): The character ID to query

**Response (200 OK):**
```json
{
  "data": {
    "id": "12345",
    "type": "marriage",
    "attributes": {
      "id": 12345,
      "characterId1": 1001,
      "characterId2": 1002,
      "status": "married",
      "proposedAt": "2023-07-15T10:30:00Z",
      "engagedAt": "2023-07-15T11:45:00Z",
      "marriedAt": "2023-07-16T14:20:00Z",
      "createdAt": "2023-07-15T10:30:00Z",
      "updatedAt": "2023-07-16T14:20:00Z",
      "partner": {
        "characterId": 1002
      },
      "ceremony": {
        "id": 5678,
        "status": "completed",
        "scheduledAt": "2023-07-16T14:00:00Z",
        "startedAt": "2023-07-16T14:00:00Z",
        "completedAt": "2023-07-16T14:20:00Z",
        "inviteeCount": 8
      }
    }
  }
}
```

**Response (404 Not Found):**
```json
{
  "error": {
    "status": 404,
    "title": "Not Found",
    "detail": "Character is not married"
  }
}
```

**Marriage Status Values:**
- `proposed` - Proposal has been sent but not yet accepted
- `engaged` - Proposal accepted, ceremony not yet completed
- `married` - Ceremony completed, marriage is active
- `divorced` - Marriage has been ended by divorce

### GET /api/characters/{characterId}/marriage/history

Returns the complete marriage history for a character, including past marriages and proposals.

**Parameters:**
- `characterId` (path, required): The character ID to query

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "12345",
      "type": "marriage",
      "attributes": {
        "id": 12345,
        "characterId1": 1001,
        "characterId2": 1002,
        "status": "divorced",
        "proposedAt": "2023-07-15T10:30:00Z",
        "engagedAt": "2023-07-15T11:45:00Z",
        "marriedAt": "2023-07-16T14:20:00Z",
        "divorcedAt": "2023-08-01T09:15:00Z",
        "createdAt": "2023-07-15T10:30:00Z",
        "updatedAt": "2023-08-01T09:15:00Z"
      }
    }
  ]
}
```

**Response (200 OK - Empty History):**
```json
{
  "data": []
}
```

### GET /api/characters/{characterId}/marriage/proposals

Returns all pending proposals for a character (both sent and received).

**Parameters:**
- `characterId` (path, required): The character ID to query

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "9876",
      "type": "proposal",
      "attributes": {
        "id": 9876,
        "proposerId": 1001,
        "targetId": 1003,
        "status": "pending",
        "proposedAt": "2023-07-16T08:30:00Z",
        "expiresAt": "2023-07-17T08:30:00Z",
        "rejectionCount": 0,
        "createdAt": "2023-07-16T08:30:00Z",
        "updatedAt": "2023-07-16T08:30:00Z"
      }
    }
  ]
}
```

**Proposal Status Values:**
- `pending` - Proposal is active and awaiting response
- `accepted` - Proposal has been accepted (leads to engagement)
- `declined` - Proposal has been declined by the target
- `expired` - Proposal has expired (24 hours without response)
- `cancelled` - Proposal has been cancelled by the proposer

### Error Responses

All endpoints may return the following error responses:

**400 Bad Request:**
```json
{
  "error": {
    "status": 400,
    "title": "Bad Request",
    "detail": "Invalid character ID"
  }
}
```

**500 Internal Server Error:**
```json
{
  "error": {
    "status": 500,
    "title": "Internal Server Error",
    "detail": "An unexpected error occurred"
  }
}
```

## Kafka Events & Commands

### Command Topics

Commands are sent to the `COMMAND_TOPIC_MARRIAGE` topic with the following structure:

```json
{
  "characterId": 1001,
  "type": "COMMAND_TYPE",
  "body": {
    // Command-specific payload
  }
}
```

### Available Commands

#### Marriage Proposal Commands

**PROPOSE** - Initiate a marriage proposal
```json
{
  "characterId": 1001,
  "type": "PROPOSE",
  "body": {
    "targetCharacterId": 1002
  }
}
```

**ACCEPT** - Accept a marriage proposal
```json
{
  "characterId": 1002,
  "type": "ACCEPT",
  "body": {
    "proposalId": 9876
  }
}
```

**DECLINE** - Decline a marriage proposal
```json
{
  "characterId": 1002,
  "type": "DECLINE",
  "body": {
    "proposalId": 9876
  }
}
```

**CANCEL** - Cancel a sent proposal
```json
{
  "characterId": 1001,
  "type": "CANCEL",
  "body": {
    "proposalId": 9876
  }
}
```

#### Marriage Commands

**DIVORCE** - Initiate divorce proceedings
```json
{
  "characterId": 1001,
  "type": "DIVORCE",
  "body": {
    "marriageId": 12345
  }
}
```

#### Ceremony Commands

**SCHEDULE_CEREMONY** - Schedule a wedding ceremony
```json
{
  "characterId": 1001,
  "type": "SCHEDULE_CEREMONY",
  "body": {
    "marriageId": 12345,
    "scheduledAt": "2023-07-16T14:00:00Z",
    "invitees": [1003, 1004, 1005]
  }
}
```

**START_CEREMONY** - Begin a scheduled ceremony
```json
{
  "characterId": 1001,
  "type": "START_CEREMONY",
  "body": {
    "ceremonyId": 5678
  }
}
```

**COMPLETE_CEREMONY** - Complete a ceremony
```json
{
  "characterId": 1001,
  "type": "COMPLETE_CEREMONY",
  "body": {
    "ceremonyId": 5678
  }
}
```

**CANCEL_CEREMONY** - Cancel a ceremony
```json
{
  "characterId": 1001,
  "type": "CANCEL_CEREMONY",
  "body": {
    "ceremonyId": 5678
  }
}
```

**POSTPONE_CEREMONY** - Postpone a ceremony
```json
{
  "characterId": 1001,
  "type": "POSTPONE_CEREMONY",
  "body": {
    "ceremonyId": 5678
  }
}
```

**RESCHEDULE_CEREMONY** - Reschedule a ceremony
```json
{
  "characterId": 1001,
  "type": "RESCHEDULE_CEREMONY",
  "body": {
    "ceremonyId": 5678,
    "scheduledAt": "2023-07-17T16:00:00Z"
  }
}
```

**ADD_INVITEE** - Add an invitee to a ceremony
```json
{
  "characterId": 1001,
  "type": "ADD_INVITEE",
  "body": {
    "ceremonyId": 5678,
    "characterId": 1006
  }
}
```

**REMOVE_INVITEE** - Remove an invitee from a ceremony
```json
{
  "characterId": 1001,
  "type": "REMOVE_INVITEE",
  "body": {
    "ceremonyId": 5678,
    "characterId": 1006
  }
}
```

### Event Topics

Events are published to the `EVENT_TOPIC_MARRIAGE_STATUS` topic with the following structure:

```json
{
  "characterId": 1001,
  "type": "EVENT_TYPE",
  "body": {
    // Event-specific payload
  }
}
```

### Available Events

#### Proposal Events

**PROPOSAL_CREATED** - A new proposal has been created
```json
{
  "characterId": 1001,
  "type": "PROPOSAL_CREATED",
  "body": {
    "proposalId": 9876,
    "proposerId": 1001,
    "targetCharacterId": 1002,
    "proposedAt": "2023-07-16T08:30:00Z",
    "expiresAt": "2023-07-17T08:30:00Z"
  }
}
```

**PROPOSAL_ACCEPTED** - A proposal has been accepted
```json
{
  "characterId": 1002,
  "type": "PROPOSAL_ACCEPTED",
  "body": {
    "proposalId": 9876,
    "proposerId": 1001,
    "targetCharacterId": 1002,
    "acceptedAt": "2023-07-16T09:15:00Z"
  }
}
```

**PROPOSAL_DECLINED** - A proposal has been declined
```json
{
  "characterId": 1002,
  "type": "PROPOSAL_DECLINED",
  "body": {
    "proposalId": 9876,
    "proposerId": 1001,
    "targetCharacterId": 1002,
    "declinedAt": "2023-07-16T09:15:00Z",
    "rejectionCount": 1,
    "cooldownUntil": "2023-07-17T09:15:00Z"
  }
}
```

**PROPOSAL_EXPIRED** - A proposal has expired
```json
{
  "characterId": 1001,
  "type": "PROPOSAL_EXPIRED",
  "body": {
    "proposalId": 9876,
    "proposerId": 1001,
    "targetCharacterId": 1002,
    "expiredAt": "2023-07-17T08:30:00Z"
  }
}
```

#### Marriage Events

**MARRIAGE_CREATED** - A new marriage has been created
```json
{
  "characterId": 1001,
  "type": "MARRIAGE_CREATED",
  "body": {
    "marriageId": 12345,
    "characterId1": 1001,
    "characterId2": 1002,
    "marriedAt": "2023-07-16T14:20:00Z"
  }
}
```

**MARRIAGE_DIVORCED** - A marriage has been divorced
```json
{
  "characterId": 1001,
  "type": "MARRIAGE_DIVORCED",
  "body": {
    "marriageId": 12345,
    "characterId1": 1001,
    "characterId2": 1002,
    "divorcedAt": "2023-08-01T09:15:00Z",
    "initiatedBy": 1001
  }
}
```

#### Ceremony Events

**CEREMONY_SCHEDULED** - A ceremony has been scheduled
```json
{
  "characterId": 1001,
  "type": "CEREMONY_SCHEDULED",
  "body": {
    "ceremonyId": 5678,
    "marriageId": 12345,
    "characterId1": 1001,
    "characterId2": 1002,
    "scheduledAt": "2023-07-16T14:00:00Z",
    "invitees": [1003, 1004, 1005]
  }
}
```

**CEREMONY_STARTED** - A ceremony has started
```json
{
  "characterId": 1001,
  "type": "CEREMONY_STARTED",
  "body": {
    "ceremonyId": 5678,
    "marriageId": 12345,
    "characterId1": 1001,
    "characterId2": 1002,
    "startedAt": "2023-07-16T14:00:00Z"
  }
}
```

**CEREMONY_COMPLETED** - A ceremony has been completed
```json
{
  "characterId": 1001,
  "type": "CEREMONY_COMPLETED",
  "body": {
    "ceremonyId": 5678,
    "marriageId": 12345,
    "characterId1": 1001,
    "characterId2": 1002,
    "completedAt": "2023-07-16T14:20:00Z"
  }
}
```

#### Error Events

**MARRIAGE_ERROR** - An error occurred during marriage operations
```json
{
  "characterId": 1001,
  "type": "MARRIAGE_ERROR",
  "body": {
    "errorType": "PROPOSAL_ERROR",
    "errorCode": "ALREADY_MARRIED",
    "message": "Character is already married",
    "characterId": 1001,
    "context": "Marriage proposal validation",
    "timestamp": "2023-07-16T08:30:00Z"
  }
}
```

### Error Codes

Common error codes that may be returned in error events:

- `ALREADY_MARRIED` - Character is already married
- `ALREADY_ENGAGED` - Character is already engaged
- `INSUFFICIENT_LEVEL` - Character does not meet level requirements
- `SELF_PROPOSAL` - Cannot propose to oneself
- `GLOBAL_COOLDOWN` - Global proposal cooldown is active
- `TARGET_COOLDOWN` - Target-specific cooldown is active
- `PROPOSAL_EXPIRED` - The proposal has expired
- `PROPOSAL_NOT_FOUND` - Proposal does not exist
- `MARRIAGE_NOT_FOUND` - Marriage does not exist
- `CEREMONY_NOT_FOUND` - Ceremony does not exist
- `INVALID_STATE` - Invalid state transition
- `INVITEE_LIMIT_EXCEEDED` - Too many invitees (max 15)
- `PARTNER_DISCONNECTED` - Partner has disconnected during ceremony
- `CEREMONY_TIMEOUT` - Ceremony timed out due to inactivity
- `TENANT_MISMATCH` - Characters are not in the same tenant

## Business Rules

### Eligibility Requirements

- Characters must be **level 10 or higher** to marry
- A character may only be in **one relationship** at a time
- Both characters must be in the **same tenant/world**
- **No gender restrictions** apply

### Proposal Constraints

- Cannot propose if currently married or engaged
- Cannot receive proposals if already engaged
- Proposals may be sent to offline characters
- Proposals expire after **24 hours**
- **Global cooldown**: 4 hours between any proposals by the same character
- **Per-target cooldown**: Starts at 24 hours, doubles on each successive rejection

### Ceremony Rules

- Must be scheduled after engagement
- Maximum of **15 invitees** allowed
- Ceremony is postponed if either partner is offline for **5+ minutes**
- Ceremony must be restarted from the beginning after postponement

### Divorce

- Either party may initiate divorce unilaterally
- Divorce cost enforcement is handled by external services
- Marriage is automatically ended if a character is deleted
