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

The service provides JSON:API compliant endpoints for family management:

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/families/{characterId}/juniors` | Add junior to family |
| DELETE | `/api/families/links/{characterId}` | Break family link |
| GET | `/api/families/tree/{characterId}` | Get family tree |
| POST | `/api/families/reputation/activities` | Process reputation activity |
| POST | `/api/families/reputation/redeem` | Redeem reputation |
| GET | `/api/families/reputation/{characterId}` | Get reputation |
| GET | `/api/families/location/{characterId}` | Get member location |

### Required Headers

All requests require the following headers:
```
Content-Type: application/json
X-Tenant-Id: {tenant-id}
X-Transaction-Id: {transaction-id}
X-World-Id: {world-id}
```

### Authentication

The service integrates with the Atlas authentication system through:
- Multi-tenant headers for request isolation
- Transaction ID tracking for idempotency
- Audit trail for all operations

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

---

**Atlas Family Service** - Enabling collaborative gameplay through hierarchical family relationships.
