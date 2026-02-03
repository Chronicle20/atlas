# atlas-asset-expiration

A background service that monitors character sessions and checks for expired items across inventories, storage, and cash shop. When items are found to be expired, commands are emitted to downstream services to handle expiration.

The service tracks online sessions by consuming session status events. Expiration checks run immediately on login and periodically for all tracked sessions.

## External Dependencies

- **Kafka**: Consumes session status events; produces asset expire commands
- **atlas-inventory**: REST client for character inventory data
- **atlas-storage**: REST client for account storage data
- **atlas-cashshop**: REST client for cash shop inventory data
- **atlas-data**: REST client for item replacement information

## Runtime Configuration

| Environment Variable | Description |
|---------------------|-------------|
| `BOOTSTRAP_SERVERS` | Kafka broker addresses |
| `EVENT_TOPIC_SESSION_STATUS` | Topic for session status events |
| `COMMAND_TOPIC_ASSET_EXPIRE` | Topic for expire commands |
| `EXPIRATION_CHECK_INTERVAL_SECONDS` | Periodic check interval (default: 60) |
| `INVENTORY_BASE_URL` | Base URL for atlas-inventory |
| `STORAGE_BASE_URL` | Base URL for atlas-storage |
| `CASHSHOP_BASE_URL` | Base URL for atlas-cashshop |
| `DATA_BASE_URL` | Base URL for atlas-data |
| `JAEGER_HOST_PORT` | OpenTelemetry collector endpoint |
| `LOG_LEVEL` | Logging level |

## Documentation

- [Domain Logic](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
