# atlas-map-actions

Executes map entry scripts (`onFirstUserEnter`, `onUserEnter`) using a JSON-based rules engine. When a character enters a map, the service loads the matching script from the database, evaluates rules in order using first-match-wins semantics, and executes the matched rule's operations via saga orchestration.

The service also provides REST endpoints for managing map scripts (CRUD, query by name, and seed from JSON files).

## External Dependencies

- **PostgreSQL** — Persistent storage for map script definitions (JSONB)
- **Kafka** — Consumes map action commands and saga status events; produces saga commands and character status events
- **atlas-query-aggregator** — Validates character conditions (gender, job, level, quest status) via REST

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `DB_HOST` | PostgreSQL host |
| `DB_PORT` | PostgreSQL port |
| `DB_USER` | Database user |
| `DB_PASSWORD` | Database password |
| `DB_NAME` | Database name |
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers |
| `REST_PORT` | REST server port |
| `QUERY_AGGREGATOR_URL` | Base URL for atlas-query-aggregator |
| `JAEGER_HOST` | OpenTelemetry collector host |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
