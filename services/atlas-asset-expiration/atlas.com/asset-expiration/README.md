# atlas-asset-expiration

A background service that monitors character sessions and checks for expired items across inventories, storage, and cash shop. When items are found to be expired, commands are emitted to downstream services to handle expiration.

The service tracks online sessions by consuming session status events. Expiration checks run immediately on login and periodically for all tracked sessions.

## External Dependencies

- **Kafka**: Consumes session status events; produces asset expire commands
- **atlas-inventory**: REST client for character inventory data
- **atlas-storage**: REST client for account storage data
- **atlas-cashshop**: REST client for cash shop inventory data
- **atlas-data**: REST client for item replacement information
- **OpenTelemetry**: Distributed tracing via OTLP/gRPC

## Runtime Configuration

| Environment Variable | Description |
|---------------------|-------------|
| `BOOTSTRAP_SERVERS` | Kafka broker addresses |
| `BASE_SERVICE_URL` | Base URL for REST service discovery |
| `EVENT_TOPIC_SESSION_STATUS` | Topic for session status events |
| `COMMAND_TOPIC_STORAGE` | Topic for storage expire commands |
| `COMMAND_TOPIC_CASH_SHOP` | Topic for cash shop expire commands |
| `COMMAND_TOPIC_COMPARTMENT` | Topic for inventory compartment expire commands |
| `EXPIRATION_CHECK_INTERVAL_SECONDS` | Periodic check interval in seconds (default: 60) |
| `TRACE_ENDPOINT` | OpenTelemetry OTLP/gRPC collector endpoint |
| `LOG_LEVEL` | Logging level |

## Documentation

- [Domain Logic](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST Interface](docs/rest.md)
- [Storage](docs/storage.md)
