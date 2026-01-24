# atlas-marriages

Character relationship management service for the Atlas ecosystem.

## Service Responsibility

The atlas-marriages service manages character relationships including proposals, engagements, marriages, ceremonies, and divorces. It handles the complete marriage lifecycle from initial proposals through ceremony completion and eventual divorce, enforcing eligibility requirements and cooldown periods.

## External Dependencies

- **PostgreSQL**: Primary data store for marriage, proposal, and ceremony records
- **Apache Kafka**: Message broker for command processing and event publication
- **Jaeger**: Distributed tracing for observability

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger collector host:port |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| DB_HOST | PostgreSQL host |
| DB_PORT | PostgreSQL port |
| DB_NAME | Database name |
| DB_USER | Database user |
| DB_PASSWORD | Database password |
| DB_SSL_MODE | SSL mode (disable/require) |
| KAFKA_BROKERS | Kafka broker addresses |
| COMMAND_TOPIC_MARRIAGE | Kafka topic for marriage commands |
| EVENT_TOPIC_MARRIAGE_STATUS | Kafka topic for marriage events |
| TENANT_ID | Tenant UUID |
| REGION | Region identifier |
| MAJOR_VERSION | Major version number |
| MINOR_VERSION | Minor version number |

## Documentation

- [Domain Model](docs/domain.md) - Core models, invariants, and processors
- [Kafka Integration](docs/kafka.md) - Topics, commands, and events
- [REST API](docs/rest.md) - HTTP endpoints
- [Storage](docs/storage.md) - Database schema
