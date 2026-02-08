# Atlas Effective Stats Service

Computes and maintains effective (temporary) character statistics in real-time by aggregating base stats with bonuses from equipment, buffs, and passive skills.

## Responsibility

This service provides computed effective stats for logged-in characters. It maintains an in-memory registry of character stat models that are lazily initialized on first query or session creation and kept up to date via Kafka events for equipment changes, buff application/expiry, and base stat modifications. The service does not persist data; state is rebuilt from external services on demand.

## External Dependencies

- **atlas-character**: Source of base character stats (STR, DEX, INT, LUK, MaxHP, MaxMP)
- **atlas-inventory**: Source of equipped item stats (flat asset fields in the equip compartment)
- **atlas-buffs**: Source of active buff stat changes
- **atlas-skills**: Source of character skill levels
- **atlas-data**: Source of skill effect data (passive skill bonuses)
- **Kafka cluster**: Event streaming for real-time updates and command publishing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `REST_PORT` | HTTP server port (default: 8080) |
| `BOOTSTRAP_SERVERS` | Kafka broker addresses |
| `CHARACTERS_SERVICE_URL` | atlas-character service URL |
| `INVENTORY_SERVICE_URL` | atlas-inventory service URL |
| `BUFFS_SERVICE_URL` | atlas-buffs service URL |
| `SKILLS_SERVICE_URL` | atlas-skills service URL |
| `DATA_SERVICE_URL` | atlas-data service URL |
| `JAEGER_HOST_PORT` | Jaeger tracing endpoint |
| `LOG_LEVEL` | Log level (default: info) |

## Documentation

- [Domain](docs/domain.md)
- [REST API](docs/rest.md)
- [Kafka Integration](docs/kafka.md)
- [Storage](docs/storage.md)
