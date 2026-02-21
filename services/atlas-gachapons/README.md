# atlas-gachapons

Gachapon management service for the Atlas platform.

The service manages gachapon machines, their item pools, and reward selection. Each gachapon has configurable tier weights (common, uncommon, rare) and a per-machine item pool that is merged with a global item pool at selection time. The service provides REST endpoints for CRUD operations on gachapons, items, and global items, as well as reward selection and seed data loading.

## External Dependencies

- PostgreSQL: Persistent storage for gachapons, items, and global items
- OpenTelemetry Collector: Distributed tracing via OTLP/gRPC

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry Collector gRPC endpoint |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | HTTP server port |
| DB_HOST | PostgreSQL host |
| DB_PORT | PostgreSQL port |
| DB_USER | PostgreSQL user |
| DB_PASSWORD | PostgreSQL password |
| DB_NAME | PostgreSQL database name |
| GACHAPONS_DATA_PATH | Override path for gachapons seed JSON file |
| GACHAPON_ITEMS_DATA_PATH | Override path for gachapon items seed JSON file |
| GLOBAL_ITEMS_DATA_PATH | Override path for global items seed JSON file |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
